package elastic

import (
	"bytes"
	"encoding/json"
	"go-task/internal/model"
	"log"
	"strconv"
	"time"

	elasticsearch "github.com/elastic/go-elasticsearch/v8"
)

const idxName string = "task-idx"

type TaskDoc struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ElasticsearchSync struct {
	esClient *elasticsearch.Client
	taskChan chan *model.Task
}

func NewElasticsearchSync(esClient *elasticsearch.Client, taskChan chan *model.Task) *ElasticsearchSync {
	es := ElasticsearchSync{
		esClient: esClient,
		taskChan: taskChan,
	}
	go es.startWorker()
	return &es
}

func (es *ElasticsearchSync) startWorker() {
	for {
		task, ok := <-es.taskChan
		if !ok {
			log.Printf("channel closed elasticsearch sync worker exiting")
			return
		}
		taskDoc := TaskDoc{
			ID:        task.ID,
			Title:     task.Title,
			Content:   task.Content,
			Status:    string(task.Status),
			CreatedAt: task.CreatedAt,
			UpdatedAt: task.UpdatedAt,
		}
		taskJson, _ := json.Marshal(taskDoc)
		_, err := es.esClient.Index(
			idxName,
			bytes.NewReader(taskJson),
			es.esClient.Index.WithDocumentID(strconv.FormatInt(task.ID, 10)),
		)
		if err != nil {
			log.Println(err.Error())
		}
	}
}
