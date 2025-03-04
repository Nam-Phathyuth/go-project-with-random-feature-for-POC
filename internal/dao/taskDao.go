package dao

import (
	"database/sql"
	"errors"
	"fmt"
	"go-task/internal/model"
	"go-task/pkg/taskError"
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

	inserted, err := mysql.db.Exec(query, task.Title, task.Content, task.Status)

	if err != nil {
		return nil, err
	}
	task.ID, _ = inserted.LastInsertId()
	mysql.taskChan <- task
	return task, nil

}

func (mysql *MysqlStore) FindById(id int64) (*model.Task, error) {
	var task model.Task
	query := "select * from tasks where id = ?"
	result := mysql.db.QueryRow(query, id)

	err := result.Scan(&task.ID, &task.Title, &task.Content, &task.Status, &task.CreatedAt, &task.UpdatedAt, &task.DeletedAt)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return nil, &taskError.ErrorNotFound{}
	}

	return &task, nil
}

func (mysql *MysqlStore) FindAll() ([]*model.Task, error) {
	q := fmt.Sprintf("select * from %s where deleted_at is null", tableName)
	results, err := mysql.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer results.Close()

	var tasks []*model.Task
	for results.Next() {
		var task model.Task
		err := results.Scan(&task.ID, &task.Title, &task.Content, &task.Status, &task.CreatedAt, &task.UpdatedAt, &task.DeletedAt)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, &task)
	}
	if err = results.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}
