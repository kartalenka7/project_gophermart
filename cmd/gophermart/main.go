package main

import (
	"net/http"

	"github.com/kartalenka7/project_gophermart/internal/config"
	"github.com/kartalenka7/project_gophermart/internal/handlers"
	"github.com/kartalenka7/project_gophermart/internal/service"
	"github.com/kartalenka7/project_gophermart/internal/storage"
)

func main() {
	log := config.InitLog()

	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err.Error())
		return
	}
	storage, err := storage.NewStorage(cfg.Database, log)
	if err != nil {
		return
	}
	service := service.NewService(storage, log)
	router := handlers.NewRouter(service)

	http.ListenAndServe(cfg.Server, router)
	defer storage.Close()
}

var _ service.Storer = &storage.DBStruct{}
