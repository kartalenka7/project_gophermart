package handlers

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/kartalenka7/project_gophermart/internal/model"
)

// интерфейс для взаимодействия с сервисом
type ServiceIntf interface {
	RgstrUser(ctx context.Context, user model.User) (string, error)
	AuthUser(ctx context.Context, user model.User) (string, error)
	AddUserOrder(ctx context.Context, number string, cookie string) error
	GetUserOrders(user model.User) error
	ParseUserCredentials(r *http.Request) (model.User, error)
	WriteWithdraw(withdraw model.OrderWithdraw)
}

func (s *server) userRegstr(rw http.ResponseWriter, r *http.Request) {
	// проверяем запрос, парсим логин и пароль
	user, err := s.service.ParseUserCredentials(r)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	cookieVal, err := s.service.RgstrUser(r.Context(), user)
	if err != nil {
		// логин уже существует
		if errors.Is(err, model.ErrLoginExists) {
			rw.WriteHeader(http.StatusConflict)
			return
		}
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	// аутентификация пользователя
	cookie := http.Cookie{
		Name:  "UserAuth",
		Value: cookieVal}
	r.AddCookie(&cookie)
	rw.WriteHeader(http.StatusOK)
}

func (s *server) userAuth(rw http.ResponseWriter, r *http.Request) {
	user, err := s.service.ParseUserCredentials(r)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	cookieVal, err := s.service.AuthUser(r.Context(), user)
	if err != nil {
		// Неверные логин или пароль
		if errors.Is(err, model.ErrAuthFailed) {
			rw.WriteHeader(http.StatusUnauthorized)
			return
		}
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	// аутентификация пользователя
	cookie := http.Cookie{
		Name:  "UserAuth",
		Value: cookieVal}
	r.AddCookie(&cookie)
	rw.WriteHeader(http.StatusOK)
}

func (s *server) addOrder(rw http.ResponseWriter, r *http.Request) {

	// получить номер заказа из body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Add order handler| %v", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()
	number := string(body)

	// проверить формат запроса
	if r.Header.Get("Content-Type") != "text/plain" {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	cookie, err := r.Cookie("UserAuth")
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = s.service.AddUserOrder(r.Context(), number, cookie.Value)
	if err != nil {
		if errors.Is(err, model.ErrOrderExistsSameUser) {
			rw.WriteHeader(http.StatusOK)
			return
		}
		if errors.Is(err, model.ErrOrderExistsDiffUser) {
			rw.WriteHeader(http.StatusConflict)
			return
		}
		if errors.Is(err, model.ErrNotValidOrderNumber) {
			rw.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		log.Printf("Add order handler| %v", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	rw.WriteHeader(http.StatusAccepted)
}

func (s *server) getOrders(rw http.ResponseWriter, r *http.Request) {
	var user model.User
	// проверить авторизацию пользователя

	s.service.GetUserOrders(user)
}

func (s *server) withdraw(rw http.ResponseWriter, r *http.Request) {
	var withdraw model.OrderWithdraw
	//проверить авторизацию пользователя

	// передать номер заказа и число баллов для списания
	s.service.WriteWithdraw(withdraw)
}

func (s *server) getWithdrawals(rw http.ResponseWriter, r *http.Request) {

}

func (s *server) getBalance(rw http.ResponseWriter, r *http.Request) {

}
