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
	// assetUUID, inventoryUUID, rawMaterialUUID, workInProgressUUID,
	// finishedProductUUID, cashUUID, equityUUID, expenseUUID, matPurchaseUUID, equipmentPurchaseUUID,
	// incomeUUID, financialIncomeUUID, nonFinancialIncomeUUID string
	// steelUUID, woodUUID, widget1UUID, widget2UUID string
	assetAcc, inventoryAcc, rawMaterialAcc, workInProgressAcc,
	finishedProductAcc, cashAcc, equityAcc, expenseAcc, matPurchaseAcc, equipmentPurchaseAcc,
	incomeAcc, financialIncomeAcc, nonFinancialIncomeAcc *inventory.Account
	steelItem, woodItem, widget1Item, widget2Item *inventory.Item
)

func createAccountsAndItems(db *sql.DB) error {
	// Account hierarchy
	var err error
	assetAcc, err = inventory.AddAccount(db, "asset", "")
	if err != nil {
		return err
	}
	inventoryAcc, err = inventory.AddAccount(db, "inventory", assetAcc.UUID)
	if err != nil {
		return err
	}
	nonFinancialIncomeAcc, err = inventory.AddAccount(db, "non-financial income", inventoryAcc.UUID)
	if err != nil {
		return err
	}
	rawMaterialAcc, err = inventory.AddAccount(db, "raw material", inventoryAcc.UUID)
	if err != nil {
		return err
	}
	workInProgressAcc, err = inventory.AddAccount(db, "work in progress", inventoryAcc.UUID)
	if err != nil {
		return err
	}
	finishedProductAcc, err = inventory.AddAccount(db, "finished product", inventoryAcc.UUID)
	if err != nil {
		return err
	}
	cashAcc, err = inventory.AddAccount(db, "cash", assetAcc.UUID)
	if err != nil {
		return err
	}
	equityAcc, err = inventory.AddAccount(db, "equity", "")
	if err != nil {
		return err
	}
	expenseAcc, err = inventory.AddAccount(db, "expense", "")
	if err != nil {
		return err
	}
	matPurchaseAcc, err = inventory.AddAccount(db, "material purchase", expenseAcc.UUID)
	if err != nil {
		return err
	}
	equipmentPurchaseAcc, err = inventory.AddAccount(db, "equipment purchase", expenseAcc.UUID)
	if err != nil {
		return err
	}
	incomeAcc, err = inventory.AddAccount(db, "income", "")
	if err != nil {
		return err
	}
	financialIncomeAcc, err = inventory.AddAccount(db, "financial income", incomeAcc.UUID)
	if err != nil {
		return err
	}

	// Items
	steelItem, err = inventory.AddItem(db, "steel", "kg", "")
	if err != nil {
		return err
	}
	woodItem, err = inventory.AddItem(db, "wood", "kg", "")
	if err != nil {
		return err
	}
	widget1Item, err = inventory.AddItem(db, "widget 1", "pcs", "")
	if err != nil {
		return err
	}
	widget2Item, err = inventory.AddItem(db, "widget 1", "pcs", "")
	if err != nil {
		return err
	}

	return nil
}

func main() {
	// Remove existing database file for a clean example run
	fmt.Println("removing inventory.db")
	_ = os.Remove("inventory.db")

	// Initialize SQLite DB
	fmt.Println("initialize sqlite")
	db, err := sql.Open("sqlite3", "file:inventory.db?cache=shared&mode=rwc")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	defer fmt.Println("closing db")

	fmt.Println("init schema")
	err = inventory.InitSchema(db)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("create accounts and items")
	err = createAccountsAndItems(db)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("apply transactions")
	// Transaction: Owner invests 1000 USD equity → Cash
	err = inventory.ApplyTransaction(db, "Owner Investment", time.Date(2025, 9, 1, 0, 0, 0, 0, time.Local), []inventory.TransactionLine{
		{Account: cashAcc, Item: nil, Quantity: 1000, Unit: "USD", Price: 1, Currency: "USD"},    // Cash
		{Account: equityAcc, Item: nil, Quantity: -1000, Unit: "USD", Price: 1, Currency: "USD"}, // Equity
	})
	if err != nil {
		log.Fatal(err)
	}

	steelPrice := 5.0
	widgetSteelNeed := 2.0 // 2 kg steel per widget
	targetWidgetProduction := 10.0
	steelNeeded := widgetSteelNeed * targetWidgetProduction

	err = inventory.ApplyTransaction(db, "Purchase Steel 100kg", time.Date(2025, 9, 2, 0, 0, 0, 0, time.Local), []inventory.TransactionLine{
		{Account: rawMaterialAcc, Item: steelItem, Quantity: 100, Unit: "kg", Price: steelPrice, Currency: ""}, // Raw Materials
		{Account: nonFinancialIncomeAcc, Item: steelItem, Quantity: -100, Unit: "kg", Price: steelPrice, Currency: "USD"},
		{Account: cashAcc, Item: nil, Quantity: -500, Unit: "USD", Price: 1, Currency: "USD"},       // Cash decreases
		{Account: matPurchaseAcc, Item: nil, Quantity: 500, Unit: "USD", Price: 1, Currency: "USD"}, // Expense recognized
	})
	if err != nil {
		log.Fatal(err)
	}

	err = inventory.ApplyTransaction(db, "Purchase Wood 100kg", time.Date(2025, 9, 3, 0, 0, 0, 0, time.Local), []inventory.TransactionLine{
		{Account: rawMaterialAcc, Item: woodItem, Quantity: 150, Unit: "kg", Price: steelPrice, Currency: ""}, // Raw Materials
		{Account: nonFinancialIncomeAcc, Item: woodItem, Quantity: -150, Unit: "kg", Price: steelPrice, Currency: "USD"},
		{Account: cashAcc, Item: nil, Quantity: -300, Unit: "USD", Price: 1, Currency: "USD"},       // Cash decreases
		{Account: matPurchaseAcc, Item: nil, Quantity: 300, Unit: "USD", Price: 1, Currency: "USD"}, // Expense recognized
	})
	if err != nil {
		log.Fatal(err)
	}

	err = inventory.ApplyTransaction(db, "Use Steel to Manufacture Widgets", time.Date(2025, 9, 4, 0, 0, 0, 0, time.Local), []inventory.TransactionLine{
		{Account: workInProgressAcc, Item: steelItem, Quantity: steelNeeded, Unit: "kg", Price: steelPrice, Currency: ""}, // WIP increases
		{Account: rawMaterialAcc, Item: steelItem, Quantity: -steelNeeded, Unit: "kg", Price: steelPrice, Currency: ""},   // Raw Materials decreases
	})
	if err != nil {
		log.Fatal(err)
	}

	widgetCost := steelNeeded * steelPrice / targetWidgetProduction // 100kg steel makes 50 widgets at 5 USD/kg
	err = inventory.ApplyTransaction(db, "Complete Widgets", time.Date(2025, 9, 5, 0, 0, 0, 0, time.Local), []inventory.TransactionLine{
		{Account: finishedProductAcc, Item: widget1Item, Quantity: targetWidgetProduction, Unit: "pcs", Price: widgetCost, Currency: ""}, // Finished Goods increases
		{Account: workInProgressAcc, Item: steelItem, Quantity: -steelNeeded, Unit: "kg", Price: steelPrice, Currency: ""},               // WIP decreases
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("update market price")
	// Market prices
	err = inventory.SetMarketPrice(db, steelItem.UUID, 8, "USD", "kg") // steel now 8 USD/kg
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
