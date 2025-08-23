package server

import (
	"encoding/json"
	"io"
	"net/http"

	"task-service/internal/server/dto"
	"task-service/pkg/http_server"
)

const (
	readRequestBodyError      = "ошибка чтения тела запроса"
	incorrectRequestBodyError = "некорректный формат запроса"
)

func (api *Api) ListTasks(w http.ResponseWriter, r *http.Request) {

}

func (api *Api) GetTask(w http.ResponseWriter, r *http.Request) {

}

func (api *Api) CreateTask(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http_server.SendErrorResponse(w, http.StatusBadRequest, readRequestBodyError, err)
		return
	}

	var params *dto.CreateTaskRequest
	err = json.Unmarshal(body, &params)
	if err != nil {
		http_server.SendErrorResponse(w, http.StatusBadRequest, incorrectRequestBodyError, err)
		return
	}
}
