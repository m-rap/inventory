package inventory

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"

	_ "github.com/vmihailenco/msgpack/v5"
)

// type HookFunc func(tx Transaction, inv *Account) error

type Item struct {
	ID          int    `msgpack:"id,omitempty"`
	UUID        string `msgpack:"uuid,omitempty"`
	Name        string `msgpack:"name,omitempty"`
	Description string `msgpack:"description,omitempty"`
	Unit        string `msgpack:"unit,omitempty"`
}

type Transaction struct {
	ID          int               `msgpack:"id,omitempty"`
	UUID        string            `msgpack:"uuid,omitempty"`
	Description string            `msgpack:"description,omitempty"`
	DatetimeMs  int64             `msgpack:"date,omitempty"`
	Year        int               `msgpack:"year,omitempty"`
	Month       uint8             `msgpack:"month,omitempty"`
	Lines       []TransactionLine `msgpack:"lines,omitempty"`
}

type TransactionLine struct {
	ID              int     `msgpack:"id,omitempty"`
	TransactionUUID string  `msgpack:"transaction_uuid,omitempty"`
	AccountUUID     string  `msgpack:"account_uuid,omitempty"`
	ItemUUID        string  `msgpack:"item_uuid,omitempty"`
	Quantity        float64 `msgpack:"quantity,omitempty"`
	Unit            string  `msgpack:"unit,omitempty"`
	Price           float64 `msgpack:"price,omitempty"`
	Currency        string  `msgpack:"currency,omitempty"`
	Note            string  `msgpack:"note,omitempty"`
}

type BalanceHistory struct {
	ID          int      `msgpack:"id,omitempty"`
	Path        []string `msgpack:"path,omitempty"`
	AccountUUID string   `msgpack:"account_uuid,omitempty"`
	ItemUUID    string   `msgpack:"item_uuid,omitempty"`
	Unit        string   `msgpack:"unit,omitempty"`
	Quantity    float64  `msgpack:"quantity,omitempty"`
	AvgCost     float64  `msgpack:"avg_cost,omitempty"`
	Value       float64  `msgpack:"value,omitempty"`
	DatetimeMs  int64    `msgpack:"date,omitempty"`
	Year        int      `msgpack:"year,omitempty"`
	Month       uint8    `msgpack:"month,omitempty"`
	Price       float64  `msgpack:"price,omitempty"`
	Currency    string   `msgpack:"currency,omitempty"`
	MarketValue float64  `msgpack:"market_value,omitempty"`
	Description string   `msgpack:"description,omitempty"`
}

type UnitConversions struct {
	FromUnit   string  `msgpack:"from_unit,omitempty"`
	ToUnit     string  `msgpack:"to_unit,omitempty"`
	Factor     float64 `msgpack:"factor,omitempty"`
	DatetimeMs int64   `msgpack:"date,omitempty"`
}

type CurrencyConversions struct {
	FromCurrency string  `msgpack:"from_currency,omitempty"`
	ToCurrency   string  `msgpack:"to_currency,omitempty"`
	Rate         float64 `msgpack:"rate,omitempty"`
	DatetimeMs   int64   `msgpack:"date,omitempty"`
}

type MarketPrices struct {
	ID         int     `msgpack:"id,omitempty"`
	ItemUUID   string  `msgpack:"item_uuid,omitempty"`
	DatetimeMs int64   `msgpack:"date,omitempty"`
	Price      float64 `msgpack:"price,omitempty"`
	Unit       string  `msgpack:"unit,omitempty"`
	Currency   string  `msgpack:"currency,omitempty"`
}

func RollupBalances(balances []BalanceHistory, paths map[string][]string) map[string]BalanceHistory {
	result := map[string]BalanceHistory{}
	for _, b := range balances {
		path := paths[b.AccountUUID]
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
		path := paths[b.AccountUUID]
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
