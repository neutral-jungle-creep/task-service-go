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
	readRequestBodyError      = "ошибка чтения тела запроса"
	incorrectRequestBodyError = "некорректный формат запроса"
	internalServerError       = "внутренняя ошибка сервера"
)

func (api *Api) ListTasks(w http.ResponseWriter, _ *http.Request) {
	tasks, err := api.taskService.List()
	if err != nil {
		protocol.SendErrorResponse(w, http.StatusInternalServerError, internalServerError, err)
		return
	}

	response := dto.ListTasksResponse{
		Items: tasksFromDomain(tasks),
		Total: int64(len(tasks)), //  когда появится пагинация, это значение будет браться из метода total репозитория
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

	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		protocol.SendErrorResponse(w, http.StatusBadRequest, incorrectRequestBodyError, err)
	}

	task, err := api.taskService.Get(id)
	if err != nil {
		protocol.SendErrorResponse(w, http.StatusInternalServerError, internalServerError, err)
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
