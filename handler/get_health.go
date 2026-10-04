package handler

import (
	"encoding/json"
	"log"
	"net/http"
)

type HealthResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := HealthResponse{
		Status:  "ok",
		Message: "hello world",
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("failed to encode health response: %v", err)
	}
}
