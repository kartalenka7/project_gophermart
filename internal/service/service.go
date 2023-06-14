package service

import (
	"context"
	"crypto/hmac"
	crypto "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"golang.org/x/crypto/bcrypt"

	"github.com/kartalenka7/project_gophermart/internal/config"
	"github.com/kartalenka7/project_gophermart/internal/model"
	"github.com/sirupsen/logrus"
)

// интерфейс взаимодействия с хранилищем
type Storer interface {
	AddUser(ctx context.Context, user model.User) error
	GetUser(ctx context.Context, user model.User) (string, error)
	AddOrder(ctx context.Context, number string) error
	GetOrders(user model.User) ([]model.Orders, error)
	WriteWithdraw() error
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

func (s ServiceStruct) RgstrUser(ctx context.Context, user model.User) (string, error) {

	s.Log.WithFields(logrus.Fields{"user": user.Login}).Info("Регистрация пользователя")

	// пароль преобразовать в хэш
	bytes, err := bcrypt.GenerateFromPassword([]byte(user.Password), 14)
	if err != nil {
		s.Log.Error(err.Error())
		return "", err
	}
	user.Password = string(bytes)

	user.Cookie, err = generateCookie(config.CryptoLen)
	if err != nil {
		s.Log.Error(err.Error())
		return "", err
	}

	if err = s.storage.AddUser(ctx, user); err != nil {
		return "", err
	}

	return user.Cookie, nil
}

func (s ServiceStruct) AuthUser(ctx context.Context, user model.User) (string, error) {
	var cookie string

	cookie, err := s.storage.GetUser(ctx, user)

	s.Log.WithFields(logrus.Fields{
		"user":   user.Login,
		"cookie": cookie}).Info("Авторизация пользователя")

	if err != nil {
		return "", model.ErrAuthFailed
	}

	return cookie, nil
}

func generateCookie(len int) (string, error) {
	// сгенерировать криптостойкий слайс случайных байт
	b := make([]byte, len)
	_, err := crypto.Read(b)
	if err != nil {
		return "", err
	}
	// кодируем, чтобы использовать его для куки
	token := hex.EncodeToString(b)

	// подписываем алгоритмом HMAC, используя SHA256
	h := hmac.New(sha256.New, model.Secretkey)
	h.Write([]byte(token))

	sign := h.Sum(nil)

	token = string(sign) + token
	token = base64.URLEncoding.EncodeToString([]byte(token))

	return token, nil
}

func (s ServiceStruct) AddUserOrder(ctx context.Context, number string) error {

	//проверить формат номера заказа
	if !config.CheckLuhnAlg(number) {
		s.Log.Error(model.ErrNotValidOrderNumber.Error())
		return model.ErrNotValidOrderNumber
	}

	err := s.storage.AddOrder(ctx, number)
	return err
}

func (s ServiceStruct) GetUserOrders(user model.User) error {

	return nil
}

func (s ServiceStruct) WriteWithdraw(withdraw model.OrderWithdraw) {

}
