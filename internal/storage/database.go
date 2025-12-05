package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Database struct {
	db *gorm.DB
}

func New(dbPath string) (*Database, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.AutoMigrate(&Monitor{}, &CheckResult{}, &Incident{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return &Database{db: db}, nil
}

func (d *Database) GetDB() *gorm.DB {
	return d.db
}

func (d *Database) Close() error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (d *Database) CreateMonitor(m *Monitor) error {
	return d.db.Create(m).Error
}

func (d *Database) GetMonitor(id uint) (*Monitor, error) {
	var m Monitor
	err := d.db.First(&m, id).Error
	return &m, err
}

func (d *Database) GetMonitorByURL(url string) (*Monitor, error) {
	var m Monitor
	err := d.db.Where("url = ?", url).First(&m).Error
	return &m, err
}

func (d *Database) ListMonitors() ([]Monitor, error) {
	var monitors []Monitor
	err := d.db.Order("id asc").Find(&monitors).Error
	return monitors, err
}

func (d *Database) ListEnabledMonitors() ([]Monitor, error) {
	var monitors []Monitor
	err := d.db.Where("enabled = ?", true).Order("id asc").Find(&monitors).Error
	return monitors, err
}

func (d *Database) UpdateMonitor(m *Monitor) error {
	return d.db.Save(m).Error
}

func (d *Database) DeleteMonitor(id uint) error {
	d.db.Where("monitor_id = ?", id).Delete(&CheckResult{})
	d.db.Where("monitor_id = ?", id).Delete(&Incident{})
	return d.db.Delete(&Monitor{}, id).Error
}

func (d *Database) ToggleMonitor(id uint, enabled bool) error {
	return d.db.Model(&Monitor{}).Where("id = ?", id).Update("enabled", enabled).Error
}

func (d *Database) CreateCheckResult(cr *CheckResult) error {
	return d.db.Create(cr).Error
}

func (d *Database) GetRecentCheckResults(monitorID uint, limit int) ([]CheckResult, error) {
	var results []CheckResult
	err := d.db.Where("monitor_id = ?", monitorID).
		Order("created_at desc").
		Limit(limit).
		Find(&results).Error
	return results, err
}

func (d *Database) GetCheckResultStats(monitorID uint, since time.Time) (total, successful int64, avgResponseTime float64, err error) {
	err = d.db.Model(&CheckResult{}).
		Where("monitor_id = ? AND created_at >= ?", monitorID, since).
		Count(&total).Error
	if err != nil {
		return
	}

	err = d.db.Model(&CheckResult{}).
		Where("monitor_id = ? AND created_at >= ? AND success = ?", monitorID, since, true).
		Count(&successful).Error
	if err != nil {
		return
	}

	var avg struct{ Avg float64 }
	err = d.db.Model(&CheckResult{}).
		Select("AVG(response_time) as avg").
		Where("monitor_id = ? AND created_at >= ? AND success = ?", monitorID, since, true).
		Scan(&avg).Error
	avgResponseTime = avg.Avg

	return
}

func (d *Database) CreateIncident(i *Incident) error {
	return d.db.Create(i).Error
}

func (d *Database) GetActiveIncident(monitorID uint) (*Incident, error) {
	var i Incident
	err := d.db.Where("monitor_id = ? AND resolved_at IS NULL", monitorID).First(&i).Error
	if err != nil {
		return nil, err
	}
	return &i, nil
}

func (d *Database) ResolveIncident(id uint) error {
	now := time.Now()
	return d.db.Model(&Incident{}).Where("id = ?", id).Update("resolved_at", now).Error
}

func (d *Database) UpdateIncident(i *Incident) error {
	return d.db.Save(i).Error
}

func (d *Database) GetRecentIncidents(monitorID uint, limit int) ([]Incident, error) {
	var incidents []Incident
	err := d.db.Where("monitor_id = ?", monitorID).
		Order("started_at desc").
		Limit(limit).
		Find(&incidents).Error
	return incidents, err
}

func (d *Database) GetAllRecentIncidents(limit int) ([]Incident, error) {
	var incidents []Incident
	err := d.db.Order("started_at desc").
		Limit(limit).
		Find(&incidents).Error
	return incidents, err
}

func ParseExpectedCodes(codes string) []int {
	if codes == "" {
		return []int{200}
	}

	parts := strings.Split(codes, ",")
	result := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		var code int
		fmt.Sscanf(p, "%d", &code)
		if code > 0 {
			result = append(result, code)
		}
	}

	if len(result) == 0 {
		return []int{200}
	}
	return result
}

func ParseKeywords(keywords string) []string {
	if keywords == "" {
		return nil
	}

	parts := strings.Split(keywords, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
