package es_repo

import "github.com/elastic/go-elasticsearch/v8"

type EsRepository struct {
	client *elasticsearch.Client
	index  string
}

func NewEsRepo(client *elasticsearch.Client, index string) EsRepository {
	return EsRepository{client: client, index: index}
}
