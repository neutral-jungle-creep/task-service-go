package repositories

type TaskRepository struct {
}

func NewTaskRepository() *TaskRepository {
	return &TaskRepository{}
}

func (r *TaskRepository) Store() {}
