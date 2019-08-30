package txlevel

import "github.com/jinzhu/gorm"

type transactionLevel struct {
	TransactionIsolation string `json:"transaction_isolation"`
}

var (
	RepeatableRead = "repeatable read"
	Serializable   = "serializable"
)

// Repeatable return true if isolation level is repeatable read or serializable
func Repeatable(tx *gorm.DB) (bool, error) {
	var level transactionLevel
	err := tx.Raw("SHOW TRANSACTION ISOLATION LEVEL").Scan(&level).Error
	if err != nil {
		return false, err
	}
	switch level.TransactionIsolation {
	case RepeatableRead:
		fallthrough
	case Serializable:
		return true, nil
	default:
		return false, nil
	}
}
