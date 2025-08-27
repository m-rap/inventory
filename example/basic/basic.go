package main

import (
	"database/sql"
	"fmt"
	"inventory"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Initialize SQLite DB
	db, err := sql.Open("sqlite3", "file:inventory.db?cache=shared&mode=rwc")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Create inventory bound to SQLite
	inv := inventory.WithSQLite(db)

	// Register item with generated UUID
	appleID := inventory.GenerateUUID()
	inv.RegisterItem(inventory.Item{
		ID:          appleID,
		Name:        "Apple",
		Unit:        "pcs",
		Currency:    "IDR",
		Description: "",
	})

	// Add exchange rates
	inventory.AddCurrencyConversionRule(inventory.CurrencyConversionRule{
		FromCurrency: "USD",
		ToCurrency:   "IDR",
		Rate:         16000,
	})
	inventory.AddCurrencyConversionRule(inventory.CurrencyConversionRule{
		FromCurrency: "EUR",
		ToCurrency:   "IDR",
		Rate:         17000,
	})

	// Add unit conversions
	inventory.AddUnitConversionRule(inventory.UnitConversionRule{
		FromUnit: "box",
		ToUnit:   "pcs",
		Factor:   10,
	}) // 1 box = 10 pcs

	// Add a transaction
	inv.AddTransaction(inventory.Transaction{
		ID:   inventory.GenerateUUID(),
		Type: inventory.TransactionTypeAdd,
		Items: []inventory.TransactionItem{
			{ItemID: appleID, Quantity: 2, Unit: "box"}, // adds 20 pcs
			{ItemID: appleID, Quantity: 5, Unit: "pcs"}, // adds 5 pcs
		},
		Note:        "Restock with box and pcs",
		Timestamp:   time.Now(),
		InventoryID: inv.ID,
	})

	// Check balance
	balances := inv.GetBalancesForItems([]string{appleID})
	appleBalance := balances[appleID]
	fmt.Printf("Apple stock: %d %s (Value: %.2f %s)\n",
		appleBalance.Quantity, appleBalance.Unit,
		appleBalance.Value, appleBalance.Currency)
}
