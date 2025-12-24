package inventory

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

var DECIMALPRECISION = 4

type Decimal struct {
	FracDivisor int32
	Data        int64
	Format      string
}

func NewDecimal(rawI64 int64) Decimal {
	fracDivisor := int32(math.Pow10(DECIMALPRECISION))
	return Decimal{
		FracDivisor: fracDivisor,
		Data:        rawI64,
		Format:      fmt.Sprintf("%%d.%%0%.0fd", math.Log10(float64(fracDivisor))),
	}
}

func NewDecimalFromIntFrac(intPart, fracPartPrecModulo int64) Decimal {
	fracDivisor := int64(math.Pow10(DECIMALPRECISION))
	if intPart >= 0 {
		return NewDecimal(intPart*fracDivisor + fracPartPrecModulo)
	} else {
		return NewDecimal(intPart*fracDivisor - fracPartPrecModulo)
	}
}

func NewDecimalFromFloat(fdata float64) Decimal {
	fracDivisor := math.Pow10(DECIMALPRECISION)
	return NewDecimal(int64(fdata * fracDivisor))
}

func NewDecimalFromStr(str string) Decimal {
	var intPart int64
	var fracPart int64
	var fracStr string
	strs := strings.Split(str, ".")
	if len(strs) >= 1 {
		iTmp, _ := strconv.Atoi(strs[0])
		intPart = int64(iTmp)
		if len(strs) > 1 {
			fracStr = strs[1]
		} else {
			fracStr = "0"
		}
	}
	frontZeroCount := 0
	nonPaddedCount := 0
	nonPaddedStr := ""
	for _, c := range fracStr {
		if nonPaddedCount == 0 {
			if c == '0' {
				frontZeroCount++
			} else {
				nonPaddedCount++
				nonPaddedStr += string(c)
			}
		} else {
			nonPaddedCount++
		}
	}
	decPrecZeroDigitCount := DECIMALPRECISION
	// misal 0.02, zeroCount = 1, nonPaddedCount = 1. (2 * 10^2)/10000
	// misal 0.0002, zeroCount = 3, nonPaddedCount = 1 -> (2 * 10^0)/10000
	// misal 0.2, zeroCount = 0, nonPaddedCount = 1 -> (2 * 10^3)/10000
	// misal 0.20, zeroCount = 0, nonPaddedCount = 2 -> (20 * 10^2)/10000
	powFactor := decPrecZeroDigitCount - (frontZeroCount + nonPaddedCount)
	nonPaddedInt, _ := strconv.Atoi(nonPaddedStr)
	fracPart = int64(nonPaddedInt) * int64(math.Pow10(powFactor))
	return NewDecimalFromIntFrac(intPart, fracPart)
}

func (d Decimal) ToFloat() float64 {
	return float64(d.Data) / float64(d.FracDivisor)
}

func (d Decimal) ToIntFrac() (int64, int64) {
	return d.Data / int64(d.FracDivisor), d.Data % int64(d.FracDivisor)
}

func (d Decimal) ToString() string {
	return fmt.Sprintf(d.Format, d.Data/int64(d.FracDivisor), d.Data%int64(d.FracDivisor))
}

func (d *Decimal) Scan(src any) error {
	iSrc, ok := src.(int64)
	if !ok {
		return errors.New("src must be int64")
	}
	dst := NewDecimal(iSrc)
	*d = dst
	return nil
}

func (d Decimal) Value() (driver.Value, error) {
	return d.Data, nil
}

func (d Decimal) Divide(divisor Decimal) Decimal {
	// d as dividend
	tmp := d.Data * int64(d.FracDivisor) / divisor.Data
	return NewDecimalFromIntFrac(tmp/int64(d.FracDivisor), tmp%int64(d.FracDivisor))
}

func (d Decimal) Multiply(multiplicand Decimal) Decimal {
	// d as multiplier
	return NewDecimal((d.Data * multiplicand.Data) / int64(d.FracDivisor))
}
