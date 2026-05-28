package tray

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ankityadav/statping/internal/checker"
	"github.com/ankityadav/statping/internal/notifier"
	"github.com/ankityadav/statping/internal/storage"
	"github.com/getlantern/systray"
)

const slowThresholdMs = 1000

type TrayApp struct {
	db       *storage.Database
	notifier *notifier.Notifier
	checker  *checker.Checker
	cancel   context.CancelFunc

	mu        sync.Mutex
	mStatus   *systray.MenuItem
	mMonitors []*systray.MenuItem
	builtIDs  []uint
}

func New(db *storage.Database) *TrayApp {
	n := notifier.New()
	return &TrayApp{
		db:       db,
		notifier: n,
		checker:  checker.New(db, n),
	}
}

func (t *TrayApp) Run() {
	systray.Run(t.onReady, t.onExit)
}

func (t *TrayApp) onReady() {
	systray.SetIcon(greenIcon)
	systray.SetTitle("")
	systray.SetTooltip("Statping - starting…")

	t.mStatus = systray.AddMenuItem("● Starting…", "Current status")
	t.mStatus.Disable()

	systray.AddSeparator()

	mHeader := systray.AddMenuItem("── Monitors ──", "")
	mHeader.Disable()

	t.rebuildMenu()

	systray.AddSeparator()

	mRefresh := systray.AddMenuItem("↻ Refresh Now", "Check all monitors immediately")
	mSettings := systray.AddMenuItem("⚙ Settings...", "Open settings window")

	systray.AddSeparator()

	mQuit := systray.AddMenuItem("Quit Statping", "Stop monitoring and exit")

	// Start the shared checker; refresh the menu whenever a check completes.
	ctx, cancel := context.WithCancel(context.Background())
	t.cancel = cancel
	t.checker.SetOnUpdate(t.refresh)
	if err := t.checker.Start(ctx); err != nil {
		t.updateStatus("red", "Failed to start: "+err.Error())
	}
	t.refresh()

	go func() {
		for {
			select {
			case <-mRefresh.ClickedCh:
				go t.checker.CheckNow()
			case <-mSettings.ClickedCh:
				go t.openSettings()
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func (t *TrayApp) onExit() {
	if t.cancel != nil {
		t.cancel()
	}
}

func (t *TrayApp) openSettings() {
	settings := NewSettingsWindow(t.db, t.onSettingsChanged, t.checker.Online)
	settings.Show()
}

func (t *TrayApp) onSettingsChanged() {
	if err := t.checker.Reload(); err != nil {
		return
	}
	t.rebuildMenu()
	t.refresh()
}

// rebuildMenu rebuilds the per-monitor menu items. systray can't remove items,
// so we hide the old ones and add fresh ones. Only call this when the set of
// monitors changes, not on every status update.
func (t *TrayApp) rebuildMenu() {
	monitors, err := t.db.ListEnabledMonitors()
	if err != nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	for _, item := range t.mMonitors {
		item.Hide()
	}
	t.mMonitors = nil
	t.builtIDs = nil

	for _, mon := range monitors {
		item := systray.AddMenuItem(mon.Name, mon.URL)
		item.Disable()
		t.mMonitors = append(t.mMonitors, item)
		t.builtIDs = append(t.builtIDs, mon.ID)
	}
}

// refresh updates menu item titles and the overall status icon based on the
// current database state and connectivity. It does not add menu items.
func (t *TrayApp) refresh() {
	monitors, err := t.db.ListEnabledMonitors()
	if err != nil {
		return
	}

	// If the set of enabled monitors changed, rebuild the menu first.
	t.mu.Lock()
	changed := len(monitors) != len(t.builtIDs)
	if !changed {
		for i, mon := range monitors {
			if mon.ID != t.builtIDs[i] {
				changed = true
				break
			}
		}
	}
	t.mu.Unlock()
	if changed {
		t.rebuildMenu()
	}

	online := t.checker.Online()

	var downCount, slowCount, upCount int

	t.mu.Lock()
	for i, mon := range monitors {
		var label string
		switch mon.CurrentStatus {
		case "down":
			label = fmt.Sprintf("✗ %s (DOWN)", mon.Name)
			downCount++
		case "up":
			rt := t.latestResponseTime(mon.ID)
			if rt > slowThresholdMs {
				label = fmt.Sprintf("◐ %s (%dms)", mon.Name, rt)
				slowCount++
			} else if rt > 0 {
				label = fmt.Sprintf("✓ %s (%dms)", mon.Name, rt)
				upCount++
			} else {
				label = fmt.Sprintf("✓ %s", mon.Name)
				upCount++
			}
		default:
			label = fmt.Sprintf("○ %s (pending)", mon.Name)
		}
		if i < len(t.mMonitors) {
			t.mMonitors[i].SetTitle(label)
		}
	}
	t.mu.Unlock()

	switch {
	case !online:
		t.updateStatus("yellow", "No internet — monitoring paused")
	case len(monitors) == 0:
		t.updateStatus("green", "No monitors configured")
	case downCount > 0:
		t.updateStatus("red", fmt.Sprintf("%d down, %d up", downCount, upCount))
	case slowCount > 0:
		t.updateStatus("yellow", fmt.Sprintf("%d slow, %d up", slowCount, upCount))
	default:
		t.updateStatus("green", fmt.Sprintf("All %d monitors operational", upCount))
	}
}

func (t *TrayApp) latestResponseTime(monitorID uint) int64 {
	results, err := t.db.GetRecentCheckResults(monitorID, 1)
	if err != nil || len(results) == 0 {
		return 0
	}
	if !results[0].Success {
		return 0
	}
	return results[0].ResponseTime
}

func (t *TrayApp) updateStatus(status, message string) {
	switch status {
	case "green":
		systray.SetIcon(greenIcon)
		if t.mStatus != nil {
			t.mStatus.SetTitle("● " + message)
		}
	case "yellow":
		systray.SetIcon(yellowIcon)
		if t.mStatus != nil {
			t.mStatus.SetTitle("◐ " + message)
		}
	case "red":
		systray.SetIcon(redIcon)
		if t.mStatus != nil {
			t.mStatus.SetTitle("✗ " + message)
		}
	}
	systray.SetTooltip("Statping - " + message + "  (updated " + time.Now().Format("15:04:05") + ")")
}
