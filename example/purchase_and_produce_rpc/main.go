package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"inventory"
	"inventorypb"
	"inventoryrpc"
	"log"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var (
	// assetUUID, inventoryUUID, rawMaterialUUID, workInProgressUUID,
	// finishedProductUUID, cashUUID, equityUUID, expenseUUID, matPurchaseUUID, equipmentPurchaseUUID,
	// incomeUUID, financialIncomeUUID, nonFinancialIncomeUUID string
	// steelUUID, woodUUID, widget1UUID, widget2UUID string
	inventoryAcc, rawMaterialAcc, workInProgressAcc,
	finishedProductAcc, cashAcc, matPurchaseAcc, equipmentPurchaseAcc,
	nonFinancialIncomeAcc, incomingMatAcc uuid.UUID
	steelItem, woodItem, widget1Item, widget2Item uuid.UUID
)

var serverPktReceiver inventorypb.PacketReceiver
var serverProcessor inventorypb.ServerProcessor
var clientPktReceiver inventorypb.PacketReceiver
var clientProcessor ClientProcessor

type ClientProcessor struct {
	PktChan chan *inventoryrpc.Packet
}

func (p *ClientProcessor) ProcessPkt(pkt *inventoryrpc.Packet) (*inventoryrpc.Packet, string, int32, error) {
	p.PktChan <- pkt
	return nil, "", -1, nil
}

func (p *ClientProcessor) PostProcessPkt(responsePkt *inventoryrpc.Packet) error {
	return nil
}

var mainAccountUUIDBytes map[string][]byte
var mainAccountUUIDs map[string]uuid.UUID

func waitResponse(pktUUID uuid.UUID) (*inventoryrpc.Packet, error) {
	responsePkt := <-clientProcessor.PktChan
	responseCodeBytes, ok := responsePkt.Body["code"]
	if !ok {
		return nil, errors.New("packet incomplete")
	}
	responseCode := int32(binary.LittleEndian.Uint32(responseCodeBytes))

	if responsePkt.UUID == pktUUID && responseCode < 0 {
		return nil, errors.New(string(responsePkt.Body["message"]))
	}
	return responsePkt, nil
}

func sendReqAndWaitResponse(funcName string, params protoreflect.ProtoMessage) (*inventoryrpc.Packet, error) {
	pktUUID, pktBytes, err := inventorypb.CreateRequest("GetMainAccounts", params)
	if err != nil {
		return nil, err
	}
	serverPktReceiver.HandleIncoming(pktBytes)
	return waitResponse(pktUUID)
}

func sendGetMainAccountsReq() error {
	responsePkt, err := sendReqAndWaitResponse("GetMainAccounts", nil)
	if err != nil {
		return err
	}

	for k, v := range responsePkt.Body {
		if k == "code" || k == "message" {
			continue
		}
		accUUID, err := uuid.FromBytes(v)
		if err != nil {
			return err
		}
		mainAccountUUIDBytes[k] = v
		mainAccountUUIDs[k] = accUUID
	}
	return nil
}

func sendCreateAccountReq(name string, parentUUID []byte) ([]byte, uuid.UUID, error) {
	var parent *inventory.Account
	if parentUUID != nil {
		UUIDBytes, err := uuid.FromBytes(parentUUID)
		if err == nil {
			return nil, uuid.UUID{}, err
		}
		parent = &inventory.Account{
			UUID: UUIDBytes,
		}
	} else {
		parent = nil
	}

	responsePkt, err := sendReqAndWaitResponse("AddAccount", inventorypb.NewAccount(&inventory.Account{
		Name:   name,
		Parent: parent,
	}, nil))
	if err != nil {
		return nil, uuid.UUID{}, err
	}

	UUIDBytes, ok := responsePkt.Body["uuid"]
	if !ok {
		return nil, uuid.UUID{}, errors.New("packet incomplete")
	}

	UUID, err := uuid.FromBytes(UUIDBytes)
	return UUIDBytes, UUID, nil
}

func sendCreateItemReq(item *inventory.Item) ([]byte, uuid.UUID, error) {
	responsePkt, err := sendReqAndWaitResponse("AddItem", inventorypb.NewItem(item, nil))
	if err != nil {
		return nil, uuid.UUID{}, err
	}

	UUIDBytes, ok := responsePkt.Body["uuid"]
	if !ok {
		return nil, uuid.UUID{}, errors.New("packet incomplete")
	}

	UUID, err := uuid.FromBytes(UUIDBytes)
	return UUIDBytes, UUID, nil
}

func sendApplyTransactionReq(transaction *inventory.Transaction) ([]byte, uuid.UUID, error) {
	responsePkt, err := sendReqAndWaitResponse("ApplyTransaction", inventorypb.NewTransaction(*transaction))
	if err != nil {
		return nil, uuid.UUID{}, err
	}

	UUIDBytes, ok := responsePkt.Body["uuid"]
	if !ok {
		return nil, uuid.UUID{}, errors.New("packet incomplete")
	}

	UUID, err := uuid.FromBytes(UUIDBytes)
	return UUIDBytes, UUID, nil
}

func createAccountsAndItems() error {
	var err error

	var (
		nonFinancialIncomeAccBytes, inventoryAccBytes []byte
	)

	nonFinancialIncomeAccBytes, nonFinancialIncomeAcc, err = sendCreateAccountReq("non-financial income", nil)
	if err != nil {
		return err
	}

	_, incomingMatAcc, err = sendCreateAccountReq("incoming material", nonFinancialIncomeAccBytes)
	if err != nil {
		return err
	}

	inventoryAccBytes, inventoryAcc, err = sendCreateAccountReq("inventory", mainAccountUUIDBytes["asset"])
	if err != nil {
		return err
	}

	_, rawMaterialAcc, err = sendCreateAccountReq("raw material", inventoryAccBytes)
	if err != nil {
		return err
	}

	_, workInProgressAcc, err = sendCreateAccountReq("work in progress", inventoryAccBytes)
	if err != nil {
		return err
	}

	_, finishedProductAcc, err = sendCreateAccountReq("finished product", inventoryAccBytes)
	if err != nil {
		return err
	}

	_, cashAcc, err = sendCreateAccountReq("cash", mainAccountUUIDBytes["asset"])
	if err != nil {
		return err
	}

	_, matPurchaseAcc, err = sendCreateAccountReq("material purchase", mainAccountUUIDBytes["expense"])
	if err != nil {
		return err
	}

	_, equipmentPurchaseAcc, err = sendCreateAccountReq("equipment purchase", mainAccountUUIDBytes["expense"])
	if err != nil {
		return err
	}

	// Items
	_, steelItem, err = sendCreateItemReq(&inventory.Item{Name: "steel", Unit: "kg"})
	if err != nil {
		return err
	}

	_, woodItem, err = sendCreateItemReq(&inventory.Item{Name: "wood", Unit: "kg"})
	if err != nil {
		return err
	}

	_, widget1Item, err = sendCreateItemReq(&inventory.Item{Name: "widget 1", Unit: "pcs"})
	if err != nil {
		return err
	}

	_, widget2Item, err = sendCreateItemReq(&inventory.Item{Name: "widget 1", Unit: "pcs"})
	if err != nil {
		return err
	}

	return nil
}

func main() {
	serverPktReceiver.Processor = &serverProcessor
	serverProcessor.ConsumeProcessingResponseFuncs = append(serverProcessor.ConsumeProcessingResponseFuncs, func(pkt []byte) {
		clientPktReceiver.HandleIncoming(pkt)
	})
	clientPktReceiver.Processor = &clientProcessor

	var err error

	fmt.Println("init db")
	_, err = sendReqAndWaitResponse("OpenOrCreateDB", nil)
	if err != nil {
		log.Fatal(err)
	}
	err = sendGetMainAccountsReq()

	fmt.Println("create accounts and items")
	err = createAccountsAndItems()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("apply transactions")
	// Transaction: Owner invests 1000 USD equity → Cash
	_, _, err = sendApplyTransactionReq(&inventory.Transaction{
		Description: "Owner Investment",
		DatetimeMs:  time.Date(2025, 9, 1, 0, 0, 0, 0, time.Local).UnixMilli(),
		TransactionLines: []*inventory.TransactionLine{
			inventory.CreateFinancialTrLineWithUUID(mainAccountUUIDs["equity"], 0, 1000, "USD"), // suntik modal
			inventory.CreateFinancialTrLineWithUUID(cashAcc, 1000, 0, "USD"),                    // masuk cash
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	steelPrice := 5.0
	widgetSteelNeed := 2.0 // 2 kg steel per widget
	targetWidgetProduction := 10.0
	steelNeeded := widgetSteelNeed * targetWidgetProduction

	_, _, err = sendApplyTransactionReq(&inventory.Transaction{
		Description: "Purchase Steel 100kg",
		DatetimeMs:  time.Date(2025, 9, 2, 0, 0, 0, 0, time.Local).UnixMilli(),
		TransactionLines: []*inventory.TransactionLine{
			inventory.CreateInventoryTrLineWithUUID(incomingMatAcc, steelItem, -100, "kg", steelPrice, "USD"), // incoming material
			inventory.CreateInventoryTrLineWithUUID(rawMaterialAcc, steelItem, 100, "kg", steelPrice, "USD"),  // added to raw material inventory
			inventory.CreateFinancialTrLineWithUUID(cashAcc, 0, 500, "USD"),                                   // Cash decreases
			inventory.CreateFinancialTrLineWithUUID(matPurchaseAcc, 500, 0, "USD"),                            // Expense recognized
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	woodPrice := 3.0
	_, _, err = sendApplyTransactionReq(&inventory.Transaction{
		Description: "Purchase Wood 100kg",
		DatetimeMs:  time.Date(2025, 9, 3, 0, 0, 0, 0, time.Local).UnixMilli(),
		TransactionLines: []*inventory.TransactionLine{
			inventory.CreateInventoryTrLineWithUUID(incomingMatAcc, woodItem, -150, "kg", woodPrice, "USD"), // incoming material
			inventory.CreateInventoryTrLineWithUUID(rawMaterialAcc, woodItem, 150, "kg", woodPrice, "USD"),  // added to raw material inventory
			inventory.CreateFinancialTrLineWithUUID(cashAcc, 0, 300, "USD"),                                 // Cash decreases
			inventory.CreateFinancialTrLineWithUUID(matPurchaseAcc, 300, 0, "USD"),                          // Expense recognized
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	_, _, err = sendApplyTransactionReq(&inventory.Transaction{
		Description: "Use Steel to Manufacture Widgets",
		DatetimeMs:  time.Date(2025, 9, 4, 0, 0, 0, 0, time.Local).UnixMilli(),
		TransactionLines: []*inventory.TransactionLine{
			inventory.CreateInventoryTrLineWithUUID(rawMaterialAcc, steelItem, -steelNeeded, "kg", steelPrice, "USD"),   // raw material decreases
			inventory.CreateInventoryTrLineWithUUID(workInProgressAcc, steelItem, steelNeeded, "kg", steelPrice, "USD"), // wip increases
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	widgetCost := steelNeeded * steelPrice / targetWidgetProduction // 100kg steel makes 50 widgets at 5 USD/kg
	_, _, err = sendApplyTransactionReq(&inventory.Transaction{
		Description: "Complete Widgets",
		DatetimeMs:  time.Date(2025, 9, 5, 0, 0, 0, 0, time.Local).UnixMilli(),
		TransactionLines: []*inventory.TransactionLine{
			inventory.CreateInventoryTrLineWithUUID(workInProgressAcc, steelItem, -steelNeeded, "kg", steelPrice, "USD"),            // wip decreases
			inventory.CreateInventoryTrLineWithUUID(finishedProductAcc, steelItem, targetWidgetProduction, "kg", widgetCost, "USD"), // Finished Goods increases
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// todo: refactor these
	// fmt.Println("update market price")

	// // Market prices
	// err = inventory.UpdateMarketPrice(db, &inventory.MarketPrices{
	// 	Item: &inventory.Item{
	// 		UUID: steelItem.UUID,
	// 	},
	// 	Price:    6,
	// 	Currency: "USD",
	// 	Unit:     "kg",
	// }) // steel now 6 USD/kg
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// // Market prices
	// err = inventory.UpdateMarketPrice(db, &inventory.MarketPrices{
	// 	Item: &inventory.Item{
	// 		UUID: steelItem.UUID,
	// 	},
	// 	Price:    6,
	// 	Currency: "USD",
	// 	Unit:     "kg",
	// }) // steel now 6 USD/kg
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// fmt.Println("=== Historical Cost Balances (Leaf Accounts) ===")
	// err = inventory.PrintBalances(db)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// fmt.Println("\n=== Market Value Balances (Leaf Accounts) ===")
	// err = inventory.PrintMarketBalances(db)
	// if err != nil {
	// 	log.Fatal(err)
	// }
}
