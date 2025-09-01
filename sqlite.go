package inventory

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"
)

const rootInventoryID = "root" // Constant for the root inventory ID

// WithSQLite initializes an Inventory object with the given SQLite database connection.
func WithSQLite(db *sql.DB) (*Inventory, error) {
	inv := NewInventory(rootInventoryID)
	inv.db = db

	// Initialize the database schema
	if err := InitSchema(db); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	// Ensure root inventory is persisted
	_, err := db.Exec("REPLACE INTO inventories (id, name, parent_id) VALUES (?, ?, ?)", inv.ID, inv.ID, inv.ParentID)
	if err != nil {
		return nil, fmt.Errorf("failed to insert root inventory: %w", err)
	}

	// Load conversion rules
	if err := LoadConversionRules(db); err != nil {
		return nil, fmt.Errorf("failed to load conversion rules: %w", err)
	}

	// Load inventory data
	if err := inv.LoadChildren(); err != nil {
		return nil, fmt.Errorf("failed to load inventory: %w", err)
	}

	return inv, nil
}

// InitSchema creates the necessary database tables if they do not already exist.
func InitSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS inventories (
		id TEXT PRIMARY KEY,
		name TEXT,
		parent_id TEXT
	);

	CREATE TABLE IF NOT EXISTS items (
		item_id TEXT PRIMARY KEY,
		name TEXT,
		description TEXT,
		unit TEXT,
		currency TEXT
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
		order_index INTEGER DEFAULT 0,
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

	// Execute the schema creation query
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("failed to execute schema creation query: %w", err)
	}

	log.Println("Database schema initialized successfully")
	return nil
}

func PersistInventory(db *sql.DB, inv *Inventory) {
	_, _ = db.Exec("REPLACE INTO inventories (id, name, parent_id) VALUES (?, ?, ?)", inv.ID, inv.ID, inv.ParentID)
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
		row := db.QueryRow("SELECT name, description, unit, currency FROM items WHERE item_id = ?", itemID)
		var item Item
		item.ID = itemID
		_ = row.Scan(&item.Name, &item.Description, &item.Unit, &item.Currency)
		PersistItem(db, "", item)
	}
}

func PersistItem(db *sql.DB, _ string, item Item) {
	_, _ = db.Exec(`REPLACE INTO items (item_id, name, description, unit, currency)
		VALUES (?, ?, ?, ?, ?)`,
		item.ID, item.Name, item.Description, item.Unit, item.Currency)
}

func DeleteItemFromDB(db *sql.DB, _ string, itemID string) {
	_, _ = db.Exec("DELETE FROM items WHERE item_id = ?", itemID)
}

func PersistTransaction(db *sql.DB, tx Transaction) {
	_, _ = db.Exec(`REPLACE INTO transactions (id, inventory_id, type, timestamp, note)
		VALUES (?, ?, ?, ?, ?)`,
		tx.ID, tx.InventoryID, tx.Type, tx.Timestamp, tx.Note)

	for idx, item := range tx.Items {
		_, _ = db.Exec(`REPLACE INTO transaction_items (transaction_id, item_id, quantity, unit, balance, unit_price, currency, order_index)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			tx.ID, item.ItemID, item.Quantity, item.Unit, item.Balance, item.UnitPrice, item.Currency, idx)
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

func (inv *Inventory) LoadChildren() error {
	// Query to fetch child inventories based on the parent inventory ID
	rows, err := inv.db.Query("SELECT id FROM inventories WHERE parent_id = ?", inv.ID)
	if err != nil {
		return fmt.Errorf("failed to query child inventories for parent_id %s: %w", inv.ID, err)
	}
	defer rows.Close() // Ensure rows are closed after processing

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("failed to scan child inventory: %w", err)
		}

		var sub *Inventory
		if existing, ok := inv.SubInventories[id]; ok {
			// Use the existing child inventory
			sub = existing
			sub.db = inv.db
			sub.ParentID = inv.ID
			sub.Parent = inv
		} else {
			// Create a new Inventory object for the child
			sub = NewInventory(id)
			sub.db = inv.db
			sub.ParentID = inv.ID
			sub.Parent = inv
			inv.SubInventories[id] = sub
		}

		// Recursively load the children of the current child inventory
		if err := sub.LoadChildren(); err != nil {
			return err
		}
	}

	// Check for errors during row iteration
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error occurred during iteration of child inventories: %w", err)
	}

	return nil
}

func LoadConversionRules(db *sql.DB) error {
	// Load unit conversion rules
	rules, err := db.Query("SELECT from_unit, to_unit, factor FROM unit_conversions")
	if err != nil {
		return fmt.Errorf("failed to query unit conversion rules: %w", err)
	}
	defer rules.Close()

	for rules.Next() {
		var from, to string
		var factor float64
		if err := rules.Scan(&from, &to, &factor); err != nil {
			return fmt.Errorf("failed to scan unit conversion rule: %w", err)
		}
		AddUnitConversionRule(
			UnitConversionRule{
				FromUnit: from,
				ToUnit:   to,
				Factor:   factor,
			})
	}

	// Check for errors during unit conversion rule iteration
	if err := rules.Err(); err != nil {
		return fmt.Errorf("error occurred during iteration of unit conversion rules: %w", err)
	}

	// Load currency conversion rules
	crules, err := db.Query("SELECT from_currency, to_currency, rate FROM currency_conversions")
	if err != nil {
		return fmt.Errorf("failed to query currency conversion rules: %w", err)
	}
	defer crules.Close()

	for crules.Next() {
		var from, to string
		var rate float64
		if err := crules.Scan(&from, &to, &rate); err != nil {
			return fmt.Errorf("failed to scan currency conversion rule: %w", err)
		}
		AddCurrencyConversionRule(
			CurrencyConversionRule{
				FromCurrency: from,
				ToCurrency:   to,
				Rate:         rate,
			})
	}

	// Check for errors during currency conversion rule iteration
	if err := crules.Err(); err != nil {
		return fmt.Errorf("error occurred during iteration of currency conversion rules: %w", err)
	}

	return nil
}

// AddChildInventory creates a new child inventory under the current inventory, persists it, and returns the new inventory.
func (inv *Inventory) AddChildInventory(childID string) (*Inventory, error) {
	if inv.db == nil {
		return nil, fmt.Errorf("inventory is not associated with a database")
	}
	if _, exists := inv.SubInventories[childID]; exists {
		return nil, fmt.Errorf("child inventory with ID %s already exists", childID)
	}

	child := NewInventory(childID)
	child.ParentID = inv.ID
	child.db = inv.db

	// Persist the new child inventory
	_, err := inv.db.Exec(
		"REPLACE INTO inventories (id, name, parent_id) VALUES (?, ?, ?)",
		child.ID, child.ID, child.ParentID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to persist child inventory: %w", err)
	}

	inv.SubInventories[childID] = child
	return child, nil
}

// LoadTransactionsForInventories loads all transactions and their items for the given inventory IDs.
// Returns a map from inventory ID to a slice of Transactions.
func LoadTransactionsForInventories(db *sql.DB, inventoryIDs []string) (map[string][]Transaction, error) {
	result := make(map[string][]Transaction)
	if len(inventoryIDs) == 0 {
		return result, nil
	}

	// Build query with IN clause
	query := "SELECT id, inventory_id, type, timestamp, note FROM transactions WHERE inventory_id IN (?" + strings.Repeat(",?", len(inventoryIDs)-1) + ")"
	args := make([]interface{}, len(inventoryIDs))
	for i, id := range inventoryIDs {
		args[i] = id
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}
	defer rows.Close()

	transactions := make(map[string]*Transaction)
	for rows.Next() {
		var tx Transaction
		var ts time.Time
		if err := rows.Scan(&tx.ID, &tx.InventoryID, &tx.Type, &ts, &tx.Note); err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		tx.Timestamp = ts
		tx.Items = []TransactionItem{}
		transactions[tx.ID] = &tx
		result[tx.InventoryID] = append(result[tx.InventoryID], tx)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error occurred during iteration of transactions: %w", err)
	}

	// Load transaction items for all found transactions
	if len(transactions) == 0 {
		return result, nil
	}
	txIDs := make([]interface{}, 0, len(transactions))
	for id := range transactions {
		txIDs = append(txIDs, id)
	}
	itemQuery := "SELECT transaction_id, item_id, quantity, unit, balance, unit_price, currency FROM transaction_items WHERE transaction_id IN (?" + strings.Repeat(",?", len(txIDs)-1) + ")"
	itemRows, err := db.Query(itemQuery, txIDs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query transaction items: %w", err)
	}
	defer itemRows.Close()
	for itemRows.Next() {
		var tid string
		var item TransactionItem
		if err := itemRows.Scan(&tid, &item.ItemID, &item.Quantity, &item.Unit, &item.Balance, &item.UnitPrice, &item.Currency); err != nil {
			return nil, fmt.Errorf("failed to scan transaction item: %w", err)
		}
		if tx, ok := transactions[tid]; ok {
			tx.Items = append(tx.Items, item)
		}
	}
	// Update result with items
	for invID, txs := range result {
		for i := range txs {
			if tx, ok := transactions[txs[i].ID]; ok {
				result[invID][i].Items = tx.Items
			}
		}
	}
	return result, nil
}

// LoadTransactionItemsSorted loads all transaction items for an inventory,
// joined with transactions, sorted by transaction timestamp and order_index.
// If ascending is true, sorts ASC; if false, sorts DESC.
func LoadTransactionItemsSorted(db *sql.DB, inventoryID string, ascending bool, itemIDs ...string) ([]struct {
	TransactionID string
	Timestamp     time.Time
	OrderIndex    int
	Item          TransactionItem
}, error) {
	order := "ASC"
	if !ascending {
		order = "DESC"
	}
	query := `
        SELECT
            ti.transaction_id,
            t.timestamp,
            ti.order_index,
            ti.item_id,
            ti.quantity,
            ti.unit,
            ti.balance,
            ti.unit_price,
            ti.currency
        FROM transaction_items ti
        JOIN transactions t ON ti.transaction_id = t.id
        WHERE t.inventory_id = ? 
    `
	args := []interface{}{inventoryID}
	if len(itemIDs) > 0 {
		placeholders := strings.Repeat(",?", len(itemIDs)-1)
		query += " AND ti.item_id IN (?" + placeholders + ")"
		for _, id := range itemIDs {
			args = append(args, id)
		}
	}
	query += fmt.Sprintf(" ORDER BY t.timestamp %s, ti.order_index %s", order, order)

	rows, err := db.Query(query, inventoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to query sorted transaction items: %w", err)
	}
	defer rows.Close()

	var results []struct {
		TransactionID string
		Timestamp     time.Time
		OrderIndex    int
		Item          TransactionItem
	}
	for rows.Next() {
		var tid string
		var ts time.Time
		var orderIdx int
		var item TransactionItem
		if err := rows.Scan(&tid, &ts, &orderIdx, &item.ItemID, &item.Quantity, &item.Unit, &item.Balance, &item.UnitPrice, &item.Currency); err != nil {
			return nil, fmt.Errorf("failed to scan transaction item: %w", err)
		}
		results = append(results, struct {
			TransactionID string
			Timestamp     time.Time
			OrderIndex    int
			Item          TransactionItem
		}{
			TransactionID: tid,
			Timestamp:     ts,
			OrderIndex:    orderIdx,
			Item:          item,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error occurred during iteration: %w", err)
	}
	return results, nil
}

// LoadLatestTransactionItemsDistinct loads the latest transaction item for each distinct item_id
// for a given inventory, optionally filtered by itemIDs.
// If itemIDs is nil or empty, loads all items.
func LoadLatestTransactionItemsDistinct(db *sql.DB, inventoryID string, itemIDs ...string) ([]TransactionItem, error) {
	baseQuery := `
        SELECT
            ti.transaction_id,
            t.timestamp,
            ti.order_index,
            ti.item_id,
            ti.quantity,
            ti.unit,
            ti.balance,
            ti.unit_price,
            ti.currency
        FROM transaction_items ti
        JOIN transactions t ON ti.transaction_id = t.id
        WHERE t.inventory_id = ?
    `
	args := []interface{}{inventoryID}
	if len(itemIDs) > 0 {
		placeholders := strings.Repeat(",?", len(itemIDs)-1)
		baseQuery += " AND ti.item_id IN (?" + placeholders + ")"
		for _, id := range itemIDs {
			args = append(args, id)
		}
	}
	baseQuery += " ORDER BY t.timestamp DESC, ti.order_index DESC"

	rows, err := db.Query(baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query transaction items: %w", err)
	}
	defer rows.Close()

	var results []TransactionItem
	seen := make(map[string]bool)
	for rows.Next() {
		var tid string
		var ts time.Time
		var orderIdx int
		var item TransactionItem
		if err := rows.Scan(&tid, &ts, &orderIdx, &item.ItemID, &item.Quantity, &item.Unit, &item.Balance, &item.UnitPrice, &item.Currency); err != nil {
			return nil, fmt.Errorf("failed to scan transaction item: %w", err)
		}
		if !seen[item.ItemID] {
			results = append(results, item)
			seen[item.ItemID] = true
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error occurred during iteration: %w", err)
	}
	return results, nil
}
