package server

import (
	"encoding/json"
	"io"
	"net/http"

	"task-service/internal/domain"
	"task-service/internal/server/dto"
	"task-service/pkg/http/protocol"
)

const (
	readRequestBodyError      = "ошибка чтения тела запроса"
	incorrectRequestBodyError = "некорректный формат запроса"
	internalServerError       = "внутренняя ошибка сервера"
)

func (api *Api) ListTasks(w http.ResponseWriter, r *http.Request) {
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
