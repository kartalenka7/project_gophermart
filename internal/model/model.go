package model

import (
	"errors"
)

type User struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	Cookie   string
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
	ErrWrongRequest        = errors.New("Wrong request")
	ErrNotAuthorized       = errors.New("User not authorized")
	ErrLoginExists         = errors.New("Login already exists")
	ErrAuthFailed          = errors.New("Authentification failed")
	ErrOrderExistsSameUser = errors.New("Order number has downloaded by current user")
	ErrOrderExistsDiffUser = errors.New("Order number has downloaded by other user")
	ErrNotValidOrderNumber = errors.New("Order number is not valid")

	Secretkey = []byte("secret key")
)
