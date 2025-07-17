package inventory

import (
	"database/sql"
	"log"
	"time"
)

func (inv *Inventory) SaveTransactionToDB(db *sql.DB, invID string, tx Transaction) error {
	tx.InventoryID = invID

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS transactions (
			inventory_id TEXT,
			id TEXT,
			type INTEGER,
			timestamp INTEGER,
			note TEXT,
			PRIMARY KEY (inventory_id, id)
		);
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS transaction_items (
			inventory_id TEXT,
			transaction_id TEXT,
			item_id TEXT,
			quantity INTEGER,
			unit TEXT,
			balance INTEGER,
			unit_price REAL,
			currency TEXT
		);
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		INSERT INTO transactions (inventory_id, id, type, timestamp, note)
		VALUES (?, ?, ?, ?, ?)
	`, tx.InventoryID, tx.ID, tx.Type, tx.Timestamp.UnixNano(), tx.Note)
	if err != nil {
		return err
	}

	for _, item := range tx.Items {
		_, err := db.Exec(`
			INSERT INTO transaction_items
			(inventory_id, transaction_id, item_id, quantity, unit, balance, unit_price, currency)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, tx.InventoryID, tx.ID, item.ItemID, item.Quantity, item.Unit, item.Balance, item.UnitPrice, item.Currency)
		if err != nil {
			log.Println("failed insert item:", err)
		}
	}

	return nil
}

func LoadTransactionsFromDB(db *sql.DB, invID string) ([]Transaction, error) {
	rows, err := db.Query(`
		SELECT id, type, timestamp, note FROM transactions
		WHERE inventory_id = ? ORDER BY timestamp ASC
	`, invID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []Transaction
	for rows.Next() {
		var tx Transaction
		var ts int64
		err := rows.Scan(&tx.ID, &tx.Type, &ts, &tx.Note)
		if err != nil {
			return nil, err
		}
		tx.InventoryID = invID
		tx.Timestamp = time.Unix(0, ts)

		itemRows, err := db.Query(`
			SELECT item_id, quantity, unit, balance, unit_price, currency
			FROM transaction_items
			WHERE inventory_id = ? AND transaction_id = ?
		`, invID, tx.ID)
		if err != nil {
			return nil, err
		}

		for itemRows.Next() {
			var item TransactionItem
			err := itemRows.Scan(&item.ItemID, &item.Quantity, &item.Unit, &item.Balance, &item.UnitPrice, &item.Currency)
			if err != nil {
				itemRows.Close()
				return nil, err
			}
			tx.Items = append(tx.Items, item)
		}
		itemRows.Close()

		txs = append(txs, tx)
	}

	return txs, nil
}
