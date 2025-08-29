package main

import (
	"database/sql"
	"fmt"
	"inventory"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Remove existing database file for a clean example run
	_ = os.Remove("inventory.db")

	// Initialize SQLite DB
	db, err := sql.Open("sqlite3", "file:inventory.db?cache=shared&mode=rwc")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Create inventory bound to SQLite
	inv, err := inventory.WithSQLite(db)
	if err != nil {
		panic(err)
	}

	// Register item with generated UUID
	appleID := inventory.GenerateUUID()
	inventory.RegisterItem(db, inventory.Item{
		ID:          appleID,
		Name:        "Apple",
		Unit:        "pcs",
		Currency:    "IDR",
		Description: "",
	})
	orangeID := inventory.GenerateUUID()
	inventory.RegisterItem(db, inventory.Item{
		ID:          orangeID,
		Name:        "Orange",
		Unit:        "pcs",
		Currency:    "IDR",
		Description: "",
	})
	bananaId := inventory.GenerateUUID()
	inventory.RegisterItem(db, inventory.Item{
		ID:          bananaId,
		Name:        "Banana",
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

	inv1, err := inv.AddChildInventory("inv1")
	if err != nil {
		panic(err)
	}

	inv2, err := inv.AddChildInventory("inv2")
	if err != nil {
		panic(err)
	}

	// Add a transaction
	inv1.AddTransaction(inventory.Transaction{
		ID:   inventory.GenerateUUID(),
		Type: inventory.TransactionTypeAdd,
		Items: []inventory.TransactionItem{
			{ItemID: appleID, Quantity: 2, Unit: "box"},  // adds 20 pcs
			{ItemID: appleID, Quantity: 5, Unit: "pcs"},  // adds 5 pcs
			{ItemID: orangeID, Quantity: 9, Unit: "pcs"}, // adds 9 pcs
		},
		Note:      "Restock with box and pcs",
		Timestamp: time.Now(),
	})

	// Add a transaction
	inv2.AddTransaction(inventory.Transaction{
		ID:   inventory.GenerateUUID(),
		Type: inventory.TransactionTypeAdd,
		Items: []inventory.TransactionItem{
			{ItemID: appleID, Quantity: 1, Unit: "box"},  // adds 10 pcs
			{ItemID: orangeID, Quantity: 3, Unit: "pcs"}, // adds 3 pcs
			{ItemID: bananaId, Quantity: 2, Unit: "box"}, // adds 3 pcs
		},
		Note:      "Restock with box and pcs",
		Timestamp: time.Now(),
	})

	// Persist changes if needed
	inventory.PersistAllInventories(inv)

	// Check balance
	balances := inv.GetBalancesForItems([]string{appleID})
	appleBalance := balances[appleID]
	fmt.Printf("Apple stock: %d %s (Value: %.2f %s)\n",
		appleBalance.Quantity, appleBalance.Unit,
		appleBalance.Value, appleBalance.Currency)

	fmt.Println()

	allBalances := inv.GetBalances()
	fmt.Println("All item balances:")
	for itemID, balance := range allBalances {
		item, _ := inventory.GetItem(db, itemID)
		fmt.Printf("- %s: %d %s (Value: %.2f %s)\n",
			item.Name, balance.Quantity, balance.Unit,
			balance.Value, balance.Currency)
	}

	inv1Balances := inv1.GetBalances()
	fmt.Println("\nInventory 1 item balances:")
	for itemID, balance := range inv1Balances {
		item, _ := inventory.GetItem(db, itemID)
		fmt.Printf("- %s: %d %s (Value: %.2f %s)\n",
			item.Name, balance.Quantity, balance.Unit,
			balance.Value, balance.Currency)
	}

	inv2Balances := inv2.GetBalances()
	fmt.Println("\nInventory 2 item balances:")
	for itemID, balance := range inv2Balances {
		item, _ := inventory.GetItem(db, itemID)
		fmt.Printf("- %s: %d %s (Value: %.2f %s)\n",
			item.Name, balance.Quantity, balance.Unit,
			balance.Value, balance.Currency)
	}
}
