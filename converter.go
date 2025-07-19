package inventory

import (
	"sync"
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

var (
	unitConversionRules     []UnitConversionRule
	currencyConversionRules []CurrencyConversionRule
	conversionMutex         sync.Mutex
)

func AddUnitConversionRule(rule UnitConversionRule) {
	conversionMutex.Lock()
	defer conversionMutex.Unlock()
	unitConversionRules = append(unitConversionRules, rule)
}

func AddCurrencyConversionRule(rule CurrencyConversionRule) {
	conversionMutex.Lock()
	defer conversionMutex.Unlock()
	currencyConversionRules = append(currencyConversionRules, rule)
}

func ConvertUnit(quantity int, fromUnit, toUnit string) int {
	conversionMutex.Lock()
	defer conversionMutex.Unlock()
	for _, rule := range unitConversionRules {
		if rule.FromUnit == fromUnit && rule.ToUnit == toUnit {
			return int(float64(quantity) * rule.Factor)
		}
	}
	return quantity
}

func ConvertCurrency(amount float64, fromCurrency, toCurrency string) float64 {
	conversionMutex.Lock()
	defer conversionMutex.Unlock()
	for _, rule := range currencyConversionRules {
		if rule.FromCurrency == fromCurrency && rule.ToCurrency == toCurrency {
			return amount * rule.Rate
		}
	}
	return amount
}
