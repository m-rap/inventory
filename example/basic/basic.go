package main

import (
	"fmt"
	"inventory"
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
	inv.AddCurrencyConversionRule("USD", "IDR", 16000)
	inv.AddCurrencyConversionRule("EUR", "IDR", 17000)

	// Optional: add unit conversions
	inv.AddUnitConversionRule("box", "pcs", 10) // 1 box = 10 pcs

	inv.AddTransaction(inventory.TransactionTypeAdd, []inventory.TransactionItem{
		{ItemID: "A1", Quantity: 2, Unit: "box"}, // adds 20 pcs
		{ItemID: "A1", Quantity: 5, Unit: "pcs"}, // adds 5 pcs
	}, "Restock with box and pcs")

	fmt.Println("Apple stock:", inv.GetBalanceForItem("A1")) // Output: 25
}
