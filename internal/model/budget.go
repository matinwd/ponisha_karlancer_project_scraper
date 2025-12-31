package model

const TomanThreshold int64 = 99_000_000

func IsAboveThreshold(amountMin, amountMax int64) bool {
	return amountMax > TomanThreshold || amountMin > TomanThreshold
}
