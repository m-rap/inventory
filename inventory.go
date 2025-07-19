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

func (inv *Inventory) AddTransaction(tx Transaction) {
	inv.loadFromPersistence()
	inv.mutex.Lock()
	defer inv.mutex.Unlock()
	inv.Transactions = append(inv.Transactions, tx)
	sort.Slice(inv.Transactions, func(i, j int) bool {
		return inv.Transactions[i].Timestamp.Before(inv.Transactions[j].Timestamp)
	})
	inv.UpdateTransactionBalances([]string{}, tx.Timestamp)
	PersistInventorySince(inv.db, inv.ID, tx.Timestamp, extractItemIDs(tx))
	_ = RunHooks(tx, inv)
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

func (inv *Inventory) GetBalances() map[string]int {
	inv.loadFromPersistence()
	inv.mutex.Lock()
	defer inv.mutex.Unlock()
	balances := make(map[string]int)
	for _, tx := range inv.Transactions {
		for _, item := range tx.Items {
			balances[item.ItemID] = item.Balance
		}
	}
	return balances
}

func (inv *Inventory) GetBalancesForItems(itemIDs []string) map[string]int {
	inv.loadFromPersistence()
	inv.mutex.Lock()
	defer inv.mutex.Unlock()
	balances := make(map[string]int)
	for _, tx := range inv.Transactions {
		for _, item := range tx.Items {
			if contains(itemIDs, item.ItemID) {
				balances[item.ItemID] = item.Balance
			}
		}
	}
	return balances
}

func (inv *Inventory) UpdateTransactionBalances(itemIDs []string, since time.Time) {
	sort.Slice(inv.Transactions, func(i, j int) bool {
		return inv.Transactions[i].Timestamp.Before(inv.Transactions[j].Timestamp)
	})

	inv.mutex.Lock()
	defer inv.mutex.Unlock()
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
