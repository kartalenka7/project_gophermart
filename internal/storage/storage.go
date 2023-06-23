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

	"github.com/kartalenka7/project_gophermart/internal/model"
	"github.com/kartalenka7/project_gophermart/internal/utils"
)

var (
	createUserTable = `CREATE TABLE IF NOT EXISTS
					    users(
							login    TEXT PRIMARY KEY,
							password TEXT
						)`
	createOrdersTable = `CREATE TABLE IF NOT EXISTS
						 orders(
							number TEXT PRIMARY KEY,
							login TEXT,
							time   TEXT,
							status TEXT,
							accrual INT
						 )`
	createHistoryTable = `CREATE TABLE IF NOT EXISTS
							ordersHistory(
								number      TEXT,
								withdraw    INT,
								time        TEXT
							)`

	insertUser = `INSERT INTO users(login, password) VALUES($1, $2)`
	selectUser = `SELECT password FROM users WHERE login = $1`

	selectOrder      = `SELECT login FROM orders WHERE number = $1`
	selectUserOrders = `SELECT number, login, time, status, accrual FROM orders WHERE login = $1 AND time IS NOT NULL`
	insertOrder      = `INSERT INTO orders(number, login, time, status, accrual) VALUES($1, $2, $3, 'NEW', 0)`

	selectProcessingOrders = `SELECT number FROM orders WHERE status != $1 AND status != $2 AND time IS NOT NULL`
	updateOrdersStatus     = `UPDATE orders SET status = $1, accrual = $2 WHERE number = $3`

	addOrderHistory   = `INSERT INTO ordersHistory(number, withdraw, time) VALUES($1, $2, $3)`
	selectUserHistory = `SELECT h.withdraw 
						 FROM ordersHistory as h 
						 JOIN orders AS o ON o.number = h.number 
						 WHERE o.login = $1`
	selectWithdrawHistory = `SELECT h.number, h.withdraw, h.time
						 FROM ordersHistory as h 
						 JOIN orders AS o ON h.number = o.number
						 WHERE o.login = $1`
)

type DBStruct struct {
	pgxPool *pgxpool.Pool
	log     *logrus.Logger
}

func NewStorage(connString string, accrualSys string, log *logrus.Logger) (*DBStruct, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	time.AfterFunc(60*time.Second, cancel)

	pgxPool, err := InitConnection(ctx, connString, accrualSys, log)
	if err != nil {
		return nil, err
	}
	return &DBStruct{
		pgxPool: pgxPool,
		log:     log,
	}, nil
}

func InitConnection(ctx context.Context, connString string, accrualSys string, log *logrus.Logger) (*pgxpool.Pool, error) {

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

	if _, err = pgxPool.Exec(ctx, createHistoryTable); err != nil {
		log.Error(err.Error())
		return nil, err
	}

	log.Info("Запускаем горутину для взаимодейтсвия с системой расчета баллов лояльности")
	go updateOrders(ctx, pgxPool, accrualSys, log)
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

func (db *DBStruct) AuthUser(ctx context.Context, user model.User) (string, error) {
	var checkUser model.User
	row := db.pgxPool.QueryRow(ctx, selectUser, user.Login)
	err := row.Scan(&checkUser.Password)
	if err != nil {
		db.log.Error(err.Error())
		return "", err
	}

	return checkUser.Password, err
}

func (db *DBStruct) AddOrder(ctx context.Context, number string, login string) error {
	var user string

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
		"login":  login,
		"t":      t}).Info("Запись заказа в таблицу orderTable")

	_, err = db.pgxPool.Exec(ctx, insertOrder, number, login, t)
	if err != nil {
		db.log.Error(err.Error())
	}
	return err
}

func (db *DBStruct) GetOrders(ctx context.Context, login string) ([]model.OrdersResponse, error) {
	var timeStr string
	var accrualInt int32
	var orderResp model.OrdersResponse
	var orders []model.OrdersResponse

	db.log.WithFields(
		logrus.Fields{
			"login": login,
		}).Info("Выбираем заказы для пользователя")
	// выбираем список запросов для авторизованного пользователя
	rows, err := db.pgxPool.Query(ctx, selectUserOrders, login)
	if err != nil {
		db.log.Error(err.Error())
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&orderResp.Number, &orderResp.Login, &timeStr, &orderResp.Status, &accrualInt)
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
		orderResp.Accrual = float64(accrualInt)
		orderResp.Accrual = orderResp.Accrual / 100
		db.log.WithFields(logrus.Fields{
			"time":    timeStr,
			"number":  orderResp.Number,
			"status":  orderResp.Status,
			"accrual": orderResp.Accrual,
		}).Info("Получили строку с заказом")
		orders = append(orders, orderResp)
	}

	if rows.Err() != nil {
		db.log.Error(rows.Err().Error())
		return nil, err
	}

	if orders == nil {
		db.log.Info("в orders пусто")
		return nil, errors.New("в orders пусто")
	}
	// сортируем ответ по времени
	sort.SliceStable(orders, func(i, j int) bool {
		return orders[i].Time.Before(orders[j].Time)
	})
	return orders, nil
}

func (db *DBStruct) CalculateBalance(ctx context.Context, login string) (int32, error) {
	var balance int32
	var balanceAll int32

	// определяем баланс пользователя
	rows, err := db.pgxPool.Query(ctx, selectUserHistory, login)
	if err != nil {
		db.log.Error(err.Error())
		return 0, err
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&balance)
		if err != nil {
			db.log.Error(err.Error())
			return 0, err
		}
		db.log.WithFields(logrus.Fields{"balance": balance}).Info("Накапливаем баланс")
		balanceAll += balance
	}
	if rows.Err() != nil {
		db.log.Error(rows.Err().Error())
		return 0, rows.Err()
	}
	return balance, nil
}

func (db *DBStruct) WriteWithdraw(ctx context.Context, withdraw model.OrderWithdraw, login string) error {
	db.log.WithFields(logrus.Fields{
		"number":   withdraw.Number,
		"withdraw": withdraw.Withdraw,
	}).Info("Запись в таблицу OrdersHistory")
	// Добавляем запись списания в OrdersHistory
	_, err := db.pgxPool.Exec(ctx, addOrderHistory, withdraw.Number, withdraw.Withdraw, time.Now().Format(time.RFC3339))
	if err != nil {
		db.log.Error(err.Error())
		return err
	}
	db.log.WithFields(logrus.Fields{
		"number": withdraw.Number,
		"login":  login,
	}).Info("Запись в таблицу orders")
	_, err = db.pgxPool.Exec(ctx, insertOrder, withdraw.Number, login, nil)
	if err != nil {
		db.log.Error(err.Error())
	}
	return nil
}

// взаимодействие с системой расчета начислений баллов лояльности
func updateOrders(ctx context.Context, pgxPool *pgxpool.Pool, accrualSys string, log *logrus.Logger) {
	var allResp []model.PointsAppResponse
	var orderNumber string
	var orderNumbers []string
	var accrual int32

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// выбрать заказы, у которых не окончательный статус
			rows, err := pgxPool.Query(ctx, selectProcessingOrders, "INVALID", "PROCESSED")
			if err != nil {
				log.Error(err.Error())
				continue
			}

			orderNumbers = orderNumbers[:0]
			for rows.Next() {
				err := rows.Scan(&orderNumber)
				if err != nil {
					log.Error(err.Error())
					continue
				}
				log.WithFields(logrus.Fields{"orderNumber": orderNumber}).Info("Выбран заказ для запроса статуса")
				orderNumbers = append(orderNumbers, orderNumber)
			}

			pointsResp := model.PointsAppResponse{}
			client := &http.Client{}
			for _, v := range orderNumbers {
				url := accrualSys + "/api/orders/" + v

				log.Info("http запрос в систему начислений баллов лояльности")
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
				log.WithFields(logrus.Fields{"status-code": resp.StatusCode}).Info("Статус ответа")
				decoder := json.NewDecoder(resp.Body)
				if err = decoder.Decode(&pointsResp); err != nil {
					log.Error(err.Error())
					continue
				}
				allResp = append(allResp, pointsResp)
				resp.Body.Close()
			}

			if allResp == nil {
				continue
			}
			// обновить статусы и баллы полученных в ответе заказов
			batch := &pgx.Batch{}
			for _, response := range allResp {
				// переводим в копейки
				accrual = int32(response.Accrual * 100)
				batch.Queue(updateOrdersStatus, response.Status, accrual, response.Number)
				batch.Queue(addOrderHistory, response.Number, accrual, time.Now().Format(time.RFC3339))
				log.WithFields(logrus.Fields{
					"number":  response.Number,
					"status":  response.Status,
					"accrual": accrual,
				}).Info("Обновление заказа")
			}
			batchReq := pgxPool.SendBatch(ctx, batch)
			_, err = batchReq.Exec()
			if err != nil {
				log.Error(err.Error())
			}
			batchReq.Close()
		case <-ctx.Done():
			log.Error("Отмена контекста")
			return
		}
	}
}

func (db *DBStruct) GetBalance(ctx context.Context, login string) (model.Balance, error) {
	var balance model.Balance
	var withdrawFloat float64
	var withdraw int32

	rows, err := db.pgxPool.Query(ctx, selectUserHistory, login)
	if err != nil {
		db.log.Error(rows.Err().Error())
		return model.Balance{}, err
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&withdraw)
		if err != nil {
			db.log.Error(err.Error())
			return model.Balance{}, err
		}
		withdrawFloat = float64(withdraw)
		db.log.WithFields(logrus.Fields{"withdraw": withdraw}).Info("Баланс")
		balance.Balance += withdrawFloat / 100
		if withdrawFloat < 0 {
			balance.Withdrawn += withdrawFloat / 100
		}
	}

	balance.Balance = utils.Round(balance.Balance, 2)
	balance.Withdrawn = utils.Round(-balance.Withdrawn, 2)
	if rows.Err() != nil {
		db.log.Error(rows.Err().Error())
		return model.Balance{}, rows.Err()
	}

	db.log.WithFields(logrus.Fields{"balance": balance}).Info("Баланс")
	return balance, nil
}

func (db *DBStruct) GetWithdrawals(ctx context.Context, login string) ([]model.OrderWithdraw, error) {
	var userWithdraw model.OrderWithdraw
	var allWithdrawals []model.OrderWithdraw
	var withdraw int32
	var timeUpl string

	rows, err := db.pgxPool.Query(ctx, selectWithdrawHistory, login)
	if err != nil {
		db.log.Error(err.Error())
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&userWithdraw.Number, &withdraw, &timeUpl)
		if err != nil {
			db.log.Error(err.Error())
			return nil, err
		}
		if withdraw > 0 {
			continue
		}
		userWithdraw.Time, err = time.Parse(time.RFC3339, timeUpl)
		if err != nil {
			db.log.Error(err.Error())
		}
		userWithdraw.Withdraw = float64(withdraw)
		userWithdraw.Withdraw = -userWithdraw.Withdraw / 100

		db.log.WithFields(logrus.Fields{
			"number":   userWithdraw.Number,
			"withdraw": userWithdraw.Withdraw,
			"time":     userWithdraw.Time,
		}).Info("Списание")
		allWithdrawals = append(allWithdrawals, userWithdraw)
	}
	if err := rows.Err(); err != nil {
		db.log.Error(err.Error())
		return nil, err
	}
	if allWithdrawals == nil {
		db.log.Error(model.ErrNoWithdrawals.Error())
		return nil, model.ErrNoWithdrawals
	}

	// сортируем записи списаний
	sort.SliceStable(allWithdrawals, func(i, j int) bool {
		return allWithdrawals[i].Time.Before(allWithdrawals[j].Time)
	})
	return allWithdrawals, nil
}
