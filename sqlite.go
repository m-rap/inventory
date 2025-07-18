package inventory

import (
	"database/sql"
	"time"
)

func WithSqlite(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	if err := InitSchema(db); err != nil {
		return nil, err
	}
	return db, nil
}

func InitSchema(db *sql.DB) error {
	schema := []string{
		`CREATE TABLE IF NOT EXISTS items (
			id TEXT PRIMARY KEY,
			inventory_id TEXT,
			name TEXT,
			description TEXT,
			unit TEXT,
			currency TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS transactions (
			id TEXT PRIMARY KEY,
			inventory_id TEXT,
			type INTEGER,
			timestamp TEXT,
			note TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS transaction_items (
			transaction_id TEXT,
			item_id TEXT,
			quantity INTEGER,
			unit TEXT,
			balance INTEGER,
			unit_price REAL,
			currency TEXT
		);`,
	}
	for _, stmt := range schema {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func PersistItem(db *sql.DB, inventoryID string, item Item) {
	_, _ = db.Exec(
		`INSERT OR REPLACE INTO items (id, inventory_id, name, description, unit, currency) VALUES (?, ?, ?, ?, ?, ?)`,
		item.ID, inventoryID, item.Name, item.Description, item.Unit, item.Currency,
	)
}

func PersistTransaction(db *sql.DB, tx Transaction) {
	txStmt, _ := db.Prepare(
		`INSERT INTO transactions (id, inventory_id, type, timestamp, note) VALUES (?, ?, ?, ?, ?)`,
	)
	_, _ = txStmt.Exec(tx.ID, tx.InventoryID, tx.Type, tx.Timestamp.Format(time.RFC3339Nano), tx.Note)

	itemStmt, _ := db.Prepare(
		`INSERT INTO transaction_items (transaction_id, item_id, quantity, unit, balance, unit_price, currency) VALUES (?, ?, ?, ?, ?, ?, ?)`,
	)
	for _, item := range tx.Items {
		_, _ = itemStmt.Exec(tx.ID, item.ItemID, item.Quantity, item.Unit, item.Balance, item.UnitPrice, item.Currency)
	}
}

func PersistInventorySince(db *sql.DB, inventoryID string, since time.Time, itemIDs []string) {
	txs := LoadTransactionsForItems(db, inventoryID, itemIDs)
	for _, tx := range txs {
		if !tx.Timestamp.Before(since) {
			PersistTransaction(db, tx)
		}
	}
}

func PersistInventory(db *sql.DB, inv *Inventory) {
	for _, item := range inv.RegisteredItems {
		PersistItem(db, inv.ID, item)
	}
	for _, tx := range inv.Transactions {
		PersistTransaction(db, tx)
	}
}

func PersistAllInventories(inventories map[string]*Inventory) {
	for _, inv := range inventories {
		PersistInventory(inv.db, inv)
	}
}

func LoadTransactionsForItems(db *sql.DB, inventoryID string, itemIDs []string) []Transaction {
	if len(itemIDs) == 0 {
		return nil
	}

	query := `
		SELECT DISTINCT t.id, t.inventory_id, t.type, t.timestamp, t.note
		FROM transactions t
		JOIN transaction_items ti ON t.id = ti.transaction_id
		WHERE t.inventory_id = ? AND ti.item_id IN (` + placeholders(len(itemIDs)) + `)
		ORDER BY t.timestamp`

	args := []interface{}{inventoryID}
	for _, id := range itemIDs {
		args = append(args, id)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var transactions []Transaction
	for rows.Next() {
		var tx Transaction
		var ts string
		if err := rows.Scan(&tx.ID, &tx.InventoryID, &tx.Type, &ts, &tx.Note); err != nil {
			continue
		}
		tx.Timestamp, _ = time.Parse(time.RFC3339Nano, ts)
		tx.Items = loadTransactionItems(db, tx.ID)
		transactions = append(transactions, tx)
	}
	return transactions
}

func loadTransactionItems(db *sql.DB, txID string) []TransactionItem {
	rows, err := db.Query(`
		SELECT item_id, quantity, unit, balance, unit_price, currency
		FROM transaction_items WHERE transaction_id = ?`, txID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var items []TransactionItem
	for rows.Next() {
		var item TransactionItem
		_ = rows.Scan(&item.ItemID, &item.Quantity, &item.Unit, &item.Balance, &item.UnitPrice, &item.Currency)
		items = append(items, item)
	}
	return items
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	s := "?"
	for i := 1; i < n; i++ {
		s += ",?"
	}
	return s
}
