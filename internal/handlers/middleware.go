package handlers

import (
	"compress/gzip"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/kartalenka7/project_gophermart/internal/model"
)

type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

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
func checkUserAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("UserAuth")
		if err != nil || cookie.Value == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		signedValue, err := base64.URLEncoding.DecodeString(cookie.Value)
		if err != nil {
			log.Printf("check authentification|%v\n", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		signature := signedValue[:sha256.Size]
		value := signedValue[sha256.Size:]

		mac := hmac.New(sha256.New, model.Secretkey)
		mac.Write([]byte(value))
		expectedSignature := mac.Sum(nil)

		if !hmac.Equal([]byte(signature), expectedSignature) {
			log.Printf("check authentification|%s\n", errors.New("подпись не совпадает"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

	})
}
