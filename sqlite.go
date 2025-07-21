package inventory

import (
	"database/sql"
	"time"
)

func WithSQLite(db *sql.DB) *Inventory {
	inv := NewInventory("root")
	inv.db = db
	InitSchema(db)
	LoadConversionRules(db)
	LoadInventory(db, inv)
	return inv
}

func InitSchema(db *sql.DB) {
	schema := `
	CREATE TABLE IF NOT EXISTS inventories (
		id TEXT PRIMARY KEY,
		name TEXT,
		parent_id TEXT
	);

	CREATE TABLE IF NOT EXISTS items (
		inventory_id TEXT,
		item_id TEXT,
		name TEXT,
		description TEXT,
		unit TEXT,
		currency TEXT,
		PRIMARY KEY (inventory_id, item_id)
	);

	CREATE TABLE IF NOT EXISTS transactions (
		id TEXT PRIMARY KEY,
		inventory_id TEXT,
		type INTEGER,
		timestamp DATETIME,
		note TEXT
	);

	CREATE TABLE IF NOT EXISTS transaction_items (
		transaction_id TEXT,
		item_id TEXT,
		quantity INTEGER,
		unit TEXT,
		balance INTEGER,
		unit_price REAL,
		currency TEXT,
		PRIMARY KEY (transaction_id, item_id)
	);

	CREATE TABLE IF NOT EXISTS unit_conversions (
		from_unit TEXT,
		to_unit TEXT,
		factor REAL,
		PRIMARY KEY (from_unit, to_unit)
	);

	CREATE TABLE IF NOT EXISTS currency_conversions (
		from_currency TEXT,
		to_currency TEXT,
		rate REAL,
		PRIMARY KEY (from_currency, to_currency)
	);
	`
	_, _ = db.Exec(schema)
}

func PersistInventory(db *sql.DB, inv *Inventory) {
	_, _ = db.Exec("REPLACE INTO inventories (id, name, parent_id) VALUES (?, ?, ?)", inv.ID, inv.ID, inv.ParentID)
	for _, item := range inv.RegisteredItems {
		PersistItem(db, inv.ID, item)
	}
	for _, tx := range inv.Transactions {
		PersistTransaction(db, tx)
	}
	for id, sub := range inv.SubInventories {
		sub.ParentID = inv.ID
		sub.ID = id
		PersistInventory(db, sub)
	}
}

func PersistInventorySince(db *sql.DB, inventoryID string, since time.Time, itemIDs []string) {
	txRows, _ := db.Query("SELECT id, type, timestamp, note FROM transactions WHERE inventory_id = ? AND timestamp >= ?", inventoryID, since)
	defer txRows.Close()
	for txRows.Next() {
		var tx Transaction
		tx.InventoryID = inventoryID
		tx.Items = []TransactionItem{}
		tx.ID = ""
		tx.Note = ""
		var ts time.Time
		_ = txRows.Scan(&tx.ID, &tx.Type, &ts, &tx.Note)
		tx.Timestamp = ts

		itemRows, _ := db.Query("SELECT item_id, quantity, unit, balance, unit_price, currency FROM transaction_items WHERE transaction_id = ?", tx.ID)
		for itemRows.Next() {
			var item TransactionItem
			_ = itemRows.Scan(&item.ItemID, &item.Quantity, &item.Unit, &item.Balance, &item.UnitPrice, &item.Currency)
			tx.Items = append(tx.Items, item)
		}
		itemRows.Close()
		PersistTransaction(db, tx)
	}

	for _, itemID := range itemIDs {
		row := db.QueryRow("SELECT name, description, unit, currency FROM items WHERE inventory_id = ? AND item_id = ?", inventoryID, itemID)
		var item Item
		item.ID = itemID
		_ = row.Scan(&item.Name, &item.Description, &item.Unit, &item.Currency)
		PersistItem(db, inventoryID, item)
	}
}

func PersistItem(db *sql.DB, inventoryID string, item Item) {
	_, _ = db.Exec(`REPLACE INTO items (inventory_id, item_id, name, description, unit, currency)
		VALUES (?, ?, ?, ?, ?, ?)`,
		inventoryID, item.ID, item.Name, item.Description, item.Unit, item.Currency)
}

func DeleteItemFromDB(db *sql.DB, inventoryID, itemID string) {
	_, _ = db.Exec("DELETE FROM items WHERE inventory_id = ? AND item_id = ?", inventoryID, itemID)
}

func PersistTransaction(db *sql.DB, tx Transaction) {
	_, _ = db.Exec(`REPLACE INTO transactions (id, inventory_id, type, timestamp, note)
		VALUES (?, ?, ?, ?, ?)`,
		tx.ID, tx.InventoryID, tx.Type, tx.Timestamp, tx.Note)

	for _, item := range tx.Items {
		_, _ = db.Exec(`REPLACE INTO transaction_items (transaction_id, item_id, quantity, unit, balance, unit_price, currency)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			tx.ID, item.ItemID, item.Quantity, item.Unit, item.Balance, item.UnitPrice, item.Currency)
	}
}

func DeleteTransactionFromDB(db *sql.DB, inventoryID, txID string) {
	_, _ = db.Exec("DELETE FROM transactions WHERE id = ? AND inventory_id = ?", txID, inventoryID)
	_, _ = db.Exec("DELETE FROM transaction_items WHERE transaction_id = ?", txID)
}

func PersistAllInventories(inv *Inventory) {
	if inv.db != nil {
		PersistInventory(inv.db, inv)
	}
}

func LoadInventory(db *sql.DB, inv *Inventory) {
	rows, _ := db.Query("SELECT id FROM inventories WHERE parent_id = ?", inv.ID)
	for rows.Next() {
		var id string
		_ = rows.Scan(&id)
		sub := NewInventory(id)
		sub.db = db
		sub.ParentID = inv.ID
		inv.SubInventories[id] = sub
		LoadInventory(db, sub)
	}
}

func LoadConversionRules(db *sql.DB) {
	rules, _ := db.Query("SELECT from_unit, to_unit, factor FROM unit_conversions")
	for rules.Next() {
		var from, to string
		var factor float64
		_ = rules.Scan(&from, &to, &factor)
		AddUnitConversionRule(
			UnitConversionRule{
				FromUnit: from,
				ToUnit:   to,
				Factor:   factor,
			})
	}
	crules, _ := db.Query("SELECT from_currency, to_currency, rate FROM currency_conversions")
	for crules.Next() {
		var from, to string
		var rate float64
		_ = crules.Scan(&from, &to, &rate)
		AddCurrencyConversionRule(
			CurrencyConversionRule{
				FromCurrency: from,
				ToCurrency:   to,
				Rate:         rate,
			})
	}
}
