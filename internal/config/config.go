package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds the application configuration
type Config struct {
	CommonPorts []int `json:"common_ports"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		CommonPorts: []int{
			// Frontend
			3000, // React, Node.js
			3001, // Create React App fallback
			4200, // Angular
			5173, // Vite
			8080, // Vue, general web

			// Backend
			4000, // Phoenix, general API
			5000, // Flask, general API
			8000, // Django, general API
			9000, // PHP-FPM, general API

			// Databases
			3306,  // MySQL/MariaDB
			5432,  // PostgreSQL
			6379,  // Redis
			27017, // MongoDB

			// Tools
			9200, // Elasticsearch
			9090, // Prometheus
			3100, // Grafana Loki

			// Other common ports
			8081, // Alternative HTTP
			8888, // Jupyter
			7000, // Cassandra
			8983, // Solr
		},
	}
}

// Load loads the configuration from file or returns default
func Load() *Config {
	cfg := DefaultConfig()

	// Try to load from config file
	configPath := getConfigPath()
	if configPath != "" {
		if data, err := os.ReadFile(configPath); err == nil {
			json.Unmarshal(data, cfg)
		}
	}

	return cfg
}

// Save saves the configuration to file
func (c *Config) Save() error {
	configPath := getConfigPath()
	if configPath == "" {
		return nil
	}

	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// getConfigPath returns the configuration file path
func getConfigPath() string {
	// Check XDG_CONFIG_HOME first
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "portfinder", "config.json")
	}

	// Fall back to ~/.config
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".config", "portfinder", "config.json")
	}

	return ""
}
