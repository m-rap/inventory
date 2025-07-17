package inventory

import (
	"fmt"
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

type Inventory struct {
	mutex          sync.Mutex
	ID             string
	Items          map[string]Item // All known items (metadata only)
	Transactions   []Transaction   // Append-only list of all transactions
	hooks          []HookFunc
	logs           []string
	Converter      *UnitConverter
	Currency       *CurrencyConverter
	SubInventories map[string]*Inventory
}

type ItemReport struct {
	Item     Item
	Quantity int
}

type ItemReportFilter struct {
	ID       string // Exact ID match (optional)
	Name     string // Contains match (optional)
	Category string // Exact category match (optional)
	MinQty   *int   // Optional: Minimum quantity
	MaxQty   *int   // Optional: Maximum quantity
}

func NewInventory(baseCurrency string) *Inventory {
	return &Inventory{
		Items:        make(map[string]Item),
		Transactions: make([]Transaction, 0),
		logs:         make([]string, 0),
		hooks:        make([]HookFunc, 0),
		Converter:    NewUnitConverter(),
		Currency:     NewCurrencyConverter(baseCurrency),
	}
}

func (inv *Inventory) RegisterItem(item Item) {
	if inv.Items == nil {
		inv.Items = make(map[string]Item)
	}
	inv.Items[item.ID] = item
}

// Backward-compatible alias
func (inv *Inventory) GetStockLevels() map[string]int {
	return inv.GetBalances()
}

func (inv *Inventory) GetInventoryValueInBaseCurrency() float64 {
	total := 0.0
	for _, tx := range inv.Transactions {
		for _, ti := range tx.Items {
			baseQty := inv.Converter.ToBase(ti.ItemID, ti.Unit, ti.Quantity)
			total += float64(baseQty) * inv.Currency.ConvertToBase(ti.UnitPrice, ti.Currency)
		}
	}
	return total
}

func (inv *Inventory) RegisterHook(hook HookFunc) {
	inv.mutex.Lock()
	defer inv.mutex.Unlock()
	inv.hooks = append(inv.hooks, hook)
}

func LowStockAlert(threshold int) HookFunc {
	return func(tx Transaction, inv *Inventory) error {
		balances := inv.GetBalances()
		for _, ti := range tx.Items {
			if qty := balances[ti.ItemID]; qty < threshold {
				item := inv.Items[ti.ItemID]
				fmt.Printf("[ALERT] Low stock: %s (ID: %s) has %d remaining\n", item.Name, item.ID, qty)
			}
		}
		return nil
	}
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
