package common

import (
	"strconv"
	"strings"
)

func FormatBudgetText(amountMin, amountMax int64) string {
	if amountMin > 0 && amountMax > 0 {
		return "از " + formatToman(amountMin) + " تا " + formatToman(amountMax) + " تومان"
	}
	if amountMax > 0 {
		return "تا " + formatToman(amountMax) + " تومان"
	}
	if amountMin > 0 {
		return "از " + formatToman(amountMin) + " تومان"
	}
	return "نامشخص"
}

func formatToman(amount int64) string {
	return formatWithCommas(strconv.FormatInt(amount, 10))
}

func formatWithCommas(input string) string {
	if len(input) <= 3 {
		return input
	}

	neg := strings.HasPrefix(input, "-")
	if neg {
		input = strings.TrimPrefix(input, "-")
	}

	n := len(input)
	first := n % 3
	if first == 0 {
		first = 3
	}

	parts := []string{input[:first]}
	for i := first; i < n; i += 3 {
		parts = append(parts, input[i:i+3])
	}

	result := strings.Join(parts, ",")
	if neg {
		return "-" + result
	}
	return result
}

func ToInt64(value any) int64 {
	switch v := value.(type) {
	case float64:
		return int64(v)
	case float32:
		return int64(v)
	case int:
		return int64(v)
	case int64:
		return v
	case int32:
		return int64(v)
	case jsonNumber:
		if i, err := v.Int64(); err == nil {
			return i
		}
		if f, err := v.Float64(); err == nil {
			return int64(f)
		}
	case string:
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i
		}
	}
	return 0
}

type jsonNumber interface {
	Int64() (int64, error)
	Float64() (float64, error)
	String() string
}

func ToString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case jsonNumber:
		return v.String()
	case float64:
		return strconv.FormatFloat(v, 'f', 0, 64)
	case int64:
		return strconv.FormatInt(v, 10)
	case int:
		return strconv.Itoa(v)
	default:
		return ""
	}
}
