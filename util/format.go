package util

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/spf13/cast"
)

func CutPointRight(s string) string {
	parts := strings.Split(s, ".")
	return parts[0]
}

func ShiftLeft(num decimal.Decimal, decimals int32) decimal.Decimal {
	return num.Shift(-decimals)
}

func ShiftLeftStr(num string, decimals string) string {
	n, _ := decimal.NewFromString(num)
	d := cast.ToInt32(decimals)
	return n.Shift(-d).String()
}

func ShiftRightStr(num string, decimals string) string {
	n, _ := decimal.NewFromString(num)
	d := cast.ToInt32(decimals)
	return n.Shift(d).String()
}

func ShiftRight(num decimal.Decimal, decimals int32) decimal.Decimal {
	return num.Shift(decimals)
}

func ParseTokenAmountByDecimals(amount string, decimals int32) (string, error) {
	amountDecimal, err := decimal.NewFromString(amount)
	if err != nil {
		return "", fmt.Errorf("invalid amount: %v", err)
	}

	exp := decimal.New(1, decimals)
	result := amountDecimal.Mul(exp)

	return result.BigInt().String(), nil
}

func FormatJSONNumber(num string) json.Number {
	if num == "" {
		return ""
	}
	bigFloat, _, err := big.ParseFloat(num, 10, 0, big.ToNearestEven)
	threshold := big.NewFloat(0.001)
	if bigFloat.Cmp(threshold) < 0 {
		text := bigFloat.Text('f', -1)
		return json.Number(text)
	}
	if err != nil {
		return ""
	}

	formattedNum := bigFloat.Text('f', 3)
	formattedNum = strings.TrimRight(formattedNum, "0")
	formattedNum = strings.TrimRight(formattedNum, ".")

	return json.Number(formattedNum)
}

func FormatNumber_old(numStr string) string {
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return numStr
	}

	if num == 0 {
		return "0"
	}

	if num < 1 {
		parts := strings.Split(numStr, ".")
		if len(parts) == 2 && len(parts[1]) > 3 {
			zeroCount := 0
			for _, ch := range parts[1] {
				if ch == '0' {
					zeroCount++
				} else {
					break
				}
			}
			if zeroCount > 3 {
				if len(parts[1]) >= zeroCount+4 {
					return fmt.Sprintf("0.0{%d}%s", zeroCount, parts[1][zeroCount:zeroCount+4])
				} else {
					return fmt.Sprintf("0.0{%d}%s", zeroCount, parts[1][zeroCount:])
				}
			} else {
				return fmt.Sprintf("%.5f", num)
			}
		}
	} else {
		num = math.Round(num*100) / 100
		if num >= 1e8 {
			return fmt.Sprintf("%.2f亿", num/1e8)
		} else if num >= 1e4 {
			return fmt.Sprintf("%.2f万", num/1e4)
		} else {
			return fmt.Sprintf("%.2f", num)
		}
	}

	return numStr
}

func FormatNumber(numStr string) string {
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return numStr
	}

	if num == 0 {
		return "0"
	}

	parts := strings.Split(numStr, ".")
	if len(parts) == 1 {
		return numStr
	}

	if num > 0 && num < 1 {
		parts := strings.Split(numStr, ".")
		zeroCount := 0
		for _, r := range parts[1] {
			if r == '0' {
				zeroCount++
			} else {
				break
			}
		}
		if zeroCount > 2 {
			rightPart := parts[1]
			restPart := truncateString(rightPart[zeroCount:], 3)
			return fmt.Sprintf("%s.{%d}%s", parts[0], zeroCount, restPart)
		} else {
			rightPart := parts[1]
			restPart := truncateString(rightPart[zeroCount:], 3)
			return fmt.Sprintf("%s.%s%s", parts[0], zero(zeroCount), restPart)
		}
	}

	if num > 1 && num < 100_0000 {
		parts := strings.Split(numStr, ".")
		rightPart := parts[1]
		restPart := truncateString(rightPart, 2)
		return truncateZero(fmt.Sprintf("%s.%s", parts[0], restPart))
	}

	if num > 100_0000 {
		parts := strings.Split(numStr, ".")
		rightPart := parts[1]
		restPart := truncateString(rightPart, 2)

		leftPart := parts[0][:3]
		str := truncateZero(fmt.Sprintf("%s.%s", leftPart, restPart))
		return str + "万"
	}
	return numStr
}

func zero(count int) string {
	runes := []rune{}
	for range count {
		runes = append(runes, '0')
	}
	return string(runes)
}

func truncateZero(s string) string {
	if !strings.Contains(s, ".") {
		return s
	}
	parts := strings.Split(s, ".")

	if parts[1] == "" || strings.Count(parts[1], "0") == len(parts[1]) {
		return parts[0]
	}

	return s
}

func truncateString(s string, maxLen int) string {
	runes := []rune(s)

	runeLength := len(runes)

	if runeLength > maxLen {
		return string(runes[:maxLen])
	}

	return s
}

func FormatTime(timestamp string) string {
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return timestamp
	}

	t := time.Unix(ts/1000, 0)

	return t.Format("2006-01-02 15:04:05")
}

func FormatPercentage(per string) string {
	num, err := strconv.ParseFloat(per, 64)
	if err != nil {
		return per
	}

	// 0.3 30%
	// 0.35 35%
	percentageRaw := num * 100

	perStr := strconv.FormatFloat(percentageRaw, 'f', 0, 64)
	return perStr + "%"
}
