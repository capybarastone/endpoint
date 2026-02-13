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
	"strconv"
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
		log.Println(err)
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

func GetCommandOutput(command string) string {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		powershellPath, err := exec.LookPath("powershell")
		if err != nil {
			log.Fatalf("PowerShell not found: %v", err)
		}
		cmd = exec.Command(powershellPath, "-NoProfile", "-NonInteractive", "-Command", command)
	default:
		bashPath, err := exec.LookPath("bash")
		if err != nil {
			log.Fatalf("bash not found in PATH: %v", err)
		}
		cmd = exec.Command(bashPath, "-lc", command)
	}

	output, err := cmd.Output()
	if err != nil {
		log.Fatalf("Error executing `%s`: %v", command, err)
	}

	return string(output)
}

func GetHostname() string {
	switch runtime.GOOS {
	case "windows":
		return GetCommandOutput("hostname")
	case "linux":
		// TODO: error validation?
		return strings.TrimSpace(GetCommandOutput("cat /etc/hostname"))
	}

	return "{something went wrong}"
}

func GetOSSubtype() string {
	switch runtime.GOOS {
	case "windows":
		var blorp = GetCommandOutput("systeminfo | findstr /B /C:\"OS Name\" /C:\"OS Version\"")
		var re = regexp.MustCompile(`(?m)\d\d \w\w\w`)
		var res = re.FindString(blorp)

		return res

	case "linux":
		if _, err := os.Stat("/etc/os-release"); err != nil {
			// TODO: could try checking lsb release or redhat release
			return "Uknown"
		}
		var glorp string = GetCommandOutput("cat /etc/os-release")
		var redistro = regexp.MustCompile(`(?m)\bNAME="\w*`)
		var distro = strings.Replace(redistro.FindString(glorp), "NAME=\"", "", 1)
		var rerel = regexp.MustCompile(`(?m)\bBUILD_ID=\w*`)
		var release = strings.Replace(rerel.FindString(glorp), "BUILD_ID=", "", 1)

		return distro + " " + release
	}

	return "{confused in this empty place}"
}

func GetCPUCores() string {
	switch runtime.GOOS {
	case "linux":
		return GetCommandOutput("nproc")
	case "windows":
		// TODO: return format untested
		// nproc equivalent would just be some INT number of cores
		return GetCommandOutput("wmic cpu get NumberOfCores,NumberOfLogicalProcessors")
	}

	return ""
}

// TODO:
// CPU usage???
// X% in use

func GetUsedMemory() string {
	// TODO: goal is X G.B. of X G.B. total
	// so that mgmt could parse into % free
	switch runtime.GOOS {
	case "linux":
		// prefer MemAvailable as a closer approximation to "free" memory
		totalRaw := strings.TrimSpace(GetCommandOutput("awk '/^MemTotal:/ {print $2}' /proc/meminfo"))
		availableRaw := strings.TrimSpace(GetCommandOutput("awk '/^MemAvailable:/ {print $2}' /proc/meminfo"))
		if availableRaw == "" {
			availableRaw = strings.TrimSpace(GetCommandOutput("awk '/^MemFree:/ {print $2}' /proc/meminfo"))
		}

		totalKB, err := strconv.ParseInt(totalRaw, 10, 64)
		if err != nil || totalKB == 0 {
			log.Printf("unable to parse MemTotal value %q: %v", totalRaw, err)
			return ""
		}

		availableKB, err := strconv.ParseInt(availableRaw, 10, 64)
		if err != nil {
			log.Printf("unable to parse MemAvailable value %q: %v", availableRaw, err)
			return ""
		}

		usedKB := totalKB - availableKB
		if usedKB < 0 {
			usedKB = 0
		}

		usedGB := float64(usedKB) / (1024 * 1024)
		totalGB := float64(totalKB) / (1024 * 1024)
		percentUsed := (float64(usedKB) / float64(totalKB)) * 100

		return fmt.Sprintf("%.2f GB used of %.2f GB total (%.0f%%)", usedGB, totalGB, percentUsed)

	case "windows":
		// TODO: I assume WMIC can do free memory?
		// but also it seems to only sometimes be available?
		return "TODO"
	}

	return ""
}

func GetDiskUsage() string {
	switch runtime.GOOS {
	case "linux":
		return GetCommandOutput("df | awk 'NR==2 {print $5}'")
	case "windows":
		return GetCommandOutput("wmic disk get") // TODO: I dunno what to do for this one
	}

	return ""
}

// TODO: re-org this file or make category-specfic files?
// IDK I could go either way
func cloneTask(task Task) Task {
	result := make(Task, len(task))
	for k, v := range task {
		result[k] = v
	}
	return result
}

func compileSystemInfo() HostInvetoryObject {
	var myinfo = HostInvetoryObject{}                         // below assumptions for Linux only
	myinfo["hostname"] = GetHostname()                        // should be good
	myinfo["os_string"] = runtime.GOOS + " " + GetOSSubtype() // should be good?
	myinfo["cpu_count"] = GetCPUCores()                       // should be good
	myinfo["memory_use"] = GetUsedMemory()                    // should be good
	myinfo["disk_usage"] = GetDiskUsage()                     // should be good

	return myinfo
}
