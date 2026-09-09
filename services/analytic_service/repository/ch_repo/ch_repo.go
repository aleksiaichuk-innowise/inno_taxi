package ch_repo

import "github.com/ClickHouse/clickhouse-go/v2"

type ClickHouseRepository struct {
	conn clickhouse.Conn
}

func NewClickHouseRepository(conn clickhouse.Conn) *ClickHouseRepository {
	return &ClickHouseRepository{conn: conn}
}
