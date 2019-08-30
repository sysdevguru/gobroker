package op

import (
	"time"

	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/jinzhu/gorm"
)

// GetDayBraggartExecutions returns list of braggart reported executions for that day, for that account.
func GetDayBraggartExecutions(tx *gorm.DB, apexAccount string, t time.Time) ([]models.Execution, error) {
	begin := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	end := begin.Add(24 * time.Hour)

	var executions []models.Execution
	q := tx.
		Where(
			"account = ? AND type IN (?) AND braggart_timestamp >= ? AND braggart_timestamp < ?",
			apexAccount,
			[]enum.ExecutionType{enum.ExecutionFill, enum.ExecutionPartialFill},
			begin.Format(time.RFC3339),
			end.Format(time.RFC3339)).
		Order("braggart_timestamp asc").
		Find(&executions)
	if q.Error != nil {
		return nil, q.Error
	}
	return executions, nil
}
