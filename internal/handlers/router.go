package handlers

import (
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

type server struct {
	service ServiceInterface
	log     *logrus.Logger
}

func NewRouter(service ServiceInterface, log *logrus.Logger) chi.Router {
	log.Info("Инициализируем роутер")
	server := &server{
		service: service,
		log:     log}

	router := chi.NewRouter()

	// маршрутизация запросов
	router.Route("/api/user", func(r chi.Router) {
		r.Use(gzipHandle)
		r.Post("/register", server.userRegstr)
		r.Post("/login", server.userAuth)
	})

	router.Group(func(r chi.Router) {
		r.Use(gzipHandle)
		r.Use(server.checkUserAuth)
		r.Post("/api/user/orders", server.addOrder)
		r.Get("/api/user/orders", server.getOrders)
		r.Get("/api/user/balance", server.getBalance)
		r.Post("/api/user/balance/withdraw", server.withdraw)
		r.Get("/api/user/withdrawals", server.getWithdrawals)
	})

	return router
}
