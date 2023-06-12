package storage

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"

	"github.com/kartalenka7/project_gophermart/internal/model"
)

var (
	createUserTable = `CREATE TABLE IF NOT EXISTS
						users(
							login    TEXT PRIMARY KEY,
							password TEXT,
							cookie   TEXT
						)`
	createOrdersTable = `CREATE TABLE IF NOT EXISTS
						 orders(
							number TEXT PRIMARY KEY,
							user   TEXT,
							time   TEXT,
							status TEXT
						 )`
	createHistoryTable = `CREATE TABLE IF NOT EXISTS
							ordersHistory(
								number   TEXT,
								withdraw INT,
								time     TEXT
							)`

	insertUser = `INSERT INTO users(login, password, cookie) VALUES($1, $2, $3)`
	selectUser = `SELECT password, cookie FROM users WHERE login = $1`

	selectOrder = `SELECT user FROM orders WHERE number = $1`
	insertOrder = `INSERT INTO orders(number, user, date) VALUES($1, $2, $3)`
)

type DBStruct struct {
	pgxPool *pgxpool.Pool
	log     *logrus.Logger
}

func NewStorage(connString string, log *logrus.Logger) (*DBStruct, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	time.AfterFunc(60*time.Second, cancel)

	pgxPool, err := InitConnection(ctx, connString, log)
	if err != nil {
		return nil, err
	}
	return &DBStruct{
		pgxPool: pgxPool,
		log:     log,
	}, nil
}

func InitConnection(ctx context.Context, connString string, log *logrus.Logger) (*pgxpool.Pool, error) {

	log.Info("Инициализируем пул соединений с Postgres, создаем таблицы")
	pgxPool, err := pgxpool.Connect(ctx, connString)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}

	// создаем таблицы
	if _, err = pgxPool.Exec(ctx, createUserTable); err != nil {
		log.Error(err.Error())
		return nil, err
	}

	if _, err = pgxPool.Exec(ctx, createOrdersTable); err != nil {
		log.Error(err.Error())
		return nil, err
	}

	return pgxPool, nil
}

func (db *DBStruct) Close() {
	db.pgxPool.Close()
}

func (db *DBStruct) AddUser(ctx context.Context, user model.User) error {
	var pgxError *pgconn.PgError
	// добавляем пользователя в таблицу users
	_, err := db.pgxPool.Exec(ctx, insertUser, user.Login, user.Password, user.Cookie)
	if errors.As(err, &pgxError) {
		// Логин уже существует
		if pgxError.Code == pgerrcode.UniqueViolation {
			db.log.Error(model.ErrLoginExists.Error())
			return model.ErrLoginExists
		}
	}
	if err != nil {
		db.log.Error(err.Error())
	}
	return err
}

func (db *DBStruct) GetUser(ctx context.Context, user model.User) (string, error) {
	var checkUser model.User
	row := db.pgxPool.QueryRow(ctx, selectUser, user.Login)
	err := row.Scan(&checkUser.Password, &checkUser.Cookie)
	if err != nil {
		db.log.Error(err.Error())
		return "", err
	}

	// проверить хэш пароля
	err = bcrypt.CompareHashAndPassword([]byte(checkUser.Password), []byte(user.Password))
	if err != nil {
		db.log.Error(err.Error())
	}
	return checkUser.Cookie, err
}

func (db *DBStruct) AddOrder(ctx context.Context, number string, cookie string) error {
	var user string
	t := time.Now().Format(time.RFC3339)

	row := db.pgxPool.QueryRow(ctx, selectOrder, number)
	err := row.Scan(&user)
	if err == nil {
		// Номер заказа уже был загружен этим пользователем
		if user == cookie {
			return model.ErrOrderExistsSameUser
		}
		// Номер заказа уже был загружен другим пользователем
		db.log.WithFields(logrus.Fields{
			"user": user}).Error(model.ErrOrderExistsDiffUser.Error())
		return model.ErrOrderExistsDiffUser
	}
	// запрос insert в таблицу orders
	_, err = db.pgxPool.Exec(ctx, insertOrder, number, t, cookie)
	if err != nil {
		db.log.Error(err.Error())
	}
	return err
}

func (db *DBStruct) GetOrders(user model.User) ([]model.Orders, error) {
	var orders []model.Orders

	// запрос select в таблицу orders

	// http запрос в систему начислений баллов лояльности
	getOrdersPoints(orders)
	return orders, nil
}

func (db *DBStruct) WriteWithdraw() error {
	// запрос select в таблицу orders, проверяем что номер заказа существует
	//422 — неверный номер заказа;

	// запрос select в таблицу users, проверка, что withdraw < balance
	//402 — на счету недостаточно средств

	// запрос insert в таблицу OrdersHistory
	return nil
}

// взаимодействие с системой расчета начислений баллов лояльности
func getOrdersPoints(orders []model.Orders) ([]model.PointsAppResponse, error) {
	var allResp []model.PointsAppResponse
	pointsResp := model.PointsAppResponse{}
	client := &http.Client{}
	for _, v := range orders {
		url := "api/orders/" + v.Number
		request, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			//обработка ошибки
			return nil, err
		}
		request.Header.Add("Content-Length", "0")
		resp, err := client.Do(request)
		if err != nil {
			//обработка ошибки
			return nil, err
		}
		decoder := json.NewDecoder(resp.Body)
		if err = decoder.Decode(&pointsResp); err != nil {
			return nil, err
		}
		allResp = append(allResp, pointsResp)
		resp.Body.Close()
	}
	return allResp, nil
}
