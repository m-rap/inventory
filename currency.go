package inventory

type CurrencyConverter struct {
	rates map[string]float64 // e.g., "USD" → rate to base currency like IDR
	base  string
}

func NewCurrencyConverter(base string) *CurrencyConverter {
	return &CurrencyConverter{
		rates: make(map[string]float64),
		base:  base,
	}
}

func (cc *CurrencyConverter) AddRate(currency string, toBase float64) {
	cc.rates[currency] = toBase
}

func (cc *CurrencyConverter) ConvertToBase(amount float64, currency string) float64 {
	if currency == cc.base {
		return amount
	}
	if rate, ok := cc.rates[currency]; ok {
		return amount * rate
	}
	return amount // fallback: treat as base if unknown
}
