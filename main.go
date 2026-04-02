package main

import (
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

type herdagentConfig struct {
	Server       string `toml:"server"`
	EnrollServer string `toml:"enrollServer"`
	PollingDelay string `toml:"pollingDelay"`
	CertFile     string `toml:"certFile"`
	KeyFile      string `toml:"keyFile"`
	CACertFile   string `toml:"caCertFile"`
	AgentID      string `toml:"agentID"`
}

func loadConfig(path string) (herdagentConfig, time.Duration) {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("failed to read config file %s: %v", path, err)
	}

	var cfg herdagentConfig
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

func saveConfig(path string, cfg herdagentConfig) {
	data, err := toml.Marshal(cfg)
	if err != nil {
		log.Fatalf("failed to marshal config: %v", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		log.Fatalf("failed to write config %s: %v", path, err)
	}
}

func resolveConfigPath(flagPath string) string {
	const localConfig = "config.toml"
	if _, err := os.Stat(localConfig); err == nil {
		return localConfig
	}
	return flagPath
}

func main() {
	configFlag := flag.String("c", "/etc/herd/config.toml", "path to config file")
	flag.Parse()

	configPath := resolveConfigPath(*configFlag)
	cfg, pollingDelay := loadConfig(configPath)

	log.Printf("The stones are capped.... Or whatever")
	log.Printf("My hostname is %s", GetHostname())

	var agentID string

	_, certErr := os.Stat(cfg.CertFile)

	switch {
	case os.IsNotExist(certErr):
		// No cert on disk — enroll now over plain HTTP.
		if cfg.AgentID != "" {
			log.Printf("Warning: agentID set in config but cert is missing; re-enrolling")
		}
		log.Printf("No client certificate found, enrolling...")
		agentID = enroll(cfg)
		cfg.AgentID = agentID
		saveConfig(configPath, cfg)
		log.Printf("Enrollment complete, agent ID: %s", agentID)

	case cfg.AgentID == "":
		// Cert exists but no agent_id — something went wrong during a previous run.
		log.Fatal("cert files exist but agentID is missing from config; delete certs/ and re-enroll")

	default:
		// Both cert and agent_id present — normal startup.
		agentID = cfg.AgentID
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
