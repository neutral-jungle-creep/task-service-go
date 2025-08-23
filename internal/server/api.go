package server

import (
	"task-service/internal/ports"
	"task-service/pkg/http/server"
)

type Api struct {
	taskService ports.TaskService
}

func NewApi(taskService ports.TaskService) *Api {
	return &Api{
		taskService: taskService,
	}
}

func (api *Api) InitRoutes(routeGroup string) *server.Router {
	router := server.NewRouter()

	router.Register("GET", routeGroup+"/tasks", api.ListTasks)
	router.Register("GET", routeGroup+"/tasks/{id}", api.GetTask)
	router.Register("POST", routeGroup+"/tasks", api.CreateTask)

	return router
}
