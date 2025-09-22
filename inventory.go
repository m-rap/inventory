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
	ID          int
	UUID        uuid.UUID
	Description string
	DatetimeMs  int64
	Year        int
	Month       uint8
	Lines       []*TransactionLine
}

type TransactionLine struct {
	ID          int
	Transaction *Transaction
	Account     *Account
	Item        *Item
	Quantity    float64
	Unit        string
	Price       float64
	Currency    string
	Note        string
}

type BalanceHistory struct {
	ID               int
	Path             []string
	TransactionLine  *TransactionLine
	Unit             string
	Quantity         float64
	AvgCost          float64
	Value            float64
	DatetimeMs       int64
	Year             int
	Month            uint8
	TransactionPrice float64
	MarketPrice      float64
	Currency         string
	MarketValue      float64
	Description      string
}

type UnitConversions struct {
	FromUnit   string
	ToUnit     string
	Factor     float64
	DatetimeMs int64
}

type CurrencyConversions struct {
	FromCurrency string
	ToCurrency   string
	Rate         float64
	DatetimeMs   int64
}

type MarketPrices struct {
	ID         int
	Item       *Item
	DatetimeMs int64
	Price      float64
	Unit       string
	Currency   string
}

func RollupBalances(balances []BalanceHistory, paths map[int][]string) map[string]BalanceHistory {
	// fmt.Println("enter rollup balances")
	result := map[string]BalanceHistory{}
	for _, b := range balances {
		path := paths[b.TransactionLine.Account.ID]
		var itemName string
		if b.TransactionLine.Item != nil {
			itemName = b.TransactionLine.Item.Name
		} else {
			itemName = ""
		}
		for i := 1; i <= len(path); i++ {
			key := strings.Join(path[:i], " > ") + " " + itemName
			agg := result[key]
			agg.Path = path[:i]
			agg.Quantity += b.Quantity
			agg.Value += b.Value
			agg.DatetimeMs = b.DatetimeMs
			agg.TransactionLine = b.TransactionLine
			result[key] = agg
		}
	}
	// fmt.Println("exit rollup balances")
	return result
}

func PrintBalances(db *sql.DB) error {
	// fmt.Println("building account tree")
	paths, accMap, err := BuildAccountTree(db)
	if err != nil {
		return err
	}
	// fmt.Println("fetching leaf balances")
	leaf, err := FetchLeafBalances(db, accMap)
	if err != nil {
		return err
	}
	// fmt.Println("rolling up balances")
	rolled := RollupBalances(leaf, paths)

	fmt.Println("=== Historical Cost Balances ===")
	keys := make([]string, 0)
	for k := range rolled {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for i := range keys {
		k := keys[i]
		b := rolled[k]
		var normQty, normVal float64
		if b.TransactionLine.Account != nil &&
			(b.TransactionLine.Account.IsChildOfOrItself(LiabilityAcc) ||
				b.TransactionLine.Account.IsChildOfOrItself(EquityAcc) ||
				b.TransactionLine.Account.IsChildOfOrItself(IncomeAcc)) {
			normQty = -b.Quantity
			normVal = -b.Value
		} else {
			normQty = b.Quantity
			normVal = b.Value
		}
		fmt.Printf("%s | Qty %.2f | Value %.2f | as of %v\n", k, normQty, normVal, time.UnixMilli(b.DatetimeMs))
	}
	return nil
}

func RollupMarketBalances(balances []BalanceHistory, paths map[int][]string) map[string]BalanceHistory {
	result := map[string]BalanceHistory{}
	for _, b := range balances {
		path := paths[b.TransactionLine.Account.ID]
		itemName := ""
		for i := 1; i <= len(path); i++ {
			key := strings.Join(path[:i], " > ") + itemName
			agg := result[key]
			agg.Path = path[:i]
			agg.Quantity += b.Quantity
			agg.MarketValue += b.MarketValue
			agg.Currency = b.Currency
			agg.TransactionPrice = b.TransactionPrice
			result[key] = agg
		}
	}
	return result
}

func PrintMarketBalances(db *sql.DB) error {
	leaf, err := FetchLeafMarketBalances(db)
	if err != nil {
		return err
	}
	paths, _, err := BuildAccountTree(db)
	if err != nil {
		return err
	}
	rolled := RollupMarketBalances(leaf, paths)

	fmt.Println("\n=== Market Value Balances ===")
	for k, b := range rolled {
		fmt.Printf("%s | Qty %.2f | MarketValue %.2f %s\n",
			k, b.Quantity, b.MarketValue, b.Currency)
	}

	return nil
}
