package storage

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"

	"github.com/kartalenka7/project_gophermart/internal/model"
)

var (
	createUserTable = `CREATE TABLE IF NOT EXISTS
					    users(
							login    TEXT PRIMARY KEY,
							password TEXT
						)`
	createOrdersTable = `DROP TABLE IF EXISTS orders
						 CREATE TABLE IF NOT EXISTS
						 orders(
							number TEXT PRIMARY KEY,
							login TEXT,
							time   TEXT,
							status TEXT,
							accrual INT
						 )`
	createHistoryTable = `CREATE TABLE IF NOT EXISTS
							ordersHistory(
								number   TEXT,
								withdraw INT,
								time     TEXT
							)`

	insertUser = `INSERT INTO users(login, password) VALUES($1, $2)`
	selectUser = `SELECT password FROM users WHERE login = $1`

	selectOrder      = `SELECT login FROM orders WHERE number = $1`
	selectUserOrders = `SELECT number, login, time, status, accrual FROM orders WHERE login = $1`
	insertOrder      = `INSERT INTO orders(number, login, time) VALUES($1, $2, $3)`

	selectProcessingOrders = `SELECT number FROM orders WHERE status != "INVALID" AND status != "PROCESSED"`
	updateOrdersStatus     = `UPDATE orders SET status = $1, accrual = $2 WHERE number = $3`
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

	log.Info("Запускаем горутину для взаимодейтсвия с системой расчета баллов лояльности")
	go updateOrders(ctx, pgxPool, log)
	return pgxPool, nil
}

func (db *DBStruct) Close() {
	db.pgxPool.Close()
}

func (db *DBStruct) AddUser(ctx context.Context, user model.User) error {
	var pgxError *pgconn.PgError
	// добавляем пользователя в таблицу users
	_, err := db.pgxPool.Exec(ctx, insertUser, user.Login, user.Password)
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

func (db *DBStruct) GetUser(ctx context.Context, user model.User) error {
	var checkUser model.User
	row := db.pgxPool.QueryRow(ctx, selectUser, user.Login)
	err := row.Scan(&checkUser.Password)
	if err != nil {
		db.log.Error(err.Error())
		return err
	}

	// проверить хэш пароля
	err = bcrypt.CompareHashAndPassword([]byte(checkUser.Password), []byte(user.Password))
	if err != nil {
		db.log.Error(err.Error())
	}
	return err
}

func (db *DBStruct) AddOrder(ctx context.Context, number string) error {
	var user string

	login := ctx.Value(model.KeyLogin).(string)

	// форматировать текущий момент времени в строку
	t := time.Now().Format(time.RFC3339)

	row := db.pgxPool.QueryRow(ctx, selectOrder, number)
	err := row.Scan(&user)

	if err == nil {
		// Номер заказа уже был загружен этим пользователем
		if user == login {
			db.log.Error(model.ErrOrderExistsSameUser.Error())
			return model.ErrOrderExistsSameUser
		}
		// Номер заказа уже был загружен другим пользователем
		db.log.WithFields(logrus.Fields{
			"user": user}).Error(model.ErrOrderExistsDiffUser.Error())
		return model.ErrOrderExistsDiffUser
	}
	db.log.WithFields(logrus.Fields{
		"number": number,
		"login":  login}).Info("Запись заказа в таблицу orderTable")

	_, err = db.pgxPool.Exec(ctx, insertOrder, number, login, t)
	if err != nil {
		db.log.Error(err.Error())
	}
	return err
}

func (db *DBStruct) GetOrders(ctx context.Context) ([]model.OrdersResponse, error) {
	var timeStr string
	var accrualInt int32
	var orderResp model.OrdersResponse
	var orders []model.OrdersResponse

	login := ctx.Value(model.KeyLogin).(string)

	// выбираем список запросов для авторизованного пользователя
	rows, err := db.pgxPool.Query(ctx, selectUserOrders, login)
	if err != nil {
		db.log.Error(err.Error())
		return nil, err
	}

	for rows.Next() {
		err := rows.Scan(&orderResp.Number, &orderResp.Login, &timeStr, &orderResp.Status, accrualInt)
		if err != nil {
			db.log.Error(err.Error())
			return nil, err
		}
		//парсим строку со временем в тип time.Time
		orderResp.Time, err = time.Parse(time.RFC3339, timeStr)
		if err != nil {
			db.log.Error(err.Error())
		}
		//переводим из копеек
		orderResp.Accrual = float32(accrualInt / 100)
		orders = append(orders, orderResp)
	}

	if rows.Err() != nil {
		db.log.Error(err.Error())
		return nil, err
	}

	// сортируем ответ по времени
	sort.SliceStable(orders, func(i, j int) bool {
		return orders[i].Time.Before(orders[j].Time)
	})
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
func updateOrders(ctx context.Context, pgxPool *pgxpool.Pool, log *logrus.Logger) {
	var allResp []model.PointsAppResponse
	var orderNumber string
	var orderNumbers []string
	var accrual int32

	// выбрать заказы, у которых не окончательный статус
	rows, err := pgxPool.Query(ctx, selectProcessingOrders)
	if err != nil {
		log.Error(err.Error())
		return
	}

	for rows.Next() {
		err := rows.Scan(&orderNumber)
		if err != nil {
			log.Error(err.Error())
			return
		}
		orderNumbers = append(orderNumbers, orderNumber)
	}

	pointsResp := model.PointsAppResponse{}
	client := &http.Client{}
	for _, v := range orderNumbers {
		url := "api/orders/" + v

		// http запрос в систему начислений баллов лояльности
		request, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			log.Error(err.Error())
			continue
		}
		request.Header.Add("Content-Length", "0")
		resp, err := client.Do(request)
		if err != nil {
			log.Error(err.Error())
			continue
		}
		decoder := json.NewDecoder(resp.Body)
		if err = decoder.Decode(&pointsResp); err != nil {
			log.Error(err.Error())
			continue
		}
		allResp = append(allResp, pointsResp)
		resp.Body.Close()
	}

	// обновить статусы и баллы полученных в ответе заказов
	batch := &pgx.Batch{}
	for _, response := range allResp {
		// переводим в копейки
		accrual = int32(response.Accrual * 100)
		batch.Queue(updateOrdersStatus, response.Status, accrual, response.Number)
		log.WithFields(logrus.Fields{
			"number":  response.Number,
			"status":  response.Status,
			"accrual": accrual,
		}).Info("Обновление заказа")
	}
	batchReq := pgxPool.SendBatch(ctx, batch)
	defer batchReq.Close()
	_, err = batchReq.Exec()
	if err != nil {
		log.Error(err.Error())
	}
}
