package main

import (
	"fmt"
	"inventory"
	"time"
)

func main() {
	inv := inventory.NewInventory("IDR")

	inv.RegisterItem(inventory.Item{
		ID:          "A1",
		Name:        "Apple",
		Unit:        "pcs",
		Currency:    "IDR",
		Description: "",
	})

	// Optional: add exchange rates
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

	// Optional: add unit conversions
	inventory.AddUnitConversionRule(inventory.UnitConversionRule{
		FromUnit: "box",
		ToUnit:   "pcs",
		Factor:   10,
	}) // 1 box = 10 pcs

	inv.AddTransaction(inventory.Transaction{
		ID:   inventory.GenerateUUID(),
		Type: inventory.TransactionTypeAdd,
		Items: []inventory.TransactionItem{
			{ItemID: "Apple", Quantity: 2, Unit: "box"}, // adds 20 pcs
			{ItemID: "Apple", Quantity: 5, Unit: "pcs"}, // adds 5 pcs
		},
		Note:        "Restock with box and pcs",
		Timestamp:   time.Now(),
		InventoryID: inv.ID,
	})

	fmt.Println("Apple stock:", inv.GetBalancesForItems([]string{"Apple"})) // Output: 25
}
