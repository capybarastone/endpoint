package main

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"strings"
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
	log.Printf("Submitting task result for task ID %s", taskValue(task, "task_id"))
}

// processTask handles a single task and returns false when the agent should stop running.
func processTask(server string, agentID string, task Task) bool {
	instruction := taskValue(task, "instruction")

	switch instruction {
	case "syscall":
		cmd := taskValue(task, "arg")
		log.Printf("Running %s for task ID %s", cmd, taskValue(task, "task_id"))

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

	case "install_av":
		var cmd string
		switch runtime.GOOS {
		case "windows":
			cmd = `$s = Get-MpComputerStatus; if ($s.AntivirusEnabled) { "Windows Defender is enabled and active" } else { "WARNING: Defender not enabled - manual intervention required" }`
		default:
			if _, err := exec.LookPath("apt-get"); err != nil {
				submitResult(server, agentID, task, true, map[string]any{
					"stdout":    "",
					"stderr":    "apt-get not found; install_av only supports apt-based systems",
					"exit_code": 1,
				})
				return true
			}
			cmd = "apt-get install -y clamav clamav-daemon 2>&1 && freshclam --quiet 2>&1; echo 'ClamAV install complete'"
		}
		out, code, err := runCommandSafe(cmd)
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		submitResult(server, agentID, task, true, map[string]any{
			"stdout":    out,
			"stderr":    errStr,
			"exit_code": code,
		})
		return true

	case "av_scan":
		scanPath := taskValue(task, "arg")
		if scanPath == "<nil>" || scanPath == "" {
			if runtime.GOOS == "windows" {
				scanPath = `C:\`
			} else {
				scanPath = "/"
			}
		}
		var cmd string
		switch runtime.GOOS {
		case "windows":
			cmd = fmt.Sprintf(`Start-MpScan -ScanPath "%s" -ScanType CustomScan; $threats = Get-MpThreat; if ($threats) { $threats | ForEach-Object { $_.Resources } } else { "No threats found" }`, scanPath)
		default:
			cmd = fmt.Sprintf("clamscan -r --infected '%s' 2>&1", scanPath)
		}
		out, code, err := runCommandSafe(cmd)

		// On Linux, clamscan --infected only prints infected files, but exit code 1
		// means "found something" (not an error). Summarise cleanly for the operator.
		flagged := out
		if runtime.GOOS != "windows" {
			var found []string
			for _, line := range strings.Split(out, "\n") {
				if strings.Contains(line, "FOUND") {
					found = append(found, strings.TrimSpace(line))
				}
			}
			if len(found) > 0 {
				flagged = strings.Join(found, "\n")
			} else if code == 0 {
				flagged = "No threats found"
			}
		}

		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		submitResult(server, agentID, task, true, map[string]any{
			"stdout":    flagged,
			"stderr":    errStr,
			"exit_code": code,
		})
		return true

	default:
		log.Printf("Unknown task instruction %q for task ID %s", instruction, taskValue(task, "task_id"))
		return true
	}
}
