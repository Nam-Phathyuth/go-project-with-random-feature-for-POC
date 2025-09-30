package model

import (
	"errors"
	"go-task/pkg"
	"time"
)

type Task struct {
	ID        int64
	Title     string
	Content   string
	Status    pkg.TaskStatus
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

func NewTask(title string, content string, status pkg.TaskStatus) (*Task, error) {
	if !validTitle(title) {
		return nil, errors.New("title cannot be empty")
	}

	if !status.IsValid() {
		return nil, errors.New("invalid status")
	}
	timestamp := time.Now()

	return &Task{
		Title:     title,
		Content:   content,
		Status:    status,
		CreatedAt: timestamp,
		UpdatedAt: timestamp,
		DeletedAt: nil}, nil
}

func (task *Task) UpdateTitle(title string) error {
	if !validTitle(title) {
		return errors.New("title cannot be empyty")
	}

	task.Title = title
	task.UpdatedAt = time.Now()
	return nil
}

func (task *Task) UpdateContent(content string) error {
	task.Content = content
	task.UpdatedAt = time.Now()
	return nil
}

func (task *Task) UpdateFrom(updateTask Task) (*Task, error) {
	if err := task.UpdateTitle(updateTask.Title); err != nil {
		return nil, err
	}
	if err := task.UpdateContent(task.Content); err != nil {
		return nil, err
	}
	task.UpdatedAt = time.Now()
	return task, nil
}

func validTitle(title string) bool {
	return title != ""
}
