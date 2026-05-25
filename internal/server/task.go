package server

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"task-service/internal/domain"
	"task-service/internal/ports"
	"task-service/internal/server/dto"
	"task-service/pkg/http/server"
)

const defaultListLimit = 50

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
	query, err := parseListQuery(r.URL.Query())
	if err != nil {
		api.responseHandler.SendErrorResponse(w, http.StatusBadRequest, err)
		return
	}
	err = api.responseHandler.Validate(&query)
	if err != nil {
		api.responseHandler.SendErrorResponse(w, http.StatusBadRequest, err)
		return
	}

	tasks, total, err := api.taskService.List(r.Context(), query.Limit, query.Offset)
	if err != nil {
		api.responseHandler.SendErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	response := dto.ListTasksResponse{
		Items:  tasksFromDomain(tasks),
		Total:  total,
		Limit:  query.Limit,
		Offset: query.Offset,
	}
	api.responseHandler.SendSuccessResponse(w, http.StatusOK, response)
}

// parseListQuery reads ?limit=N&offset=M and applies defaults. Validation of
// the resulting struct is performed by ResponseHandler.Validate.
func parseListQuery(q url.Values) (dto.ListTasksQuery, error) {
	out := dto.ListTasksQuery{Limit: defaultListLimit}
	if raw := q.Get("limit"); raw != "" {
		v, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return out, errors.New("limit must be a non-negative integer")
		}
		out.Limit = v
	}
	if raw := q.Get("offset"); raw != "" {
		v, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return out, errors.New("offset must be a non-negative integer")
		}
		out.Offset = v
	}
	return out, nil
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
	id, err := parsePathID(r)
	if err != nil {
		api.responseHandler.SendErrorResponse(w, http.StatusBadRequest, err)
		return
	}

	task, err := api.taskService.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			api.responseHandler.SendErrorResponse(w, http.StatusNotFound, err)
			return
		}
		api.responseHandler.SendErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	response := taskFromDomain(task)
	api.responseHandler.SendSuccessResponse(w, http.StatusOK, response)
}

// UpdateTask applies a partial update to a task.
//
//	@Summary	Update task
//	@Tags		tasks
//	@Accept		json
//	@Produce	json
//	@Param		id		path		uint64				true	"Task id"
//	@Param		payload	body		dto.UpdateTaskRequest	true	"Fields to update"
//	@Success	200		{object}	dto.GetTaskResponse
//	@Failure	400		{object}	protocol.ExceptionResponse
//	@Failure	404		{object}	protocol.ExceptionResponse
//	@Failure	500		{object}	protocol.ExceptionResponse
//	@Router		/tasks/{id} [patch]
func (api *API) UpdateTask(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathID(r)
	if err != nil {
		api.responseHandler.SendErrorResponse(w, http.StatusBadRequest, err)
		return
	}

	var body dto.UpdateTaskRequest
	err = api.responseHandler.BindJSON(r.Body, &body)
	if err != nil {
		api.responseHandler.SendErrorResponse(w, http.StatusBadRequest, err)
		return
	}

	updated, err := api.taskService.Update(r.Context(), id, ports.UpdateTaskParams{
		Name:   body.Name,
		Body:   body.Body,
		Status: body.Status,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrTaskNotFound):
			api.responseHandler.SendErrorResponse(w, http.StatusNotFound, err)
		case errors.Is(err, domain.ErrInvalidStatusTransition), errors.Is(err, domain.ErrUnknownStatus):
			api.responseHandler.SendErrorResponse(w, http.StatusBadRequest, err)
		default:
			api.responseHandler.SendErrorResponse(w, http.StatusInternalServerError, err)
		}
		return
	}

	api.responseHandler.SendSuccessResponse(w, http.StatusOK, taskFromDomain(updated))
}

// DeleteTask removes a task by id.
//
//	@Summary	Delete task
//	@Tags		tasks
//	@Produce	json
//	@Param		id	path	uint64	true	"Task id"
//	@Success	204	"No Content"
//	@Failure	400	{object}	protocol.ExceptionResponse
//	@Failure	404	{object}	protocol.ExceptionResponse
//	@Failure	500	{object}	protocol.ExceptionResponse
//	@Router		/tasks/{id} [delete]
func (api *API) DeleteTask(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathID(r)
	if err != nil {
		api.responseHandler.SendErrorResponse(w, http.StatusBadRequest, err)
		return
	}

	err = api.taskService.Delete(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			api.responseHandler.SendErrorResponse(w, http.StatusNotFound, err)
			return
		}
		api.responseHandler.SendErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parsePathID(r *http.Request) (uint64, error) {
	params := server.RequestParams(r)
	idParam := params["id"]
	if len(idParam) == 0 {
		return 0, errors.New("task id is required")
	}
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		return 0, err
	}
	return id, nil
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
	var params dto.CreateTaskRequest
	err := api.responseHandler.BindJSON(r.Body, &params)
	if err != nil {
		api.responseHandler.SendErrorResponse(w, http.StatusBadRequest, err)
		return
	}

	id, err := api.taskService.Create(r.Context(), domain.NewTask(params.Name, params.Body))
	if err != nil {
		api.responseHandler.SendErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	response := dto.CreateTaskResponse{
		ID: id,
	}
	api.responseHandler.SendSuccessResponse(w, http.StatusOK, response)
}
