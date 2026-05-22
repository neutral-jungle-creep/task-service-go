package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

	defaultListLimit = 50
	maxListLimit     = 500
)

// ListTasks returns a paginated page of tasks plus the total row count.
//
//	@Summary	List tasks (paginated)
//	@Tags		tasks
//	@Produce	json
//	@Param		limit	query		integer	false	"Items per page (default 50, max 500)"
//	@Param		offset	query		integer	false	"Items to skip (default 0)"
//	@Success	200		{object}	dto.ListTasksResponse
//	@Failure	400		{object}	protocol.ExceptionResponse
//	@Failure	500		{object}	protocol.ExceptionResponse
//	@Router		/tasks [get]
func (api *API) ListTasks(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := parsePagination(r.URL.Query())
	if err != nil {
		protocol.SendErrorResponse(w, http.StatusBadRequest, incorrectRequestBodyError, err)
		return
	}

	tasks, total, err := api.taskService.List(limit, offset)
	if err != nil {
		protocol.SendErrorResponse(w, http.StatusInternalServerError, internalServerError, err)
		return
	}

	response := dto.ListTasksResponse{
		Items:  tasksFromDomain(tasks),
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}
	protocol.SendSuccessResponse(w, http.StatusOK, response)
}

// parsePagination reads ?limit=N&offset=M with the rules:
//   - missing → defaults (50, 0);
//   - non-numeric or negative → 400;
//   - limit > maxListLimit → 400.
func parsePagination(q url.Values) (limit, offset uint64, err error) {
	limit = defaultListLimit
	if raw := q.Get("limit"); raw != "" {
		v, parseErr := strconv.ParseUint(raw, 10, 64)
		if parseErr != nil {
			return 0, 0, errors.New("limit must be a non-negative integer")
		}
		if v == 0 {
			return 0, 0, errors.New("limit must be > 0")
		}
		if v > maxListLimit {
			return 0, 0, fmt.Errorf("limit must be <= %d", maxListLimit)
		}
		limit = v
	}
	if raw := q.Get("offset"); raw != "" {
		v, parseErr := strconv.ParseUint(raw, 10, 64)
		if parseErr != nil {
			return 0, 0, errors.New("offset must be a non-negative integer")
		}
		offset = v
	}
	return limit, offset, nil
}

// GetTask returns a single task by id.
//
//	@Summary	Get task by id
//	@Tags		tasks
//	@Produce	json
//	@Param		id	path		uint64	true	"Task id"
//	@Success	200	{object}	dto.GetTaskResponse
//	@Failure	400	{object}	protocol.ExceptionResponse
//	@Failure	404	{object}	protocol.ExceptionResponse
//	@Failure	500	{object}	protocol.ExceptionResponse
//	@Router		/tasks/{id} [get]
func (api *API) GetTask(w http.ResponseWriter, r *http.Request) {
	params := server.RequestParams(r)
	idParam := params["id"]
	if len(idParam) == 0 {
		protocol.SendErrorResponse(w, http.StatusBadRequest, incorrectRequestBodyError, errors.New("id is required"))
		return
	}

	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		protocol.SendErrorResponse(w, http.StatusBadRequest, incorrectRequestBodyError, err)
		return
	}

	task, err := api.taskService.Get(id)
	if err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			protocol.SendErrorResponse(w, http.StatusNotFound, "", err)
			return
		}
		protocol.SendErrorResponse(w, http.StatusInternalServerError, internalServerError, err)
		return
	}

	response := taskFromDomain(task)
	protocol.SendSuccessResponse(w, http.StatusOK, response)
}

// CreateTask creates a new task and returns its id.
//
//	@Summary	Create task
//	@Tags		tasks
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		dto.CreateTaskRequest	true	"Task to create"
//	@Success	200		{object}	dto.CreateTaskResponse
//	@Failure	400		{object}	protocol.ExceptionResponse
//	@Failure	500		{object}	protocol.ExceptionResponse
//	@Router		/tasks [post]
func (api *API) CreateTask(w http.ResponseWriter, r *http.Request) {
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

	if params == nil || params.Name == "" || params.Body == "" {
		protocol.SendErrorResponse(w, http.StatusBadRequest, incorrectRequestBodyError,
			errors.New("name and body are required"))
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
