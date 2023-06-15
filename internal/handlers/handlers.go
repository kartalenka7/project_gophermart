package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/dgrijalva/jwt-go"
	"github.com/kartalenka7/project_gophermart/internal/model"
)

// интерфейс для взаимодействия с сервисом
type ServiceIntf interface {
	RgstrUser(ctx context.Context, user model.User) error
	AuthUser(ctx context.Context, user model.User) error
	AddUserOrder(ctx context.Context, number string) error
	GetUserOrders(ctx context.Context) ([]model.OrdersResponse, error)
	ParseUserCredentials(r *http.Request) (model.User, error)
	WriteWithdraw(withdraw model.OrderWithdraw)
}

func (s server) userRegstr(rw http.ResponseWriter, r *http.Request) {
	s.log.Info("Хэндлер регистрация пользователя")
	// проверяем запрос, парсим логин и пароль
	user, err := s.service.ParseUserCredentials(r)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	err = s.service.RgstrUser(r.Context(), user)
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
	//Создать новый токен JWT для новой зарегистрированной учётной записи
	tk := &model.Token{Login: user.Login}
	token := jwt.NewWithClaims(jwt.GetSigningMethod("HS256"), tk)
	tokenString, _ := token.SignedString([]byte("secret"))

	rw.Header().Add("Authorization", tokenString)

	s.log.Info("Пользователь успешно зарегистрирован и аутентифицирован")
	rw.WriteHeader(http.StatusOK)
}

func (s server) userAuth(rw http.ResponseWriter, r *http.Request) {
	user, err := s.service.ParseUserCredentials(r)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	err = s.service.AuthUser(r.Context(), user)
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
	tk := &model.Token{Login: user.Login}
	token := jwt.NewWithClaims(jwt.GetSigningMethod("HS256"), tk)
	tokenString, _ := token.SignedString([]byte("secret"))

	rw.Header().Add("Authorization", tokenString)

	s.log.Info("Пользователь успешно аутентифицирован")
	rw.WriteHeader(http.StatusOK)
}

func (s server) addOrder(rw http.ResponseWriter, r *http.Request) {

	s.log.Info("Добавить заказ")
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
		s.log.Error("Неверный Content-Type")
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	err = s.service.AddUserOrder(r.Context(), number)
	if err != nil {
		//номер заказа уже был зарегистрирован текущим пользователем
		if errors.Is(err, model.ErrOrderExistsSameUser) {
			rw.WriteHeader(http.StatusOK)
			return
		}
		// номер заказа был зарегистрирова другим пользователем
		if errors.Is(err, model.ErrOrderExistsDiffUser) {
			rw.WriteHeader(http.StatusConflict)
			return
		}
		// неверный формат номера заказа
		if errors.Is(err, model.ErrNotValidOrderNumber) {
			rw.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	rw.WriteHeader(http.StatusAccepted)
}

func (s server) getOrders(rw http.ResponseWriter, r *http.Request) {
	s.log.Info("Получение списка заказов")
	orders, err := s.service.GetUserOrders(r.Context())
	if err != nil {
		rw.WriteHeader(http.StatusNoContent)
		return
	}

	// пишем в тело ответа закодированный в JSON объект
	buf := bytes.NewBuffer([]byte{})
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	err = encoder.Encode(orders)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	s.log.Info("Список заказов успешно возвращен")
	fmt.Fprint(rw, buf)

	// устанавливаем заголовок Content-Type
	// для передачи клиенту информации, кодированной в JSO
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)

}

func (s server) withdraw(rw http.ResponseWriter, r *http.Request) {
	var withdraw model.OrderWithdraw
	//проверить авторизацию пользователя

	// передать номер заказа и число баллов для списания
	s.service.WriteWithdraw(withdraw)
}

func (s server) getWithdrawals(rw http.ResponseWriter, r *http.Request) {

}

func (s server) getBalance(rw http.ResponseWriter, r *http.Request) {

}
