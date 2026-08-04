package mysql

import (
	"encoding/json"
	"time"
)

type jobRecord struct {
	ID          string          `gorm:"column:id;type:char(36);primaryKey"`
	Type        string          `gorm:"column:type;type:varchar(50);not null"`
	Status      string          `gorm:"column:status;type:varchar(20);not null"`
	Payload     json.RawMessage `gorm:"column:payload;type:json;not null"`
	RetryCount  int             `gorm:"column:retry_count;not null"`
	MaxRetries  int             `gorm:"column:max_retries;not null"`
	LastError   *string         `gorm:"column:last_error"`
	CreatedAt   time.Time       `gorm:"column:created_at;not null"`
	UpdatedAt   time.Time       `gorm:"column:updated_at;not null"`
	StartedAt   *time.Time      `gorm:"column:started_at"`
	CompletedAt *time.Time      `gorm:"column:completed_at"`
}

func (jobRecord) TableName() string {
	return "jobs"
}
