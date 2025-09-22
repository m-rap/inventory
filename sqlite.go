package inventory

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
)

var (
	AssetAcc, EquityAcc, LiabilityAcc, IncomeAcc, ExpenseAcc *Account = nil, nil, nil, nil, nil
)

func InitSchema(db *sql.DB) error {
	schema := `
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS accounts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid BLOB UNIQUE NOT NULL,
    name TEXT,
    parent_id INTEGER
);

CREATE TABLE IF NOT EXISTS items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid BLOB UNIQUE NOT NULL,
    name TEXT,
    description TEXT,
    unit TEXT
);

CREATE TABLE IF NOT EXISTS transactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid BLOB UNIQUE NOT NULL,
    description TEXT,
    datetime_ms INTEGER NOT NULL,
    year INTEGER NOT NULL,
    month INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_transactions_year_month
    ON transactions(year, month);

CREATE TABLE IF NOT EXISTS transaction_lines (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
	uuid BLOB UNIQUE NOT NULL,
    transaction_id INTEGER NOT NULL,
    account_id INTEGER NOT NULL,
    item_id INTEGER,
    quantity REAL,
    unit TEXT,
    price REAL,
    currency TEXT,
    note TEXT,
    FOREIGN KEY (transaction_id) REFERENCES transactions(id) ON DELETE CASCADE,
    FOREIGN KEY (item_id) REFERENCES items(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS balance_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
	uuid BLOB UNIQUE NOT NULL,
    account_id INTEGER NOT NULL,
	transaction_id INTEGER NOT NULL,
    item_id INTEGER,
    unit TEXT,
    quantity REAL,
	total_cost REAL,
    avg_cost REAL,
    value REAL,
    price REAL,
    currency TEXT,
    market_value REAL,
    description TEXT
);

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
    item_id BLOB NOT NULL,
    datetime_ms INTEGER NOT NULL,
    price REAL,
    unit TEXT,
    currency TEXT,
    FOREIGN KEY (item_id) REFERENCES items(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_market_prices_item_date
    ON market_prices(item_id, datetime_ms);
`
	_, err := db.Exec(schema)
	if err != nil {
		return err
	}

	_, err = AddAccount(db, "asset", nil)
	if err != nil {
		return err
	}
	_, err = AddAccount(db, "equity", nil)
	if err != nil {
		return err
	}
	_, err = AddAccount(db, "liability", nil)
	if err != nil {
		return err
	}
	_, err = AddAccount(db, "income", nil)
	if err != nil {
		return err
	}
	_, err = AddAccount(db, "expense", nil)
	if err != nil {
		return err
	}
	_, _, err = BuildAccountTree(db)

	return err
}

func GetAccountFromUUID(db *sql.DB, accUUID []byte) (*Account, error) {
	rows, err := db.Query(`SELECT * FROM accounts WHERE uuid=?`, accUUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, sql.ErrNoRows
	}
	var id, parentId int
	var name string
	err = rows.Scan(&id, &accUUID, &name, &parentId)
	if err != nil {
		return nil, err
	}

	bUUID, err := uuid.FromBytes(accUUID)
	if err != nil {
		return nil, err
	}

	return &Account{
		ID: id, UUID: bUUID, Name: name, Parent: nil,
	}, nil
}

func GetAccountFromID(db *sql.DB, accID int) (*Account, error) {
	rows, err := db.Query(`SELECT * FROM accounts WHERE id=?`, accID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, sql.ErrNoRows
	}
	var parentId int
	var accUUID []byte
	var name string
	err = rows.Scan(&accID, &accUUID, &name, &parentId)
	if err != nil {
		return nil, err
	}

	bUUID, err := uuid.FromBytes(accUUID)
	if err != nil {
		return nil, err
	}

	return &Account{
		ID: accID, UUID: bUUID, Name: name, Parent: nil,
	}, nil
}

// func getLastIDOrErr(res sql.Result, err error) (int, error) {
// 	if err != nil {
// 		return -1, err
// 	}

// 	id, err := res.LastInsertId()
// 	if err != nil {
// 		return -1, err
// 	}

// 	return int(id), nil
// }

func AddAccount(db *sql.DB, name string, parentUUID []byte) ([]byte, error) {
	accUUID, err := uuid.NewV6()
	if err != nil {
		return accUUID[:], err
	}

	var parentID int
	acc, err := GetAccountFromUUID(db, parentUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			parentID = -1
		} else {
			return accUUID[:], err
		}
	}
	if acc != nil {
		parentID = acc.ID
	} else {
		parentID = -1
	}

	_, err = db.Exec("INSERT INTO accounts(uuid,name,parent_id) VALUES(?,?,?)", accUUID[:], name, parentID)

	return accUUID[:], err
}

func GetItemFromUUID(db *sql.DB, itUUID []byte) (*Item, error) {
	rows, err := db.Query(`SELECT * FROM items WHERE uuid=?`, itUUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, sql.ErrNoRows
	}

	var id int
	var name, description, unit string

	err = rows.Scan(&id, &itUUID, &name, &description, &unit)
	if err != nil {
		return nil, err
	}

	bUUID, err := uuid.FromBytes(itUUID)
	if err != nil {
		return nil, err
	}

	return &Item{
		ID: id, UUID: bUUID, Name: name, Description: description, Unit: unit,
	}, nil
}

func AddItem(db *sql.DB, name, unit, description string) ([]byte, error) {
	itUUID, err := uuid.NewV6()
	if err != nil {
		return itUUID[:], err
	}

	_, err = db.Exec("INSERT INTO items(uuid,name,unit,description) VALUES(?,?,?,?)", itUUID[:], name, unit, description)
	return itUUID[:], err
}

func CreateInventoryTrLine(acc *Account, item *Item, qty float64, unit string, price float64, currency string) TransactionLine {
	return TransactionLine{
		Account:  acc,
		Item:     item,
		Quantity: qty,
		Unit:     unit,
		Price:    price,
		Currency: currency,
	}
}

func CreateFinancialTrLine(acc *Account, debet float64, kredit float64, currency string) TransactionLine {
	amount := debet - kredit
	return TransactionLine{
		Account:  acc,
		Item:     nil,
		Quantity: amount,
		Unit:     "",
		Currency: currency,
	}
}

func ApplyTransaction(db *sql.DB, desc string, date time.Time, lines []TransactionLine) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	trUUID, err := uuid.NewV6()
	if err != nil {
		return err
	}

	var res sql.Result

	// fmt.Println("inserting transaction")
	res, err = tx.Exec("INSERT INTO transactions(uuid,datetime_ms,year,month,description) VALUES(?,?,?,?,?)", trUUID, date.UnixMilli(), date.Year(), int(date.Month()), desc)

	if err != nil {
		tx.Rollback()
		return err
	}

	trID, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, l := range lines {
		// fmt.Println("inserting line")
		lineUUID, err := uuid.NewV6()
		if err != nil {
			tx.Rollback()
			return err
		}
		var itemID int
		if l.Item != nil {
			itemID = l.Item.ID
		} else {
			itemID = -1
		}
		// fmt.Println(lineUUID, trID, l.Account.ID, sql.NullInt64{Int64: int64(itemID), Valid: itemID != -1}, l.Quantity, l.Unit, l.Price, l.Currency, l.Note)
		_, err = tx.Exec(
			"INSERT INTO transaction_lines(uuid,transaction_id,account_id,item_id,quantity,unit,price,currency,note) VALUES(?,?,?,?,?,?,?,?,?)",
			lineUUID, trID, l.Account.ID, sql.NullInt64{Int64: int64(itemID), Valid: itemID != -1}, l.Quantity, l.Unit, l.Price, l.Currency, l.Note)
		if err != nil {
			tx.Rollback()
			return err
		}

		// fmt.Println("finding prev qty and prev total")
		var prevQty, prevTotal float64
		err = tx.QueryRow(`
			SELECT h.quantity, h.total_cost
			FROM balance_history h
			JOIN transactions t ON h.transaction_id = t.id
			WHERE h.item_id=? AND h.account_id=? AND t.datetime_ms <= ?
			ORDER BY t.datetime_ms DESC
			LIMIT 1`,
			itemID, l.Account.ID, date.UnixMilli()).Scan(&prevQty, &prevTotal)

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

		// fmt.Printf("new qty %v new total %v new avg cost %v. inserting balance history.\n", newQty, newTotal, avgCost)
		histUUID, err := uuid.NewV6()
		if err != nil {
			tx.Rollback()
			return err
		}

		_, err = tx.Exec(`INSERT INTO balance_history(uuid,item_id,account_id,transaction_id,quantity,total_cost,avg_cost)
		                  VALUES(?,?,?,?,?,?,?)`,
			histUUID, itemID, l.Account.ID, trID, newQty, newTotal, avgCost)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	// fmt.Println("committing")
	tx.Commit()
	return nil
}

func UpdateMarketPrice(db *sql.DB, itemUUID []byte, price float64, currency string, unit string) error {
	item, err := GetItemFromUUID(db, itemUUID)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		INSERT INTO market_prices(item_id,datetime_ms,price,currency,unit)
		VALUES(?,?,?,?,?)
	`, item.ID, time.Now().UnixMilli(), price, currency, unit)

	return err
}

func BuildAccountTree(db *sql.DB) (map[int][]string, map[int]*Account, error) {
	// fmt.Println("querying accounts")
	rows, err := db.Query(`SELECT id,uuid,name,parent_id FROM accounts`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	// type node struct {
	// 	uuid, name, parent string
	// }

	accMap := make(map[int]*Account)
	for rows.Next() {
		var id, parent int
		var accUUID []byte
		var name string
		rows.Scan(&id, &accUUID, &name, &parent)
		// fmt.Println(id, accUUID, name, parent)

		var acc *Account
		switch name {
		case "asset":
			if AssetAcc == nil {
				AssetAcc = new(Account)
			}
			acc = AssetAcc
		case "equity":
			if EquityAcc == nil {
				EquityAcc = new(Account)
			}
			acc = EquityAcc
		case "liability":
			if LiabilityAcc == nil {
				LiabilityAcc = new(Account)
			}
			acc = LiabilityAcc
		case "income":
			if IncomeAcc == nil {
				IncomeAcc = new(Account)
			}
			acc = IncomeAcc
		case "expense":
			if ExpenseAcc == nil {
				ExpenseAcc = new(Account)
			}
			acc = ExpenseAcc
		default:
			acc = new(Account)
		}

		acc.ID = id
		acc.UUID, err = uuid.FromBytes(accUUID)
		if err != nil {
			return nil, nil, err
		}

		acc.Name = name

		parentAcc, ok := accMap[parent]
		if ok {
			acc.Parent = parentAcc
		}
		accMap[id] = acc
	}

	paths := map[int][]string{}

	// fmt.Println(accMap)

	for id := range accMap {
		// fmt.Printf("id %v\n", id)
		cur := id
		var path []string
		for cur != -1 {
			// fmt.Printf("cur %v\n", cur)
			n, ok := accMap[cur]
			if !ok {
				return nil, nil, os.ErrNotExist
			}
			path = append([]string{n.Name}, path...)
			if n.Parent == nil {
				cur = -1
				continue
			}
			// fmt.Printf("parent %v\n", n.Parent)
			cur = n.Parent.ID
		}
		paths[id] = path
	}
	return paths, accMap, nil
}

// --- Fetch & Rollup Historical Balances ---

func FetchLeafBalances(db *sql.DB, accountMap map[int]*Account) ([]BalanceHistory, error) {
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
				l1.id as transaction_line_id,
				i.id as item_id,
				i.name as item_name,
				t.id as transaction_id,
				t.description,
				l1.price as transaction_price,
				m.price as market_price,
				b.quantity,
				i.unit,
				b.avg_cost,
				b.quantity*b.avg_cost,
				b.quantity*m.price,
				t.datetime_ms
			from balance_history b
			join accounts a on b.account_id = a.id
			join transactions t on b.transaction_id = t.id
			join transaction_lines l1 on l1.transaction_id=b.transaction_id and l1.account_id=b.account_id
			left join items i on b.item_id = i.id
			left join accounts p on a.parent_id = p.id
			left join market_prices m on b.item_id = m.item_id
			order by t.datetime_ms desc
		) 
		group by account_id,item_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var balances []BalanceHistory
	for rows.Next() {
		var itemName, unit sql.NullString
		var desc string
		var accID, lineID, trID int
		var itemID sql.NullInt64
		var date int64
		var trPrice, qty, avgCost, value float64
		var marketPrice, marketValue sql.NullFloat64
		if err := rows.Scan(&accID, &lineID, &itemID, &itemName, &trID, &desc, &trPrice, &marketPrice, &qty, &unit, &avgCost, &value, &marketValue, &date); err != nil {
			return nil, err
		}
		acc, ok := accountMap[accID]
		if !ok {
			acc = nil
			fmt.Printf("uuid %v not found on map\n", accID)
		}
		h := BalanceHistory{
			TransactionLine: &TransactionLine{
				Account: acc,
				Transaction: &Transaction{
					ID:          trID,
					Description: desc,
					DatetimeMs:  date,
				},
			},
			Unit:             unit.String,
			TransactionPrice: trPrice,
			Quantity:         qty,
			AvgCost:          avgCost,
			Value:            value,
			DatetimeMs:       date,
			Description:      desc,
		}
		if itemID.Valid {
			h.TransactionLine.Item = &Item{
				ID:   int(itemID.Int64),
				Name: itemName.String,
			}
		}
		if marketPrice.Valid {
			h.MarketPrice = marketPrice.Float64
		}
		if marketValue.Valid {
			h.MarketValue = marketValue.Float64
		}
		balances = append(balances, h)
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
