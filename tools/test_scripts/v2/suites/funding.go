package suites

// import (
// 	"fmt"

// 	"github.com/alpacahq/apex"
// 	"github.com/alpacahq/gobroker/models"
// 	"github.com/alpacahq/gobroker/service/relationship"
// 	"github.com/alpacahq/gobroker/service/transfer"
// 	"github.com/alpacahq/gopaca/db"
// 	"github.com/gofrs/uuid"
// 	"github.com/shopspring/decimal"
// )

// type FundingLog struct {
// 	Relationships        []string // 10
// 	Deposits             []string // 10
// 	Withdrawals          []string // 10
// 	CanceledRelationship string
// 	CanceledTransfer     string
// 	ReturnTransfer       string
// 	NOCTransfer          string
// }

// func CreateRelationship(srv relationship.RelationshipService, acctID uuid.UUID) (*models.ACHRelationship, error) {
// 	bInfo := relationship.BankAcctInfo{
// 		Token:       "token",
// 		Item:        "item",
// 		Account:     "021000021",
// 		Institution: "bofa",
// 		BankAccount: "021000021",
// 		Routing:     "091000019",
// 		AccountType: "CHECKING",
// 		Nickname:    "my favorite checking account",
// 	}
// 	return srv.Create(acctID, bInfo)
// }

// func CreateRelationships(accountIDs []uuid.UUID) ([]string, error) {
// 	fmt.Print("\nCreating 10 ACH relationships... ")
// 	relIDs := []string{}
// 	for i := 0; i < 10; i++ {
// 		tx := db.Begin()
// 		srv := relationship.Service().WithTx(tx)

// 		rel, err := CreateRelationship(srv, accountIDs[i])
// 		if err != nil {
// 			tx.Rollback()
// 			return nil, err
// 		}
// 		relIDs = append(relIDs, rel.ID)
// 	}
// 	fmt.Print("Done.")
// 	return relIDs, nil
// }

// func CancelRelationship(accountID uuid.UUID, relID string) (err error) {
// 	fmt.Print("\nCanceling an ACH relationship... ")
// 	tx := db.Begin()
// 	srv := relationship.Service().WithTx(tx)

// 	if err = srv.Cancel(accountID, relID); err != nil {
// 		tx.Rollback()
// 	} else {
// 		tx.Commit()
// 	}
// 	fmt.Print("Done.")
// 	return
// }

// func MakeDeposits(acctIDs []uuid.UUID, relIDs []string) ([]string, error) {
// 	fmt.Print("\nMaking 10 deposits... ")
// 	transferIDs := []string{}
// 	for i := 0; i < 10; i++ {
// 		tx := db.Begin()
// 		srv := transfer.Service().WithTx(tx)

// 		transfer, err := srv.Create(
// 			acctIDs[i],
// 			relIDs[i],
// 			apex.Incoming,
// 			decimal.NewFromFloat(1000),
// 		)
// 		if err != nil {
// 			tx.Rollback()
// 			return nil, err
// 		}
// 		if err := apex.Client().SimulateTransferApproval(*transfer.ApexID); err != nil {
// 			tx.Rollback()
// 			return nil, err
// 		}

// 		tx.Commit()
// 		transferIDs = append(transferIDs, transfer.ID)
// 	}
// 	fmt.Print("Done.")
// 	return transferIDs, nil
// }

// func MakeWithdrawals(acctIDs []uuid.UUID, relIDs []string) ([]string, error) {
// 	fmt.Print("\nMaking 10 withdrawals... ")
// 	transferIDs := []string{}
// 	for i := 0; i < 10; i++ {
// 		tx := db.Begin()
// 		srv := transfer.Service().WithTx(tx)

// 		transfer, err := srv.Create(
// 			acctIDs[i],
// 			relIDs[i],
// 			apex.Outgoing,
// 			decimal.NewFromFloat(100),
// 		)
// 		if err != nil {
// 			tx.Rollback()
// 			return nil, err
// 		}
// 		if err := apex.Client().SimulateTransferApproval(*transfer.ApexID); err != nil {
// 			tx.Rollback()
// 			return nil, err
// 		}
// 		tx.Commit()
// 		transferIDs = append(transferIDs, transfer.ID)
// 	}
// 	fmt.Print("Done.")
// 	return transferIDs, nil
// }

// func CancelTransfer(acctID uuid.UUID, relID string) (transferID *string, err error) {
// 	fmt.Print("\nCanceling a transfer... ")
// 	tx := db.Begin()
// 	srv := transfer.Service().WithTx(tx)

// 	transfer, err := srv.Create(
// 		acctID,
// 		relID,
// 		apex.Outgoing,
// 		decimal.NewFromFloat(100),
// 	)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if err = srv.Cancel(acctID, transfer.ID); err != nil {
// 		tx.Rollback()
// 	} else {
// 		tx.Commit()
// 	}
// 	transferID = &transfer.ID
// 	fmt.Print("Done.")
// 	return
// }

// func SimReturn(transferID string) (err error) {
// 	fmt.Print("\nSimulating an ACH return... ")

// 	err = apex.Client().SimulateAchReturn(transferID, false)
// 	if err != nil {
// 		return err
// 	}

// 	fmt.Print("Done.")
// 	return nil
// }

// func SimNOC(transferID string) (err error) {
// 	fmt.Print("\nSimulating an ACH NOC... ")
// 	nocReq := apex.NOCRequest{NewBankAccountType: "SAVINGS"}

// 	err = apex.Client().SimulateAchNOC(transferID, nocReq)
// 	if err != nil {
// 		return err
// 	}

// 	fmt.Print("Done.")
// 	return nil
// }
