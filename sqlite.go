package inventory

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

// InitSchema creates the necessary database tables if they do not already exist.
func InitSchema(db *sql.DB) error {
	schema := `
	-- Table to store inventories or financial accounts
	CREATE TABLE IF NOT EXISTS accounts (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		parent_id TEXT REFERENCES accounts(id)
	);

	CREATE TABLE IF NOT EXISTS items(
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		unit TEXT NOT NULL,
		description TEXT
	);

	CREATE TABLE IF NOT EXISTS transactions (
		id TEXT PRIMARY KEY,
		date DATETIME NOT NULL,
		description TEXT
	);

	CREATE TABLE IF NOT EXISTS transaction_lines (
		id TEXT PRIMARY KEY,
		transaction_id TEXT NOT NULL REFERENCES transactions(id),
		account_id TEXT NOT NULL REFERENCES accounts(id),
		item_id TEXT NOT NULL REFERENCES items(item_id),
		quantity REAL NOT NULL,
		price REAL, -- price per unit
		unit TEXT,
		currency TEXT,
		note TEXT
	);

	CREATE TABLE balance_history (
		id TEXT PRIMARY KEY,
		item_id TEXT NOT NULL REFERENCES items(id),
		account_id TEXT NOT NULL REFERENCES accounts(id),
		transaction_id TEXT NOT NULL REFERENCES transactions(id),
		quantity REAL NOT NULL,
		total_cost REAL NOT NULL,
		avg_cost REAL NOT NULL
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

	CREATE TABLE market_prices (
		id TEXT PRIMARY KEY,
		item_id TEXT NOT NULL REFERENCES items(id),
		date DATETIME NOT NULL,
		price REAL NOT NULL,   -- market price per unit
		unit TEXT NOT NULL,    -- e.g. kg
		currency TEXT NOT NULL -- e.g. USD
	);
	`

	// Execute the schema creation query
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("failed to execute schema creation query: %w", err)
	}

	log.Println("Database schema initialized successfully")
	return nil
}

func AddAccount(db *sql.DB, name, parentID string) (string, error) {
	id := uuid.NewString()
	_, err := db.Exec("INSERT INTO accounts(id,name,parent_id) VALUES(?,?,?)", id, name, sql.NullString{String: parentID, Valid: parentID != ""})
	return id, err
}

func AddItem(db *sql.DB, name, unit, description string) (string, error) {
	id := uuid.NewString()
	_, err := db.Exec("INSERT INTO items(id,name,unit,description) VALUES(?,?,?,?)", id, name, unit, description)
	return id, err
}

func ApplyTransaction(db *sql.DB, desc string, lines []Line) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	now := time.Now()
	tid := uuid.NewString()

	_, err = tx.Exec("INSERT INTO transactions(id,date,description) VALUES(?,?,?)", tid, now, desc)
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, l := range lines {
		lid := uuid.NewString()
		_, err := tx.Exec(
			"INSERT INTO transaction_lines(id,transaction_id,account_id,item_id,quantity,unit,price,currency,note) VALUES(?,?,?,?,?,?,?,?)",
			lid, tid, l.AccountID, sql.NullString{String: l.ItemID, Valid: l.ItemID != ""}, l.Quantity, l.Unit, l.Price, l.Currency, l.Note)
		if err != nil {
			tx.Rollback()
			return err
		}

		var prevQty, prevTotal float64
		err = tx.QueryRow(`
			SELECT h.quantity, h.total_cost
			FROM balance_history h
			JOIN transactions t ON h.transaction_id = t.id
			WHERE h.item_id=? AND h.account_id=? AND t.date <= ?
			ORDER BY t.date DESC, h.id DESC
			LIMIT 1`,
			l.ItemID, l.AccountID, now).Scan(&prevQty, &prevTotal)

		if err == sql.ErrNoRows {
			prevQty, prevTotal = 0, 0
		} else if err != nil {
			tx.Rollback()
			return err
		}

		newQty := prevQty + l.Quantity
		newTotal := prevTotal + l.Quantity*l.Price
		avgCost := 0.0
		if newQty != 0 {
			avgCost = newTotal / newQty
		}

		hid := uuid.NewString()
		_, err = tx.Exec(`INSERT INTO balance_history(id,item_id,account_id,transaction_id,quantity,total_cost,avg_cost)
		                  VALUES(?,?,?,?,?,?,?)`,
			hid, l.ItemID, l.AccountID, tid, newQty, newTotal, avgCost)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	tx.Commit()
	return nil
}

func SetMarketPrice(db *sql.DB, itemID string, price float64, currency string) error {
	_, err := db.Exec(`
		INSERT INTO market_prices(id,item_id,date,price,currency)
		VALUES(?,?,?,?,?)
	`, uuid.NewString(), itemID, time.Now(), price, currency)
	return err
}

func BuildAccountTree(db *sql.DB) (map[string][]string, error) {
	rows, err := db.Query(`SELECT id,name,parent_id FROM accounts`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type node struct {
		id, name, parent string
	}
	accMap := map[string]node{}
	for rows.Next() {
		var id, name string
		var parent sql.NullString
		rows.Scan(&id, &name, &parent)
		parentID := ""
		if parent.Valid {
			parentID = parent.String
		}
		accMap[id] = node{id: id, name: name, parent: parentID}
	}

	paths := map[string][]string{}
	for id := range accMap {
		cur := id
		var path []string
		for cur != "" {
			n := accMap[cur]
			path = append([]string{n.name}, path...)
			cur = n.parent
		}
		paths[id] = path
	}
	return paths, nil
}

// --- Fetch & Rollup Historical Balances ---

func FetchLeafBalances(db *sql.DB) ([]Balance, error) {
	rows, err := db.Query(`
		SELECT a.id, COALESCE(i.name,''), COALESCE(i.unit,''),
		       h.quantity, h.avg_cost, h.quantity*h.avg_cost, t.date
		FROM balance_history h
		JOIN (
		    SELECT item_id, account_id, MAX(t.date) as last_date
		    FROM balance_history bh
		    JOIN transactions t ON bh.transaction_id = t.id
		    GROUP BY item_id, account_id
		) last ON h.item_id=last.item_id AND h.account_id=last.account_id
		JOIN transactions t ON h.transaction_id=t.id AND t.date=last.last_date
		LEFT JOIN items i ON h.item_id=i.id
		JOIN accounts a ON h.account_id=a.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var balances []Balance
	for rows.Next() {
		var accID, item, unit string
		var date time.Time
		var qty, avgCost, value float64
		if err := rows.Scan(&accID, &item, &unit, &qty, &avgCost, &value, &date); err != nil {
			return nil, err
		}
		balances = append(balances, Balance{
			AccountID: accID,
			Item:      item,
			Unit:      unit,
			Quantity:  qty,
			AvgCost:   avgCost,
			Value:     value,
			Date:      date,
		})
	}
	return balances, nil
}

// --- Fetch & Rollup Market Balances ---

func FetchLeafMarketBalances(db *sql.DB) ([]Balance, error) {
	rows, err := db.Query(`
		SELECT a.id, COALESCE(i.name,''), COALESCE(i.unit,''),
		       h.quantity,
		       COALESCE(mp.price,0), COALESCE(mp.currency,''),
		       (h.quantity*COALESCE(mp.price,0)) as market_value
		FROM balance_history h
		JOIN (
		    SELECT item_id, account_id, MAX(t.date) as last_date
		    FROM balance_history bh
		    JOIN transactions t ON bh.transaction_id = t.id
		    GROUP BY item_id, account_id
		) last ON h.item_id=last.item_id AND h.account_id=last.account_id
		JOIN transactions t ON h.transaction_id=t.id AND t.date=last.last_date
		LEFT JOIN items i ON h.item_id=i.id
		JOIN accounts a ON h.account_id=a.id
		LEFT JOIN (
		    SELECT m1.item_id, m1.price, m1.currency
		    FROM market_prices m1
		    JOIN (
		        SELECT item_id, MAX(date) as max_date
		        FROM market_prices
		        GROUP BY item_id
		    ) m2 ON m1.item_id=m2.item_id AND m1.date=m2.max_date
		) mp ON i.id=mp.item_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var balances []Balance
	for rows.Next() {
		var accID, item, unit, currency string
		var qty, price, marketValue float64
		if err := rows.Scan(&accID, &item, &unit, &qty, &price, &currency, &marketValue); err != nil {
			return nil, err
		}
		balances = append(balances, Balance{
			AccountID:   accID,
			Item:        item,
			Unit:        unit,
			Quantity:    qty,
			Price:       price,
			Currency:    currency,
			MarketValue: marketValue,
		})
	}
	return balances, nil
}

func AddUnitConversionRule(db *sql.DB, rule UnitConversionRule) error {
	_, err := db.Exec("INSERT INTO unit_conversions(from_unit,to_unit,factor) VALUES(?,?,?)", rule.FromUnit, rule.ToUnit, rule.Factor)
	return err
}

func AddCurrencyConversionRule(db *sql.DB, rule CurrencyConversionRule) error {
	_, err := db.Exec("INSERT INTO currency_conversions(from_currency,to_currency,rate) VALUES(?,?,?)", rule.FromCurrency, rule.ToCurrency, rule.Rate)
	return err
}

func LoadConversionRule(db *sql.DB, fromUnit, toUnit string) (UnitConversionRule, error) {
	var rule UnitConversionRule
	err := db.QueryRow("SELECT from_unit,to_unit,factor FROM unit_conversions WHERE from_unit=? AND to_unit=?", fromUnit, toUnit).
		Scan(&rule.FromUnit, &rule.ToUnit, &rule.Factor)
	if err != nil {
		return UnitConversionRule{}, err
	}
	return rule, nil
}

func LoadCurrencyConversionRule(db *sql.DB, fromCurrency, toCurrency string) (CurrencyConversionRule, error) {
	var rule CurrencyConversionRule
	err := db.QueryRow("SELECT from_currency,to_currency,rate FROM currency_conversions WHERE from_currency=? AND to_currency=?", fromCurrency, toCurrency).
		Scan(&rule.FromCurrency, &rule.ToCurrency, &rule.Rate)
	if err != nil {
		return CurrencyConversionRule{}, err
	}
	return rule, nil
}
