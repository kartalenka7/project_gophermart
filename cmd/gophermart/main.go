package main

import (
	"context"
	"net/http"
	"time"

	"github.com/kartalenka7/project_gophermart/internal/config"
	"github.com/kartalenka7/project_gophermart/internal/handlers"
	"github.com/kartalenka7/project_gophermart/internal/logger"
	"github.com/kartalenka7/project_gophermart/internal/service"
	"github.com/kartalenka7/project_gophermart/internal/storage"
)

func main() {
	log := logger.InitLog()

	cfg, err := config.GetConfig(log)
	if err != nil {
		log.Error(err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	storage, err := storage.NewStorage(ctx, cfg.Database, log)
	if err != nil {
		return
	}
	service := service.NewService(ctx, storage, log, cfg.AccrualSys)
	router := handlers.NewRouter(service, log)

	err = http.ListenAndServe(cfg.Server, router)
	if err != nil {
		log.Error(err.Error())
		return
	}
	defer storage.Close()
}

var _ service.Storer = &storage.DBStruct{}
