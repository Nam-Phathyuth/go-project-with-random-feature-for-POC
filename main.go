package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/elastic/go-elasticsearch/v8"
	"go-task/internal/dao"
	"go-task/internal/db"
	"go-task/internal/elastic"
	"go-task/internal/model"
	"go-task/internal/service"
	"go-task/internal/template"
	"go-task/pkg"
	"go-task/pkg/request"
	"go-task/pkg/response"
	"log"
	"log/slog"
	"net/http"
	"strconv"
)

var (
	mysqlDb     *db.MysqlDB
	dbInst      *sql.DB
	taskChannel chan *model.Task
	storage     *dao.MysqlStore
	serviceInst *service.Service
	controller  *Controller
	esClient    *elasticsearch.Client
	_           *elastic.ElasticsearchSync
	router      *http.ServeMux
)

func init() {
	taskChannel = make(chan *model.Task, 200)
	log.Printf("initializing database")
	mysqlDb = &db.MysqlDB{}
	dbInst = mysqlDb.Init()
	storage = dao.NewMysqlStore(dbInst, taskChannel)

	log.Printf("initializing task service")
	serviceInst = service.NewService(storage)
	log.Printf("initializing task controller")
	controller = NewController(serviceInst)
	log.Printf("initializing elasticsearch")
	esClient = elastic.NewElasticsearch()
	_ = elastic.NewElasticsearchSync(esClient, taskChannel, dbInst)
	router = initRouter()
}

func renderIndex(service Service) []*model.Task {
	tasks, _ := serviceInst.FindAll()
	return tasks
}

func initRouter() *http.ServeMux {
	log.Println("init router")
	router := http.NewServeMux()
	//router.Handle("/", templ.Handler(template.Index(renderIndex(serviceInst))))
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tasks, _ := serviceInst.FindAll()
		w.Header().Set("Cache-Control", "no-cache")
		err := template.Index(tasks).Render(context.Background(), w)
		if err != nil {
			return
		}
	})
	router.Handle("GET /{id}", taskHandler(taskByIDHandler))
	router.HandleFunc("DELETE /{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		slog.Info("deleting task", "id", r.PathValue("id"))
		dbInst.Exec("DELETE FROM tasks WHERE id = ?", r.PathValue("id"))
		return
	})
	router.HandleFunc("PUT /{id}/{status}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		slog.Info("updating task", "id", r.PathValue("id"), "status", r.PathValue("status"))
		dbInst.Exec("UPDATE tasks SET status = ? WHERE id = ?", r.PathValue("status"), r.PathValue("id"))
		id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
		task, _ := serviceInst.FindById(id)
		template.UpdateTask(*task).Render(context.Background(), w)
	})
	router.HandleFunc("PUT /{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		slog.Info("updating task", "id", r.PathValue("id"), "title", r.FormValue("title"))
		dbInst.Exec("UPDATE tasks SET title = ? WHERE id = ?", r.FormValue("title"), r.PathValue("id"))
		id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
		task, _ := serviceInst.FindById(id)
		template.UpdateTask(*task).Render(context.Background(), w)
	})
	return router
}

type HttpErr struct {
	Err  error  `json:"error"`
	Code int    `json:"code"`
	Msg  string `json:"message"`
}

func (e HttpErr) Error() string { return e.Msg }

type taskHandler func(w http.ResponseWriter, r *http.Request) error

func (th taskHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := th(w, r)
	if err != nil {
		httpErr := HttpErr{
			Err:  err,
			Code: http.StatusInternalServerError,
			Msg:  "Internal Server Error",
		}
		httpErrJson, _ := json.Marshal(httpErr)
		w.Header().Set("Content-Type", "Application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(httpErrJson)
	}
}
func main() {
	defer close(taskChannel)
	defer func(dbInst *sql.DB) {
		err := dbInst.Close()
		if err != nil {
			log.Printf("db conn closed")
		}
	}(dbInst)
	log.Fatal(http.ListenAndServe(":7000", router))
}

func taskByIDHandler(w http.ResponseWriter, r *http.Request) error {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Printf(err.Error())
		return err
	}
	task, err := serviceInst.FindById(id)
	if err != nil {
		if errors.Is(err, pkg.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
			log.Printf(err.Error())
			return err
		}
		log.Printf(err.Error())
		return err
	}
	taskRs, err := mapToTaskRes(*task)
	if err != nil {
		log.Printf(err.Error())
		return err
	}
	jsonData, err := json.Marshal(taskRs)
	if err != nil {
		return fmt.Errorf("error serialize task: %w", err)
	}

	w.Header().Set("Content-Type", "Application/jsonData")
	_, err = w.Write([]byte(jsonData))
	if err != nil {
		return err
	}
	return nil
}

type Service interface {
	Create(task *model.Task) (*model.Task, error)
	Update(task model.Task, id int64) (*model.Task, error)
	Delete(int64) error
	FindAll() ([]*model.Task, error)
	FindById(int64) (*model.Task, error)
}

type Controller struct {
	service Service
}

func NewController(service Service) *Controller {
	return &Controller{
		service: service,
	}
}

func (controller *Controller) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		var tasksRS []*response.TaskResponse
		tasks, _ := controller.service.FindAll()
		for _, t := range tasks {
			task, err := mapToTaskRes(*t)
			if err != nil {
				return
			}
			tasksRS = append(tasksRS, task)
		}
		json, _ := json.Marshal(tasksRS)
		w.Header().Set("Content-Type", "Application/json")
		w.Write(json)
		return
	case http.MethodPost:
		var taskReq request.TaskRequest
		err := json.NewDecoder(r.Body).Decode(&taskReq)

		if err != nil {
			w.Write([]byte(err.Error()))
			fmt.Errorf(err.Error())
			return
		}

		task, err := mapToTask(taskReq)
		if err != nil {
			http.Error(w, err.Error(), 400)
			fmt.Errorf(err.Error())
			return
		}
		createdTask, err := controller.service.Create(task)
		if err != nil {
			w.Write([]byte(err.Error()))
			log.Printf(err.Error())
			return
		}
		res, _ := mapToTaskRes(*createdTask)
		jsonData, err := json.Marshal(res)
		if err != nil {
			w.Write([]byte(err.Error()))
			fmt.Errorf(err.Error())
			return
		}
		w.Header().Set("Content-Type", "Application/jsonData")
		w.Write([]byte(jsonData))
		return
	default:
		w.Write([]byte("method not allowed"))
	}
}

func mapToTask(req request.TaskRequest) (*model.Task, error) {
	task, err := model.NewTask(req.Title, req.Content, pkg.TaskStatus(req.Status))
	if err != nil {
		return nil, err
	}

	return task, nil
}

func mapToTaskRes(task model.Task) (*response.TaskResponse, error) {
	res := &response.TaskResponse{
		ID:        task.ID,
		Title:     task.Title,
		Content:   task.Content,
		Status:    string(task.Status),
		CreatedAt: task.CreatedAt,
		UpdatedAt: task.UpdatedAt,
	}

	return res, nil
}
