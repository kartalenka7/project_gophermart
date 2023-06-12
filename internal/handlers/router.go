package handlers

import (
	"github.com/go-chi/chi/v5"
)

type server struct {
	service ServiceIntf
}

func NewRouter(service ServiceIntf) chi.Router {
	router := chi.NewRouter()
	server := &server{service: service}

	// маршрутизация запросов
	router.Route("/api/user/", func(r chi.Router) {
		r.Use(gzipHandle)
		r.Post("/api/user/register", server.userRegstr)
		r.Post("/api/user/login", server.userAuth)

		router.Route("/api/user/orders", func(r chi.Router) {
			r.Use(checkUserAuth)
			r.Post("/api/user/orders", server.addOrder)
			r.Get("/api/user/orders", server.getOrders)
		})

		router.Route("/api/user/balance", func(r chi.Router) {
			r.Use(checkUserAuth)
			r.Get("/api/user/balance", server.getBalance)
			r.Post("/api/user/balance/withdraw", server.withdraw)
		})

		router.Route("/api/user/withdrawals", func(r chi.Router) {
			r.Use(checkUserAuth)
			r.Get("/api/user/withdrawals", server.getWithdrawals)
		})

	})

	return router
}
