package elastic

import (
	"log"

	"github.com/elastic/go-elasticsearch/v8"
)

type Elasticsearch struct {
}

func NewElasticsearch() *elasticsearch.Client {
	cfg := elasticsearch.Config{
		Addresses: []string{"http://127.0.0.1:9200"},
	}
	esClient, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Fatalf(err.Error())
	}

	log.Printf("connected to elasticsearch")
	return esClient
}
