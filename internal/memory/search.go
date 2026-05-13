package memory

import (
	"fmt"
	"math"
	"sort"
)

func CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProduct float64
	var normA float64
	var normB float64

	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return float32(dotProduct / (math.Sqrt(normA) * math.Sqrt(normB)))
}

func (m *memoryStore) SemanticSearch(sessionID string, queryVector []float32, topK int) ([]*MemoryUnit, error) {
	allMemories, err := m.SearchMemories(&MemoryQuery{
		SessionID: sessionID,
		TopK:      10000,
	})
	if err != nil {
		return nil, fmt.Errorf("search for semantic: %w", err)
	}

	type scoredMemory struct {
		memory *MemoryUnit
		score  float32
	}

	var scored []scoredMemory
	for _, mem := range allMemories {
		if len(mem.Embedding) == 0 {
			continue
		}
		score := CosineSimilarity(queryVector, mem.Embedding)
		scored = append(scored, scoredMemory{memory: mem, score: score})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	if topK <= 0 {
		topK = m.cfg.DefaultTopK
	}
	if topK > len(scored) {
		topK = len(scored)
	}

	results := make([]*MemoryUnit, topK)
	for i := 0; i < topK; i++ {
		results[i] = scored[i].memory
	}

	return results, nil
}