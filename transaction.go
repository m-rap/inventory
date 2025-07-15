package inventory

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type TransactionType string

const (
	TransactionAdd    TransactionType = "ADD"
	TransactionRemove TransactionType = "REMOVE"
	TransactionAdjust TransactionType = "ADJUST" // for corrections
)

const (
	UnitPiece = "pcs"
	UnitKg    = "kg"
	UnitBox   = "box"
	// Add more as needed
)

type TransactionItem struct {
	ItemID   string
	Quantity int
	Unit     string // e.g., "pcs", "kg", "box"
	Balance  int    // New: stock level *after* this transaction

	UnitPrice float64 // Price per unit in given currency at time of transaction
	Currency  string  // Currency at the time of transaction
}

type Transaction struct {
	ID        string
	Type      TransactionType
	Items     []TransactionItem // Now supports multiple items
	Timestamp time.Time
	Note      string
}

func (inv *Inventory) AddTransaction(ttype TransactionType, items []TransactionItem, note string) {
	inv.mu.Lock()
	defer inv.mu.Unlock()

	currentBalances := inv.GetLastBalances()

	for i := range items {
		current := currentBalances[items[i].ItemID]
		switch ttype {
		case TransactionAdd:
			items[i].Balance = current + items[i].Quantity
		case TransactionRemove:
			items[i].Balance = current - items[i].Quantity
		case TransactionAdjust:
			items[i].Balance = current + items[i].Quantity
		}
	}

	tx := Transaction{
		ID:        uuid.New().String(),
		Type:      ttype,
		Items:     items,
		Timestamp: time.Now(),
		Note:      note,
	}
	inv.Transactions = append(inv.Transactions, tx)

	var logEntry string
	for _, ti := range tx.Items {
		logEntry = fmt.Sprintf("Transaction: %s %d %s of item %s (new balance: %d %s)\n",
			tx.Type, ti.Quantity, ti.Unit, ti.ItemID, ti.Balance, ti.Unit)
	}
	logEntry += fmt.Sprintf("[%s] %s: %s", tx.Timestamp.Format(time.RFC3339), tx.Type, tx.Note)
	inv.logs = append(inv.logs, logEntry)

	inv.runHooks(tx)
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
	inv.mu.Lock()
	defer inv.mu.Unlock()
	return append([]string(nil), inv.logs...) // return a copy
}
