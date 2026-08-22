package auth

// ValidateTargetResponse represents the JSON response from /api/validate-target.
type ValidateTargetResponse struct {
	Valid bool   `json:"valid"`
	Token string `json:"token,omitempty"`
	Error string `json:"error,omitempty"`
}

// TicketResponse represents the JSON response from /api/auth/ticket.
type TicketResponse struct {
	Success bool   `json:"success"`
	Ticket  string `json:"ticket"`
	Error   string `json:"error,omitempty"`
	Details string `json:"details,omitempty"`
}

// LoginResponse represents the JSON response from /api/login.
type LoginResponse struct {
	Token   string   `json:"token"`
	Targets []string `json:"targets,omitempty"`
	Error   string   `json:"error,omitempty"`
}
