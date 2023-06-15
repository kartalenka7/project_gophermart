package handlers

import (
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

type server struct {
	service ServiceIntf
	log     *logrus.Logger
}

func NewRouter(service ServiceIntf, log *logrus.Logger) chi.Router {
	log.Info("Инициализируем роутер")
	router := chi.NewRouter()
	server := &server{
		service: service,
		log:     log}

	// маршрутизация запросов
	router.Route("/api/user", func(r chi.Router) {
		r.Use(gzipHandle)
		r.Post("/register", server.userRegstr)
		r.Post("/login", server.userAuth)
	})

	router.Route("/api/user/orders", func(r chi.Router) {
		r.Use(gzipHandle)
		r.Use(server.checkUserAuth)
		r.Post("/", server.addOrder)
		r.Get("/", server.getOrders)
	})

	router.Route("/api/user/balance", func(r chi.Router) {
		r.Use(gzipHandle)
		r.Use(server.checkUserAuth)
		r.Get("/", server.getBalance)
		r.Post("/withdraw", server.withdraw)
	})

	router.Route("/api/user/withdrawals", func(r chi.Router) {
		r.Use(gzipHandle)
		r.Use(server.checkUserAuth)
		r.Get("/", server.getWithdrawals)
	})

	return router
}
