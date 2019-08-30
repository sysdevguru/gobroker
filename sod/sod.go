package sod

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"path"
	"path/filepath"
	"runtime"
	"time"

	"github.com/alpacahq/gobroker/external/slack"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/s3man"
	"github.com/alpacahq/gobroker/sod/files"
	"github.com/alpacahq/gobroker/utils"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/env"
	"github.com/alpacahq/gopaca/log"
	"github.com/pkg/errors"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

const (
	branch     = "APCA"
	dateFormat = "20060102"
)

type SoDProcessor struct {
	client *sftp.Client
	rsaKey ssh.Signer
	conn   ssh.Conn
}

func (sp *SoDProcessor) loadRSA(file string) error {
	_, f, _, _ := runtime.Caller(0)
	dir, err := filepath.Abs(filepath.Dir(f))
	buf, err := ioutil.ReadFile(dir + "/keys/" + file)
	if err != nil {
		return err
	}
	key, err := ssh.ParsePrivateKey(buf)
	if err != nil {
		return err
	}
	sp.rsaKey = key
	return nil
}

func (sp *SoDProcessor) InitClient() (err error) {
	if sp.client == nil {
		addr := "files.apexclearing.com:22"
		err = sp.loadRSA(env.GetVar("APEX_RSA"))
		if err != nil {
			log.Error("sod file pull error", "action", "load rsa key", "error", err)
			return err
		}
		config := &ssh.ClientConfig{
			User: env.GetVar("APEX_SFTP_USER"),
			Auth: []ssh.AuthMethod{ssh.PublicKeys(sp.rsaKey)},
			HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
				return nil
			},
		}
		conn, err := ssh.Dial("tcp", addr, config)
		if err != nil {
			log.Error("sod file pull error", "action", "connect sftp", "error", err)
			return err
		}
		sp.conn = conn
		sp.client, err = sftp.NewClient(conn)
		if err != nil {
			sp.conn.Close()
			log.Error("sod file pull error", "action", "connect sftp", "error", err)
			return err
		}
	}
	return nil
}

func (sp *SoDProcessor) Close() error {
	if sp.conn != nil {
		return sp.conn.Close()
	}
	return nil
}

func (sp *SoDProcessor) Pull(asof time.Time, securitiesOnly bool, retries uint) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Warn("sod processing recovered from panic", "retries", retries, "wait", time.Minute)
			<-time.After(time.Minute)
			if retries > 0 {
				sp.client.Close()
				sp.client = nil
				err = sp.Pull(asof, securitiesOnly, retries-1)
			} else {
				log.Error("failed to pull sod files", "attempts", retries)
				return
			}
		}
	}()

	if utils.Dev() && !securitiesOnly {
		return nil
	}

	if err := sp.InitClient(); err != nil {
		return err
	}
	defer sp.Close()

	var sodFiles []files.SODFile

	log.Info("pulling sod files", "asOf", asof.Format("2006-01-02"))

	if securitiesOnly {
		sodFiles = []files.SODFile{&files.SecurityMaster{}}
	} else {
		// WARNING: do not modify the order of this array
		// unless you know what you're doing
		sodFiles = []files.SODFile{
			&files.AccountMaster{},
			&files.AmountAvailableDetailReport{},
			&files.BuyingPowerDetailReport{},
			&files.BuyingPowerSummaryReport{},
			&files.DividendReport{},
			// cash activity need to wait dividend reports to confirm activity related to them.
			&files.CashActivityReport{},
			&files.ElectronicCommPrefReport{},
			&files.MarginCallReport{},
			&files.ReturnedMailReport{},
			&files.MandatoryActionReport{},
			&files.SecurityMaster{},
			&files.EasyToBorrowReport{},
			&files.PositionReport{},
			// &files.SecurityOverrideReport{},
			// &files.StockActivityReport{},
			&files.TradeActivityReport{},
			// &files.TradesMovedToErrorReport{},
			// &files.VoluntaryActionReport{},
		}
	}

	if err := sp.ProcessFiles(asof, sodFiles); err != nil {
		return err
	}

	return sp.notifySlack(asof)
}

func (sp *SoDProcessor) ProcessFiles(asof time.Time, sodFiles []files.SODFile) (err error) {
	date := asof.Format(dateFormat)

	for _, file := range sodFiles {

		dirName := fmt.Sprintf(
			"/download/%v/%v/",
			date,
			file.ExtCode(),
		)

		data, _ := sp.DownloadDir(dirName)

		if data == nil || len(data) == 0 {
			log.Warn("sod file is empty", "file", file.ExtCode())
			continue
		}

		log.Info("downloaded sod file", "file", file.ExtCode())

		if err = sp.BackupFile(asof, file, data); err != nil {
			// We can manually upload it later, so not skip processing.
			log.Error("sod backup failure", "error", err)
		}

		start := clock.Now()
		if err = files.Parse(data, file); err != nil {
			return err
		}

		log.Info(
			"parsed sod file",
			"file", file.ExtCode(),
			"elapsed", clock.Now().Sub(start),
		)

		log.Info("syncing sod file", "file", file.ExtCode())

		start = clock.Now()
		processed, errors := file.Sync(asof)

		log.Info("synced sod file", "file", file.ExtCode())

		if err := sp.storeMetric(
			file.ExtCode(),
			asof,
			clock.Now().Sub(start),
			processed,
			errors); err != nil {
			log.Panic(
				"failed to store sod file metrics",
				"file", file.ExtCode(),
				"error", err)
		}
	}
	return
}

// BackupFile to s3 and egnyte (if needed)
func (sp *SoDProcessor) BackupFile(asof time.Time, file files.SODFile, data []byte) (err error) {
	s3 := s3man.New()

	// books & records
	pushBooksAndRecords := func(category string) error {
		s3Path := path.Join(
			"books_and_records",
			category,
			asof.Format(dateFormat),
			file.ExtCode(),
			"APCA.csv",
		)

		if err := s3.Upload(bytes.NewReader(data), s3Path); err != nil {
			return errors.Wrapf(err, "failed to upload %v to S3 (books & records)", file.ExtCode())
		}

		return nil
	}

	// standard backup
	pushBackup := func() error {
		s3Path := fmt.Sprintf(
			"/apex/download/%v/%v/APCA.csv",
			asof.Format(dateFormat),
			file.ExtCode(),
		)

		if err := s3.Upload(bytes.NewReader(data), s3Path); err != nil {
			// We can manually upload it later, so not skip processing.
			return errors.Wrapf(err, "failed to upload %v to S3", file.ExtCode())
		}

		return nil
	}

	switch file.ExtCode() {
	case (&files.CashActivityReport{}).ExtCode():
		err = pushBooksAndRecords("money_movements")
	case (&files.TradeActivityReport{}).ExtCode():
		err = pushBooksAndRecords("trades")
	}

	if err != nil {
		return
	}

	return pushBackup()
}

func (sp *SoDProcessor) DownloadDir(dir string) ([]byte, error) {
	fileInfos, err := sp.client.ReadDir(dir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read dir info")
	}
	b := bytes.Buffer{}
	w := bufio.NewWriter(&b)
	// should only be one
	for _, fileInfo := range fileInfos {
		file, err := sp.client.Open(dir + fileInfo.Name())
		if err != nil {
			return nil, errors.Wrap(err, "failed to open sftp file")
		}
		defer file.Close()
		_, err = file.WriteTo(w)
		if err != nil {
			return nil, errors.Wrap(err, "failed to write content to buffer")
		}
		err = w.Flush()
		if err != nil {
			return nil, errors.Wrap(err, "failed to flush the buffer")
		}
		break
	}
	return b.Bytes(), nil
}

func (sp *SoDProcessor) DownloadFile(fileName string) ([]byte, error) {
	file, err := sp.client.Open(fileName)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	b := bytes.Buffer{}
	w := bufio.NewWriter(&b)

	if _, err = file.WriteTo(w); err != nil {
		return nil, err
	}

	if err = w.Flush(); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func (sp *SoDProcessor) storeMetric(code string, asOf time.Time, elapsed time.Duration, processed, errors uint) error {
	if errors > 0 {
		log.Error(
			"synced sod file",
			"succeeded", processed,
			"errors", errors,
			"file", code,
			"elapsed", elapsed)
	} else {
		log.Info(
			"synced sod file",
			"succeeded", processed,
			"errors", errors,
			"file", code,
			"elapsed", elapsed)
	}

	metric := &models.BatchMetric{
		ProcessDate:     asOf.Format("2006-01-02"),
		FileCode:        code,
		ProcessDuration: int(elapsed.Seconds()),
		RecordCount:     processed,
		ErrorCount:      errors,
	}
	tx := db.Begin()
	if err := tx.Where(
		&models.BatchMetric{
			ProcessDate: metric.ProcessDate,
			FileCode:    metric.FileCode}).
		Attrs(metric).
		FirstOrCreate(&metric).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (sp *SoDProcessor) notifySlack(asOf time.Time) error {
	metrics := []models.BatchMetric{}
	db.DB().Where("process_date = ?", asOf.Format("2006-01-02")).Find(&metrics)

	msg := slack.NewBatchStatus()
	msg.SetBody(metrics)
	slack.Notify(msg)

	return nil
}
