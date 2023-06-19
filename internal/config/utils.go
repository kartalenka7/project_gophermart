package config

import "math"

const (
	asciiZero = 48
)

//Проверка номера заказа алгоритмом Луна
func CheckLuhnAlg(number string) bool {
	var luhn int64

	p := (len(number)) % 2

	for i, v := range number {
		v = v - asciiZero
		// умножаем на 2 каждую вторую цифру номера
		if i%2 == p {
			v *= 2
			// если получилось больше 9, то вычитаем из произведения 9
			if v > 9 {
				v -= 9
			}
		}
		// складываем все числа
		luhn += int64(v)
	}
	// Полученная сумма должна быть кратна 10
	return luhn%10 == 0
}

// округление float до n знаков после запятой
func Round(x float64, prec int) float64 {
	var rounder float64

	pow := math.Pow(10, float64(prec))
	intermed := x * pow
	_, frac := math.Modf(intermed)

	if frac >= 0.5 {
		rounder = math.Ceil(intermed)
	} else {
		rounder = math.Floor(intermed)
	}

	return rounder / pow
}
