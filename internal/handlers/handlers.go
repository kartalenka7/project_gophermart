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

	"github.com/kartalenka7/project_gophermart/internal/model"
	"github.com/kartalenka7/project_gophermart/internal/utils"
	"github.com/sirupsen/logrus"
)

// интерфейс для взаимодействия с сервисом
//go:generate mockery --name ServiceInterface --with-expecter
type ServiceInterface interface {
	RgstrUser(ctx context.Context, user model.User) error
	AuthUser(ctx context.Context, user model.User) error
	AddUserOrder(ctx context.Context, number string, login string) error
	GetUserOrders(ctx context.Context, login string) ([]model.OrdersResponse, error)
	ParseUserCredentials(r *http.Request) (model.User, error)
	WriteWithdraw(ctx context.Context, withdraw model.OrderWithdraw, login string) error
	GetBalance(ctx context.Context, login string) (model.Balance, error)
	GetWithdrawals(ctx context.Context, login string) ([]model.OrderWithdraw, error)
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
	if err = utils.AddAuthoriztionHeader(rw, user); err != nil {
		s.log.Error(err.Error())
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	s.log.Info("Пользователь успешно зарегистрирован и аутентифицирован")
	rw.WriteHeader(http.StatusOK)
}

func (s server) userAuth(rw http.ResponseWriter, r *http.Request) {
	user, err := s.service.ParseUserCredentials(r)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	s.log.WithFields(logrus.Fields{
		"user": user.Login}).Info("Аутентификация пользователя")

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
	if err = utils.AddAuthoriztionHeader(rw, user); err != nil {
		s.log.Error(err.Error())
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

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

	login, ok := r.Context().Value(model.KeyLogin).(string)
	if !ok {
		s.log.Error(model.ErrCastingType)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = s.service.AddUserOrder(r.Context(), number, login)
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

	login, ok := r.Context().Value(model.KeyLogin).(string)
	if !ok {
		s.log.Error(model.ErrCastingType.Error())
		rw.WriteHeader(http.StatusInternalServerError)
	}

	orders, err := s.service.GetUserOrders(r.Context(), login)
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

	// устанавливаем заголовок Content-Type
	// для передачи клиенту информации, кодированной в JSO
	rw.Header().Add("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)

	s.log.Info("Список заказов успешно возвращен")
	fmt.Fprint(rw, buf)
}

func (s server) withdraw(rw http.ResponseWriter, r *http.Request) {
	var withdraw model.OrderWithdraw

	s.log.Info("Попытка списания средств")

	// проверить формат запроса
	if r.Header.Get("Content-Type") != "application/json" {
		s.log.Error("Неверный Content-Type")
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	// получаем из body сумму списания и номер заказа
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&withdraw); err != nil {
		s.log.Error(err.Error())
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	login, ok := r.Context().Value(model.KeyLogin).(string)
	if !ok {
		s.log.Error(model.ErrCastingType.Error())
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := s.service.WriteWithdraw(r.Context(), withdraw, login); err != nil {
		if errors.Is(err, model.ErrNotValidOrderNumber) {
			//422 — неверный номер заказа;
			rw.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		if errors.Is(err, model.ErrInsufficientBalance) {
			//402 — на счету недостаточно средств
			rw.WriteHeader(http.StatusPaymentRequired)
			return
		}
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	s.log.Info("Списание произошло")
	rw.WriteHeader(http.StatusOK)
}

func (s server) getWithdrawals(rw http.ResponseWriter, r *http.Request) {
	s.log.Info("Получение информации о выводе средств")
	login, ok := r.Context().Value(model.KeyLogin).(string)
	if !ok {
		s.log.Error(model.ErrCastingType.Error())
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	withdrawals, err := s.service.GetWithdrawals(r.Context(), login)
	if err != nil {
		if errors.Is(err, model.ErrNoWithdrawals) {
			rw.WriteHeader(http.StatusNoContent)
			return
		}
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	buf := bytes.NewBuffer([]byte{})
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	err = encoder.Encode(withdrawals)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	s.log.WithFields(logrus.Fields{"withdrawals": withdrawals}).Info("Информация о выводе средств получена")
	rw.Header().Add("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	fmt.Fprint(rw, buf)
}

func (s server) getBalance(rw http.ResponseWriter, r *http.Request) {
	s.log.Info("Получение баланса")

	login, ok := r.Context().Value(model.KeyLogin).(string)
	if !ok {
		s.log.Error(model.ErrCastingType.Error())
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	balance, err := s.service.GetBalance(r.Context(), login)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
	}

	buf := bytes.NewBuffer([]byte{})
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	err = encoder.Encode(balance)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	// устанавливаем заголовок Content-Type
	// для передачи клиенту информации, кодированной в JSON
	rw.Header().Add("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)

	s.log.Info("Баланс пользователя успешно возвращен")
	fmt.Fprint(rw, buf)

}
