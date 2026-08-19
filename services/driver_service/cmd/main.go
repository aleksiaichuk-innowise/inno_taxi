package main

import (
	"log/slog"
	"os"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/app"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/config"
)

func main() {
	cfg := config.Load()
	if err := app.Run(cfg); err != nil {
		slog.Error("Failed to run driver_service", "error", err)
		os.Exit(1)
	}
}
