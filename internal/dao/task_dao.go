package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	db "go-task/internal/db/go-task"
	"go-task/internal/model"
	"go-task/pkg"
	"log"
)

const tableName string = "tasks"

type MysqlStore struct {
	db       *sql.DB
	taskChan chan *model.Task
}

func NewMysqlStore(db *sql.DB, taskChan chan *model.Task) *MysqlStore {
	return &MysqlStore{
		db:       db,
		taskChan: taskChan,
	}
}

func (mysql *MysqlStore) Save(task *model.Task) (*model.Task, error) {
	query := fmt.Sprintf("INSERT INTO %s (title, content, status) VALUES (?, ?, ?)", tableName)
	sqlc := db.New(mysql.db)
	_, err := sqlc.SaveTask(
		context.Background(),
		db.SaveTaskParams{
			Title:   task.Title,
			Content: sql.NullString{String: task.Content},
			Status:  db.TasksStatus(string(task.Status)),
		})
	if err != nil {
		return nil, err
	}
	inserted, err := mysql.db.Exec(query, task.Title, task.Content, task.Status)

	if err != nil {
		return nil, &pkg.TaskError{Message: fmt.Sprintf("Failed to insert task: %s", err.Error()), Err: err}
	}
	task.ID, err = inserted.LastInsertId()
	if err != nil {
		return nil, &pkg.TaskError{Message: fmt.Sprintf("Failed to insert task: %s", err.Error()), Err: err}
	}
	mysql.taskChan <- task
	return task, nil
}

func (mysql *MysqlStore) FindById(id int64) (*model.Task, error) {
	var task model.Task
	query := "select * from tasks where id = ?"
	result := mysql.db.QueryRow(query, id)

	err := result.Scan(&task.ID, &task.Title, &task.Content, &task.Status, &task.CreatedAt, &task.UpdatedAt, &task.DeletedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("task not found", &pkg.ErrNotFound)
		}
		return nil, &pkg.TaskError{Message: fmt.Sprintf("Data Access Error: %s", err.Error()), Err: err}
	}

	return &task, nil
}

func (mysql *MysqlStore) FindAll() ([]*model.Task, error) {
	q := fmt.Sprintf("select * from %s where deleted_at is null", tableName)
	results, err := mysql.db.Query(q)
	if err != nil {
		return nil, &pkg.TaskError{Message: fmt.Sprintf("Failed to insert task: %s", err.Error()), Err: err}
	}
	defer func(results *sql.Rows) {
		err := results.Close()
		if err != nil {
			log.Println(err)
		}

	}(results)

	var tasks []*model.Task
	for results.Next() {
		var task model.Task
		err := results.Scan(&task.ID, &task.Title, &task.Content, &task.Status, &task.CreatedAt, &task.UpdatedAt, &task.DeletedAt)
		if err != nil {
			return nil, &pkg.TaskError{Message: fmt.Sprintf("Failed to insert task: %s", err.Error()), Err: err}
		}
		tasks = append(tasks, &task)
	}
	if err = results.Err(); err != nil {
		return nil, &pkg.TaskError{Message: fmt.Sprintf("Failed to insert task: %s", err.Error()), Err: err}
	}

	return tasks, nil
}
