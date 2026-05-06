package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"task-service/internal/domain"
	"task-service/internal/server/dto"
	"task-service/pkg/http/protocol"
	"task-service/pkg/http/server"
)

const (
	readRequestBodyError      = "failed to read request body"
	incorrectRequestBodyError = "incorrect request body"
	internalServerError       = "internal server error"
)

func (api *Api) ListTasks(w http.ResponseWriter, _ *http.Request) {
	tasks, err := api.taskService.List()
	if err != nil {
		protocol.SendErrorResponse(w, http.StatusInternalServerError, internalServerError, err)
		return
	}

	response := dto.ListTasksResponse{
		Items: tasksFromDomain(tasks),
		Total: uint64(len(tasks)), // once pagination is added this value will come from a repository total method
	}
	protocol.SendSuccessResponse(w, http.StatusOK, response)
}

func (api *Api) GetTask(w http.ResponseWriter, r *http.Request) {
	params := server.RequestParams(r)
	idParam := params["id"]
	if len(idParam) == 0 {
		protocol.SendErrorResponse(w, http.StatusBadRequest, incorrectRequestBodyError, errors.New("id is required"))
		return
	}

	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		protocol.SendErrorResponse(w, http.StatusBadRequest, incorrectRequestBodyError, err)
	}

	task, err := api.taskService.Get(id)
	if err != nil {
		protocol.SendErrorResponse(w, http.StatusInternalServerError, internalServerError, err)
		return
	}

	// not-found is detected here (and not inside the service) so that the service can stay
	// quiet about missing rows and other callers can decide on their own how to react
	if task.ID == 0 {
		protocol.SendErrorResponse(w, http.StatusNotFound, "", errors.New("task not found"))
		return
	}

	response := taskFromDomain(task)
	protocol.SendSuccessResponse(w, http.StatusOK, response)
}

func (api *Api) CreateTask(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		protocol.SendErrorResponse(w, http.StatusBadRequest, readRequestBodyError, err)
		return
	}

	var params *dto.CreateTaskRequest
	err = json.Unmarshal(body, &params)
	if err != nil {
		protocol.SendErrorResponse(w, http.StatusBadRequest, incorrectRequestBodyError, err)
		return
	}

	id, err := api.taskService.Create(domain.NewTask(params.Name, params.Body))
	if err != nil {
		protocol.SendErrorResponse(w, http.StatusInternalServerError, internalServerError, err)
		return
	}

	response := dto.CreateTaskResponse{
		ID: id,
	}
	protocol.SendSuccessResponse(w, http.StatusOK, response)
}
