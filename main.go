package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

type endpointConfig struct {
	Server       string `toml:"server"`
	EnrollServer string `toml:"enrollServer"`
	PollingDelay string `toml:"pollingDelay"`
	CertFile     string `toml:"certFile"`
	KeyFile      string `toml:"keyFile"`
	CACertFile   string `toml:"caCertFile"`
}

func loadConfig(path string) (endpointConfig, time.Duration) {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("failed to read config file %s: %v", path, err)
	}

	var cfg endpointConfig
	if err := toml.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("failed to parse config file %s: %v", path, err)
	}

	cfg.Server = strings.TrimSpace(cfg.Server)
	if cfg.Server == "" {
		log.Fatalf("config %s missing server value", path)
	}

	cfg.EnrollServer = strings.TrimSpace(cfg.EnrollServer)
	if cfg.EnrollServer == "" {
		log.Fatalf("config %s missing enrollServer value", path)
	}

	pollingDelayStr := strings.TrimSpace(cfg.PollingDelay)
	if pollingDelayStr == "" {
		log.Fatalf("config %s missing pollingDelay value", path)
	}

	pollingDelay, err := time.ParseDuration(pollingDelayStr)
	if err != nil {
		log.Fatalf("invalid pollingDelay in %s: %v", path, err)
	}

	if cfg.CertFile == "" {
		cfg.CertFile = "certs/client.crt"
	}
	if cfg.KeyFile == "" {
		cfg.KeyFile = "certs/client.key"
	}
	if cfg.CACertFile == "" {
		cfg.CACertFile = "certs/ca.crt"
	}

	return cfg, pollingDelay
}

func main() {
	cfg, pollingDelay := loadConfig("config.toml")

	log.Printf("The stones are capped.... Or whatever")
	log.Printf("My hostname is %s", GetHostname())

	dirname, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	agentIDPath := filepath.Join(dirname, ".agent_id")

	var agentID string

	_, certErr := os.Stat(cfg.CertFile)
	_, idErr := os.Stat(agentIDPath)

	switch {
	case os.IsNotExist(certErr):
		// No cert on disk — enroll now over plain HTTP.
		if !os.IsNotExist(idErr) {
			log.Printf("Warning: agent_id file exists but cert is missing; re-enrolling")
		}
		log.Printf("No client certificate found, enrolling...")
		agentID = enroll(cfg)
		SaveAgentID(agentIDPath, agentID)
		log.Printf("Enrollment complete, agent ID: %s", agentID)

	case os.IsNotExist(idErr):
		// Cert exists but no agent_id — something went wrong during a previous run.
		log.Fatal("cert files exist but agent_id file is missing; delete certs/ and re-enroll")

	default:
		// Both cert and agent_id exist — normal startup.
		raw, err := os.ReadFile(agentIDPath)
		if err != nil {
			log.Fatal(err)
		}
		agentID = strings.TrimSpace(string(raw))
		log.Printf("Using existing cert, agent ID: %s", agentID)
	}

	// Build the mTLS client once; all subsequent requests use it.
	mtlsClient = newMTLSClient(cfg)

	time.Sleep(pollingDelay)

	for {
		log.Printf("Checking for tasks.....")

		res := checkin(cfg.Server, agentID)

		if len(res) == 0 {
			log.Printf("No tasks found for me.")
		} else {
			log.Printf("Task found for me.")
			for i, task := range res {
				log.Printf("Task %d: %s", i+1, formatTask(task))
				if !processTask(cfg.Server, agentID, task) {
					break
				}
			}
		}

		time.Sleep(pollingDelay)
	}
}
