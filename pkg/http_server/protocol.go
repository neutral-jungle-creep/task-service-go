package http_server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type ExceptionResponse struct {
	ErrorMessage string    `json:"errorMessage"`
	Status       int       `json:"status"`
	Timestamp    time.Time `json:"timestamp"`
}

func SendErrorResponse(w http.ResponseWriter, status int, message string, err error) {
	errDTO := &ExceptionResponse{
		ErrorMessage: fmt.Sprintf("%s %v", message, err),
		Status:       status,
		Timestamp:    time.Now().Truncate(time.Second),
	}

	bytes, err := json.Marshal(errDTO)
	if err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte(err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes)
	w.WriteHeader(status)
}
