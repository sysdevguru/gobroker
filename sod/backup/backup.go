package backup

import (
	"bytes"
	"fmt"

	"github.com/pkg/errors"

	"github.com/alpacahq/gobroker/s3man"
	"github.com/alpacahq/gobroker/sod"
	"github.com/alpacahq/gobroker/sod/files"
	"github.com/alpacahq/gobroker/utils/backfiller"
	"github.com/alpacahq/gobroker/utils/tradingdate"
	"github.com/alpacahq/gopaca/log"
)

var Files = []files.SODFile{
	&files.AccountMaster{},
	&files.AmountAvailableDetailReport{},
	&files.BuyingPowerDetailReport{},
	&files.BuyingPowerSummaryReport{},
	&files.CashActivityReport{},
	&files.DividendReport{},
	&files.ElectronicCommPrefReport{},
	&files.EasyToBorrowReport{},
	&files.MarginCallReport{},
	&files.ReturnedMailReport{},
	&files.MandatoryActionReport{},
	&files.SecurityMaster{},
	&files.PositionReport{},
	&files.SecurityOverrideReport{},
	// &files.StockActivityReport{},  // never produced by apex
	&files.TradeActivityReport{},
	&files.TradesMovedToErrorReport{},
	&files.VoluntaryActionReport{},
}

func Load(f files.SODFile, date tradingdate.TradingDate) error {
	m := s3man.New()

	s3Path := fmt.Sprintf("/apex/download/%v/%v/APCA.csv", date.MarketOpen().Format("20060102"), f.ExtCode())
	ok, err := m.Exists(s3Path)
	if !ok {
		return errors.New("backup file not found")
	}

	if err != nil {
		return errors.Wrap(err, "failed to check s3 file")
	}

	buf, err := m.DownloadInMemory(s3Path)

	if err != nil {
		return errors.Wrap(err, "failed to download file from s3")
	}

	return files.Parse(buf, f)
}

// Backup sync SoD files to S3
func Backup(frm, to tradingdate.TradingDate) {
	m := s3man.New()

	sp := sod.SoDProcessor{}
	sp.InitClient()
	defer sp.Close()

	b := backfiller.NewWithTradingDate(frm, to)

	for b.Next() {

		date := b.Value()

		for _, f := range Files {
			s3Path := fmt.Sprintf("/apex/download/%v/%v/APCA.csv", date.MarketOpen().Format("20060102"), f.ExtCode())
			ok, err := m.Exists(s3Path)
			if ok {
				continue
			}

			if err != nil {
				log.Info("failed to check existence of file in s3", "file", f.ExtCode(), "error", err)
				continue
			}

			dirname := fmt.Sprintf(
				"/download/%v/%v/",
				date.MarketOpen().Format("20060102"),
				f.ExtCode(),
			)

			buf, err := sp.DownloadDir(dirname)
			if err != nil {
				log.Error("failed to download file to s3", "file", f.ExtCode(), "error", err)
				continue
			}

			r := bytes.NewReader(buf)

			if err := m.Upload(r, s3Path); err != nil {
				log.Error("failed to upload file to s3", "file", f.ExtCode(), "error", err)
			}
		}
	}
}
