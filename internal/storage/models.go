package storage

import (
	"time"
)

type Monitor struct {
	ID               uint          `gorm:"primarykey" json:"id"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
	Name             string        `gorm:"not null" json:"name"`
	URL              string        `gorm:"not null;uniqueIndex" json:"url"`
	Enabled          bool          `gorm:"default:true" json:"enabled"`
	CheckInterval    int           `gorm:"default:60" json:"check_interval"`
	ExpectedCodes    string        `json:"expected_codes"`
	Keywords         string        `json:"keywords"`
	Timeout          int           `gorm:"default:10" json:"timeout"`
	CurrentStatus    string        `gorm:"default:unknown" json:"current_status"`
	ConsecutiveFails int           `json:"consecutive_fails"`
	LastCheckAt      *time.Time    `json:"last_check_at"`
	CheckResults     []CheckResult `gorm:"foreignKey:MonitorID" json:"-"`
	Incidents        []Incident    `gorm:"foreignKey:MonitorID" json:"-"`
}

type CheckResult struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	MonitorID    uint      `gorm:"index;not null" json:"monitor_id"`
	StatusCode   int       `json:"status_code"`
	ResponseTime int64     `json:"response_time"`
	Success      bool      `json:"success"`
	ErrorMessage string    `json:"error_message"`
}

type Incident struct {
	ID               uint       `gorm:"primarykey" json:"id"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	MonitorID        uint       `gorm:"index;not null" json:"monitor_id"`
	StartedAt        time.Time  `json:"started_at"`
	ResolvedAt       *time.Time `json:"resolved_at"`
	ErrorMessage     string     `json:"error_message"`
	Notified         bool       `gorm:"default:false" json:"notified"`
	RecoveryNotified bool       `gorm:"default:false" json:"recovery_notified"`
}

func (i *Incident) IsResolved() bool {
	return i.ResolvedAt != nil
}

func (i *Incident) Duration() time.Duration {
	if i.ResolvedAt != nil {
		return i.ResolvedAt.Sub(i.StartedAt)
	}
	return time.Since(i.StartedAt)
}
