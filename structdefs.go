package main

// Dear future us: I do not currently understand the syntax of Go struct definitions
// (especially the json syntax)
// (but I know it has saved me from worse pain)
// so I am putting them in their own file to keep things organized.

// ClientInfo represents the information sent by the client during registration.
type ClientInfo struct {
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	OSName   string `json:"os_name"`
}

// RegisterResult represents the response from the server upon registration.
type RegisterResult struct {
	AgentID string `json:"agent_id"`
}

// TaskResult represents the tasks assigned to the client by the server. (not parsed)
type TaskResult struct {
	Tasks string `json:"tasks"`
}
