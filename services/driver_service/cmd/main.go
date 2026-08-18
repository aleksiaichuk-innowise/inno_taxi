package main

import (
	"log"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/app"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/config"
)

func main() {
	cfg := config.Load()
	if err := app.Run(cfg); err != nil {
		log.Fatal(err)
	}
}
