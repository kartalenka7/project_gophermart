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
		authToken  string
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
				Login:    "user7",
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
				Login:    "user7",
				Password: "1234",
			},
			want: want{
				statusCode: http.StatusConflict,
			},
		},
	}

	// инициализация нужных структур

	log := config.InitLog()
	cfg, err := config.GetConfig(log)
	require.NoError(t, err)
	storage, err := storage.NewStorage(cfg.Database, log)
	require.NoError(t, err)
	service := service.NewService(storage, log)
	r := handlers.NewRouter(service, log)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			//запускаем тестовый сервер
			ts := httptest.NewServer(r)
			defer ts.Close()

			//формируем json объект для записи в body запроса
			buf := bytes.NewBuffer([]byte{})
			encoder := json.NewEncoder(buf)
			encoder.SetEscapeHTML(false)
			err = encoder.Encode(tt.user)
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
