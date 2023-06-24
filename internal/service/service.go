package service

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

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
	GetWithdrawals(ctx context.Context, login string) ([]model.OrderWithdraw, error)
	GetOrdersForUpdate(ctx context.Context) ([]string, error)
	UpdateOrders(ctx context.Context, accrualSysResponse []model.PointsAppResponse)
}

type ServiceStruct struct {
	storage Storer
	Log     *logrus.Logger
}

func NewService(ctx context.Context, storage Storer, log *logrus.Logger, accrualSys string) *ServiceStruct {
	var service *ServiceStruct
	log.Info("Инициализируем сервис")
	service = &ServiceStruct{
		storage: storage,
		Log:     log,
	}
	log.Info("Запускаем горутину для взаимодейтсвия с системой расчета баллов лояльности")
	go service.GetUpdatesFromAccrualSystem(ctx, accrualSys)
	return service
}

// взаимодействие с системой расчета начислений баллов лояльности
func (s ServiceStruct) GetUpdatesFromAccrualSystem(ctx context.Context, accrualSys string) {
	var allResp []model.PointsAppResponse

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			orderNumbers, err := s.storage.GetOrdersForUpdate(ctx)
			if err != nil {
				continue
			}

			pointsResp := model.PointsAppResponse{}
			client := &http.Client{}
			for _, v := range orderNumbers {
				url := accrualSys + "/api/orders/" + v

				s.Log.Info("http запрос в систему начислений баллов лояльности")
				request, err := http.NewRequest(http.MethodGet, url, nil)
				if err != nil {
					s.Log.Error(err.Error())
					continue
				}
				request.Header.Add("Content-Length", "0")
				resp, err := client.Do(request)
				if err != nil {
					s.Log.Error(err.Error())
					continue
				}
				s.Log.WithFields(logrus.Fields{"status-code": resp.StatusCode}).Info("Статус ответа")
				decoder := json.NewDecoder(resp.Body)
				if err = decoder.Decode(&pointsResp); err != nil {
					s.Log.Error(err.Error())
					continue
				}
				allResp = append(allResp, pointsResp)
				resp.Body.Close()
			}

			if allResp == nil {
				continue
			}
			s.storage.UpdateOrders(ctx, allResp)
		case <-ctx.Done():
			s.Log.Error("Отмена контекста")
			return
		}
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

	balance, err := s.storage.GetBalance(ctx, login)
	if err != nil {
		return err
	}

	// переводим обратно в рубли
	balanceFloat = balance.Balance / 100

	// проверяем, что у пользователя достаточно баллов для списания
	if balanceFloat < float64(withdraw.Withdraw) {
		s.Log.Error(model.ErrInsufficientBalance.Error())
		return model.ErrInsufficientBalance
	}
	withdraw.Withdraw = -withdraw.Withdraw * 100

	return s.storage.WriteWithdraw(ctx, withdraw, login)
}

func (s ServiceStruct) GetBalance(ctx context.Context, login string) (model.Balance, error) {
	balance, err := s.storage.GetBalance(ctx, login)
	if err != nil {
		return model.Balance{}, err
	}

	balance.Balance = balance.Balance / 100
	balance.Withdrawn = balance.Withdrawn / 100

	balance.Balance = utils.Round(balance.Balance, 2)
	balance.Withdrawn = utils.Round(-balance.Withdrawn, 2)

	s.Log.WithFields(logrus.Fields{"balance": balance}).Info("Баланс с округлением")

	return balance, err
}

func (s ServiceStruct) GetWithdrawals(ctx context.Context, login string) ([]model.OrderWithdraw, error) {
	return s.storage.GetWithdrawals(ctx, login)
}
