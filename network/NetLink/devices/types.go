package devices

// Device represents a network device discovered by net-graph.
type Device struct {
	IP       string `json:"ip"`
	Hostname string `json:"hostname,omitempty"`
}
