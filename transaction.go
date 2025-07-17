package inventory

import (
	"time"

	"github.com/google/uuid"
)

const (
	TransactionTypeAdd    = 1
	TransactionTypeRemove = -1
)

const (
	UnitPiece = "pcs"
	UnitKg    = "kg"
	UnitBox   = "box"
	// Add more as needed
)

type TransactionItem struct {
	ItemID      string
	InventoryID string
	Quantity    int
	Unit        string // e.g., "pcs", "kg", "box"
	Balance     int    // New: stock level *after* this transaction

	UnitPrice float64 // Price per unit in given currency at time of transaction
	Currency  string  // Currency at the time of transaction
}

type Transaction struct {
	ID          string
	InventoryID string
	Type        int
	Timestamp   time.Time
	Items       []TransactionItem
	Note        string
}

func (inv *Inventory) AddTransaction(tType int, items []TransactionItem, note string) Transaction {
	return inv.AddTransactionToSub("", tType, items, note)
}

func generateID() string {
	return uuid.New().String()
}

func (inv *Inventory) AddTransactionToSub(subID string, tType int, items []TransactionItem, note string) Transaction {
	inv.mutex.Lock()
	defer inv.mutex.Unlock()

	target := inv
	inventoryID := inv.ID
	if subID != "" {
		if _, exists := inv.SubInventories[subID]; !exists {
			inv.SubInventories[subID] = NewInventory(subID)
		}
		target = inv.SubInventories[subID]
		inventoryID = subID
	}

	tx := Transaction{
		ID:          generateID(),
		InventoryID: inventoryID,
		Type:        tType,
		Timestamp:   time.Now(),
		Items:       items,
		Note:        note,
	}

	balances := target.GetBalances()
	for i := range tx.Items {
		change := tx.Items[i].Quantity
		if tx.Type < 0 {
			change *= -1
		}
		tx.Items[i].Balance = balances[tx.Items[i].ItemID] + change
		balances[tx.Items[i].ItemID] = tx.Items[i].Balance
	}

	target.Transactions = append(target.Transactions, tx)
	target.logs = append(target.logs, "Transaction "+tx.ID+" added")
	target.runHooks(tx)
	return tx
}

func (inv *Inventory) GetLastBalances() map[string]int {
	balances := make(map[string]int)
	for _, tx := range inv.Transactions {
		for _, ti := range tx.Items {
			balances[ti.ItemID] = ti.Balance
		}
	}
	return balances
}

func (inv *Inventory) GetBalanceForItem(itemID string) int {
	for i := len(inv.Transactions) - 1; i >= 0; i-- {
		for _, ti := range inv.Transactions[i].Items {
			if ti.ItemID == itemID {
				return ti.Balance
			}
		}
	}
	return 0
}

func (inv *Inventory) GetLogs() []string {
	inv.mutex.Lock()
	defer inv.mutex.Unlock()
	return append([]string(nil), inv.logs...) // return a copy
}
