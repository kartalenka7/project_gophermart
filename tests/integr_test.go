package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
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
	"golang.org/x/net/publicsuffix"
)

var jar *cookiejar.Jar

func init() {
	jar, _ = cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
}

func TestUserRegstr(t *testing.T) {
	type want struct {
		statusCode int
		cookie     http.Cookie
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
				cookie:     http.Cookie{Name: "UserAuth"},
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
				cookie:     http.Cookie{Name: ""},
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
			encoder.Encode(tt.user)
			// создаем запрос
			request, err := http.NewRequest(tt.method, ts.URL+tt.request, buf)
			assert.NoError(t, err)
			request.Header.Add("Content-Type", "application/json")

			// настраиваем клиента и куки
			client := new(http.Client)
			client.Jar = jar
			resp, err := client.Do(request)

			require.NoError(t, err)
			assert.Equal(t, tt.want.statusCode, resp.StatusCode)

			for _, cookie := range jar.Cookies(request.URL) {
				log.WithFields(logrus.Fields{"cookie": cookie}).Info("Получены куки")
				assert.Equal(t, tt.want.cookie.Name, cookie.Name)
			}

		})
	}
}
