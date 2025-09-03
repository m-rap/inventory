package main

import (
	"database/sql"
	"fmt"
	"inventory"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

var (
	assetID, inventoryID, rawMaterialsID, workInProgressID, finishedGoodsID, cashID, equityID, expensesID string
	steelID, widgetID                                                                                     string
)

func seedData(db *sql.DB) error {
	// Account hierarchy
	var err error
	assetID, err = inventory.AddAccount(db, "Assets", "")
	if err != nil {
		return err
	}
	inventoryID, err = inventory.AddAccount(db, "Inventory", assetID)
	if err != nil {
		return err
	}
	rawMaterialsID, err = inventory.AddAccount(db, "Raw Materials", inventoryID)
	if err != nil {
		return err
	}
	workInProgressID, err = inventory.AddAccount(db, "Work In Progress", inventoryID)
	if err != nil {
		return err
	}
	finishedGoodsID, err = inventory.AddAccount(db, "Finished Goods", inventoryID)
	if err != nil {
		return err
	}
	cashID, err = inventory.AddAccount(db, "Cash", assetID)
	if err != nil {
		return err
	}
	equityID, err = inventory.AddAccount(db, "Equity", "")
	if err != nil {
		return err
	}
	expensesID, err = inventory.AddAccount(db, "Expenses", "")
	if err != nil {
		return err
	}

	// Items
	steelID, err = inventory.AddItem(db, "Steel", "kg", "")
	if err != nil {
		return err
	}
	widgetID, err = inventory.AddItem(db, "Widget", "pcs", "")
	if err != nil {
		return err
	}

	return nil
}

func main() {
	// Remove existing database file for a clean example run
	_ = os.Remove("inventory.db")

	// Initialize SQLite DB
	db, err := sql.Open("sqlite3", "file:inventory.db?cache=shared&mode=rwc")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	inventory.InitSchema(db)
	err = seedData(db)
	if err != nil {
		log.Fatal(err)
	}

	// Transaction: Owner invests 1000 USD equity → Cash
	inventory.ApplyTransaction(db, "Owner Investment", []inventory.Line{
		{AccountID: cashID, ItemID: "", Quantity: 1000, Unit: "USD", Price: 1, Currency: "USD"},   // Cash
		{AccountID: equityID, ItemID: "", Quantity: 1000, Unit: "USD", Price: 1, Currency: "USD"}, // Equity
	})

	// Transaction: Purchase Steel (100kg @ 5 USD/kg)
	inventory.ApplyTransaction(db, "Purchase Steel 100kg", []inventory.Line{
		{AccountID: rawMaterialsID, ItemID: steelID, Quantity: 100, Unit: "kg", Price: 5, Currency: "USD"}, // Raw Materials
		{AccountID: cashID, ItemID: "", Quantity: -500, Unit: "USD", Price: 1, Currency: "USD"},            // Cash decreases
		{AccountID: expensesID, ItemID: "", Quantity: 500, Unit: "USD", Price: 1, Currency: "USD"},         // Expense recognized
	})

	// Market prices
	inventory.SetMarketPrice(db, steelID, 8, "USD") // steel now 8 USD/kg

	fmt.Println("=== Historical Cost Balances (Leaf Accounts) ===")
	inventory.PrintBalances(db)

	fmt.Println("\n=== Market Value Balances (Leaf Accounts) ===")
	inventory.PrintMarketBalances(db)
}
