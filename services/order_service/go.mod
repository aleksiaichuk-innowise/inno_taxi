module github.com/aleksiaichuk-innowise/inno_taxi/services/order_service

go 1.26.1

replace github.com/aleksiaichuk-innowise/inno_taxi/shared => ../../shared

require (
	github.com/aleksiaichuk-innowise/inno_taxi/shared v0.0.0-00010101000000-000000000000
	github.com/elastic/go-elasticsearch/v8 v8.19.7
	github.com/jackc/pgx/v5 v5.10.0
	github.com/joho/godotenv v1.5.1
)

require (
	github.com/elastic/elastic-transport-go/v8 v8.9.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	go.opentelemetry.io/otel v1.29.0 // indirect
	go.opentelemetry.io/otel/metric v1.29.0 // indirect
	go.opentelemetry.io/otel/trace v1.29.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/text v0.41.0 // indirect
)
