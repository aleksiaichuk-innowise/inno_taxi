package elastic

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/aleksiaichuk-innowise/inno_taxi/shared/config"
	"github.com/elastic/go-elasticsearch/v8"
)

func NewESClient(ctx context.Context, cfg config.ElasticsearchConfig) (*elasticsearch.Client, error) {
	transport := &http.Transport{
		MaxIdleConnsPerHost:   cfg.MaxIdleConnsPerHost,
		ResponseHeaderTimeout: cfg.Timeout,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	esCfg := elasticsearch.Config{
		Addresses: cfg.Addresses,
		Username:  cfg.Username,
		Password:  cfg.Password,
		APIKey:    cfg.APIKey,
		Transport: transport,
	}

	client, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		return nil, err
	}

	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	res, err := client.Info(client.Info.WithContext(pingCtx))
	if err != nil {
		return nil, fmt.Errorf("cluster ElasticSearch is unavailable: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("status err ES Info: %s", res.Status())
	}

	return client, nil
}
