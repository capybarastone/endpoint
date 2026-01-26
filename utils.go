package main

// https://grugbrain.dev/

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

type ClientInfo struct {
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	OSName   string `json:"os_name"`
}

func register(baseurl string) string {
	log.Printf("%s", "Registering with server at "+baseurl)

	var result struct {
		AgentID string `json:"agent_id"`
	}

	client := req.C() // .DevMode()

	resp, err := client.R().
		SetBody(&ClientInfo{Hostname: GetHostname(), OS: runtime.GOOS, OSName: GetOSSubtype()}).
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

func checkin(baseurl string, agentid string) string {
	//log.Printf("Checking with server at " + baseurl)
	client := req.C() // .DevMode()
	var result struct {
		Tasks string `json:"tasks"`
	}
	post, err := client.R().SetSuccessResult(result).Post(baseurl + "/checkin?agentid=" + agentid)
	if err != nil {
		log.Fatal(err)
		return ""
	}

	if !post.IsSuccessState() {
		// TODO: logger.fatal
		fmt.Println("bad response status:", post.Status)
		fmt.Println("body:", post.String())
		log.Fatalf("register failed")
		return "Failure"
	}

	return result.Tasks
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
		var redistro = regexp.MustCompile(`(?m)\bNAME=\"\w*`)
		var distro = strings.Replace(redistro.FindString(glorp), "NAME=\"", "", 1)
		var rerel = regexp.MustCompile(`(?m)\bBUILD_ID=\w*`)
		var release = strings.Replace(rerel.FindString(glorp), "BUILD_ID=", "", 1)

		return distro + " " + release
	}

	return "{confused in this empty place}"
}
