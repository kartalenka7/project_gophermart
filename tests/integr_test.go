package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kartalenka7/project_gophermart/internal/config"
	"github.com/kartalenka7/project_gophermart/internal/handlers"
	"github.com/kartalenka7/project_gophermart/internal/model"
	"github.com/kartalenka7/project_gophermart/internal/service"
	"github.com/kartalenka7/project_gophermart/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRegstr(t *testing.T) {
	type want struct {
		statusCode int
	}

	tests := []struct {
		name    string
		method  string
		request string
		user    model.User
		want    want
	}{
		{
			name:    "User registration test",
			method:  http.MethodPost,
			request: "/api/user/register",
			user: model.User{
				Login:    "user2",
				Password: "1234",
			},
			want: want{
				statusCode: http.StatusOK,
			},
		},
		{
			name:    "Login already exists registration test",
			method:  http.MethodPost,
			request: "/api/user/register",
			user: model.User{
				Login:    "user2",
				Password: "1234",
			},
			want: want{
				statusCode: http.StatusConflict,
			},
		},
	}

	log := config.InitLog()
	cfg, err := config.GetConfig(log)
	require.NoError(t, err)
	storage, err := storage.NewStorage(cfg.Database, cfg.AccrualSys, log)
	require.NoError(t, err)
	service := service.NewService(storage, log)
	router := handlers.NewRouter(service, log)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			//запускаем тестовый сервер
			ts := httptest.NewServer(router)
			defer ts.Close()

			//формируем json объект для записи в body запроса
			buf := bytes.NewBuffer([]byte{})
			encoder := json.NewEncoder(buf)
			encoder.SetEscapeHTML(false)
			err := encoder.Encode(tt.user)
			require.NoError(t, err)
			// создаем запрос
			request, err := http.NewRequest(tt.method, ts.URL+tt.request, buf)
			assert.NoError(t, err)
			request.Header.Add("Content-Type", "application/json")

			// настраиваем клиента и куки
			client := new(http.Client)
			resp, err := client.Do(request)
			resp.Body.Close()

			require.NoError(t, err)
			assert.Equal(t, tt.want.statusCode, resp.StatusCode)

			token := resp.Header.Get("Authorization")
			log.WithFields(logrus.Fields{"token": token}).Info("Получен токен авторизации")
			assert.NotNil(t, token)

		})
	}
}

func TestOrders(t *testing.T) {
	type want struct {
		statusCode int
	}
	type request struct {
		url         string
		contentType string
	}

	tests := []struct {
		name    string
		method  string
		request request
		number  string
		user    model.User
		want    want
	}{
		{
			name:   "Add order test",
			method: http.MethodPost,
			request: request{
				url:         "/api/user/orders",
				contentType: "text/plain"},
			number: "45612652",
			user: model.User{
				Login:    "user2",
				Password: "1234",
			},
			want: want{
				statusCode: http.StatusAccepted,
			},
		},
	}

	log := config.InitLog()
	cfg, err := config.GetConfig(log)
	require.NoError(t, err)
	storage, err := storage.NewStorage(cfg.Database, cfg.AccrualSys, log)
	require.NoError(t, err)
	service := service.NewService(storage, log)
	router := handlers.NewRouter(service, log)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			//запускаем тестовый сервер
			ts := httptest.NewServer(router)
			defer ts.Close()

			//формируем json объект для записи в body запроса на аутентификацию
			buf := bytes.NewBuffer([]byte{})
			encoder := json.NewEncoder(buf)
			encoder.SetEscapeHTML(false)
			err := encoder.Encode(tt.user)
			require.NoError(t, err)

			// запрос на аутентификацию пользователя
			reqAuth, err := http.NewRequest(tt.method, ts.URL+"/api/user/login", buf)
			assert.NoError(t, err)
			reqAuth.Header.Add("Content-Type", "application/json")

			// запрос на добавление заказа
			reqOrder, err := http.NewRequest(tt.method, ts.URL+"/api/user/orders", bytes.NewBufferString(tt.number))
			assert.NoError(t, err)
			reqOrder.Header.Add("Content-Type", tt.request.contentType)

			// настраиваем клиента
			client := new(http.Client)

			// аутентификация
			respAuth, err := client.Do(reqAuth)
			respAuth.Body.Close()
			require.NoError(t, err)

			assert.Equal(t, http.StatusOK, respAuth.StatusCode)
			token := respAuth.Header.Get("Authorization")
			assert.NotNil(t, token)

			// добавление заказа
			reqOrder.Header.Add("Authorization", token)
			resp, err := client.Do(reqOrder)
			resp.Body.Close()

			require.NoError(t, err)
			assert.Equal(t, tt.want.statusCode, resp.StatusCode)

			// запрос на получение списка заказов
			reqOrdersAll, err := http.NewRequest(http.MethodGet, ts.URL+"/api/user/orders", nil)
			assert.NoError(t, err)
			reqOrdersAll.Header.Add("Authorization", token)
			respOrdersAll, err := client.Do(reqOrdersAll)
			require.NoError(t, err)
			defer respOrdersAll.Body.Close()

			assert.Equal(t, http.StatusOK, respOrdersAll.StatusCode)

		})
	}
}
