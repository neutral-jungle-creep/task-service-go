package server

import (
	"task-service/internal/ports"
	"task-service/pkg/http_server"
)

type Api struct {
	taskService ports.TaskService
}

func NewApi(taskService ports.TaskService) *Api {
	return &Api{
		taskService: taskService,
	}
}

func (api *Api) InitRoutes(routeGroup string) *http_server.Router {
	router := http_server.NewRouter()

	router.Register("GET", routeGroup+"/tasks", api.ListTasks)
	router.Register("GET", routeGroup+"/tasks/{id}", api.GetTask)
	router.Register("POST", routeGroup+"/tasks", api.CreateTask)

	return router
}
