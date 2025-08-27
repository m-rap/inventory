package inventory

import (
	"database/sql"
	"fmt"
	"sort"
	"sync"
	"time"
)

const (
	TransactionTypeAdd    = 1
	TransactionTypeRemove = -1
)

type HookFunc func(tx Transaction, inv *Inventory) error

type Item struct {
	ID          string
	Name        string
	Description string
	Unit        string
	Currency    string
}

type Balance struct {
	Quantity int
	Value    float64
	Unit     string
	Currency string
}

type Inventory struct {
	ID             string
	ParentID       string
	Transactions   []Transaction
	mutex          sync.Mutex
	logs           []string
	hooks          []HookFunc
	SubInventories map[string]*Inventory

	RegisteredItems map[string]Item
	db              *sql.DB
}

type Transaction struct {
	ID          string
	InventoryID string
	Type        int
	Timestamp   time.Time
	Items       []TransactionItem
	Note        string
}

type TransactionItem struct {
	ItemID    string
	Quantity  int
	Unit      string
	Balance   int
	UnitPrice float64
	Currency  string
}

func NewInventory(id string) *Inventory {
	return &Inventory{
		ID:              id,
		SubInventories:  make(map[string]*Inventory),
		RegisteredItems: make(map[string]Item),
	}
}

func (inv *Inventory) RegisterItem(item Item) {
	inv.loadFromPersistence()
	inv.mutex.Lock()
	defer inv.mutex.Unlock()
	inv.RegisteredItems[item.ID] = item
	PersistItem(inv.db, inv.ID, item)
}

func (inv *Inventory) UpdateItem(item Item) {
	inv.loadFromPersistence()
	inv.mutex.Lock()
	defer inv.mutex.Unlock()
	if _, exists := inv.RegisteredItems[item.ID]; exists {
		inv.RegisteredItems[item.ID] = item
		PersistItem(inv.db, inv.ID, item)
	}
}

func (inv *Inventory) DeleteItem(itemID string) {
	inv.loadFromPersistence()
	inv.mutex.Lock()
	defer inv.mutex.Unlock()
	delete(inv.RegisteredItems, itemID)
	DeleteItemFromDB(inv.db, inv.ID, itemID)
}

func (inv *Inventory) AddTransaction(tx Transaction) {
	inv.loadFromPersistence()
	inv.mutex.Lock()
	defer inv.mutex.Unlock()
	inv.Transactions = append(inv.Transactions, tx)
	sort.Slice(inv.Transactions, func(i, j int) bool {
		return inv.Transactions[i].Timestamp.Before(inv.Transactions[j].Timestamp)
	})

	inv.updateTransactionBalancesNoLock([]string{}, tx.Timestamp)
	PersistInventorySince(inv.db, inv.ID, tx.Timestamp, extractItemIDs(tx))
	_ = RunHooks(tx, inv)
}

func (inv *Inventory) UpdateTransaction(updatedTx Transaction) {
	inv.loadFromPersistence()
	inv.mutex.Lock()
	defer inv.mutex.Unlock()
	for i, tx := range inv.Transactions {
		if tx.ID == updatedTx.ID {
			inv.Transactions[i] = updatedTx
			inv.updateTransactionBalancesNoLock([]string{}, updatedTx.Timestamp)
			PersistInventorySince(inv.db, inv.ID, updatedTx.Timestamp, extractItemIDs(updatedTx))
			return
		}
	}
}

func (inv *Inventory) DeleteTransaction(txID string) {
	inv.loadFromPersistence()
	inv.mutex.Lock()
	defer inv.mutex.Unlock()
	for i, tx := range inv.Transactions {
		if tx.ID == txID {
			inv.Transactions = append(inv.Transactions[:i], inv.Transactions[i+1:]...)
			inv.updateTransactionBalancesNoLock([]string{}, time.Now())
			DeleteTransactionFromDB(inv.db, inv.ID, txID)
			return
		}
	}
}

func (inv *Inventory) AddTransactionToSub(subID string, tx Transaction) {
	inv.loadFromPersistence()
	inv.mutex.Lock()
	sub, ok := inv.SubInventories[subID]
	inv.mutex.Unlock()
	if !ok {
		return
	}
	sub.AddTransaction(tx)
}

func (inv *Inventory) RemoveItems(items []TransactionItem, note string, timestamp time.Time) {
	inv.loadFromPersistence()
	tx := Transaction{
		ID:          GenerateUUID(),
		InventoryID: inv.ID,
		Type:        TransactionTypeRemove,
		Timestamp:   timestamp,
		Items:       items,
		Note:        note,
	}
	inv.AddTransaction(tx)
}

func (inv *Inventory) AddItems(items []TransactionItem, note string, timestamp time.Time) {
	inv.loadFromPersistence()
	tx := Transaction{
		ID:          GenerateUUID(),
		InventoryID: inv.ID,
		Type:        TransactionTypeAdd,
		Timestamp:   timestamp,
		Items:       items,
		Note:        note,
	}
	inv.AddTransaction(tx)
}

func (inv *Inventory) GetTransactionsForItems(itemIDs []string) []Transaction {
	inv.loadFromPersistence()
	inv.mutex.Lock()
	defer inv.mutex.Unlock()
	var filtered []Transaction
	for _, tx := range inv.Transactions {
		for _, item := range tx.Items {
			if contains(itemIDs, item.ItemID) {
				filtered = append(filtered, tx)
				break
			}
		}
	}
	return filtered
}

func (inv *Inventory) GetItemReports() map[string][]TransactionItem {
	inv.loadFromPersistence()
	reports := make(map[string][]TransactionItem)
	inv.mutex.Lock()
	defer inv.mutex.Unlock()
	for _, tx := range inv.Transactions {
		for _, item := range tx.Items {
			reports[item.ItemID] = append(reports[item.ItemID], item)
		}
	}
	return reports
}

func normalizeItem(item TransactionItem, base Item) (int, float64) {
	qty := ConvertUnit(item.Quantity, item.Unit, base.Unit)
	cost := ConvertCurrency(item.UnitPrice, item.Currency, base.Currency)
	return qty, cost
}

func (inv *Inventory) GetBalances() map[string]Balance {
	inv.loadFromPersistence()
	inv.mutex.Lock()
	defer inv.mutex.Unlock()
	balances := make(map[string]Balance)
	for _, tx := range inv.Transactions {
		for _, item := range tx.Items {
			base, ok := inv.RegisteredItems[item.ItemID]
			if !ok {
				continue
			}
			qty, cost := normalizeItem(item, base)
			bal := balances[item.ItemID]
			if tx.Type == TransactionTypeAdd {
				bal.Quantity += qty
				bal.Value += float64(qty) * cost
			} else if tx.Type == TransactionTypeRemove {
				bal.Quantity -= qty
				bal.Value -= float64(qty) * cost
			}
			bal.Unit = base.Unit
			bal.Currency = base.Currency
			balances[item.ItemID] = bal
		}
	}
	return balances
}

func (inv *Inventory) GetBalancesRecursive() map[string]Balance {
	inv.loadFromPersistence()
	balances := inv.GetBalances()
	for _, sub := range inv.SubInventories {
		subBalances := sub.GetBalancesRecursive()
		for k, v := range subBalances {
			bal := balances[k]
			bal.Quantity += v.Quantity
			bal.Value += v.Value
			if bal.Unit == "" {
				bal.Unit = v.Unit
				bal.Currency = v.Currency
			}
			balances[k] = bal
		}
	}
	return balances
}

func (inv *Inventory) GetBalancesForItems(itemIDs []string) map[string]Balance {
	inv.loadFromPersistence()
	inv.mutex.Lock()
	defer inv.mutex.Unlock()
	balances := make(map[string]Balance)
	for _, tx := range inv.Transactions {
		for _, item := range tx.Items {
			if contains(itemIDs, item.ItemID) {
				base, ok := inv.RegisteredItems[item.ItemID]
				if !ok {
					continue
				}
				qty, cost := normalizeItem(item, base)
				bal := balances[item.ItemID]
				if tx.Type == TransactionTypeAdd {
					bal.Quantity += qty
					bal.Value += float64(qty) * cost
				} else if tx.Type == TransactionTypeRemove {
					bal.Quantity -= qty
					bal.Value -= float64(qty) * cost
				}
				bal.Unit = base.Unit
				bal.Currency = base.Currency
				balances[item.ItemID] = bal
			}
		}
	}
	return balances
}

// Public safe wrapper
func (inv *Inventory) UpdateTransactionBalances(itemIDs []string, since time.Time) {
	inv.mutex.Lock()
	defer inv.mutex.Unlock()
	inv.updateTransactionBalancesNoLock(itemIDs, since)
}

// Internal: assumes lock is already held
func (inv *Inventory) updateTransactionBalancesNoLock(itemIDs []string, since time.Time) {
	sort.Slice(inv.Transactions, func(i, j int) bool {
		return inv.Transactions[i].Timestamp.Before(inv.Transactions[j].Timestamp)
	})

	balances := make(map[string]int)
	for i := range inv.Transactions {
		tx := &inv.Transactions[i]
		if tx.Timestamp.Before(since) {
			continue
		}
		for j := range tx.Items {
			item := &tx.Items[j]
			if len(itemIDs) > 0 && !contains(itemIDs, item.ItemID) {
				continue
			}
			bal := balances[item.ItemID]
			if tx.Type == TransactionTypeAdd {
				bal += item.Quantity
			} else if tx.Type == TransactionTypeRemove {
				bal -= item.Quantity
			}
			item.Balance = bal
			balances[item.ItemID] = bal
		}
	}
}

func (inv *Inventory) loadFromPersistence() {
	if inv.db != nil {
		LoadInventory(inv.db, inv)
	}
}

func RunHooks(tx Transaction, inv *Inventory) error {
	for _, hook := range inv.hooks {
		if err := hook(tx, inv); err != nil {
			return err
		}
	}
	return nil
}

func extractItemIDs(tx Transaction) []string {
	ids := make([]string, 0, len(tx.Items))
	for _, item := range tx.Items {
		ids = append(ids, item.ItemID)
	}
	return ids
}

func GenerateUUID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func contains(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}
