package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/elastic/go-elasticsearch/v8"
	"go-task/internal/dao"
	"go-task/internal/db"
	"go-task/internal/elastic"
	"go-task/internal/model"
	"go-task/internal/service"
	"go-task/pkg"
	"go-task/pkg/request"
	"go-task/pkg/response"
	"log"
	"net/http"
	"strconv"
)

var (
	mysqlDb     *db.MysqlDB
	dbInst      *sql.DB // Assuming db.Init() returns *db.DBConnection
	taskChannel chan *model.Task
	storage     *dao.MysqlStore
	serviceInst *service.Service
	controller  *Controller
	esClient    *elasticsearch.Client
	_           *elastic.ElasticsearchSync
)

func init() {
	taskChannel = make(chan *model.Task)
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
	_ = elastic.NewElasticsearchSync(esClient, taskChannel)
}

func main() {
	router := http.NewServeMux()
	defer close(taskChannel)
	defer func(dbInst *sql.DB) {
		err := dbInst.Close()
		if err != nil {
			log.Printf("db conn closed")
		}
	}(dbInst)

	router.Handle("/", controller)
	router.HandleFunc("GET /{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("invalid id"))
			log.Printf(err.Error())
			return
		}
		task, err := serviceInst.FindById(id)
		if err != nil {
			w.Write([]byte(err.Error()))
			log.Printf(err.Error())
			return
		}

		taskRs, err := mapToTaskRes(*task)
		if err != nil {
			w.Write([]byte(err.Error()))
			log.Printf(err.Error())
			return
		}
		jsonData, _ := json.Marshal(taskRs)

		w.Header().Set("Content-Type", "Application/jsonData")
		w.Write([]byte(jsonData))
		return
	})

	log.Fatal(http.ListenAndServe(":7000", router))
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
		tasks, err := controller.service.FindAll()
		if err != nil {
			w.Write([]byte(err.Error()))
			fmt.Errorf(err.Error())
			return
		}

		for _, t := range tasks {
			task, err := mapToTaskRes(*t)
			if err != nil {
				w.Write([]byte(err.Error()))
				fmt.Errorf(err.Error())
				return
			}
			tasksRS = append(tasksRS, task)
		}

		json, err := json.Marshal(tasksRS)
		if err != nil {
			w.Write([]byte(err.Error()))
			fmt.Errorf(err.Error())
			return
		}

		w.Header().Set("Content-Type", "Application/jsonData")
		w.Write([]byte(json))
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
