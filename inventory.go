package inventory

import (
	"sync"
)

type Item struct {
	ID          string            // Unique identifier
	Name        string            // Display name
	Category    string            // Optional grouping
	Price       float64           // Optional static price
	Currency    string            // e.g., "USD", "IDR"
	Metadata    map[string]string // Optional extra fields
	DefaultUnit string            // Optional: used if not overridden per transaction
}

type HookFunc func(tx Transaction, inv *Inventory) error

type UnitConversionRule struct {
	From   string
	To     string
	Factor float64
}

type CurrencyConversionRule struct {
	From string
	To   string
	Rate float64
}

type Inventory struct {
	ID             string
	Transactions   []Transaction
	mutex          sync.Mutex
	logs           []string
	hooks          []HookFunc
	SubInventories map[string]*Inventory

	RegisteredItems map[string]Item

	unitConversions     []UnitConversionRule
	currencyConversions []CurrencyConversionRule
}

func NewInventory(id string) *Inventory {
	return &Inventory{
		ID:                  id,
		SubInventories:      make(map[string]*Inventory),
		RegisteredItems:     make(map[string]Item),
		unitConversions:     []UnitConversionRule{},
		currencyConversions: []CurrencyConversionRule{},
	}
}

func (inv *Inventory) RegisterItem(item Item) {
	inv.mutex.Lock()
	defer inv.mutex.Unlock()
	inv.RegisteredItems[item.ID] = item
}

func (inv *Inventory) AddUnitConversionRule(from, to string, factor float64) {
	inv.unitConversions = append(inv.unitConversions, UnitConversionRule{from, to, factor})
}

func (inv *Inventory) AddCurrencyConversionRule(from, to string, rate float64) {
	inv.currencyConversions = append(inv.currencyConversions, CurrencyConversionRule{from, to, rate})
}

func (inv *Inventory) convertUnit(qty int, fromUnit, toUnit string) int {
	if fromUnit == toUnit {
		return qty
	}
	for _, rule := range inv.unitConversions {
		if rule.From == fromUnit && rule.To == toUnit {
			return int(float64(qty) * rule.Factor)
		}
		if rule.From == toUnit && rule.To == fromUnit {
			return int(float64(qty) / rule.Factor)
		}
	}
	return qty // fallback to original if no rule found
}

func (inv *Inventory) convertCurrency(amount float64, fromCur, toCur string) float64 {
	if fromCur == toCur {
		return amount
	}
	for _, rule := range inv.currencyConversions {
		if rule.From == fromCur && rule.To == toCur {
			return amount * rule.Rate
		}
		if rule.From == toCur && rule.To == fromCur {
			return amount / rule.Rate
		}
	}
	return amount // fallback
}

func (inv *Inventory) runHooks(tx Transaction) {
	for _, hook := range inv.hooks {
		_ = hook(tx, inv) // ignore errors for now
	}
}

func (inv *Inventory) GetBalances() map[string]int {
	balances := make(map[string]int)
	for _, tx := range inv.Transactions {
		for _, item := range tx.Items {
			balances[item.ItemID] = item.Balance
		}
	}
	return balances
}

func (inv *Inventory) GetItemReports(filter func(TransactionItem) bool) []TransactionItem {
	seen := make(map[string]bool)
	reports := []TransactionItem{}
	balances := inv.GetBalances()

	for i := len(inv.Transactions) - 1; i >= 0; i-- {
		for _, item := range inv.Transactions[i].Items {
			if seen[item.ItemID] {
				continue
			}
			if filter == nil || filter(item) {
				item.Balance = balances[item.ItemID]
				reports = append(reports, item)
				seen[item.ItemID] = true
			}
		}
	}

	return reports
}

func (inv *Inventory) GetTransactionsForItems(itemIDs []string) []Transaction {
	lookup := make(map[string]bool)
	for _, id := range itemIDs {
		lookup[id] = true
	}

	var result []Transaction
	for _, tx := range inv.Transactions {
		for _, item := range tx.Items {
			if lookup[item.ItemID] {
				result = append(result, tx)
				break
			}
		}
	}
	return result
}

func (inv *Inventory) AddItems(items []TransactionItem, note string) Transaction {
	return inv.AddTransaction(TransactionTypeAdd, items, note)
}

func (inv *Inventory) RemoveItems(items []TransactionItem, note string) Transaction {
	return inv.AddTransaction(TransactionTypeRemove, items, note)
}
