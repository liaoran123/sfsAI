package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"sfsAI/internal/config"
	"sfsAI/internal/sidecar"
)

func main() {
	cfgPath := flag.String("config", "", "path to config file")
	dbPath := flag.String("db", "", "path to database directory")
	httpAddr := flag.String("addr", ":8630", "HTTP API server address")
	flag.Parse()

	cfg := config.DefaultConfig()

	if *dbPath != "" {
		cfg.Sidecar.DBPath = *dbPath
	}
	if *httpAddr != "" {
		cfg.API.HTTPAddr = *httpAddr
	}

	if *cfgPath != "" {
		cfgFile, err := os.ReadFile(*cfgPath)
		if err != nil {
			log.Printf("warning: cannot read config file %s: %v", *cfgPath, err)
		} else {
			lines := string(cfgFile)
			fmt.Println("config loaded:", len(lines), "bytes")
		}
	}

	log.Printf("sfsAI Sidecar v0.1.0")
	log.Printf("database path: %s", cfg.Sidecar.DBPath)
	log.Printf("HTTP API: %s", cfg.API.HTTPAddr)
	log.Printf("encryption: %v", cfg.Memory.EnableEncryption)

	sc, err := sidecar.New(cfg)
	if err != nil {
		log.Fatalf("failed to create sidecar: %v", err)
	}

	if err := sc.Start(); err != nil {
		log.Fatalf("failed to start sidecar: %v", err)
	}

	sc.WaitForShutdown()
}