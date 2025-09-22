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
	inventoryAcc, rawMaterialAcc, workInProgressAcc,
	finishedProductAcc, cashAcc, matPurchaseAcc, equipmentPurchaseAcc,
	nonFinancialIncomeAcc, incomingMatAcc *inventory.Account
	steelItem, woodItem, widget1Item, widget2Item *inventory.Item
)

func createAccountsAndItems(db *sql.DB) error {
	// Account hierarchy
	var err error
	var sUuid []byte

	sUuid, err = inventory.AddAccount(db, "non-financial income", nil)
	if err != nil {
		return err
	}
	nonFinancialIncomeAcc, _ = inventory.GetAccountFromUUID(db, sUuid)

	sUuid, err = inventory.AddAccount(db, "incoming material", nonFinancialIncomeAcc.UUID[:])
	if err != nil {
		return err
	}
	incomingMatAcc, _ = inventory.GetAccountFromUUID(db, sUuid)

	sUuid, err = inventory.AddAccount(db, "inventory", inventory.AssetAcc.UUID[:])
	if err != nil {
		return err
	}
	inventoryAcc, _ = inventory.GetAccountFromUUID(db, sUuid)

	sUuid, err = inventory.AddAccount(db, "raw material", inventoryAcc.UUID[:])
	if err != nil {
		return err
	}
	rawMaterialAcc, _ = inventory.GetAccountFromUUID(db, sUuid)

	sUuid, err = inventory.AddAccount(db, "work in progress", inventoryAcc.UUID[:])
	if err != nil {
		return err
	}
	workInProgressAcc, _ = inventory.GetAccountFromUUID(db, sUuid)

	sUuid, err = inventory.AddAccount(db, "finished product", inventoryAcc.UUID[:])
	if err != nil {
		return err
	}
	finishedProductAcc, _ = inventory.GetAccountFromUUID(db, sUuid)

	sUuid, err = inventory.AddAccount(db, "cash", inventory.AssetAcc.UUID[:])
	if err != nil {
		return err
	}
	cashAcc, _ = inventory.GetAccountFromUUID(db, sUuid)

	sUuid, err = inventory.AddAccount(db, "material purchase", inventory.ExpenseAcc.UUID[:])
	if err != nil {
		return err
	}
	matPurchaseAcc, _ = inventory.GetAccountFromUUID(db, sUuid)

	sUuid, err = inventory.AddAccount(db, "equipment purchase", inventory.ExpenseAcc.UUID[:])
	if err != nil {
		return err
	}
	equipmentPurchaseAcc, _ = inventory.GetAccountFromUUID(db, sUuid)

	// Items
	sUuid, err = inventory.AddItem(db, "steel", "kg", "")
	if err != nil {
		return err
	}
	steelItem, _ = inventory.GetItemFromUUID(db, sUuid)

	sUuid, err = inventory.AddItem(db, "wood", "kg", "")
	if err != nil {
		return err
	}
	woodItem, _ = inventory.GetItemFromUUID(db, sUuid)

	sUuid, err = inventory.AddItem(db, "widget 1", "pcs", "")
	if err != nil {
		return err
	}
	widget1Item, _ = inventory.GetItemFromUUID(db, sUuid)

	sUuid, err = inventory.AddItem(db, "widget 1", "pcs", "")
	if err != nil {
		return err
	}
	widget2Item, _ = inventory.GetItemFromUUID(db, sUuid)

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
		inventory.CreateFinancialTrLine(inventory.EquityAcc, 0, 1000, "USD"), // suntik modal
		inventory.CreateFinancialTrLine(cashAcc, 1000, 0, "USD"),             // masuk cash
	})
	if err != nil {
		log.Fatal(err)
	}

	steelPrice := 5.0
	widgetSteelNeed := 2.0 // 2 kg steel per widget
	targetWidgetProduction := 10.0
	steelNeeded := widgetSteelNeed * targetWidgetProduction

	err = inventory.ApplyTransaction(db, "Purchase Steel 100kg", time.Date(2025, 9, 2, 0, 0, 0, 0, time.Local), []inventory.TransactionLine{
		inventory.CreateInventoryTrLine(incomingMatAcc, steelItem, -100, "kg", steelPrice, "USD"), // incoming material
		inventory.CreateInventoryTrLine(rawMaterialAcc, steelItem, 100, "kg", steelPrice, "USD"),  // added to raw material inventory
		inventory.CreateFinancialTrLine(cashAcc, 0, 500, "USD"),                                   // Cash decreases
		inventory.CreateFinancialTrLine(matPurchaseAcc, 500, 0, "USD"),                            // Expense recognized
	})
	if err != nil {
		log.Fatal(err)
	}

	woodPrice := 3.0
	err = inventory.ApplyTransaction(db, "Purchase Wood 100kg", time.Date(2025, 9, 3, 0, 0, 0, 0, time.Local), []inventory.TransactionLine{
		inventory.CreateInventoryTrLine(incomingMatAcc, woodItem, -150, "kg", woodPrice, "USD"), // incoming material
		inventory.CreateInventoryTrLine(rawMaterialAcc, woodItem, 150, "kg", woodPrice, "USD"),  // added to raw material inventory
		inventory.CreateFinancialTrLine(cashAcc, 0, 300, "USD"),                                 // Cash decreases
		inventory.CreateFinancialTrLine(matPurchaseAcc, 300, 0, "USD"),                          // Expense recognized
	})
	if err != nil {
		log.Fatal(err)
	}

	err = inventory.ApplyTransaction(db, "Use Steel to Manufacture Widgets", time.Date(2025, 9, 4, 0, 0, 0, 0, time.Local), []inventory.TransactionLine{
		inventory.CreateInventoryTrLine(rawMaterialAcc, steelItem, -steelNeeded, "kg", steelPrice, "USD"),   // raw material decreases
		inventory.CreateInventoryTrLine(workInProgressAcc, steelItem, steelNeeded, "kg", steelPrice, "USD"), // wip increases
	})
	if err != nil {
		log.Fatal(err)
	}

	widgetCost := steelNeeded * steelPrice / targetWidgetProduction // 100kg steel makes 50 widgets at 5 USD/kg
	err = inventory.ApplyTransaction(db, "Complete Widgets", time.Date(2025, 9, 5, 0, 0, 0, 0, time.Local), []inventory.TransactionLine{
		inventory.CreateInventoryTrLine(workInProgressAcc, steelItem, -steelNeeded, "kg", steelPrice, "USD"),            // wip decreases
		inventory.CreateInventoryTrLine(finishedProductAcc, steelItem, targetWidgetProduction, "kg", widgetCost, "USD"), // Finished Goods increases
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("update market price")

	// Market prices
	err = inventory.UpdateMarketPrice(db, steelItem.UUID[:], 6, "USD", "kg") // steel now 6 USD/kg
	if err != nil {
		log.Fatal(err)
	}

	// Market prices
	err = inventory.UpdateMarketPrice(db, steelItem.UUID[:], 6, "USD", "kg") // steel now 6 USD/kg
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
