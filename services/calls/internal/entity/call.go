package entity

import (
	"time"
)

// CallStatus mirrors the gRPC CallStatus enum for DB storage.
type CallStatus string

const (
	CallStatusRinging   CallStatus = "ringing"
	CallStatusActive    CallStatus = "active"
	CallStatusEnded     CallStatus = "ended"
	CallStatusMissed    CallStatus = "missed"
	CallStatusRejected  CallStatus = "rejected"
	CallStatusCancelled CallStatus = "cancelled"
)

// CallType mirrors the gRPC CallType enum for DB storage.
type CallType string

const (
	CallTypeAudio CallType = "audio"
	CallTypeVideo CallType = "video"
)

// Call records a single call session between two users.
type Call struct {
	ID           uint       `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time  `json:"created_at"`
	CallerID     uint       `gorm:"index;not null" json:"caller_id"`
	CalleeID     uint       `gorm:"index;not null" json:"callee_id"`
	ChatID       uint       `gorm:"index" json:"chat_id,omitempty"`
	CallType     CallType   `gorm:"size:10;default:'audio'" json:"call_type"`
	Status       CallStatus `gorm:"size:20;default:'ringing';index" json:"status"`
	StartedAtMs  int64      `json:"started_at_ms"`
	AcceptedAtMs int64      `json:"accepted_at_ms,omitempty"`
	EndedAtMs    int64      `json:"ended_at_ms,omitempty"`
	DurationMs   int64      `json:"duration_ms"`
	EndReason    string     `gorm:"size:20" json:"end_reason,omitempty"`
	TimeToDieAt  int64      `json:"-"`
}
