package memory

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/liaoran123/sfsDb/engine"
	"github.com/liaoran123/sfsDb/storage"
)

const memoryTableName = "ai_memories"

type memoryStore struct {
	table *engine.Table
	cfg   StoreConfig
}

type StoreConfig struct {
	DefaultTopK int
}

func NewMemoryStore(s storage.Store, cfg StoreConfig) (AIMemoryStore, error) {
	table, err := engine.NewTable(memoryTableName)
	if err != nil {
		return nil, fmt.Errorf("create memory table: %w", err)
	}

	if err := table.SetFields(map[string]any{
		"memory_id":  "",
		"session_id": "",
		"content":    "",
		"embedding":  "",
		"metadata":   "",
		"created_at": time.Time{},
		"expires_at": time.Time{},
	}); err != nil {
		return nil, fmt.Errorf("set fields: %w", err)
	}

	if err := table.CreatePrimaryKey("memory_id"); err != nil {
		return nil, fmt.Errorf("create primary key: %w", err)
	}

	if err := table.CreateSimpleIndex("session_time_idx", "session_id", "created_at"); err != nil {
		return nil, fmt.Errorf("create session-time index: %w", err)
	}

	return &memoryStore{table: table, cfg: cfg}, nil
}

func (m *memoryStore) InsertMemory(mem *MemoryUnit) error {
	if mem.ID == "" {
		mem.ID = uuid.New().String()
	}
	if mem.CreatedAt.IsZero() {
		mem.CreatedAt = time.Now()
	}

	metaJSON, err := json.Marshal(mem.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	row := &map[string]any{
		"memory_id":  mem.ID,
		"session_id": mem.SessionID,
		"content":    mem.Content,
		"embedding":  embeddingToStr(mem.Embedding),
		"metadata":   string(metaJSON),
		"created_at": mem.CreatedAt,
	}
	if mem.ExpiresAt != nil {
		(*row)["expires_at"] = *mem.ExpiresAt
	}

	_, err = m.table.Insert(row)
	return err
}

func (m *memoryStore) SearchMemories(query *MemoryQuery) ([]*MemoryUnit, error) {
	if query == nil || query.SessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}

	topK := m.cfg.DefaultTopK
	if query.TopK > 0 {
		topK = query.TopK
	}

	start := map[string]any{
		"session_id": query.SessionID,
		"created_at": time.Time{},
	}
	limit := map[string]any{
		"session_id": query.SessionID,
		"created_at": time.Now(),
	}
	if !query.TimeStart.IsZero() {
		start["created_at"] = query.TimeStart
	}
	if !query.TimeEnd.IsZero() {
		limit["created_at"] = query.TimeEnd
	}

	iter, err := m.table.SearchRange(nil, &start, &limit)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	defer iter.Release()

	records := iter.GetRecords(true)
	results := make([]*MemoryUnit, 0, len(records))

	for _, record := range records {
		r := record
		mem := decodeRecord((*map[string]any)(&r))
		if mem != nil {
			results = append(results, mem)
			if len(results) >= topK {
				break
			}
		}
	}

	return results, nil
}

func (m *memoryStore) GetMemory(sessionID, memoryID string) (*MemoryUnit, error) {
	iter, err := m.table.SearchRange(nil, &map[string]any{
		"session_id": sessionID,
		"created_at": time.Time{},
	}, &map[string]any{
		"session_id": sessionID,
		"created_at": time.Now(),
	})
	if err != nil {
		return nil, fmt.Errorf("get memory: %w", err)
	}
	defer iter.Release()

	for _, record := range iter.GetRecords(true) {
		idStr := fmt.Sprintf("%v", record["memory_id"])
		if idStr == memoryID {
			r := record
			mem := decodeRecord((*map[string]any)(&r))
			if mem != nil {
				return mem, nil
			}
		}
	}
	return nil, fmt.Errorf("memory not found: %s/%s", sessionID, memoryID)
}

func (m *memoryStore) GetRecentMemories(sessionID string, limit int, timeWindow time.Duration) ([]*MemoryUnit, error) {
	return m.SearchMemories(&MemoryQuery{
		SessionID: sessionID,
		TimeStart: time.Now().Add(-timeWindow),
		TimeEnd:   time.Now(),
		TopK:      limit,
	})
}

func (m *memoryStore) WipeMemories(filter MemoryFilter) error {
	if filter.SessionID == "" {
		return fmt.Errorf("session_id is required")
	}

	iter, err := m.table.SearchRange(nil, &map[string]any{
		"session_id": filter.SessionID,
		"created_at": time.Time{},
	}, &map[string]any{
		"session_id": filter.SessionID,
		"created_at": time.Now(),
	})
	if err != nil {
		return fmt.Errorf("search for wipe: %w", err)
	}
	defer iter.Release()

	records := iter.GetRecords(true)
	for _, record := range records {
		if len(filter.MemoryIDs) > 0 {
			idStr := fmt.Sprintf("%v", record["memory_id"])
			match := false
			for _, id := range filter.MemoryIDs {
				if idStr == id {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}

		rec := record
		if err := m.table.Delete((*map[string]any)(&rec)); err != nil {
			return fmt.Errorf("delete: %w", err)
		}
	}

	return nil
}

func (m *memoryStore) CompressMemories(sessionID string, olderThan time.Time) error {
	memories, err := m.SearchMemories(&MemoryQuery{
		SessionID: sessionID,
		TimeEnd:   olderThan,
		TopK:      10000,
	})
	if err != nil {
		return err
	}

	if len(memories) == 0 {
		return nil
	}

	summary := ""
	for _, mem := range memories {
		if mem.Content == "" {
			continue
		}
		if summary != "" {
			summary += "\n---\n"
		}
		if len(mem.Content) > 200 {
			summary += mem.Content[:200] + "..."
		} else {
			summary += mem.Content
		}
	}

	if err := m.InsertMemory(&MemoryUnit{
		SessionID: sessionID,
		Content:   summary,
		Metadata:  map[string]interface{}{"type": "compressed_summary", "original_count": len(memories)},
	}); err != nil {
		return fmt.Errorf("insert summary: %w", err)
	}

	ids := make([]string, len(memories))
	for i, mem := range memories {
		ids[i] = mem.ID
	}
	return m.WipeMemories(MemoryFilter{SessionID: sessionID, MemoryIDs: ids})
}

func (m *memoryStore) GetStats() (memoryUsageMB int, totalRecords int, err error) {
	iter := m.table.ForData()
	if iter == nil {
		return 0, 0, nil
	}
	defer iter.Release()

	count := 0
	for iter.Next() {
		count++
	}

	mb := count * 1024 / (1024 * 1024)
	if mb < 1 {
		mb = 1
	}
	return mb, count, nil
}

func (m *memoryStore) Close() error {
	return nil
}

func embeddingToStr(emb []float32) string {
	if len(emb) == 0 {
		return ""
	}
	data, _ := json.Marshal(emb)
	return string(data)
}

func decodeRecord(record *map[string]any) *MemoryUnit {
	if record == nil {
		return nil
	}

	mem := &MemoryUnit{Metadata: make(map[string]interface{})}

	if v, ok := (*record)["memory_id"]; ok {
		mem.ID = fmt.Sprintf("%v", v)
	}
	if v, ok := (*record)["session_id"]; ok {
		mem.SessionID = fmt.Sprintf("%v", v)
	}
	if v, ok := (*record)["content"]; ok {
		mem.Content = fmt.Sprintf("%v", v)
	}
	if v, ok := (*record)["embedding"]; ok {
		s := fmt.Sprintf("%v", v)
		if s != "" {
			json.Unmarshal([]byte(s), &mem.Embedding)
		}
	}
	if v, ok := (*record)["metadata"]; ok {
		s := fmt.Sprintf("%v", v)
		if s != "" {
			var meta map[string]interface{}
			if json.Unmarshal([]byte(s), &meta) == nil {
				mem.Metadata = meta
			}
		}
	}
	if v, ok := (*record)["created_at"]; ok {
		switch t := v.(type) {
		case time.Time:
			mem.CreatedAt = t
		case string:
			mem.CreatedAt, _ = time.Parse(time.RFC3339, t)
		}
	}
	if v, ok := (*record)["expires_at"]; ok {
		switch t := v.(type) {
		case time.Time:
			if !t.IsZero() {
				mem.ExpiresAt = &t
			}
		case string:
			if parsed, err := time.Parse(time.RFC3339, t); err == nil {
				mem.ExpiresAt = &parsed
			}
		}
	}

	return mem
}
