package sidecar

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"sfsAI/internal/config"
	"sfsAI/internal/memory"
)

type Server struct {
	sidecar *Sidecar
	cfg     *config.Config
	engine  *gin.Engine
	httpSrv *http.Server
}

func NewServer(sc *Sidecar, cfg *config.Config) *Server {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())

	s := &Server{sidecar: sc, cfg: cfg, engine: engine}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	v1 := s.engine.Group("/api/v1")
	{
		v1.GET("/health", s.handleHealth)
		v1.GET("/stats", s.handleStats)

		m := v1.Group("/memories")
		{
			m.POST("", s.handleInsert)
			m.GET("", s.handleSearch)
			m.GET("/:sid/:mid", s.handleGet)
			m.GET("/recent/:sid", s.handleRecent)
			m.DELETE("/wipe", s.handleWipe)
			m.POST("/compress/:sid", s.handleCompress)
			m.POST("/semantic", s.handleSemantic)
		}
	}
}

func (s *Server) Start() error {
	s.httpSrv = &http.Server{
		Addr:         s.cfg.API.HTTPAddr,
		Handler:      s.engine,
		ReadTimeout:  s.cfg.API.ReadTimeout,
		WriteTimeout: s.cfg.API.WriteTimeout,
	}

	log.Printf("[sfsAI] listening on %s", s.cfg.API.HTTPAddr)
	go func() {
		if err := s.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[sfsAI] http error: %v", err)
		}
	}()
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpSrv != nil {
		return s.httpSrv.Shutdown(ctx)
	}
	return nil
}

func (s *Server) handleHealth(c *gin.Context) {
	if err := s.sidecar.Health(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "healthy", "uptime": s.sidecar.Uptime().String(), "version": "0.1.0"})
}

func (s *Server) handleStats(c *gin.Context) {
	mb, total, err := s.sidecar.MemoryStore().GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"memory_usage_mb": mb, "total_records": total, "uptime": s.sidecar.Uptime().String(), "running": s.sidecar.IsRunning()})
}

type insertReq struct {
	SessionID string                 `json:"session_id" binding:"required"`
	Content   string                 `json:"content" binding:"required"`
	Embedding []float32              `json:"embedding"`
	Metadata  map[string]interface{} `json:"metadata"`
}

func (s *Server) handleInsert(c *gin.Context) {
	var req insertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	mem := &memory.MemoryUnit{
		SessionID: req.SessionID,
		Content:   req.Content,
		Embedding: req.Embedding,
		Metadata:  req.Metadata,
	}

	if err := s.sidecar.MemoryStore().InsertMemory(mem); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"memory_id": mem.ID, "created_at": mem.CreatedAt})
}

func (s *Server) handleSearch(c *gin.Context) {
	var req struct {
		SessionID string `json:"session_id" binding:"required"`
		TimeStart string `json:"time_start"`
		TimeEnd   string `json:"time_end"`
		TopK      int    `json:"top_k"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	q := &memory.MemoryQuery{SessionID: req.SessionID, TopK: req.TopK}
	if req.TimeStart != "" {
		q.TimeStart, _ = time.Parse(time.RFC3339, req.TimeStart)
	}
	if req.TimeEnd != "" {
		q.TimeEnd, _ = time.Parse(time.RFC3339, req.TimeEnd)
	}

	results, err := s.sidecar.MemoryStore().SearchMemories(q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"memories": results, "count": len(results)})
}

func (s *Server) handleGet(c *gin.Context) {
	mem, err := s.sidecar.MemoryStore().GetMemory(c.Param("sid"), c.Param("mid"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, mem)
}

func (s *Server) handleRecent(c *gin.Context) {
	limit := 10
	window := 10 * time.Minute

	fmt.Sscanf(c.DefaultQuery("limit", "10"), "%d", &limit)
	if w, err := time.ParseDuration(c.DefaultQuery("window", "10m")); err == nil {
		window = w
	}

	results, err := s.sidecar.MemoryStore().GetRecentMemories(c.Param("sid"), limit, window)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"memories": results, "count": len(results)})
}

func (s *Server) handleWipe(c *gin.Context) {
	var req struct {
		SessionID string   `json:"session_id" binding:"required"`
		MemoryIDs []string `json:"memory_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.sidecar.MemoryStore().WipeMemories(memory.MemoryFilter{
		SessionID: req.SessionID,
		MemoryIDs: req.MemoryIDs,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) handleCompress(c *gin.Context) {
	olderThan := 72 * time.Hour
	if d, err := time.ParseDuration(c.DefaultQuery("older_than", "72h")); err == nil {
		olderThan = d
	}

	if err := s.sidecar.MemoryStore().CompressMemories(c.Param("sid"), time.Now().Add(-olderThan)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) handleSemantic(c *gin.Context) {
	var req struct {
		SessionID string    `json:"session_id" binding:"required"`
		Vector    []float32 `json:"vector" binding:"required"`
		TopK      int       `json:"top_k"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	searcher, ok := s.sidecar.MemoryStore().(interface {
		SemanticSearch(string, []float32, int) ([]*memory.MemoryUnit, error)
	})
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "semantic search not supported"})
		return
	}

	results, err := searcher.SemanticSearch(req.SessionID, req.Vector, req.TopK)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"memories": results, "count": len(results)})
}