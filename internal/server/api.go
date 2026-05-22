package server

import (
	"task-service/internal/ports"
	"task-service/pkg/http/server"
)

type API struct {
	taskService ports.TaskService
}

func NewAPI(taskService ports.TaskService) *API {
	return &API{
		taskService: taskService,
	}
}

func (api *API) InitRoutes(routeGroup string) *server.Router {
	router := server.NewRouter()

	router.Register("GET", routeGroup+"/tasks", api.ListTasks)
	router.Register("GET", routeGroup+"/tasks/{id}", api.GetTask)
	router.Register("POST", routeGroup+"/tasks", api.CreateTask)

	return router
}
