package op

import (
	"fmt"

	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/models"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
)

func GetOrderByClientOrderID(tx *gorm.DB, apexAccount string, clientOrderID string) (*models.Order, error) {
	var order models.Order
	if err := tx.Where(
		"client_order_id = ? AND account = ?",
		clientOrderID, apexAccount).Find(&order).Error; err != nil {
		switch {
		case gorm.IsRecordNotFoundError(err):
			return nil, gberrors.NotFound.WithMsg(fmt.Sprintf("order not found for %v", clientOrderID))
		default:
			return nil, err
		}
	}
	return &order, nil
}

func GetOrderByID(tx *gorm.DB, apexAccount string, orderID uuid.UUID) (*models.Order, error) {
	var order models.Order
	if err := tx.Where(
		"id = ? AND account = ?",
		orderID, apexAccount).Find(&order).Error; err != nil {
		switch {
		case gorm.IsRecordNotFoundError(err):
			return nil, gberrors.NotFound.WithMsg(fmt.Sprintf("order not found for %v", orderID))
		default:
			return nil, err
		}
	}
	return &order, nil
}
