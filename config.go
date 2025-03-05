package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"time"
)

// Config holds configuration data loaded from config.json
type Config struct {
	Extensions []string `json:"extensions"` // List of file extensions to filter
	NumWorkers int      `json:"numWorkers"` // Number of concurrent workers
}

// Constants for hardcoded values
const (
	DefaultTimeout     = 120 * time.Second
	DefaultWorkerDelay = 2 * time.Second
	DefaultNumWorkers  = 5
)

// loadConfig reads and parses the config.json file
func loadConfig() Config {
	file, err := os.ReadFile("config.json")
	if err != nil {
		fmt.Println("Failed to read config file:", err)
		os.Exit(1)
	}

	var config Config
	if err := json.Unmarshal(file, &config); err != nil {
		fmt.Println("Failed to parse config file:", err)
		os.Exit(1)
	}

	// Validate and set defaults
	if len(config.Extensions) == 0 {
		fmt.Println("No extensions specified in config.json")
		os.Exit(1)
	}
	if config.NumWorkers <= 0 {
		config.NumWorkers = DefaultNumWorkers
	}
	if config.NumWorkers > runtime.NumCPU()*2 {
		config.NumWorkers = runtime.NumCPU() * 2 // Reasonable upper limit
	}

	return config
}