package testop

import (
	"github.com/jinzhu/gorm"
	gormigrate "gopkg.in/gormigrate.v1"
)

type ApiKey struct {
	AccountID string `sql:"type:uuid;"`
	KeyID     string `gorm:"primary_key;type:text"`
	SecretKey string `gorm:"type:text;"`
}

func Migration(db *gorm.DB) *gormigrate.Gormigrate {
	return gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "2018032301",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.AutoMigrate(&ApiKey{}).Error; err != nil {
					return err
				}
				return nil
			},
		},
	})
}
