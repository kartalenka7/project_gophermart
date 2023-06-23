package service

import (
	"context"
	"encoding/json"
	"net/http"

	"golang.org/x/crypto/bcrypt"

	"github.com/kartalenka7/project_gophermart/internal/model"
	"github.com/kartalenka7/project_gophermart/internal/utils"
	"github.com/sirupsen/logrus"
)

// интерфейс взаимодействия с хранилищем
//go:generate mockery --name Storer --with-expecter
type Storer interface {
	AddUser(ctx context.Context, user model.User) error
	AuthUser(ctx context.Context, user model.User) (string, error)
	AddOrder(ctx context.Context, number string, login string) error
	GetOrders(ctx context.Context, login string) ([]model.OrdersResponse, error)
	WriteWithdraw(ctx context.Context, withdraw model.OrderWithdraw, login string) error
	GetBalance(ctx context.Context, login string) (model.Balance, error)
	CalculateBalance(ctx context.Context, login string) (int32, error)
	GetWithdrawals(ctx context.Context, login string) ([]model.OrderWithdraw, error)
}

type ServiceStruct struct {
	storage Storer
	Log     *logrus.Logger
}

func NewService(storage Storer, log *logrus.Logger) *ServiceStruct {
	log.Info("Инициализируем сервис")
	return &ServiceStruct{
		storage: storage,
		Log:     log,
	}
}

func (s ServiceStruct) ParseUserCredentials(r *http.Request) (model.User, error) {
	var user model.User
	// проверить у запроса content-type = application/json
	if r.Header.Get("Content-Type") != "application/json" {
		contType := r.Header.Get("Content-Type")
		s.Log.WithFields(logrus.Fields{"Content-Type": contType}).Error("Неверный Content-Type")
		return model.User{}, model.ErrWrongRequest
	}

	// парсим из json логин и пароль
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&user); err != nil {
		s.Log.Error(err.Error())
		return model.User{}, err
	}
	// проверить, что логин и пароль не пустые
	if user.Login == "" || user.Password == "" {
		s.Log.Error(model.ErrWrongRequest.Error())
		return model.User{}, model.ErrWrongRequest
	}
	return user, nil
}

func (s ServiceStruct) RgstrUser(ctx context.Context, user model.User) error {

	s.Log.WithFields(logrus.Fields{"user": user.Login}).Info("Регистрация пользователя")

	// пароль преобразовать в хэш
	bytes, err := bcrypt.GenerateFromPassword([]byte(user.Password), 14)
	if err != nil {
		s.Log.Error(err.Error())
		return err
	}
	user.Password = string(bytes)

	if err = s.storage.AddUser(ctx, user); err != nil {
		return err
	}

	return nil
}

func (s ServiceStruct) AuthUser(ctx context.Context, user model.User) error {

	checkPassword, err := s.storage.AuthUser(ctx, user)
	if err != nil {
		return model.ErrAuthFailed
	}

	// проверить хэш пароля
	err = bcrypt.CompareHashAndPassword([]byte(checkPassword), []byte(user.Password))
	if err != nil {
		s.Log.Error(err.Error())
		return model.ErrAuthFailed
	}

	return nil
}

func (s ServiceStruct) AddUserOrder(ctx context.Context, number string, login string) error {

	//проверить формат номера заказа
	if !utils.CheckLuhnAlg(number) {
		s.Log.Error(model.ErrNotValidOrderNumber.Error())
		return model.ErrNotValidOrderNumber
	}

	err := s.storage.AddOrder(ctx, number, login)
	return err
}

func (s ServiceStruct) GetUserOrders(ctx context.Context, login string) ([]model.OrdersResponse, error) {
	return s.storage.GetOrders(ctx, login)
}

func (s ServiceStruct) WriteWithdraw(ctx context.Context, withdraw model.OrderWithdraw, login string) error {
	var balanceFloat float64

	//проверить формат номера заказа
	if !utils.CheckLuhnAlg(withdraw.Number) {
		s.Log.Error(model.ErrNotValidOrderNumber.Error())
		return model.ErrNotValidOrderNumber
	}

	balance, err := s.storage.CalculateBalance(ctx, login)
	if err != nil {
		return err
	}

	// проверяем, что у пользователя достаточно баллов для списания
	balanceFloat = float64(balance)
	// переводим обратно в рубли
	balanceFloat = balanceFloat / 100

	if balanceFloat < float64(withdraw.Withdraw) {
		s.Log.Error(model.ErrInsufficientBalance.Error())
		return model.ErrInsufficientBalance
	}
	withdraw.Withdraw = -withdraw.Withdraw * 100

	return s.storage.WriteWithdraw(ctx, withdraw, login)
}

func (s ServiceStruct) GetBalance(ctx context.Context, login string) (model.Balance, error) {
	return s.storage.GetBalance(ctx, login)
}

func (s ServiceStruct) GetWithdrawals(ctx context.Context, login string) ([]model.OrderWithdraw, error) {
	return s.storage.GetWithdrawals(ctx, login)
}
