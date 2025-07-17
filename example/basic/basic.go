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
		DefaultUnit: "pcs",
	})

	// Optional: add exchange rates
	inv.Currency.AddRate("USD", 16000)
	inv.Currency.AddRate("EUR", 17000)

	// Optional: add unit conversions
	inv.Converter.AddRule("A1", "box", 10) // 1 box = 10 pcs

	inv.AddTransaction(inventory.TransactionTypeAdd, []inventory.TransactionItem{
		{ItemID: "A1", Quantity: 2, Unit: "box"}, // adds 20 pcs
		{ItemID: "A1", Quantity: 5, Unit: "pcs"}, // adds 5 pcs
	}, "Restock with box and pcs")

	fmt.Println("Apple stock:", inv.GetBalanceForItem("A1")) // Output: 25
}
