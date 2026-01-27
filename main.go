package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TODO: client config file
var server = "http://127.0.0.1:8443/api/end/"
var pollingDelay = 3 * time.Second

func main() {
	log.Printf("The stones are capped.... Or whatever")
	//log.Printf("We'll be connecting to " + server)
	log.Printf("My hostname is " + GetHostname()) // Right now, on Windows we fail here (not exactly sure why. Didn't debug it for now)

	dirname, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	var fp = filepath.Join(dirname, ".agent_id")

	_, err = os.Stat(fp)
	var id = ""

	if os.IsNotExist(err) {
		// No agent ID file
		log.Printf("We have not previously registered on the server.")
		id = register(server)
		SaveAgentID(fp, id)
		log.Printf("We are now registered on the server.")
	} else {
		log.Printf("We have already registered on the server.")
		cont, err := os.ReadFile(fp)
		if err != nil {
			log.Fatal(err)
		}
		id = strings.TrimSpace(string(cont))
	}

	time.Sleep(pollingDelay)

	var running = true

	for running {
		log.Printf("Checking for tasks.....")

		var res = checkin(server, id)

		if len(res) == 0 {
			log.Printf("No tasks found for me.")
		} else {
			log.Printf("Task found for me.")
			for i, task := range res {
				log.Printf("Task %d: %s", i+1, formatTask(task))
			}

			// Now we'd have to decode the JSON of the tasks to do
			// Which should have an ID and command(s) to run

			// Then we would post it back

		}

		time.Sleep(pollingDelay)
	}

}
