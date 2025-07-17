package inventory

import (
	"database/sql"
	"sync"
)

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

type Item struct {
	ID          string
	Name        string
	Description string
	Unit        string
	Currency    string
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

	db *sql.DB
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
	if inv.db != nil {
		_ = inv.persistInventory()
	}
}

func (inv *Inventory) AddUnitConversionRule(from, to string, factor float64) {
	inv.unitConversions = append(inv.unitConversions, UnitConversionRule{from, to, factor})
	if inv.db != nil {
		_ = inv.persistInventory()
	}
}

func (inv *Inventory) AddCurrencyConversionRule(from, to string, rate float64) {
	inv.currencyConversions = append(inv.currencyConversions, CurrencyConversionRule{from, to, rate})
	if inv.db != nil {
		_ = inv.persistInventory()
	}
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

func (inv *Inventory) persistInventory() error {
	_, err := inv.db.Exec(`INSERT OR REPLACE INTO inventories (id) VALUES (?)`, inv.ID)
	if err != nil {
		return err
	}
	for _, item := range inv.RegisteredItems {
		_, err := inv.db.Exec(`INSERT OR REPLACE INTO items (id, name, description, unit, currency) VALUES (?, ?, ?, ?, ?)`,
			item.ID, item.Name, item.Description, item.Unit, item.Currency)
		if err != nil {
			return err
		}
	}
	for _, tx := range inv.Transactions {
		if err := inv.persistTransaction(tx); err != nil {
			return err
		}
	}
	return nil
}

func PersistAllInventories(inventories map[string]*Inventory) error {
	for _, inv := range inventories {
		if inv.db != nil {
			if err := inv.persistInventory(); err != nil {
				return err
			}
		}
	}
	return nil
}

func LoadInventoriesFromDB(db *sql.DB) (map[string]*Inventory, error) {
	rows, err := db.Query(`SELECT id FROM inventories`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	inventories := make(map[string]*Inventory)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		inv := NewInventory(id)
		inv.db = db
		// Optionally load items
		itemRows, err := db.Query(`SELECT id, name, description, unit, currency FROM items`)
		if err != nil {
			return nil, err
		}
		for itemRows.Next() {
			var it Item
			if err := itemRows.Scan(&it.ID, &it.Name, &it.Description, &it.Unit, &it.Currency); err == nil {
				inv.RegisteredItems[it.ID] = it
			}
		}
		itemRows.Close()
		// Transactions are loaded in persistTransaction or lazily if needed
		inventories[id] = inv
	}
	return inventories, nil
}

func (inv *Inventory) runHooks(tx Transaction) {
	for _, hook := range inv.hooks {
		_ = hook(tx, inv) // ignore errors for now
	}
}

func (inv *Inventory) GetBalances() map[string]int {
	balances := make(map[string]int)
	seen := make(map[string]bool)

	for i := len(inv.Transactions) - 1; i >= 0; i-- {
		for _, item := range inv.Transactions[i].Items {
			if !seen[item.ItemID] {
				balances[item.ItemID] = item.Balance
				seen[item.ItemID] = true
			}
		}
		if len(seen) == len(inv.RegisteredItems) {
			break
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
