package elastic

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"go-task/internal/model"
	"log"
	"strconv"
	"time"

	elasticsearch "github.com/elastic/go-elasticsearch/v8"
)

const idxName string = "task-idx"
const deadLetterTableName string = "dead_letter_tasks"

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
	db       *sql.DB
}

func NewElasticsearchSync(esClient *elasticsearch.Client, taskChan chan *model.Task, db *sql.DB) *ElasticsearchSync {
	es := ElasticsearchSync{
		esClient: esClient,
		taskChan: taskChan,
		db:       db,
	}
	go es.startWorker()
	go es.runReplayDeadLetter()
	return &es
}
func (es *ElasticsearchSync) runReplayDeadLetter() {
	log.Println("starting dead letter replay")
	ticker := time.NewTicker(30 * time.Second)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				log.Println("resynchronise task")
				es.replayDeadLetter()
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}
func (es *ElasticsearchSync) startWorker() {
	log.Println("starting elasticsearch sync worker")
	const maxRetry = 3
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
			for i := 0; i < maxRetry; i++ {
				_, err = es.esClient.Index(
					idxName,
					bytes.NewReader(taskJson),
					es.esClient.Index.WithDocumentID(strconv.FormatInt(task.ID, 10)),
				)
				if err == nil {
					break
				}
				log.Printf("Retry %d: %s", i+1, err.Error())
				time.Sleep(2 * time.Second)
			}
			if err != nil {
				log.Println("Failed to index document after retries:", err.Error())
				log.Println("sending task to dead letter queue")
				es.storeDeadLetter(task, err.Error())
			}
		}
	}
}
func (es *ElasticsearchSync) storeDeadLetter(task *model.Task, errorMsg string) {
	query := fmt.Sprintf(`INSERT INTO %s (task_id, payload, error_msg, retry_count) VALUES (?, ?, ?, ?)`, deadLetterTableName)
	taskJson, _ := json.Marshal(task)

	_, err := es.db.Exec(query, task.ID, string(taskJson), errorMsg, 3)
	if err != nil {
		log.Println("Failed to insert into dead letter queue:", err)
	}
}

func (es *ElasticsearchSync) replayDeadLetter() {
	query := fmt.Sprintf("SELECT id, task_id, payload FROM %s LIMIT 100", deadLetterTableName)
	results, err := es.db.Query(query)
	if err != nil {
		log.Println("Failed to query dead letter queue:", err)
		return
	}
	defer func(results *sql.Rows) {
		err := results.Close()
		if err != nil {
			log.Println("Failed to close dead letter queue:", err)
		}
	}(results)

	for results.Next() {
		var id int64
		var taskID int64
		var payload []byte
		err = results.Scan(&id, &taskID, &payload)
		if err != nil {
			log.Println("Failed to scan dead letter row:", err)
			continue
		}
		task := &model.Task{}
		err = json.Unmarshal(payload, task)
		if err != nil {
			log.Println("Failed to unmarshal task from dead letter queue:", err)
			continue
		}
		task.ID = taskID
		es.taskChan <- task
		deleteQuery := fmt.Sprintf("DELETE FROM %s WHERE id = ?", deadLetterTableName)
		_, err = es.db.Exec(deleteQuery, id)
		if err != nil {
			log.Println("Failed to delete from dead letter queue:", err)
		}
	}
	if err = results.Err(); err != nil {
		log.Println("Failed to iterate over dead letter queue:", err)
	}
}
