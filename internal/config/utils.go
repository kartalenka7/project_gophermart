package config

import "fmt"

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
			fmt.Printf("v, i %d %d\n", v, i)
		}

		// складываем все числа
		luhn += int64(v)
		fmt.Printf("luhn %d\n", luhn)

	}
	// Полученная сумма должна быть кратна 10
	return luhn%10 == 0
}
