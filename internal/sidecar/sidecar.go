package sidecar

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/liaoran123/sfsDb/storage"
	"sfsAI/internal/config"
	"sfsAI/internal/memory"
	"sfsAI/internal/security"
)

type Sidecar struct {
	cfg         *config.Config
	store       storage.Store
	memoryStore memory.AIMemoryStore
	encryptor   *security.Encryptor
	server      *Server

	healthChecks []func() error
	mu           sync.RWMutex
	running      bool
	startedAt    time.Time
}

func New(cfg *config.Config) (*Sidecar, error) {
	sc := &Sidecar{
		cfg:          cfg,
		healthChecks: make([]func() error, 0),
	}

	if err := sc.initDB(); err != nil {
		return nil, fmt.Errorf("init database: %w", err)
	}
	if err := sc.initEncryption(); err != nil {
		return nil, fmt.Errorf("init encryption: %w", err)
	}
	if err := sc.initMemory(); err != nil {
		return nil, fmt.Errorf("init memory store: %w", err)
	}

	sc.initHealthChecks()
	return sc, nil
}

func (sc *Sidecar) initDB() error {
	dbManager := storage.GetDBManager()
	store, err := dbManager.OpenDB(sc.cfg.Sidecar.DBPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	sc.store = store
	return nil
}

func (sc *Sidecar) initEncryption() error {
	if !sc.cfg.Memory.EnableEncryption {
		return nil
	}

	key := sc.cfg.Memory.EncryptionKey
	if len(key) == 0 {
		deviceKey, err := security.NewDeviceBoundKey("sfsai-default-device")
		if err != nil {
			return fmt.Errorf("device key: %w", err)
		}
		key = deviceKey.DeriveKey()
	}

	enc, err := security.NewEncryptor(key)
	if err != nil {
		return fmt.Errorf("encryptor: %w", err)
	}
	sc.encryptor = enc
	return nil
}

func (sc *Sidecar) initMemory() error {
	memStore, err := memory.NewMemoryStore(sc.store, memory.StoreConfig{
		DefaultTopK: sc.cfg.Memory.DefaultTopK,
	})
	if err != nil {
		return fmt.Errorf("memory store: %w", err)
	}
	sc.memoryStore = memStore
	return nil
}

func (sc *Sidecar) initHealthChecks() {
	sc.healthChecks = append(sc.healthChecks, func() error {
		if sc.store == nil {
			return fmt.Errorf("store is nil")
		}
		return nil
	})
	sc.healthChecks = append(sc.healthChecks, func() error {
		if sc.memoryStore == nil {
			return fmt.Errorf("memory store is nil")
		}
		return nil
	})
}

func (sc *Sidecar) MemoryStore() memory.AIMemoryStore {
	return sc.memoryStore
}

func (sc *Sidecar) Encryptor() *security.Encryptor {
	return sc.encryptor
}

func (sc *Sidecar) Store() storage.Store {
	return sc.store
}

func (sc *Sidecar) Start() error {
	sc.mu.Lock()
	if sc.running {
		sc.mu.Unlock()
		return fmt.Errorf("sidecar is already running")
	}
	sc.running = true
	sc.startedAt = time.Now()
	sc.mu.Unlock()

	log.Printf("[sfsAI Sidecar] starting on %s (db: %s)", sc.cfg.API.HTTPAddr, sc.cfg.Sidecar.DBPath)
	log.Printf("[sfsAI Sidecar] encryption: %v, auto-compress: %v", sc.cfg.Memory.EnableEncryption, sc.cfg.Memory.AutoCompress)

	sc.server = NewServer(sc, sc.cfg)
	return sc.server.Start()
}

func (sc *Sidecar) Shutdown(ctx context.Context) error {
	sc.mu.Lock()
	sc.running = false
	sc.mu.Unlock()

	log.Printf("[sfsAI Sidecar] shutting down...")

	if sc.server != nil {
		if err := sc.server.Shutdown(ctx); err != nil {
			log.Printf("[sfsAI Sidecar] server shutdown error: %v", err)
		}
	}
	if sc.memoryStore != nil {
		sc.memoryStore.Close()
	}
	if sc.store != nil {
		storage.GetDBManager().CloseDB()
	}

	log.Printf("[sfsAI Sidecar] shutdown complete")
	return nil
}

func (sc *Sidecar) WaitForShutdown() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	sig := <-sigCh
	log.Printf("[sfsAI Sidecar] received signal: %v", sig)

	ctx, cancel := context.WithTimeout(context.Background(), sc.cfg.Sidecar.GracefulShutdown)
	defer cancel()

	sc.Shutdown(ctx)
}

func (sc *Sidecar) IsRunning() bool {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.running
}

func (sc *Sidecar) Uptime() time.Duration {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	if !sc.running {
		return 0
	}
	return time.Since(sc.startedAt)
}

func (sc *Sidecar) Health() error {
	for _, check := range sc.healthChecks {
		if err := check(); err != nil {
			return err
		}
	}
	return nil
}