package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

var verbose bool

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

	var basePath string
	if runtime.GOOS == "windows" {
		basePath = "C:\\herd\\"
	} else {
		basePath = "/etc/herd/"
	}

	if cfg.CertFile == "" {
		cfg.CertFile = basePath + "client.crt"
		if verbose {
			log.Printf("[verbose] certFile not set in config; using absolute path %s", cfg.CertFile)
		}
	}
	if cfg.KeyFile == "" {
		cfg.KeyFile = basePath + "client.key"
		if verbose {
			log.Printf("[verbose] keyFile not set in config; using absolute path %s", cfg.KeyFile)
		}
	}
	if cfg.CACertFile == "" {
		cfg.CACertFile = basePath + "ca.crt"
		if verbose {
			log.Printf("[verbose] caCertFile not set in config; using absolute path %s", cfg.CACertFile)
		}
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
		if verbose {
			log.Printf("[verbose] found %s in CWD; using it instead of %s", localConfig, flagPath)
		}
		return localConfig
	}
	if verbose {
		log.Printf("[verbose] no local config.toml found; using absolute path %s", flagPath)
	}
	return flagPath
}

func main() {

	if runtime.GOOS != "windows" && os.Geteuid() != 0 {
		log.Fatal("herd-agent should be run as root (required for /etc/herd cert permissions)")
		os.Exit(1)
	}

	var defPath string
	if runtime.GOOS == "windows" {
		defPath = "C:\\herd\\config.toml"
	} else {
		defPath = "/etc/herd/config.toml"
	}

	configFlag := flag.String("c", defPath, "path to config file")
	flag.BoolVar(&verbose, "verbose", false, "log when absolute paths are used instead of CWD")
	flag.Parse()

	configPath := resolveConfigPath(*configFlag)
	cfg, pollingDelay := loadConfig(configPath)

	log.Printf("The stones are capped.... Or whatever")
	log.Printf("My hostname is %s", GetHostname())

	var agentID string

	_, certErr := os.Stat(cfg.CertFile)

	agentIDFile := filepath.Join(filepath.Dir(cfg.CertFile), "agent_id")

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
		SaveAgentID(agentIDFile, agentID)
		log.Printf("Enrollment complete, agent ID: %s", agentID)

	case cfg.AgentID == "":
		// Cert exists but agentID missing from config (e.g. config was overwritten by an update).
		// Fall back to the agent_id file written alongside the certs.
		data, err := os.ReadFile(agentIDFile)
		if err != nil {
			log.Fatalf("cert files exist but agentID is missing from config and %s is unreadable: %v; delete certs and re-enroll", agentIDFile, err)
		}
		agentID = strings.TrimSpace(string(data))
		log.Printf("Recovered agent ID from %s: %s", agentIDFile, agentID)

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
