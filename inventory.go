package inventory

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"

	_ "github.com/vmihailenco/msgpack/v5"
)

// type HookFunc func(tx Transaction, inv *Account) error

type Account struct {
	ID     int
	UUID   string
	Name   string
	Parent *Account
}

type Item struct {
	ID          int
	UUID        string
	Name        string
	Description string
	Unit        string
}

type Transaction struct {
	ID          int
	UUID        string
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
	ID          int
	Path        []string
	Account     *Account
	Item        *Item
	Unit        string
	Quantity    float64
	AvgCost     float64
	Value       float64
	DatetimeMs  int64
	Year        int
	Month       uint8
	Price       float64
	Currency    string
	MarketValue float64
	Description string
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

func RollupBalances(balances []BalanceHistory, paths map[string][]string) map[string]BalanceHistory {
	result := map[string]BalanceHistory{}
	for _, b := range balances {
		path := paths[b.Account.UUID]
		itemName := ""
		for i := 1; i <= len(path); i++ {
			key := strings.Join(path[:i], " > ") + " " + itemName
			agg := result[key]
			agg.Path = path[:i]
			agg.Quantity += b.Quantity
			agg.Value += b.Value
			agg.DatetimeMs = b.DatetimeMs
			result[key] = agg
		}
	}
	return result
}

func PrintBalances(db *sql.DB) error {
	leaf, err := FetchLeafBalances(db)
	if err != nil {
		return err
	}
	paths, err := BuildAccountTree(db)
	if err != nil {
		return err
	}
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
		fmt.Printf("%s | Qty %.2f | Value %.2f | as of %d\n", k, b.Quantity, b.Value, b.DatetimeMs)
	}
	return nil
}

func RollupMarketBalances(balances []BalanceHistory, paths map[string][]string) map[string]BalanceHistory {
	result := map[string]BalanceHistory{}
	for _, b := range balances {
		path := paths[b.Account.UUID]
		itemName := ""
		for i := 1; i <= len(path); i++ {
			key := strings.Join(path[:i], " > ") + itemName
			agg := result[key]
			agg.Path = path[:i]
			agg.Quantity += b.Quantity
			agg.MarketValue += b.MarketValue
			agg.Currency = b.Currency
			agg.Price = b.Price
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
	paths, err := BuildAccountTree(db)
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
