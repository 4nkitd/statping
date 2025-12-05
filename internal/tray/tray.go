package tray

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
	"github.com/getlantern/systray"
)

type TrayApp struct {
	db        *storage.Database
	notifier  *notifier.Notifier
	monitors  []storage.Monitor
	mu        sync.RWMutex
	stopChan  chan struct{}
	status    string
	mStatus   *systray.MenuItem
	mMonitors []*systray.MenuItem
}

func New(db *storage.Database) *TrayApp {
	return &TrayApp{
		db:       db,
		notifier: notifier.New(),
		stopChan: make(chan struct{}),
		status:   "green",
	}
}

func (t *TrayApp) Run() {
	systray.Run(t.onReady, t.onExit)
}

func (t *TrayApp) onReady() {
	systray.SetIcon(greenIcon)
	systray.SetTitle("")
	systray.SetTooltip("Statping - All systems operational")

	t.mStatus = systray.AddMenuItem("● All Systems Operational", "Current status")
	t.mStatus.Disable()

	systray.AddSeparator()

	mHeader := systray.AddMenuItem("── Monitors ──", "")
	mHeader.Disable()

	t.loadMonitors()

	systray.AddSeparator()

	mRefresh := systray.AddMenuItem("↻ Refresh Now", "Check all monitors immediately")

	systray.AddSeparator()

	mQuit := systray.AddMenuItem("Quit Statping", "Stop monitoring and exit")

	go t.runChecker()

	go func() {
		for {
			select {
			case <-mRefresh.ClickedCh:
				go t.checkAllMonitors()
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			case <-t.stopChan:
				return
			}
		}
	}()
}

func (t *TrayApp) onExit() {
	close(t.stopChan)
}

func (t *TrayApp) loadMonitors() {
	monitors, err := t.db.ListEnabledMonitors()
	if err != nil {
		return
	}

	t.mu.Lock()
	t.monitors = monitors

	for _, item := range t.mMonitors {
		item.Hide()
	}
	t.mMonitors = nil

	for _, mon := range monitors {
		statusIcon := "○"
		switch mon.CurrentStatus {
		case "up":
			statusIcon = "✓"
		case "down":
			statusIcon = "✗"
		}
		item := systray.AddMenuItem(fmt.Sprintf("%s %s", statusIcon, mon.Name), mon.URL)
		item.Disable()
		t.mMonitors = append(t.mMonitors, item)
	}
	t.mu.Unlock()
}

func (t *TrayApp) runChecker() {
	t.checkAllMonitors()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.checkAllMonitors()
		case <-t.stopChan:
			return
		}
	}
}

func (t *TrayApp) checkAllMonitors() {
	monitors, err := t.db.ListEnabledMonitors()
	if err != nil {
		return
	}

	t.mu.Lock()
	t.monitors = monitors
	t.mu.Unlock()

	if len(monitors) == 0 {
		t.updateStatus("green", "No monitors configured")
		return
	}

	var hasDown, hasSlow bool
	var downCount, slowCount, upCount int

	for i, mon := range monitors {
		statusCode, responseTime, checkErr := t.checkMonitor(&mon)

		now := time.Now()
		result := &storage.CheckResult{
			MonitorID:    mon.ID,
			StatusCode:   statusCode,
			ResponseTime: responseTime,
			Success:      checkErr == nil,
			CreatedAt:    now,
		}
		if checkErr != nil {
			result.ErrorMessage = checkErr.Error()
		}
		t.db.CreateCheckResult(result)

		t.mu.Lock()
		var label string
		if checkErr != nil {
			label = fmt.Sprintf("✗ %s (DOWN)", mon.Name)
			hasDown = true
			downCount++

			mon.ConsecutiveFails++
			if mon.ConsecutiveFails >= config.DefaultMaxFailures {
				wasUp := mon.CurrentStatus != "down"
				mon.CurrentStatus = "down"
				if wasUp {
					t.notifier.NotifyDown(mon.Name, mon.URL, checkErr.Error())
				}
			}
		} else if responseTime > 1000 {
			label = fmt.Sprintf("◐ %s (%dms)", mon.Name, responseTime)
			hasSlow = true
			slowCount++

			wasDown := mon.CurrentStatus == "down"
			mon.CurrentStatus = "up"
			mon.ConsecutiveFails = 0
			if wasDown {
				t.notifier.NotifyRecovery(mon.Name, mon.URL)
			}
		} else {
			label = fmt.Sprintf("✓ %s (%dms)", mon.Name, responseTime)
			upCount++

			wasDown := mon.CurrentStatus == "down"
			mon.CurrentStatus = "up"
			mon.ConsecutiveFails = 0
			if wasDown {
				t.notifier.NotifyRecovery(mon.Name, mon.URL)
			}
		}

		if i < len(t.mMonitors) {
			t.mMonitors[i].SetTitle(label)
		}
		t.mu.Unlock()

		mon.LastCheckAt = &now
		t.db.UpdateMonitor(&mon)
	}

	if hasDown {
		t.updateStatus("red", fmt.Sprintf("%d down, %d up", downCount, upCount))
	} else if hasSlow {
		t.updateStatus("yellow", fmt.Sprintf("%d slow, %d up", slowCount, upCount))
	} else {
		t.updateStatus("green", fmt.Sprintf("All %d monitors operational", upCount))
	}
}

func (t *TrayApp) checkMonitor(mon *storage.Monitor) (int, int64, error) {
	timeout := time.Duration(mon.Timeout) * time.Second
	if timeout == 0 {
		timeout = time.Duration(config.DefaultTimeout) * time.Second
	}

	client := &http.Client{Timeout: timeout}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", mon.URL, nil)
	if err != nil {
		return 0, 0, err
	}
	req.Header.Set("User-Agent", "Statping/1.0")

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	responseTime := time.Since(start).Milliseconds()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, responseTime, fmt.Errorf("failed to read body: %w", err)
	}

	expectedCodes := storage.ParseExpectedCodes(mon.ExpectedCodes)
	statusOK := false
	for _, code := range expectedCodes {
		if resp.StatusCode == code {
			statusOK = true
			break
		}
	}

	if !statusOK {
		return resp.StatusCode, responseTime, fmt.Errorf("status %d", resp.StatusCode)
	}

	keywords := storage.ParseKeywords(mon.Keywords)
	if len(keywords) > 0 {
		bodyStr := string(body)
		for _, keyword := range keywords {
			pattern := "(?i)" + regexp.QuoteMeta(keyword)
			matched, _ := regexp.MatchString(pattern, bodyStr)
			if !matched {
				return resp.StatusCode, responseTime, fmt.Errorf("keyword '%s' not found", keyword)
			}
		}
	}

	return resp.StatusCode, responseTime, nil
}

func (t *TrayApp) updateStatus(status, message string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.status = status

	switch status {
	case "green":
		systray.SetIcon(greenIcon)
		systray.SetTooltip("Statping - " + message)
		if t.mStatus != nil {
			t.mStatus.SetTitle("● " + message)
		}
	case "yellow":
		systray.SetIcon(yellowIcon)
		systray.SetTooltip("Statping - " + message)
		if t.mStatus != nil {
			t.mStatus.SetTitle("◐ " + message)
		}
	case "red":
		systray.SetIcon(redIcon)
		systray.SetTooltip("Statping - " + message)
		if t.mStatus != nil {
			t.mStatus.SetTitle("✗ " + message)
		}
	}
}
