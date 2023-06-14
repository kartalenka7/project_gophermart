package handlers

import (
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/kartalenka7/project_gophermart/internal/model"
)

// Обработка запросов с поддержкой сжатия данных
func gzipHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.Header.Get("Content-Encoding") != "gzip" {
			// если gzip не поддерживается, передаём управление
			// дальше без изменений
			next.ServeHTTP(w, r)
			return
		}

		// Распаковать длинный url из body с помощью gzip
		gz, err := gzip.NewReader(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer gz.Close()

		// при чтении вернётся распакованный слайс байт
		b, err := io.ReadAll(gz)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// пишем в тело распакованные данные и передаем дальше в хэндлеры
		r.Body = io.NopCloser(strings.NewReader(string(b)))

		// замыкание — используем ServeHTTP следующего хендлера
		next.ServeHTTP(w, r)

	})
}

// Проверить, что пользователь аутентифицирован
func (s server) checkUserAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//Получение токена
		tokenHeader := r.Header.Get("Authorization")
		if tokenHeader == "" {
			s.log.Error("Токен пуст")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		tk := &model.Token{}
		token, err := jwt.ParseWithClaims(tokenHeader, tk, func(token *jwt.Token) (interface{}, error) {
			return []byte("secret"), nil
		})

		if err != nil {
			s.log.Error(err.Error())
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if !token.Valid {
			s.log.Error("Token not valid")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// передаем логин через контекст
		ctx := context.WithValue(r.Context(), "login", tk.Login)
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}
