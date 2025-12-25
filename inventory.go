package inventory

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/vmihailenco/msgpack/v5"
)

// type HookFunc func(tx Transaction, inv *Account) error

type Account struct {
	ID     int
	UUID   uuid.UUID
	Name   string
	Parent *Account
}

func (a *Account) IsChildOfOrItself(parent *Account) bool {
	if parent == nil {
		return false
	}
	tmp := a
	for tmp != nil {
		if tmp == parent {
			return true
		}
		tmp = tmp.Parent
	}
	return false
}

type Item struct {
	ID          int
	UUID        uuid.UUID
	Name        string
	Description string
	Unit        string
}

type Transaction struct {
	ID               int
	UUID             uuid.UUID
	Description      string
	DatetimeMs       int64
	Year             int
	Month            uint8
	TransactionLines []*TransactionLine
}

type TransactionLine struct {
	ID          int
	UUID        uuid.UUID
	Transaction *Transaction
	Account     *Account
	Item        *Item
	Quantity    Decimal
	Unit        string
	Price       Decimal
	Currency    string
	Note        string
}

type BalanceHistory struct {
	ID               int
	UUID             uuid.UUID
	Path             []string
	TransactionLine  *TransactionLine
	Unit             string
	Quantity         Decimal
	AvgCost          Decimal
	Value            Decimal
	DatetimeMs       int64
	Year             int
	Month            uint8
	TransactionPrice Decimal
	MarketPrice      Decimal
	Currency         string
	MarketValue      Decimal
	Description      string
}

type UnitConversions struct {
	FromUnit   string
	ToUnit     string
	Factor     Decimal
	DatetimeMs int64
}

type CurrencyConversion struct {
	FromCurrency string
	ToCurrency   string
	Rate         Decimal
	DatetimeMs   int64
}

type MarketPrice struct {
	ID         int
	Item       *Item
	DatetimeMs int64
	Price      Decimal
	Unit       string
	Currency   string
}

func RollupBalances(balances []BalanceHistory, paths map[int][]string) map[string]BalanceHistory {
	// fmt.Println("enter rollup balances")
	result := map[string]BalanceHistory{}
	for _, b := range balances {
		path := paths[b.TransactionLine.Account.ID]
		var itemName string
		// var itemID int
		if b.TransactionLine.Item != nil {
			itemName = b.TransactionLine.Item.Name
			// itemID = b.TransactionLine.Item.ID

		} else {
			itemName = ""
		}
		for i := 1; i <= len(path); i++ {
			key := strings.Join(path[:i], " > ") + " " + itemName
			agg, ok := result[key]
			if !ok {
				agg.Quantity = NewDecimal(0)
				agg.Value = NewDecimal(0)
				agg.MarketValue = NewDecimal(0)
			}

			// fmt.Printf("agg %s to tl %d acc %d it %d: qty %s+%s val %s+%s mval %s+%s\n", key, b.TransactionLine.ID, b.TransactionLine.Account.ID, itemID,
			// 	agg.Quantity.ToString(), b.Quantity.ToString(),
			// 	agg.Value.ToString(), b.Value.ToString(),
			// 	agg.MarketValue.ToString(), b.MarketValue.ToString())
			agg.Path = path[:i]
			agg.Quantity.Data += b.Quantity.Data
			agg.Value.Data += b.Value.Data
			agg.MarketValue.Data += b.MarketValue.Data
			agg.Currency = b.Currency
			agg.DatetimeMs = b.DatetimeMs
			agg.TransactionLine = b.TransactionLine
			agg.TransactionPrice = b.TransactionPrice
			result[key] = agg
		}
	}
	// fmt.Println("exit rollup balances")
	return result
}

func SprintBalances(db *sql.DB) (string, error) {
	outStr := ""
	// fmt.Println("building account tree")
	paths, accMap, err := BuildAccountTree(db)
	if err != nil {
		return outStr, err
	}
	// fmt.Println("fetching leaf balances")
	leaf, err := FetchLeafBalances(db, accMap)
	if err != nil {
		return outStr, err
	}
	// fmt.Println("rolling up balances")
	rolled := RollupBalances(leaf, paths)

	outStr += fmt.Sprintln("=== Historical Cost Balances ===")
	keys := make([]string, 0)
	for k := range rolled {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for i := range keys {
		k := keys[i]
		b := rolled[k]
		normQty := NewDecimal(0)
		normVal := NewDecimal(0)
		if b.TransactionLine.Account != nil &&
			(b.TransactionLine.Account.IsChildOfOrItself(LiabilityAcc) ||
				b.TransactionLine.Account.IsChildOfOrItself(EquityAcc) ||
				b.TransactionLine.Account.IsChildOfOrItself(IncomeAcc)) {
			normQty.Data = -b.Quantity.Data
			normVal.Data = -b.Value.Data
		} else {
			normQty = b.Quantity
			normVal = b.Value
		}
		outStr += fmt.Sprintf("%s | Qty %.2f | Value %.2f | as of %v\n", k, normQty.ToFloat(), normVal.ToFloat(), time.UnixMilli(b.DatetimeMs))
	}
	return outStr, nil
}

func PrintBalances(db *sql.DB) error {
	str, err := SprintBalances(db)
	fmt.Print(str)
	return err
}

func SprintMarketBalances(db *sql.DB) (string, error) {
	// fmt.Println("building account tree")
	outStr := ""
	paths, accMap, err := BuildAccountTree(db)
	if err != nil {
		return outStr, err
	}
	// fmt.Println("fetching leaf balances")
	leaf, err := FetchLeafBalances(db, accMap)
	if err != nil {
		return outStr, err
	}
	// fmt.Println("rolling up balances")
	rolled := RollupBalances(leaf, paths)

	outStr += fmt.Sprintln("\n=== Market Value Balances ===")
	keys := make([]string, 0)
	for k := range rolled {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for i := range keys {
		k := keys[i]
		b := rolled[k]
		outStr += fmt.Sprintf("%s | Qty %.2f | MarketValue %.2f %s\n",
			k, b.Quantity.ToFloat(), b.MarketValue.ToFloat(), b.Currency)
	}

	return outStr, nil
}

func PrintMarketBalances(db *sql.DB) error {
	str, err := SprintMarketBalances(db)
	fmt.Print(str)
	return err
}
