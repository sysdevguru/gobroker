package gc

import (
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/log"
)

func Work() {
	tx := db.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err := tx.Where("expire_at < ?", clock.Now()).Delete(&models.EmailVerificationCode{}).Error; err != nil {
		log.Error("failed to clean up email_verification codes", "error", err)
		tx.Rollback()
		return
	}

	tx.Commit()
}
