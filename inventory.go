package inventory

import (
	"database/sql"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/vmihailenco/msgpack/v5"
)

// type HookFunc func(tx Transaction, inv *Account) error

const DECIMALPRECISION = 10000

type Decimal struct {
	Precision uint32
	Data      int64
	Format    string
}

func NewDecimal(rawI64 int64) Decimal {
	return Decimal{
		Precision: DECIMALPRECISION,
		Data:      rawI64,
		Format:    fmt.Sprintf("%%d.%%%dd", DECIMALPRECISION/10),
	}
}

func NewDecimalFromIntFrac(intPart, fracPartPrecModulo int64) Decimal {
	if intPart >= 0 {
		return NewDecimal(intPart*DECIMALPRECISION + fracPartPrecModulo)
	} else {
		return NewDecimal(intPart*DECIMALPRECISION - fracPartPrecModulo)
	}
}

func NewDecimalFromFloat(fdata float64) Decimal {
	return NewDecimal(int64(fdata * DECIMALPRECISION))
}

func NewDecimalFromStr(str string) Decimal {
	var intPart int64
	var fracPart int64
	var fracStr string
	strs := strings.Split(str, ".")
	if len(strs) >= 1 {
		iTmp, _ := strconv.Atoi(strs[0])
		intPart = int64(iTmp)
		if len(strs) > 1 {
			fracStr = strs[1]
		} else {
			fracStr = "0"
		}
	}
	frontZeroCount := 0
	nonPaddedCount := 0
	nonPaddedStr := ""
	for _, c := range fracStr {
		if nonPaddedCount == 0 {
			if c == '0' {
				frontZeroCount++
			} else {
				nonPaddedCount++
				nonPaddedStr += string(c)
			}
		} else {
			nonPaddedCount++
		}
	}
	decPrecZeroDigitCount := DECIMALPRECISION / 10
	// misal 0.02, zeroCount = 1, nonPaddedCount = 1. (2 * 10^2)/10000
	// misal 0.0002, zeroCount = 3, nonPaddedCount = 1 -> (2 * 10^0)/10000
	// misal 0.2, zeroCount = 0, nonPaddedCount = 1 -> (2 * 10^3)/10000
	// misal 0.20, zeroCount = 0, nonPaddedCount = 2 -> (20 * 10^2)/10000
	powFactor := decPrecZeroDigitCount - (frontZeroCount + nonPaddedCount)
	nonPaddedInt, _ := strconv.Atoi(nonPaddedStr)
	fracPart = int64(nonPaddedInt) * int64(math.Pow10(powFactor))
	return NewDecimalFromIntFrac(intPart, fracPart)
}

func (d Decimal) ToFloat() float64 {
	return float64(d.Data) / float64(d.Precision)
}

func (d Decimal) ToIntFrac() (int64, int64) {
	return d.Data / DECIMALPRECISION, d.Data % DECIMALPRECISION
}

func (d Decimal) ToString() string {
	return fmt.Sprintf(d.Format, d.Data/DECIMALPRECISION, d.Data%DECIMALPRECISION)
}

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
	Quantity    int64
	Unit        string
	Price       int64
	Currency    string
	Note        string
}

type BalanceHistory struct {
	ID               int
	UUID             uuid.UUID
	Path             []string
	TransactionLine  *TransactionLine
	Unit             string
	Quantity         int64
	AvgCost          int64
	Value            int64
	DatetimeMs       int64
	Year             int
	Month            uint8
	TransactionPrice int64
	MarketPrice      int64
	Currency         string
	MarketValue      int64
	Description      string
}

type UnitConversions struct {
	FromUnit   string
	ToUnit     string
	Factor     int64
	DatetimeMs int64
}

type CurrencyConversion struct {
	FromCurrency string
	ToCurrency   string
	Rate         float64
	DatetimeMs   int64
}

type MarketPrice struct {
	ID         int
	Item       *Item
	DatetimeMs int64
	Price      int64
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
			agg.MarketValue += b.MarketValue
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
		var normQty, normVal int64
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
		outStr += fmt.Sprintf("%s | Qty %.2f | Value %.2f | as of %v\n", k, NewDecimal(normQty).ToFloat(), NewDecimal(normVal).ToFloat(), time.UnixMilli(b.DatetimeMs))
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
	for k, b := range rolled {
		outStr += fmt.Sprintf("%s | Qty %.2f | MarketValue %.2f %s\n",
			k, NewDecimal(b.Quantity).ToFloat(), NewDecimal(b.MarketValue).ToFloat(), b.Currency)
	}

	return outStr, nil
}

func PrintMarketBalances(db *sql.DB) error {
	str, err := SprintMarketBalances(db)
	fmt.Print(str)
	return err
}
