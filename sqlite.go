package inventory

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

func InitSchema(db *sql.DB) error {
	schema := `
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS accounts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT UNIQUE NOT NULL,
    name TEXT,
    parent_uuid TEXT
);

CREATE TABLE IF NOT EXISTS items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT UNIQUE NOT NULL,
    name TEXT,
    description TEXT,
    unit TEXT
);

CREATE TABLE IF NOT EXISTS transactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT UNIQUE NOT NULL,
    description TEXT,
    datetime_ms INTEGER NOT NULL,
    year INTEGER NOT NULL,
    month INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_transactions_year_month
    ON transactions(year, month);

CREATE TABLE IF NOT EXISTS transaction_lines (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    transaction_uuid TEXT NOT NULL,
    account_uuid TEXT NOT NULL,
    item_uuid TEXT,
    quantity REAL,
    unit TEXT,
    price REAL,
    currency TEXT,
    note TEXT,
    FOREIGN KEY (transaction_uuid) REFERENCES transactions(uuid) ON DELETE CASCADE,
    FOREIGN KEY (item_uuid) REFERENCES items(uuid) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS balance_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_uuid TEXT NOT NULL,
    item_uuid TEXT,
    unit TEXT,
    quantity REAL,
    avg_cost REAL,
    value REAL,
    datetime_ms INTEGER NOT NULL,
    year INTEGER NOT NULL,
    month INTEGER NOT NULL,
    price REAL,
    currency TEXT,
    market_value REAL,
    description TEXT,
    FOREIGN KEY (item_uuid) REFERENCES items(uuid) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_balance_history_year_month
    ON balance_history(year, month);

CREATE TABLE IF NOT EXISTS unit_conversions (
    from_unit TEXT NOT NULL,
    to_unit TEXT NOT NULL,
    factor REAL NOT NULL,
    datetime_ms INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS currency_conversions (
    from_currency TEXT NOT NULL,
    to_currency TEXT NOT NULL,
    rate REAL NOT NULL,
    datetime_ms INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS market_prices (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    item_uuid TEXT NOT NULL,
    datetime_ms INTEGER NOT NULL,
    price REAL,
    unit TEXT,
    currency TEXT,
    FOREIGN KEY (item_uuid) REFERENCES items(uuid) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_market_prices_item_date
    ON market_prices(item_uuid, datetime_ms);
`
	_, err := db.Exec(schema)
	return err
}

func AddAccount(db *sql.DB, name, parentID string) (string, error) {
	id := uuid.NewString()
	_, err := db.Exec("INSERT INTO accounts(id,name,parent_uuid) VALUES(?,?,?)", id, name, sql.NullString{String: parentID, Valid: parentID != ""})
	return id, err
}

func AddItem(db *sql.DB, name, unit, description string) (string, error) {
	id := uuid.NewString()
	_, err := db.Exec("INSERT INTO items(id,name,unit,description) VALUES(?,?,?,?)", id, name, unit, description)
	return id, err
}

func ApplyTransaction(db *sql.DB, desc string, date time.Time, lines []TransactionLine) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	tid := uuid.NewString()

	_, err = tx.Exec("INSERT INTO transactions(id,datetime_ms,description) VALUES(?,?,?)", tid, date.UnixMilli(), desc)
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, l := range lines {
		lid := uuid.NewString()
		_, err := tx.Exec(
			"INSERT INTO transaction_lines(id,transaction_id,account_id,item_id,quantity,unit,price,currency,note) VALUES(?,?,?,?,?,?,?,?,?)",
			lid, tid, l.AccountUUID, sql.NullString{String: l.ItemUUID, Valid: l.ItemUUID != ""}, l.Quantity, l.Unit, l.Price, l.Currency, l.Note)
		if err != nil {
			tx.Rollback()
			return err
		}

		var prevQty, prevTotal float64
		err = tx.QueryRow(`
			SELECT h.quantity, h.total_cost
			FROM balance_history h
			JOIN transactions t ON h.transaction_id = t.id
			WHERE h.item_id=? AND h.account_id=? AND t.datetime_ms <= ?
			ORDER BY t.datetime_ms DESC, h.id DESC
			LIMIT 1`,
			l.ItemUUID, l.AccountUUID, date.UnixMilli()).Scan(&prevQty, &prevTotal)

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
			hid, l.ItemUUID, l.AccountUUID, tid, newQty, newTotal, avgCost)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	tx.Commit()
	return nil
}

func SetMarketPrice(db *sql.DB, itemID string, price float64, currency string, unit string) error {
	_, err := db.Exec(`
		INSERT INTO market_prices(id,item_id,datetime_ms,price,currency,unit)
		VALUES(?,?,?,?,?,?)
	`, uuid.NewString(), itemID, time.Now().UnixMilli(), price, currency, unit)
	return err
}

func BuildAccountTree(db *sql.DB) (map[string][]string, error) {
	rows, err := db.Query(`SELECT id,name,parent_uuid FROM accounts`)
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

func FetchLeafBalances(db *sql.DB) ([]BalanceHistory, error) {
	// rows, err := db.Query(`
	// 	SELECT a.id, COALESCE(i.name,''), COALESCE(i.unit,''),
	// 	       h.quantity, h.avg_cost, h.quantity*h.avg_cost, t.date
	// 	FROM balance_history h
	// 	JOIN (
	// 	    SELECT item_id, account_id, MAX(t.date) as last_date
	// 	    FROM balance_history bh
	// 	    JOIN transactions t ON bh.transaction_id = t.id
	// 	    GROUP BY item_id, account_id
	// 	) last ON h.item_id=last.item_id AND h.account_id=last.account_id
	// 	JOIN transactions t ON h.transaction_id=t.id AND t.date=last.last_date
	// 	LEFT JOIN items i ON h.item_id=i.id
	// 	JOIN accounts a ON h.account_id=a.id
	// `)
	// rows, err := db.Query(`
	// 	select * from (
	// 		select a.id as account_id, a.parent_uuid as parent_uuid, a.name as account,p.name as parent,t.date,t.description,i.id as item_id,i.name as item,l1.quantity as qty,b.quantity as bal,b.total_cost,b.avg_cost,i.unit from balance_history b
	// 		join accounts a on b.account_id = a.id
	// 		join transactions t on b.transaction_id = t.id
	// 		join transaction_lines l1 on l1.transaction_id=b.transaction_id and l1.account_id=b.account_id
	// 		left join items i on b.item_id = i.id
	// 		left join accounts p on a.parent_uuid = p.id
	// 		order by date desc
	// 	)
	// 	group by account_id,item_id
	// `)
	rows, err := db.Query(`
		select * from (
			select
				a.id as account_id,
				i.name as item,
				i.unit,
				b.quantity,
				b.avg_cost,
				b.quantity*b.avg_cost,
				t.datetime_ms,
				t.description
			from balance_history b
			join accounts a on b.account_id = a.id
			join transactions t on b.transaction_id = t.id
			join transaction_lines l1 on l1.transaction_id=b.transaction_id and l1.account_id=b.account_id
			left join items i on b.item_id = i.id
			left join accounts p on a.parent_uuid = p.id
			order by datetime_ms desc
		) 
		group by account_id,item
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var balances []BalanceHistory
	for rows.Next() {
		var item, unit sql.NullString
		var accID, desc string
		var date int64
		var qty, avgCost, value float64
		if err := rows.Scan(&accID, &item, &unit, &qty, &avgCost, &value, &date, &desc); err != nil {
			return nil, err
		}
		balances = append(balances, BalanceHistory{
			AccountUUID: accID,
			ItemUUID:    item.String,
			Unit:        unit.String,
			Quantity:    qty,
			AvgCost:     avgCost,
			Value:       value,
			DatetimeMs:  date,
			Description: desc,
		})
	}
	return balances, nil
}

// --- Fetch & Rollup Market Balances ---

func FetchLeafMarketBalances(db *sql.DB) ([]BalanceHistory, error) {
	rows, err := db.Query(`
		SELECT a.id, COALESCE(i.name,''), COALESCE(i.unit,''),
		       h.quantity,
		       COALESCE(mp.price,0), COALESCE(mp.currency,''),
		       (h.quantity*COALESCE(mp.price,0)) as market_value
		FROM balance_history h
		JOIN (
		    SELECT item_id, account_id, MAX(t.datetime_ms) as last_date
		    FROM balance_history bh
		    JOIN transactions t ON bh.transaction_id = t.id
		    GROUP BY item_id, account_id
		) last ON h.item_id=last.item_id AND h.account_id=last.account_id
		JOIN transactions t ON h.transaction_id=t.id AND t.datetime_ms=last.last_date
		LEFT JOIN items i ON h.item_id=i.id
		JOIN accounts a ON h.account_id=a.id
		LEFT JOIN (
		    SELECT m1.item_id, m1.price, m1.currency
		    FROM market_prices m1
		    JOIN (
		        SELECT item_id, MAX(datetime_ms) as max_date
		        FROM market_prices
		        GROUP BY item_id
		    ) m2 ON m1.item_id=m2.item_id AND m1.datetime_ms=m2.max_date
		) mp ON i.id=mp.item_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var balances []BalanceHistory
	for rows.Next() {
		var accID, item, unit, currency string
		var qty, price, marketValue float64
		if err := rows.Scan(&accID, &item, &unit, &qty, &price, &currency, &marketValue); err != nil {
			return nil, err
		}
		balances = append(balances, BalanceHistory{
			AccountUUID: accID,
			ItemUUID:    item,
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
