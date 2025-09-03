package inventory

import (
	"database/sql"
)

type UnitConversionRule struct {
	FromUnit string
	ToUnit   string
	Factor   float64
}

type CurrencyConversionRule struct {
	FromCurrency string
	ToCurrency   string
	Rate         float64
}

func ConvertUnit(db *sql.DB, quantity float64, fromUnit, toUnit string) (float64, error) {
	rule, err := LoadConversionRule(db, fromUnit, toUnit)
	if err != nil {
		return quantity, err
	}
	return quantity * rule.Factor, nil
}

func ConvertCurrency(db *sql.DB, amount float64, fromCurrency, toCurrency string) float64 {
	rule, err := LoadCurrencyConversionRule(db, fromCurrency, toCurrency)
	if err != nil {
		return amount
	}
	return amount * rule.Rate
}
