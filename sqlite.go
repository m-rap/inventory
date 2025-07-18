package inventory

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func (inv *Inventory) WithSQLite(path string) error {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return err
	}
	inv.db = db
	return inv.initSchema()
}

func (inv *Inventory) initSchema() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS inventories (
			id TEXT PRIMARY KEY
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
	for _, q := range queries {
		if _, err := inv.db.Exec(q); err != nil {
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
