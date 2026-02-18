package handlers

// StatsResponse represents statistics response
type StatsResponse struct {
	Status string                 `json:"status" example:"ok"`
	Stats  map[string]interface{} `json:"stats"`
}

// DatabaseStatusResponse represents database status response
type DatabaseStatusResponse struct {
	Status   string `json:"status" example:"ok"`
	Database struct {
		Connected bool   `json:"connected" example:"true"`
		Type      string `json:"type" example:"sqlite"`
	} `json:"database"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error" example:"error message"`
}


