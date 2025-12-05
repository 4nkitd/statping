package checker

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/ankityadav/statping/internal/config"
	"github.com/ankityadav/statping/internal/notifier"
	"github.com/ankityadav/statping/internal/storage"
)

type Checker struct {
	db       *storage.Database
	notifier *notifier.Notifier
	client   *http.Client
	stopChan chan struct{}
	wg       sync.WaitGroup
	mu       sync.RWMutex
	monitors map[uint]*monitorState
}

type monitorState struct {
	monitor      *storage.Monitor
	ticker       *time.Ticker
	stopChan     chan struct{}
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

	go func() {
		<-ctx.Done()
		c.Stop()
	}()

	return nil
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
		case <-ms.stopChan:
			return
		case <-c.stopChan:
			return
		}
	}
}

func (c *Checker) performCheck(m *storage.Monitor) {
	startTime := time.Now()

	timeout := time.Duration(m.Timeout) * time.Second
	if timeout == 0 {
		timeout = time.Duration(config.DefaultTimeout) * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", m.URL, nil)
	if err != nil {
		c.recordFailure(m, 0, err)
		return
	}

	req.Header.Set("User-Agent", "Statping/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		c.recordFailure(m, 0, err)
		return
	}
	defer resp.Body.Close()

	responseTime := time.Since(startTime).Milliseconds()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.recordFailure(m, resp.StatusCode, fmt.Errorf("failed to read response body: %w", err))
		return
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
		c.recordFailure(m, resp.StatusCode, fmt.Errorf("unexpected status code: got %d, expected one of %v", resp.StatusCode, expectedCodes))
		return
	}

	keywords := storage.ParseKeywords(m.Keywords)
	if len(keywords) > 0 {
		bodyStr := string(body)
		for _, keyword := range keywords {
			pattern := "(?i)" + regexp.QuoteMeta(keyword)
			matched, err := regexp.MatchString(pattern, bodyStr)
			if err != nil || !matched {
				c.recordFailure(m, resp.StatusCode, fmt.Errorf("keyword '%s' not found in response", keyword))
				return
			}
		}
	}

	c.recordSuccess(m, resp.StatusCode, responseTime)
}

func (c *Checker) recordSuccess(m *storage.Monitor, statusCode int, responseTime int64) {
	now := time.Now()

	result := &storage.CheckResult{
		MonitorID:    m.ID,
		StatusCode:   statusCode,
		ResponseTime: responseTime,
		Success:      true,
		CreatedAt:    now,
	}
	c.db.CreateCheckResult(result)

	wasDown := m.CurrentStatus == "down"
	m.CurrentStatus = "up"
	m.ConsecutiveFails = 0
	m.LastCheckAt = &now
	c.db.UpdateMonitor(m)

	if wasDown {
		incident, err := c.db.GetActiveIncident(m.ID)
		if err == nil && incident != nil {
			c.db.ResolveIncident(incident.ID)

			if !incident.RecoveryNotified {
				c.notifier.NotifyRecovery(m.Name, m.URL)
				incident.RecoveryNotified = true
				c.db.UpdateIncident(incident)
			}
		}
	}
}

func (c *Checker) recordFailure(m *storage.Monitor, statusCode int, err error) {
	now := time.Now()

	errorMsg := err.Error()

	result := &storage.CheckResult{
		MonitorID:    m.ID,
		StatusCode:   statusCode,
		ResponseTime: 0,
		Success:      false,
		ErrorMessage: errorMsg,
		CreatedAt:    now,
	}
	c.db.CreateCheckResult(result)

	m.ConsecutiveFails++
	m.LastCheckAt = &now

	if m.ConsecutiveFails >= config.DefaultMaxFailures {
		wasUp := m.CurrentStatus != "down"
		m.CurrentStatus = "down"

		if wasUp {
			incident := &storage.Incident{
				MonitorID:    m.ID,
				StartedAt:    now,
				ErrorMessage: errorMsg,
			}
			c.db.CreateIncident(incident)

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
				c.db.UpdateIncident(incident)

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

	c.db.UpdateMonitor(m)
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
