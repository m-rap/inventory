package inventory

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"
)

// type HookFunc func(tx Transaction, inv *Account) error

type Item struct {
	ID          string
	Name        string
	Description string
	Unit        string
}

type Transaction struct {
	ID          string
	Description string
	Date        time.Time
	Lines       []TransactionLine
}

type TransactionLine struct {
	TransactionID string
	AccountID     string
	ItemID        string
	Quantity      float64
	Unit          string
	Price         float64
	Currency      string
	Note          string
}

type Balance struct {
	Path        []string
	AccountID   string
	Item        string
	Unit        string
	Quantity    float64
	AvgCost     float64
	Value       float64
	Date        time.Time
	Price       float64
	Currency    string
	MarketValue float64
	Description string
}

func RollupBalances(balances []Balance, paths map[string][]string) map[string]Balance {
	result := map[string]Balance{}
	for _, b := range balances {
		path := paths[b.AccountID]
		for i := 1; i <= len(path); i++ {
			key := strings.Join(path[:i], " > ") + " " + b.Item
			agg := result[key]
			agg.Path = path[:i]
			agg.Quantity += b.Quantity
			agg.Value += b.Value
			agg.Date = b.Date
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
		fmt.Printf("%s | Qty %.2f | Value %.2f | as of %s\n", k, b.Quantity, b.Value, b.Date)
	}
	return nil
}

func RollupMarketBalances(balances []Balance, paths map[string][]string) map[string]Balance {
	result := map[string]Balance{}
	for _, b := range balances {
		path := paths[b.AccountID]
		for i := 1; i <= len(path); i++ {
			key := strings.Join(path[:i], " > ") + b.Item
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
