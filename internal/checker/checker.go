package checker

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/ankityadav/statping/internal/config"
	"github.com/ankityadav/statping/internal/notifier"
	"github.com/ankityadav/statping/internal/storage"
)

func logIfErr(action string, err error) {
	if err != nil {
		log.Printf("checker: %s failed: %v", action, err)
	}
}

type Checker struct {
	db       *storage.Database
	notifier *notifier.Notifier
	client   *http.Client
	stopChan chan struct{}
	wg       sync.WaitGroup
	mu       sync.RWMutex
	monitors map[uint]*monitorState

	onlineMu sync.RWMutex
	online   bool

	onUpdate func()
}

type monitorState struct {
	monitor      *storage.Monitor
	ticker       *time.Ticker
	stopChan     chan struct{}
	trigger      chan struct{}
	lastNotified time.Time
}

func New(db *storage.Database, n *notifier.Notifier) *Checker {
	return &Checker{
		db:       db,
		notifier: n,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		stopChan: make(chan struct{}),
		monitors: make(map[uint]*monitorState),
		online:   true,
	}
}

// SetOnUpdate registers a callback invoked after each check completes. Useful
// for UIs (e.g. the tray) that need to refresh when statuses change.
func (c *Checker) SetOnUpdate(fn func()) {
	c.onUpdate = fn
}

func (c *Checker) notifyUpdate() {
	if c.onUpdate != nil {
		c.onUpdate()
	}
}

// Online reports whether the checker currently considers the system to have
// internet connectivity.
func (c *Checker) Online() bool {
	c.onlineMu.RLock()
	defer c.onlineMu.RUnlock()
	return c.online
}

// setOnline updates connectivity state and fires notifications on transitions.
func (c *Checker) setOnline(v bool) {
	c.onlineMu.Lock()
	changed := c.online != v
	c.online = v
	c.onlineMu.Unlock()

	if !changed {
		return
	}

	if v {
		log.Printf("checker: internet connectivity restored")
		c.notifier.NotifySystemOnline()
	} else {
		log.Printf("checker: internet connectivity lost — monitoring paused")
		c.notifier.NotifySystemOffline()
	}
	c.notifyUpdate()
}

func (c *Checker) runConnectivityWatch() {
	defer c.wg.Done()

	interval := time.Duration(config.ConnectivityCheckInterval) * time.Second
	timeout := time.Duration(config.ConnectivityTimeout) * time.Second

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.setOnline(IsOnline(timeout))
		case <-c.stopChan:
			return
		}
	}
}

func (c *Checker) Start(ctx context.Context) error {
	monitors, err := c.db.ListEnabledMonitors()
	if err != nil {
		return fmt.Errorf("failed to load monitors: %w", err)
	}

	for _, m := range monitors {
		monitor := m
		c.startMonitor(&monitor)
	}

	c.wg.Add(1)
	go c.runPruner()

	c.wg.Add(1)
	go c.runConnectivityWatch()

	go func() {
		<-ctx.Done()
		c.Stop()
	}()

	return nil
}

func (c *Checker) runPruner() {
	defer c.wg.Done()

	c.prune()

	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.prune()
		case <-c.stopChan:
			return
		}
	}
}

func (c *Checker) prune() {
	before := time.Now().AddDate(0, 0, -config.CheckResultRetentionDays)
	deleted, err := c.db.PruneCheckResults(before)
	if err != nil {
		logIfErr("prune check results", err)
		return
	}
	if deleted > 0 {
		log.Printf("checker: pruned %d check results older than %d days", deleted, config.CheckResultRetentionDays)
	}
}

func (c *Checker) Stop() {
	close(c.stopChan)

	c.mu.Lock()
	for _, ms := range c.monitors {
		if ms.ticker != nil {
			ms.ticker.Stop()
		}
		close(ms.stopChan)
	}
	c.mu.Unlock()

	c.wg.Wait()
}

func (c *Checker) startMonitor(m *storage.Monitor) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ms, exists := c.monitors[m.ID]; exists {
		if ms.ticker != nil {
			ms.ticker.Stop()
		}
		close(ms.stopChan)
	}

	interval := time.Duration(m.CheckInterval) * time.Second
	if interval < time.Second {
		interval = time.Duration(config.DefaultCheckInterval) * time.Second
	}

	ms := &monitorState{
		monitor:  m,
		ticker:   time.NewTicker(interval),
		stopChan: make(chan struct{}),
		trigger:  make(chan struct{}, 1),
	}
	c.monitors[m.ID] = ms

	c.wg.Add(1)
	go c.runMonitor(ms)
}

func (c *Checker) runMonitor(ms *monitorState) {
	defer c.wg.Done()

	c.performCheck(ms.monitor)

	for {
		select {
		case <-ms.ticker.C:
			c.performCheck(ms.monitor)
		case <-ms.trigger:
			c.performCheck(ms.monitor)
		case <-ms.stopChan:
			return
		case <-c.stopChan:
			return
		}
	}
}

// CheckNow triggers an immediate check of every active monitor. Checks run in
// their own goroutines so this returns promptly.
func (c *Checker) CheckNow() {
	c.mu.RLock()
	triggers := make([]chan struct{}, 0, len(c.monitors))
	for _, ms := range c.monitors {
		triggers = append(triggers, ms.trigger)
	}
	c.mu.RUnlock()

	for _, tr := range triggers {
		select {
		case tr <- struct{}{}:
		default:
		}
	}
}

// Reload re-syncs the running monitors with the database: it stops monitors
// that were removed or disabled and (re)starts the ones that are enabled.
func (c *Checker) Reload() error {
	monitors, err := c.db.ListMonitors()
	if err != nil {
		return err
	}

	enabled := make(map[uint]bool)
	for _, m := range monitors {
		if m.Enabled {
			enabled[m.ID] = true
		}
	}

	c.mu.RLock()
	existing := make([]uint, 0, len(c.monitors))
	for id := range c.monitors {
		existing = append(existing, id)
	}
	c.mu.RUnlock()

	for _, id := range existing {
		if !enabled[id] {
			c.RemoveMonitor(id)
		}
	}

	for _, m := range monitors {
		if !m.Enabled {
			continue
		}
		monitor := m
		c.startMonitor(&monitor)
	}

	return nil
}

// Probe performs a single HTTP check against a monitor and returns the result
// without persisting anything. It is safe to call from anywhere.
func Probe(client *http.Client, m *storage.Monitor) storage.CheckResult {
	startTime := time.Now()

	result := storage.CheckResult{
		MonitorID: m.ID,
		CreatedAt: startTime,
	}

	timeout := time.Duration(m.Timeout) * time.Second
	if timeout == 0 {
		timeout = time.Duration(config.DefaultTimeout) * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", m.URL, nil)
	if err != nil {
		result.ErrorMessage = err.Error()
		return result
	}

	req.Header.Set("User-Agent", "Statping/1.0")

	resp, err := client.Do(req)
	if err != nil {
		result.ErrorMessage = err.Error()
		return result
	}
	defer resp.Body.Close()

	result.ResponseTime = time.Since(startTime).Milliseconds()
	result.StatusCode = resp.StatusCode

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to read response body: %v", err)
		return result
	}

	expectedCodes := storage.ParseExpectedCodes(m.ExpectedCodes)
	statusOK := false
	for _, code := range expectedCodes {
		if resp.StatusCode == code {
			statusOK = true
			break
		}
	}

	if !statusOK {
		result.ErrorMessage = fmt.Sprintf("unexpected status code: got %d, expected one of %v", resp.StatusCode, expectedCodes)
		return result
	}

	keywords := storage.ParseKeywords(m.Keywords)
	if len(keywords) > 0 {
		bodyStr := string(body)
		for _, keyword := range keywords {
			pattern := "(?i)" + regexp.QuoteMeta(keyword)
			matched, err := regexp.MatchString(pattern, bodyStr)
			if err != nil || !matched {
				result.ErrorMessage = fmt.Sprintf("keyword '%s' not found in response", keyword)
				return result
			}
		}
	}

	result.Success = true
	return result
}

func (c *Checker) performCheck(m *storage.Monitor) {
	result := Probe(c.client, m)
	if result.Success {
		c.recordSuccess(m, &result)
	} else {
		c.recordFailure(m, &result)
	}
	c.notifyUpdate()
}

func (c *Checker) recordSuccess(m *storage.Monitor, result *storage.CheckResult) {
	now := result.CreatedAt

	// A successful HTTP response proves the system is online.
	c.setOnline(true)

	logIfErr("create check result", c.db.CreateCheckResult(result))

	wasDown := m.CurrentStatus == "down"
	m.CurrentStatus = "up"
	m.ConsecutiveFails = 0
	m.LastCheckAt = &now
	logIfErr("update monitor", c.db.UpdateMonitor(m))

	if wasDown {
		incident, err := c.db.GetActiveIncident(m.ID)
		if err == nil && incident != nil {
			logIfErr("resolve incident", c.db.ResolveIncident(incident.ID))

			if !incident.RecoveryNotified {
				c.notifier.NotifyRecovery(m.Name, m.URL)
				incident.RecoveryNotified = true
				logIfErr("update incident", c.db.UpdateIncident(incident))
			}
		}
	}
}

func (c *Checker) recordFailure(m *storage.Monitor, result *storage.CheckResult) {
	// Don't blame the site if the whole system is offline. If we already know
	// we're offline, skip immediately; otherwise confirm with an on-demand probe
	// so a connectivity drop is detected faster than the periodic watcher.
	if !c.Online() {
		return
	}
	if !IsOnline(time.Duration(config.ConnectivityTimeout) * time.Second) {
		c.setOnline(false)
		return
	}

	now := result.CreatedAt
	errorMsg := result.ErrorMessage

	logIfErr("create check result", c.db.CreateCheckResult(result))

	m.ConsecutiveFails++
	m.LastCheckAt = &now

	maxFailures := m.MaxFailures
	if maxFailures < 1 {
		maxFailures = config.DefaultMaxFailures
	}

	if m.ConsecutiveFails >= maxFailures {
		wasUp := m.CurrentStatus != "down"
		m.CurrentStatus = "down"

		if wasUp {
			incident := &storage.Incident{
				MonitorID:    m.ID,
				StartedAt:    now,
				ErrorMessage: errorMsg,
			}
			logIfErr("create incident", c.db.CreateIncident(incident))

			c.mu.Lock()
			ms := c.monitors[m.ID]
			if ms != nil {
				if time.Since(ms.lastNotified).Seconds() >= config.NotificationCooldown {
					c.notifier.NotifyDown(m.Name, m.URL, errorMsg)
					ms.lastNotified = now
				}
			}
			c.mu.Unlock()
		} else {
			incident, err := c.db.GetActiveIncident(m.ID)
			if err == nil && incident != nil {
				incident.ErrorMessage = errorMsg
				logIfErr("update incident", c.db.UpdateIncident(incident))

				c.mu.Lock()
				ms := c.monitors[m.ID]
				if ms != nil && time.Since(ms.lastNotified).Seconds() >= config.NotificationCooldown {
					c.notifier.NotifyDown(m.Name, m.URL, errorMsg)
					ms.lastNotified = now
				}
				c.mu.Unlock()
			}
		}
	}

	logIfErr("update monitor", c.db.UpdateMonitor(m))
}

func (c *Checker) AddMonitor(m *storage.Monitor) {
	if m.Enabled {
		c.startMonitor(m)
	}
}

func (c *Checker) RemoveMonitor(id uint) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ms, exists := c.monitors[id]; exists {
		if ms.ticker != nil {
			ms.ticker.Stop()
		}
		close(ms.stopChan)
		delete(c.monitors, id)
	}
}

func (c *Checker) UpdateMonitor(m *storage.Monitor) {
	c.RemoveMonitor(m.ID)
	if m.Enabled {
		c.startMonitor(m)
	}
}

func (c *Checker) GetStatus() map[uint]string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	status := make(map[uint]string)
	for id, ms := range c.monitors {
		status[id] = ms.monitor.CurrentStatus
	}
	return status
}
