module github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service

go 1.26.1

require (
	github.com/aleksiaichuk-innowise/inno_taxi/shared v0.0.0-00010101000000-000000000000
	github.com/joho/godotenv v1.5.1
	go.mongodb.org/mongo-driver/v2 v2.8.0
)

replace github.com/aleksiaichuk-innowise/inno_taxi/shared => ../../shared

require (
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/klauspost/compress v1.17.6 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.2.0 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	golang.org/x/crypto v0.48.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/text v0.34.0 // indirect
)
