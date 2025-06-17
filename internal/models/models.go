package models

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"regexp"
	"time"
)

type User struct {
	ID        int32      `db:"id"`
	Name      string     `db:"name"`
	Email     string     `db:"email"`
	Password  string     `db:"password"`
	CreatedAt *time.Time `db:"created_at"`
	UpdatedAt *time.Time `db:"updated_at"`
}

type Device struct {
	ID          int32            `db:"id"`
	Name        string           `db:"name"`
	DeviceType  string           `db:"device_type"`
	Address     string           `db:"address"`
	Responsible SqlJsonbIntArray `db:"responsible"`
	CreatedAt   *time.Time       `db:"created_at"`
	UpdatedAt   *time.Time       `db:"updated_at"`
}

type SqlJsonbIntArray []int32

func (arr SqlJsonbIntArray) Value() (driver.Value, error) {
	res, err := json.Marshal(arr)
	return res, err
}
func (arr *SqlJsonbIntArray) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		err := fmt.Errorf("SqlJsonbStringArray: item not created")
		return err
	}
	err := json.Unmarshal(b, &arr)
	return err
}

type Tag struct {
	ID            int32      `db:"id"`
	Name          string     `db:"name"`
	DeviceId      int32      `db:"device_id"`
	Regexp        string     `db:"regexp"`
	CompareType   string     `db:"compare_type"`
	Value         string     `db:"value"`
	ArrayIndex    int32      `db:"array_index"`
	Subject       string     `db:"subject"`
	SeverityLevel string     `db:"severity_level"`
	CreatedAt     *time.Time `db:"created_at"`
	UpdatedAt     *time.Time `db:"updated_at"`

	CompareFunc    func(string, string) bool `json:"-" db:"-"`
	CompiledRegexp *regexp.Regexp            `json:"-" db:"-"`
}

func (t Tag) Compare(val string) bool {
	return t.CompareFunc(val, t.Value)
}

type Message struct {
	Id            int32     `db:"id"`
	GotAt         time.Time `db:"got_at"`
	DeviceId      int32     `db:"device_id"`
	Message       string    `db:"message"`
	MessageType   string    `db:"message_type"`
	SeverityLevel string    `db:"severity_level"`
	Component     string    `db:"component"`
	DeviceIP      string    `db:"-"`
}

type CountByDeviceID struct {
	DeviceId int32 `db:"device_id"`
	Count    int32 `db:"count"`
}

type SendedNotification struct {
	Message   string
	DeviceId  int32
	ExpiredAt time.Time
}

type MonthReportRow struct {
	DeviceID               int32           `db:"device_id"`
	MessageType            string          `db:"message_type"`
	ActiveDays             int32           `db:"active_days"`
	TotalMessages          int64           `db:"total_messages"`
	AvgDailyMessages       float64         `db:"avg_daily_messages"`
	MaxDailyMessages       int64           `db:"max_daily_messages"`
	MedianDailyMessages    float64         `db:"median_daily_messages"`
	TotalCritical          int64           `db:"total_critical"`
	MaxDailyCritical       int64           `db:"max_daily_critical"`
	MaxDailyComponents     int32           `db:"max_daily_components"`
	MostActiveComponent    sql.NullString  `db:"most_active_component"`
	FirstCriticalTime      sql.NullTime    `db:"first_critical_time"`
	LastCriticalTime       sql.NullTime    `db:"last_critical_time"`
	AvgCriticalIntervalSec sql.NullFloat64 `db:"avg_critical_interval_sec"`
	CriticalPercentage     float64         `db:"critical_percentage"`
	OverallVolumeRank      int32           `db:"overall_volume_rank"`
}
