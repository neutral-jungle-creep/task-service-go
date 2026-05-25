package server

import (
	"io"
	"net/http"

	"task-service/internal/ports"
	"task-service/pkg/http/server"
)

type ResponseHandler interface {
	SendErrorResponse(w http.ResponseWriter, status int, err error)
	SendSuccessResponse(w http.ResponseWriter, status int, body any)
	BindJSON(body io.ReadCloser, params any) error
	Validate(params any) error
}

type API struct {
	responseHandler ResponseHandler
	taskService     ports.TaskService
}

func NewAPI(responseHandler ResponseHandler, taskService ports.TaskService) *API {
	return &API{
		responseHandler: responseHandler,
		taskService:     taskService,
	}
}

func (api *API) InitRoutes(routeGroup string) *server.Router {
	router := server.NewRouter()

	router.Register("GET", routeGroup+"/tasks", api.ListTasks)
	router.Register("GET", routeGroup+"/tasks/{id}", api.GetTask)
	router.Register("POST", routeGroup+"/tasks", api.CreateTask)
	router.Register("PATCH", routeGroup+"/tasks/{id}", api.UpdateTask)
	router.Register("DELETE", routeGroup+"/tasks/{id}", api.DeleteTask)

	return router
}
