package trading

import (
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"

	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/service/assetcache"
	"github.com/alpacahq/gopaca/log"
	"github.com/jinzhu/gorm"
)

// ProcessExecution handles an inbound execution update from GoTrader,
// and creates/closes positions as required for the specific execution
func ProcessExecution(tx *gorm.DB, acct *models.TradeAccount, exec *models.Execution) (err error) {
	switch exec.Type {
	case enum.ExecutionPartialFill:
		fallthrough
	case enum.ExecutionFill:
		err = handleFill(tx, acct, exec)
	}

	return err
}

// handleFill handles fill and partial fill executions
func handleFill(tx *gorm.DB, acct *models.TradeAccount, exec *models.Execution) (err error) {
	switch exec.Side {
	case enum.Buy:
		return handleBuyFill(tx, acct, exec)
	case enum.Sell:
		return handleSellFill(tx, acct, exec)
	default:
		tx.Rollback()
		log.Panic("invalid execution side", "side", exec.Side)
	}
	return
}

// handleBuyFill handles an incoming buy fill execution and will
// create a new position in the DB
func handleBuyFill(tx *gorm.DB, acct *models.TradeAccount, exec *models.Execution) error {
	asset := assetcache.Get(exec.Symbol)
	if asset == nil {
		return fmt.Errorf("could not find asset for \"%s\"", exec.Symbol)
	}

	assetID, _ := uuid.FromString(asset.ID)

	return tx.Create(&models.Position{
		AccountID:      acct.ID,
		EntryOrderID:   exec.OrderID,
		Status:         models.Open,
		Side:           models.Long,
		AssetID:        assetID,
		Qty:            *exec.Qty,
		EntryPrice:     *exec.Price,
		EntryTimestamp: exec.TransactionTime,
	}).Error
}

// handleSellFill handles an incoming sell fill execution and closes/splits
// positions as required by the specific execution
func handleSellFill(tx *gorm.DB, acct *models.TradeAccount, exec *models.Execution) (err error) {
	qtyToProcess := *exec.Qty

	positions := []models.Position{}
	asset := assetcache.Get(exec.Symbol)

	if asset == nil {
		return fmt.Errorf("could not find asset for \"%s\"", exec.Symbol)
	}

	if err = tx.Where(
		"account_id = ? AND asset_id = ? AND side = ? AND status = ?",
		acct.ID,
		asset.ID,
		models.Long,
		models.Open,
	).Order("created_at").Find(&positions).Error; err != nil {
		return
	}

	if len(positions) == 0 {
		log.Error(
			"received sell fill with no positions",
			"symbol", asset.Symbol,
			"account", acct.ID,
			"qty", exec.Qty.String(),
		)
		return
	}

	for _, position := range positions {
		switch {
		// no more shares to process
		case qtyToProcess.Equal(decimal.Zero):
			return
		// the quantity will completely close this position
		case qtyToProcess.GreaterThanOrEqual(position.Qty):
			position.ExitOrderID = &exec.OrderID
			position.ExitPrice = exec.AvgPrice
			position.ExitTimestamp = &exec.TransactionTime
			position.Status = models.Closed

			qtyToProcess = qtyToProcess.Sub(position.Qty)

			if err = tx.Save(&position).Error; err != nil {
				return
			}
		// the quantity is smaller than this position, and
		// so it requires a split
		default:
			if _, _, err = splitPosition(tx, &position, qtyToProcess, exec); err != nil {
				return err
			}
			qtyToProcess = decimal.Zero
		}
	}

	return
}

// splitPosition splits the position as needed when partial fills or
// sell orders for less than the size of the position are processed.
// The position is split into a closed position (for the quantity)
// specified, and a remaining open position. The previous position
// is marked with status = SPLIT.
func splitPosition(
	tx *gorm.DB,
	position *models.Position,
	qty decimal.Decimal,
	exec *models.Execution) (*models.Position, *models.Position, error) {

	if position.Qty.LessThanOrEqual(qty) {
		return nil, nil, fmt.Errorf("not enough qty for split (%v)", qty)
	}

	// the closing position
	p1 := &models.Position{
		AccountID:          position.AccountID,
		EntryOrderID:       position.EntryOrderID,
		OriginalPositionID: position.OriginalPositionID,
		AssetID:            position.AssetID,
		Side:               position.Side,
		Qty:                qty,
		EntryPrice:         position.EntryPrice,
		EntryTimestamp:     position.EntryTimestamp,
		Status:             models.Closed,
		ExitTimestamp:      &exec.TransactionTime,
		ExitOrderID:        &exec.OrderID,
		ExitPrice:          exec.AvgPrice,
	}

	// the split remaining position
	p2 := &models.Position{
		Status:             models.Open,
		AccountID:          position.AccountID,
		EntryOrderID:       position.EntryOrderID,
		OriginalPositionID: position.OriginalPositionID,
		AssetID:            position.AssetID,
		Side:               position.Side,
		Qty:                position.Qty.Sub(p1.Qty),
		EntryPrice:         position.EntryPrice,
		EntryTimestamp:     position.EntryTimestamp,
	}

	position.Status = models.Split

	// store it all in the DB
	err := tx.Save(&position).Create(&p1).Create(&p2).Error

	return p1, p2, err
}
