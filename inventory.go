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
	Parent         *Inventory
	Transactions   []Transaction
	mutex          sync.Mutex
	logs           []string
	hooks          []HookFunc
	SubInventories map[string]*Inventory

	db *sql.DB
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

var (
	items     = make(map[string]Item) // in-memory global item cache
	itemsLock sync.RWMutex
)

func NewInventory(id string) *Inventory {
	return &Inventory{
		ID:             id,
		SubInventories: make(map[string]*Inventory),
	}
}

// RegisterItem now persists globally, not per-inventory, and updates in-memory cache
func RegisterItem(db *sql.DB, item Item) {
	PersistItem(db, "", item)
	itemsLock.Lock()
	items[item.ID] = item
	itemsLock.Unlock()
}

// UpdateItem now persists globally, not per-inventory, and updates in-memory cache
func UpdateItem(db *sql.DB, item Item) {
	PersistItem(db, "", item)
	itemsLock.Lock()
	items[item.ID] = item
	itemsLock.Unlock()
}

// DeleteItem now deletes globally, not per-inventory, and removes from in-memory cache
func DeleteItem(db *sql.DB, itemID string) {
	DeleteItemFromDB(db, "", itemID)
	itemsLock.Lock()
	delete(items, itemID)
	itemsLock.Unlock()
}

// GetItem fetches a single item from in-memory cache, falls back to DB if not found
func GetItem(db *sql.DB, itemID string) (Item, bool) {
	itemsLock.RLock()
	item, ok := items[itemID]
	itemsLock.RUnlock()
	if ok {
		return item, true
	}
	// Fallback to DB and update cache
	row := db.QueryRow("SELECT name, description, unit, currency FROM items WHERE item_id = ?", itemID)
	item = Item{ID: itemID}
	err := row.Scan(&item.Name, &item.Description, &item.Unit, &item.Currency)
	if err != nil {
		return Item{}, false
	}
	itemsLock.Lock()
	items[itemID] = item
	itemsLock.Unlock()
	return item, true
}

// GetAllItems fetches all items from in-memory cache if populated, otherwise loads from DB
func GetAllItems(db *sql.DB) ([]Item, error) {
	itemsLock.RLock()
	if len(items) > 0 {
		all := make([]Item, 0, len(items))
		for _, item := range items {
			all = append(all, item)
		}
		itemsLock.RUnlock()
		return all, nil
	}
	itemsLock.RUnlock()

	rows, err := db.Query("SELECT item_id, name, description, unit, currency FROM items")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var all []Item
	itemsLock.Lock()
	defer itemsLock.Unlock()
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.Unit, &item.Currency); err != nil {
			return nil, err
		}
		items[item.ID] = item
		all = append(all, item)
	}
	return all, nil
}

func (inv *Inventory) AddTransaction(tx Transaction) {
	inv.LoadChildren()
	inv.mutex.Lock()
	defer inv.mutex.Unlock()
	tx.InventoryID = inv.ID
	inv.Transactions = append(inv.Transactions, tx)
	sort.Slice(inv.Transactions, func(i, j int) bool {
		return inv.Transactions[i].Timestamp.Before(inv.Transactions[j].Timestamp)
	})

	inv.updateTransactionBalancesNoLock([]string{}, tx.Timestamp)
	PersistInventorySince(inv.db, inv.ID, tx.Timestamp, extractItemIDs(tx))
	_ = RunHooks(tx, inv)
}

func (inv *Inventory) UpdateTransaction(updatedTx Transaction) {
	inv.LoadChildren()
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
	inv.LoadChildren()
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
	inv.LoadChildren()
	inv.mutex.Lock()
	sub, ok := inv.SubInventories[subID]
	inv.mutex.Unlock()
	if !ok {
		return
	}
	sub.AddTransaction(tx)
}

func (inv *Inventory) RemoveItems(items []TransactionItem, note string, timestamp time.Time) {
	inv.LoadChildren()
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
	inv.LoadChildren()
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
	inv.LoadChildren()
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
	inv.LoadChildren()
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
	inv.LoadChildren()
	inv.mutex.Lock()
	defer inv.mutex.Unlock()
	balances := make(map[string]Balance)
	for _, tx := range inv.Transactions {
		for _, item := range tx.Items {
			base, ok := GetItem(inv.db, item.ItemID)
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
	inv.LoadChildren()
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

func (inv *Inventory) getRegisteredItem(itemID string) (Item, bool) {
	// Now fetch globally
	if inv.db == nil {
		return Item{}, false
	}
	return GetItem(inv.db, itemID)
}

func (inv *Inventory) GetBalancesForItems(itemIDs []string) map[string]Balance {
	inv.LoadChildren()
	inv.mutex.Lock()
	defer inv.mutex.Unlock()
	balances := make(map[string]Balance)
	// Calculate balances for current inventory
	for _, tx := range inv.Transactions {
		for _, item := range tx.Items {
			if contains(itemIDs, item.ItemID) {
				base, ok := inv.getRegisteredItem(item.ItemID)
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
	// Recursively add balances from sub-inventories
	for _, sub := range inv.SubInventories {
		subBalances := sub.GetBalancesForItems(itemIDs)
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
