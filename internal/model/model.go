package model

import (
	"errors"

	"github.com/dgrijalva/jwt-go"
)

type User struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	Cookie   string
}

// Структура прав доступа JWT
type Token struct {
	Login string
	jwt.StandardClaims
}

type Orders struct {
	Number string
	User   string
	Date   string
	Status string
}

type PointsAppResponse struct {
	Number  string `json:"order"`
	Status  string `json:"status"`
	Accrual int32  `json:"accrual"` //кол-во баллов, начисленное за заказ
}

type OrderWithdraw struct {
	Number string `json:"order"`
	Sum    int32  `json:"sum"`
}

//описать ошибки для разных кодов ответа

var (
	ErrWrongRequest        = errors.New("wrong request")
	ErrNotAuthorized       = errors.New("user not authorized")
	ErrLoginExists         = errors.New("login already exists")
	ErrAuthFailed          = errors.New("authentification failed")
	ErrOrderExistsSameUser = errors.New("order number has downloaded by current user")
	ErrOrderExistsDiffUser = errors.New("order number has downloaded by other user")
	ErrNotValidOrderNumber = errors.New("order number is not valid")

	Secretkey = []byte("secret key")
)
