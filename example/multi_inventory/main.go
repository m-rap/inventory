package main

import (
	"database/sql"
	"fmt"
	"inventory"
	"log"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var (
	assetID, inventoryID, rawMaterialUUID, workInProgressID,
	finishedProductID, cashUUID, equityUUID, expenseID, matPurchaseUUID, equipmentPurchaseID,
	incomeID, financialIncomeID, nonFinancialIncomeUUID string
	steelUUID, woodID, widget1ID, widget2ID string
)

func seedData(db *sql.DB) error {
	// Account hierarchy
	var err error
	assetID, err = inventory.AddAccount(db, "asset", "")
	if err != nil {
		return err
	}
	inventoryID, err = inventory.AddAccount(db, "inventory", assetID)
	if err != nil {
		return err
	}
	rawMaterialUUID, err = inventory.AddAccount(db, "raw material", inventoryID)
	if err != nil {
		return err
	}
	workInProgressID, err = inventory.AddAccount(db, "work in progress", inventoryID)
	if err != nil {
		return err
	}
	finishedProductID, err = inventory.AddAccount(db, "finished product", inventoryID)
	if err != nil {
		return err
	}
	cashUUID, err = inventory.AddAccount(db, "cash", assetID)
	if err != nil {
		return err
	}
	equityUUID, err = inventory.AddAccount(db, "equity", "")
	if err != nil {
		return err
	}
	expenseID, err = inventory.AddAccount(db, "expense", "")
	if err != nil {
		return err
	}
	matPurchaseUUID, err = inventory.AddAccount(db, "material purchase", expenseID)
	if err != nil {
		return err
	}
	equipmentPurchaseID, err = inventory.AddAccount(db, "equipment purchase", expenseID)
	if err != nil {
		return err
	}
	incomeID, err = inventory.AddAccount(db, "income", "")
	if err != nil {
		return err
	}
	financialIncomeID, err = inventory.AddAccount(db, "financial income", incomeID)
	if err != nil {
		return err
	}
	nonFinancialIncomeUUID, err = inventory.AddAccount(db, "non-financial income", incomeID)
	if err != nil {
		return err
	}

	// Items
	steelUUID, err = inventory.AddItem(db, "steel", "kg", "")
	if err != nil {
		return err
	}
	woodID, err = inventory.AddItem(db, "wood", "kg", "")
	if err != nil {
		return err
	}
	widget1ID, err = inventory.AddItem(db, "widget 1", "pcs", "")
	if err != nil {
		return err
	}
	widget2ID, err = inventory.AddItem(db, "widget 1", "pcs", "")
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

	err = inventory.InitSchema(db)
	if err != nil {
		log.Fatal(err)
	}
	err = seedData(db)
	if err != nil {
		log.Fatal(err)
	}

	// Transaction: Owner invests 1000 USD equity → Cash
	err = inventory.ApplyTransaction(db, "Owner Investment", time.Date(2025, 9, 1, 0, 0, 0, 0, time.Local), []inventory.TransactionLine{
		{AccountUUID: cashUUID, ItemUUID: "", Quantity: 1000, Unit: "USD", Price: 1, Currency: "USD"},    // Cash
		{AccountUUID: equityUUID, ItemUUID: "", Quantity: -1000, Unit: "USD", Price: 1, Currency: "USD"}, // Equity
	})
	if err != nil {
		log.Fatal(err)
	}

	steelPrice := 5.0
	widgetSteelNeed := 2.0 // 2 kg steel per widget
	targetWidgetProduction := 10.0
	steelNeeded := widgetSteelNeed * targetWidgetProduction

	err = inventory.ApplyTransaction(db, "Purchase Steel 100kg", time.Date(2025, 9, 2, 0, 0, 0, 0, time.Local), []inventory.TransactionLine{
		{AccountUUID: rawMaterialUUID, ItemUUID: steelUUID, Quantity: 100, Unit: "kg", Price: steelPrice, Currency: ""}, // Raw Materials
		{AccountUUID: nonFinancialIncomeUUID, ItemUUID: steelUUID, Quantity: -100, Unit: "kg", Price: steelPrice, Currency: "USD"},
		{AccountUUID: cashUUID, ItemUUID: "", Quantity: -500, Unit: "USD", Price: 1, Currency: "USD"},       // Cash decreases
		{AccountUUID: matPurchaseUUID, ItemUUID: "", Quantity: 500, Unit: "USD", Price: 1, Currency: "USD"}, // Expense recognized
	})
	if err != nil {
		log.Fatal(err)
	}

	err = inventory.ApplyTransaction(db, "Purchase Wood 100kg", time.Date(2025, 9, 3, 0, 0, 0, 0, time.Local), []inventory.TransactionLine{
		{AccountUUID: rawMaterialUUID, ItemUUID: woodID, Quantity: 150, Unit: "kg", Price: steelPrice, Currency: ""}, // Raw Materials
		{AccountUUID: nonFinancialIncomeUUID, ItemUUID: woodID, Quantity: -150, Unit: "kg", Price: steelPrice, Currency: "USD"},
		{AccountUUID: cashUUID, ItemUUID: "", Quantity: -300, Unit: "USD", Price: 1, Currency: "USD"},       // Cash decreases
		{AccountUUID: matPurchaseUUID, ItemUUID: "", Quantity: 300, Unit: "USD", Price: 1, Currency: "USD"}, // Expense recognized
	})
	if err != nil {
		log.Fatal(err)
	}

	err = inventory.ApplyTransaction(db, "Use Steel to Manufacture Widgets", time.Date(2025, 9, 4, 0, 0, 0, 0, time.Local), []inventory.TransactionLine{
		{AccountUUID: workInProgressID, ItemUUID: steelUUID, Quantity: steelNeeded, Unit: "kg", Price: steelPrice, Currency: ""}, // WIP increases
		{AccountUUID: rawMaterialUUID, ItemUUID: steelUUID, Quantity: -steelNeeded, Unit: "kg", Price: steelPrice, Currency: ""}, // Raw Materials decreases
	})
	if err != nil {
		log.Fatal(err)
	}

	widgetCost := steelNeeded * steelPrice / targetWidgetProduction // 100kg steel makes 50 widgets at 5 USD/kg
	err = inventory.ApplyTransaction(db, "Complete Widgets", time.Date(2025, 9, 5, 0, 0, 0, 0, time.Local), []inventory.TransactionLine{
		{AccountUUID: finishedProductID, ItemUUID: widget1ID, Quantity: targetWidgetProduction, Unit: "pcs", Price: widgetCost, Currency: ""}, // Finished Goods increases
		{AccountUUID: workInProgressID, ItemUUID: steelUUID, Quantity: -steelNeeded, Unit: "kg", Price: steelPrice, Currency: ""},             // WIP decreases
	})
	if err != nil {
		log.Fatal(err)
	}

	// Market prices
	err = inventory.SetMarketPrice(db, steelUUID, 8, "USD", "kg") // steel now 8 USD/kg
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Historical Cost Balances (Leaf Accounts) ===")
	err = inventory.PrintBalances(db)
	if err != nil {
		log.Fatal(err)
	}

	// fmt.Println("\n=== Market Value Balances (Leaf Accounts) ===")
	// err = inventory.PrintMarketBalances(db)
	// if err != nil {
	// 	log.Fatal(err)
	// }
}
