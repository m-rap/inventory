package inventory

type UnitConversion struct {
	FromUnit string
	ToUnit   string
	Factor   float64 // e.g., FromUnit * Factor = ToUnit
}

type UnitConverter struct {
	rules map[string]map[string]float64 // itemID -> fromUnit -> toBaseUnit factor
}

func NewUnitConverter() *UnitConverter {
	return &UnitConverter{
		rules: make(map[string]map[string]float64),
	}
}

// Add a rule like: 1 box = 10 pcs → (From: box, To: pcs, Factor: 10)
func (uc *UnitConverter) AddRule(itemID, fromUnit string, factorToBase float64) {
	if uc.rules[itemID] == nil {
		uc.rules[itemID] = make(map[string]float64)
	}
	uc.rules[itemID][fromUnit] = factorToBase
}

// Convert any quantity to base unit (e.g., pcs)
func (uc *UnitConverter) ToBase(itemID, unit string, qty int) int {
	if convs, ok := uc.rules[itemID]; ok {
		if factor, ok := convs[unit]; ok {
			return int(float64(qty) * factor)
		}
	}
	return qty // assume already base
}
