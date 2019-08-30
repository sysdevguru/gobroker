package files

import (
	"reflect"
	"strings"
	"time"

	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/log"
	"github.com/lib/pq"
)

type SoDEasyToBorrow struct {
	Symbols []string `json:"symbols" gorm:"type:varchar(10)[]"`
}

type EasyToBorrowReport struct {
	list SoDEasyToBorrow
}

func (e2b *EasyToBorrowReport) ExtCode() string {
	return "EXT001"
}

func (e2b *EasyToBorrowReport) Delimiter() string {
	return ""
}

func (e2b *EasyToBorrowReport) Header() bool {
	return false
}

func (e2b *EasyToBorrowReport) Extension() string {
	return "txt"
}

func (e2b *EasyToBorrowReport) Value() reflect.Value {
	return reflect.ValueOf(e2b.list.Symbols)
}

func (e2b *EasyToBorrowReport) Append(v interface{}) {
	syms := append([]string(e2b.list.Symbols), v.(string))
	e2b.list.Symbols = pq.StringArray(syms)
}

func (e2b *EasyToBorrowReport) Sync(asOf time.Time) (uint, uint) {
	assets := e2b.gatherAssets()

	tx := db.Begin()

	for _, symbol := range e2b.list.Symbols {
		if asset, ok := assets[strings.TrimSpace(symbol)]; ok {
			asset.Shortable = true
			if err := tx.Save(&asset).Error; err != nil {
				tx.Rollback()
				log.Panic(
					"start of day database error",
					"file", e2b.ExtCode(),
					"error", err)
			}
		} else {
			// we don't know about this symbol, let's warn
			log.Warn("unknown symbol on the easy to borrow list", "symbol", symbol)
		}
	}

	if err := tx.Commit().Error; err != nil {
		log.Panic(
			"start of day database error",
			"file", e2b.ExtCode(),
			"error", err)
	}

	return uint(len(e2b.list.Symbols)), 0
}

func (e2b *EasyToBorrowReport) gatherAssets() map[string]models.Asset {
	assets := []models.Asset{}
	m := map[string]models.Asset{}

	if err := db.DB().Where("class = ?", enum.AssetClassUSEquity).Find(&assets).Error; err != nil {
		log.Panic("failed to gather assets", "file", e2b.ExtCode(), "error", err)
	}

	for _, asset := range assets {
		m[models.ApexFormat(asset.Symbol)] = asset
	}

	return m
}
