package service

import (
	"go-task/internal/model"
	"time"
)

type DataStore interface {
	Save(task *model.Task) (*model.Task, error)
	FindById(id int64) (*model.Task, error)
	FindAll() ([]*model.Task, error)
}

type Service struct {
	datastore DataStore
}

func NewService(datastore DataStore) *Service {
	return &Service{
		datastore: datastore,
	}
}

func (service *Service) Create(task *model.Task) (*model.Task, error) {

	savedTask, err := service.datastore.Save(task)
	if err != nil {
		return nil, err
	}
	return savedTask, nil
}

func (service *Service) Update(task model.Task, id int64) (*model.Task, error) {
	oldTask, err := service.datastore.FindById(id)
	if err != nil {
		return nil, err
	}

	oldTask, err = oldTask.UpdateFrom(task)
	if err != nil {
		return nil, err
	}

	savedTask, err := service.datastore.Save(oldTask)
	if err != nil {
		return nil, err
	}

	return savedTask, nil
}

func (service *Service) Delete(id int64) error {
	task, err := service.datastore.FindById(id)
	if err != nil {
		return err
	}
	now := time.Now()
	task.DeletedAt = &now
	_, err = service.datastore.Save(task)
	if err != nil {
		return err
	}
	return nil
}

func (service *Service) FindAll() ([]*model.Task, error) {
	return service.datastore.FindAll()
}

func (service *Service) FindById(id int64) (*model.Task, error) {
	task, err := service.datastore.FindById(id)

	if err != nil {
		return nil, err
	}

	return task, nil
}
