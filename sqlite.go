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

func (inv *Inventory) persistTransaction(tx Transaction) error {
	_, err := inv.db.Exec(`INSERT INTO transactions (id, inventory_id, type, timestamp, note) VALUES (?, ?, ?, ?, ?)`,
		tx.ID, tx.InventoryID, tx.Type, tx.Timestamp.Format(time.RFC3339), tx.Note)
	if err != nil {
		return err
	}
	for _, item := range tx.Items {
		_, err := inv.db.Exec(`INSERT INTO transaction_items (transaction_id, item_id, quantity, unit, balance, unit_price, currency) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			tx.ID, item.ItemID, item.Quantity, item.Unit, item.Balance, item.UnitPrice, item.Currency)
		if err != nil {
			return err
		}
	}
	return nil
}
