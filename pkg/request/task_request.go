package request

import "go-task/pkg"

type TaskRequest struct {
	Title   string         `json:"title"`
	Content string         `json:"content"`
	Status  pkg.TaskStatus `json:"status"`
}
