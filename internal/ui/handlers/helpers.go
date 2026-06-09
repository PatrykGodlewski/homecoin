package handlers

import "fmt"

func cupFill(percent int) (fillY, fillH int) {
	if percent > 100 {
		percent = 100
	}
	if percent < 0 {
		percent = 0
	}
	fillY = 85 - (percent * 70 / 100)
	fillH = percent * 70 / 100
	if fillH < 2 && percent > 0 {
		fillH = 2
	}
	return fillY, fillH
}

func fmtMoney(cents int64, currency string) string {
	if currency == "" {
		currency = "USD"
	}
	sign := "$"
	switch currency {
	case "EUR":
		sign = "€"
	case "GBP":
		sign = "£"
	case "PLN":
		sign = " zł"
	}
	return fmt.Sprintf("%s%.2f", sign, float64(cents)/100)
}
