package inventory

import (
	"fmt"
	"strings"
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
	mu           sync.Mutex
	Items        map[string]Item // All known items (metadata only)
	Transactions []Transaction   // Append-only list of all transactions
	hooks        []HookFunc
	logs         []string
	Converter    *UnitConverter
	Currency     *CurrencyConverter
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
func (inv *Inventory) GetStockLevels() (map[string]int, error) {
	return inv.GetBalances()
}

func (inv *Inventory) GetBalances() (map[string]int, error) {
	stock := make(map[string]int)
	for _, tx := range inv.Transactions {
		for _, ti := range tx.Items {
			if len(tx.Items) == 0 {
				continue
			}
			// item := inv.Items[ti.ItemID]
			// if item.DefaultUnit != "" && item.DefaultUnit != ti.Unit {
			// 	return stock, fmt.Errorf("unit mismatch: expected %s, got %s", item.DefaultUnit, ti.Unit)
			// }
			baseQty := inv.Converter.ToBase(ti.ItemID, ti.Unit, ti.Quantity)
			switch tx.Type {
			case TransactionAdd:
				stock[ti.ItemID] += baseQty
			case TransactionRemove:
				stock[ti.ItemID] -= baseQty
			case TransactionAdjust:
				stock[ti.ItemID] += baseQty
			}
		}
	}
	return stock, nil
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

func (inv *Inventory) GetItemReports(filter ItemReportFilter) []ItemReport {
	// Compute current quantity per item
	stock := make(map[string]int)
	for _, tx := range inv.Transactions {
		for _, ti := range tx.Items {
			switch tx.Type {
			case TransactionAdd:
				stock[ti.ItemID] += ti.Quantity
			case TransactionRemove:
				stock[ti.ItemID] -= ti.Quantity
			case TransactionAdjust:
				stock[ti.ItemID] += ti.Quantity
			}
		}
	}

	var reports []ItemReport
	for id, item := range inv.Items {
		qty := stock[id]

		// Apply filters
		if filter.ID != "" && item.ID != filter.ID {
			continue
		}
		if filter.Name != "" && !strings.Contains(strings.ToLower(item.Name), strings.ToLower(filter.Name)) {
			continue
		}
		if filter.Category != "" && item.Category != filter.Category {
			continue
		}
		if filter.MinQty != nil && qty < *filter.MinQty {
			continue
		}
		if filter.MaxQty != nil && qty > *filter.MaxQty {
			continue
		}

		reports = append(reports, ItemReport{
			Item:     item,
			Quantity: qty,
		})
	}

	return reports
}

func (inv *Inventory) GetTransactionsForItem(itemID string) []Transaction {
	var result []Transaction
	for _, tx := range inv.Transactions {
		for _, ti := range tx.Items {
			if ti.ItemID == itemID {
				result = append(result, tx)
				break // No need to check the rest of the items in this tx
			}
		}
	}
	return result
}

func (inv *Inventory) AddItems(items []TransactionItem, note string) {
	inv.AddTransaction(TransactionAdd, items, note)
}

func (inv *Inventory) RemoveItems(items []TransactionItem, note string) {
	inv.AddTransaction(TransactionRemove, items, note)
}

func (inv *Inventory) RegisterHook(hook HookFunc) {
	inv.mu.Lock()
	defer inv.mu.Unlock()
	inv.hooks = append(inv.hooks, hook)
}

func (inv *Inventory) runHooks(tx Transaction) {
	for _, hook := range inv.hooks {
		hook(tx, inv)
	}
}

func LowStockAlert(threshold int) HookFunc {
	return func(tx Transaction, inv *Inventory) error {
		balances, err := inv.GetBalances()
		if err != nil {
			return err
		}
		for _, ti := range tx.Items {
			if qty := balances[ti.ItemID]; qty < threshold {
				item := inv.Items[ti.ItemID]
				fmt.Printf("[ALERT] Low stock: %s (ID: %s) has %d remaining\n", item.Name, item.ID, qty)
			}
		}
		return nil
	}
}
