package suites

// import (
// 	"fmt"
// 	"math/rand"
// 	"time"

// 	"github.com/alpacahq/gobroker/gbreg"

// 	"github.com/alpacahq/gobroker/models"
// 	"github.com/alpacahq/gobroker/models/enum"
// 	"github.com/alpacahq/gobroker/workers/braggart"
// 	"github.com/alpacahq/gopaca/clock"
// 	"github.com/alpacahq/gopaca/db"
// 	"github.com/gofrs/uuid"
// 	"github.com/shopspring/decimal"
// )

// var buys = map[string]decimal.Decimal{
// 	"IBM":  decimal.NewFromFloat(154.32).Round(2),
// 	"AAPL": decimal.NewFromFloat(173.14).Round(2),
// 	"TSLA": decimal.NewFromFloat(299.96).Round(2),
// 	"X":    decimal.NewFromFloat(36.29).Round(2),
// }

// var sells = map[string]decimal.Decimal{
// 	"IBM":  decimal.NewFromFloat(156.21).Round(2),
// 	"AAPL": decimal.NewFromFloat(172.02).Round(2),
// 	"TSLA": decimal.NewFromFloat(305.91).Round(2),
// 	"X":    decimal.NewFromFloat(37.45).Round(2),
// }

// func PostBuys(acctIDs []uuid.UUID) error {
// 	executions := []models.Execution{}
// 	srv := gbreg.Services.Account().WithTx(db.DB())

// 	i := 0

// 	for symbol, price := range buys {
// 		sym := symbol
// 		px := price
// 		now := clock.Now()
// 		qty := decimal.NewFromFloat(float64(rand.Intn(100) + 1))

// 		acct, _ := srv.GetByID(acctIDs[i])

// 		id, _ := uuid.NewV4()
// 		tx := db.Begin()
// 		e := models.Execution{
// 			Account:         *acct.ApexAccount,
// 			Side:            enum.Buy,
// 			Qty:             &qty,
// 			Price:           &px,
// 			TransactionTime: now,
// 			Symbol:          sym,
// 			OrderID:         id.String(),
// 		}
// 		if err := tx.Create(&e).Commit().Error; err != nil {
// 			tx.Rollback()
// 			return err
// 		}
// 		executions = append(executions, e)
// 		i++
// 	}
// 	braggart.PostExecutions(executions)
// 	return nil
// }

// func PostSells(acctIDs []uuid.UUID) error {
// 	executions := []models.Execution{}
// 	srv := gbreg.Services.Account().WithTx(db.DB())

// 	i := 0

// 	for symbol, price := range sells {
// 		sym := symbol
// 		px := price
// 		now := clock.Now()
// 		qty := decimal.NewFromFloat(float64(rand.Intn(100) + 1))

// 		acct, _ := srv.GetByID(acctIDs[i])

// 		id, _ := uuid.NewV4()

// 		tx := db.Begin()
// 		e := models.Execution{
// 			Account:         *acct.ApexAccount,
// 			Side:            enum.Sell,
// 			Qty:             &qty,
// 			Price:           &px,
// 			TransactionTime: now,
// 			Symbol:          sym,
// 			OrderID:         id.String(),
// 		}
// 		if err := tx.Create(&e).Commit().Error; err != nil {
// 			tx.Rollback()
// 			return err
// 		}
// 		executions = append(executions, e)
// 		i++
// 	}
// 	braggart.PostExecutions(executions)
// 	return nil
// }

// func PostFailure() {

// }

// func VerifyOrdersPosted(acctIDs, orderIDs []uuid.UUID) error {
// 	postedOrders := []uuid.UUID{}
// 	start := clock.Now()
// 	for {
// 		if time.Since(start) > time.Minute {
// 			return fmt.Errorf("order post verification timed out [%v/%v]", len(postedOrders), len(orderIDs))
// 		}
// 		for i, orderID := range orderIDs {
// 			service := gbreg.Services.Order().WithTx(db.DB())

// 			order, err := service.GetByID(acctIDs[i], orderID)
// 			if err != nil {
// 				return err
// 			}

// 			db.DB().Model(order).Related(&order.Executions, "Executions")

// 			if len(order.Executions) == 0 {
// 				break
// 			}

// 			if order.Executions[0].BraggartTimestamp != nil {
// 				postedOrders = append(postedOrders, order.IDAsUUID())
// 			}
// 		}
// 		if len(postedOrders) == len(orderIDs) {
// 			return nil
// 		}
// 	}
// }
