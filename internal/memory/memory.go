package memory

import (
	"time"
)

type MemoryUnit struct {
	ID        string                 `json:"id"`
	SessionID string                 `json:"session_id"`
	Content   string                 `json:"content"`
	Embedding []float32              `json:"embedding"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"created_at"`
	ExpiresAt *time.Time             `json:"expires_at,omitempty"`
}

type MemoryQuery struct {
	SessionID string
	TimeStart time.Time
	TimeEnd   time.Time
	TopK      int
}

type MemoryFilter struct {
	SessionID string
	MemoryIDs []string
}

type AIMemoryStore interface {
	InsertMemory(memory *MemoryUnit) error
	SearchMemories(query *MemoryQuery) ([]*MemoryUnit, error)
	GetMemory(sessionID, memoryID string) (*MemoryUnit, error)
	GetRecentMemories(sessionID string, limit int, timeWindow time.Duration) ([]*MemoryUnit, error)
	WipeMemories(filter MemoryFilter) error
	CompressMemories(sessionID string, olderThan time.Time) error
	GetStats() (memoryUsageMB int, totalRecords int, err error)
	Close() error
}