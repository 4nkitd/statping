package notifier

import (
	"fmt"
	"log"

	"github.com/gen2brain/beeep"
)

type Notifier struct {
	enabled bool
}

func New() *Notifier {
	return &Notifier{
		enabled: true,
	}
}

func (n *Notifier) NotifyDown(name, url, errorMsg string) {
	if !n.enabled {
		return
	}

	title := fmt.Sprintf("🔴 %s is DOWN", name)
	message := fmt.Sprintf("URL: %s\nError: %s", url, errorMsg)

	if err := beeep.Alert(title, message, ""); err != nil {
		log.Printf("Failed to send notification: %v", err)
	}
}

func (n *Notifier) NotifyRecovery(name, url string) {
	if !n.enabled {
		return
	}

	title := fmt.Sprintf("✅ %s is UP", name)
	message := fmt.Sprintf("URL: %s has recovered", url)

	if err := beeep.Notify(title, message, ""); err != nil {
		log.Printf("Failed to send notification: %v", err)
	}
}

func (n *Notifier) NotifySystemOffline() {
	if !n.enabled {
		return
	}

	if err := beeep.Notify("🌐 No internet connection", "Monitoring paused — your system appears to be offline.", ""); err != nil {
		log.Printf("Failed to send notification: %v", err)
	}
}

func (n *Notifier) NotifySystemOnline() {
	if !n.enabled {
		return
	}

	if err := beeep.Notify("🌐 Internet restored", "Connection is back — monitoring resumed.", ""); err != nil {
		log.Printf("Failed to send notification: %v", err)
	}
}

func (n *Notifier) SetEnabled(enabled bool) {
	n.enabled = enabled
}
