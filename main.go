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
	PollingDelay string `toml:"pollingDelay"`
}

func loadConfig(path string) (string, time.Duration) {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("failed to read config file %s: %v", path, err)
	}

	var cfg endpointConfig
	if err := toml.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("failed to parse config file %s: %v", path, err)
	}

	server := strings.TrimSpace(cfg.Server)
	if server == "" {
		log.Fatalf("config %s missing server value", path)
	}

	pollingDelayStr := strings.TrimSpace(cfg.PollingDelay)
	if pollingDelayStr == "" {
		log.Fatalf("config %s missing pollingDelay value", path)
	}

	pollingDelay, err := time.ParseDuration(pollingDelayStr)
	if err != nil {
		log.Fatalf("invalid pollingDelay in %s: %v", path, err)
	}

	return server, pollingDelay
}

func main() {
	server, pollingDelay := loadConfig("config.toml")

	log.Printf("The stones are capped.... Or whatever")
	//log.Printf("We'll be connecting to " + server)
	log.Printf("My hostname is %s", GetHostname()) // Right now, on Windows we fail here (not exactly sure why. Didn't debug it for now)

	dirname, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	var fp = filepath.Join(dirname, ".agent_id")

	_, err = os.Stat(fp)
	var agentId = ""

	if os.IsNotExist(err) {
		// No agent ID file
		log.Printf("We have not previously registered on the server.")
		agentId = register(server)
		SaveAgentID(fp, agentId)
		log.Printf("We are now registered on the server.")
	} else {
		log.Printf("We have already registered on the server.")
		cont, err := os.ReadFile(fp)
		if err != nil {
			log.Fatal(err)
		}
		agentId = strings.TrimSpace(string(cont))
	}

	time.Sleep(pollingDelay)

	for {
		log.Printf("Checking for tasks.....")

		var res = checkin(server, agentId)

		if len(res) == 0 {
			log.Printf("No tasks found for me.")
		} else {
			log.Printf("Task found for me.")
			for i, task := range res {
				log.Printf("Task %d: %s", i+1, formatTask(task))

				if !processTask(server, agentId, task) {
					break
				}
			}
		}
		time.Sleep(pollingDelay)
	}

}
