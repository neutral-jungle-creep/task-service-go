package protocol

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

	body, err := json.Marshal(errDTO)
	if err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func SendSuccessResponse(w http.ResponseWriter, status int, body any) {
	payload, err := json.Marshal(body)
	if err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(payload)
}
