package backup

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/alpacahq/apex"
	"github.com/alpacahq/gobroker/external/egnyte"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/s3man"
	"github.com/alpacahq/gobroker/service/account"
	"github.com/alpacahq/gobroker/service/assetcache"
	"github.com/alpacahq/gobroker/service/trustedcontact"
	"github.com/alpacahq/gopaca/calendar"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/encryption"
	"github.com/alpacahq/gopaca/env"
	"github.com/alpacahq/gopaca/log"
	"github.com/alpacahq/gopaca/pool"
	"github.com/cloudfoundry/bytefmt"
	"github.com/gocarina/gocsv"
	"github.com/mholt/archiver"
	"github.com/shopspring/decimal"
	"github.com/valyala/fasthttp"
	"gopkg.in/matryer/try.v1"
)

const (
	apexFormat     = "1/2/2006"
	dateFormat     = "2006-01-02"
	fileDateFormat = "20060102"
	monthFormat    = "200601"
	// pulling documents from Apex can be very slow
	timeout = time.Minute
)

type backupWorker struct {
	uploadS3     func(file io.ReadSeeker, path string) error
	downloadS3   func(local, remote string) error
	uploadEgnyte func(filePath string, data []byte) error
	getDocuments func(account string, start, end time.Time, docType apex.DocumentType) ([]apex.Document, error)
	getHTTP      func(dst []byte, url string, timeout time.Duration) (statusCode int, body []byte, err error)
	asOf         time.Time
	parallelism  int
}

func newWorker(asOf time.Time) *backupWorker {
	s3 := s3man.New()

	parallelism, err := strconv.ParseInt(env.GetVar("BACKUP_PARALLELISM"), 10, 32)
	if err != nil {
		parallelism = 1
	}

	return &backupWorker{
		uploadS3:     s3.Upload,
		downloadS3:   s3.DownloadDirectory,
		uploadEgnyte: egnyte.Upload,
		getDocuments: apex.Client().GetDocuments,
		getHTTP:      fasthttp.GetTimeout,
		asOf:         asOf,
		parallelism:  int(parallelism),
	}
}

// WorkDaily backs up the required records on a daily basis to S3
func WorkDaily(asOf time.Time, workers ...*backupWorker) {
	var worker *backupWorker

	if len(workers) > 0 {
		worker = workers[0]
	} else {
		worker = newWorker(asOf)
	}

	dailyJobs := []func(acct *models.Account){
		worker.backupAccount,
		worker.backupTrades,
	}

	worker.runJobs(dailyJobs)
}

// WorkWeekly backs up the required records on a weekly basis to S3
func WorkWeekly(asOf time.Time, workers ...*backupWorker) {
	var worker *backupWorker

	if len(workers) > 0 {
		worker = workers[0]
	} else {
		worker = newWorker(asOf)
	}

	weeklyJobs := []func(acct *models.Account){
		worker.backupConfirms,
	}

	worker.runJobs(weeklyJobs)
}

// WorkMonthly backs up the required records on a monthly basis to S3
func WorkMonthly(asOf time.Time, workers ...*backupWorker) {
	var worker *backupWorker

	if len(workers) > 0 {
		worker = workers[0]
	} else {
		worker = newWorker(asOf)
	}

	monthlyJobs := []func(acct *models.Account){
		worker.backupStatements,
	}

	worker.runJobs(monthlyJobs)
}

// Sync the books and records from S3 to Egnyte
func Sync(asOf time.Time, workers ...*backupWorker) {
	var (
		worker                   *backupWorker
		dirPath, filePath, s3Dir string
		zipName, zipDir          string
		fInfo                    os.FileInfo
		err                      error
		buf                      []byte
	)

	if len(workers) > 0 {
		worker = workers[0]
	} else {
		worker = newWorker(asOf)
	}

	if s3Dir, err = ioutil.TempDir("", "egnyte"); err != nil {
		log.Error("failed to create temp dir for S3 -> Egnyte sync", "error", err)
		goto Cleanup
	}

	if err = worker.downloadS3(s3Dir, "books_and_records"); err != nil {
		log.Error("failed to download books & records from S3", "error", err)
		goto Cleanup
	}

	if zipDir, err = ioutil.TempDir("", "zip"); err != nil {
		log.Error("failed to create temp dir for zip file", "error", err)
		goto Cleanup
	}

	zipName = fmt.Sprintf("%s/%s.zip", zipDir, asOf.Format(fileDateFormat))

	if err = archiver.Zip.Make(zipName, []string{s3Dir}); err != nil {
		log.Error("failed to make zip file for Egnyte", "error", err)
		goto Cleanup
	}

	if fInfo, err = os.Stat(zipName); err != nil {
		log.Error("failed to stat zip file", "error", err)
	} else {
		log.Info("created zip file", "size", bytefmt.ByteSize(uint64(fInfo.Size())))
	}

	if buf, err = ioutil.ReadFile(zipName); err != nil {
		log.Error("failed to read zip file into memory", "error", err)
		goto Cleanup
	}

	dirPath = "books_and_records"

	// if err = egnyte.CreateDirectory(dirPath); err != nil {
	// 	log.Error("failed to create directory in egnyte", "error", err)
	// 	goto Cleanup
	// }

	filePath = fmt.Sprintf("%s/%s", dirPath, filepath.Base(zipName))

	if err = worker.uploadEgnyte(filePath, buf); err != nil {
		log.Error("failed to upload zip file to egnyte", "error", err)
		goto Cleanup
	}

Cleanup:
	if s3Dir != "" {
		if err = os.RemoveAll(s3Dir); err != nil {
			log.Error("failed to clean up local S3 dir", "dir", s3Dir, "error", err)
		}
	}

	if zipDir != "" {
		if err = os.RemoveAll(zipDir); err != nil {
			log.Error("failed to clean up zip dir", "dir", zipDir, "error", err)
		}
	}

}

func (w *backupWorker) runJobs(jobs []func(acct *models.Account)) {
	srv := account.Service().WithTx(db.DB())

	accounts, _, err := srv.List(
		account.AccountQuery{
			Per: math.MaxInt32,
		},
	)

	if err != nil {
		log.Panic("failed to query accounts for backup", "error", err)
	}

	jobRunner := func(v interface{}) {
		acct := v.(*models.Account)
		for _, jobFunc := range jobs {
			jobFunc(acct)
		}
	}

	c := make(chan interface{}, w.parallelism)
	p := pool.NewPool(w.parallelism, jobRunner)

	go p.Work(c)

	for i, acct := range accounts {
		if acct.PrimaryOwner() != nil && acct.ApexAccount != nil {
			c <- &accounts[i]
		}
	}

	close(c)

	p.Wait()
}

func (w *backupWorker) backupAccount(acct *models.Account) {
	records, err := genAccountRecords(acct)
	if err != nil {
		log.Error(
			"failed to generate account records",
			"account", *acct.ApexAccount,
			"error", err)
		return
	}

	buf, err := gocsv.MarshalBytes(records)
	if err != nil {
		log.Error(
			"failed to marshal account records csv",
			"account", *acct.ApexAccount,
			"error", err)
		return
	}

	filePath := fmt.Sprintf("/books_and_records/accounts/%s.csv", *acct.ApexAccount)

	if err = w.uploadS3(bytes.NewReader(buf), filePath); err != nil {
		log.Error(
			"failed to upload account record csv to S3",
			"account", *acct.ApexAccount,
			"error", err)
	}
}

func (w *backupWorker) backupTrades(acct *models.Account) {
	start := w.asOf
	end := start.AddDate(0, 0, 1)

	executions := []models.Execution{}

	if err := db.DB().
		Where(
			"account = ? AND created_at >= ? AND created_at < ?",
			*acct.ApexAccount,
			start.Format(dateFormat),
			end.Format(dateFormat)).
		Find(&executions).Error; err != nil {

		log.Panic(
			"failed to query executions for S3 backup",
			"account", *acct.ApexAccount,
			"error", err)
	}

	for _, execution := range executions {
		order := &models.Order{}
		if err := db.DB().Where("id = ?", execution.OrderID).First(order).Error; err != nil {
			log.Panic(
				"failed to query execution order",
				"account", *acct.ApexAccount,
				"execution", execution.ID,
				"error", err)
		}

		tickets, err := genOrderTickets(acct, order)
		if err != nil {
			log.Error(
				"failed to generate order tickets",
				"account", *acct.ApexAccount,
				"order", order.ID,
				"error", err)
			continue
		}

		buf, err := gocsv.MarshalBytes(tickets)
		if err != nil {
			log.Error(
				"failed to marshal order tickets csv",
				"account", *acct.ApexAccount,
				"order", order.ID,
				"error", err)
			continue
		}

		filePath := fmt.Sprintf("/books_and_records/order_tickets/%s/%s.csv", *acct.ApexAccount, start.Format(dateFormat))

		if err = w.uploadS3(bytes.NewReader(buf), filePath); err != nil {
			log.Error(
				"failed to upload order ticket csv to S3",
				"account", *acct.ApexAccount,
				"order", order.ID,
				"error", err)
		}
	}
}

func (w *backupWorker) backupConfirms(acct *models.Account) {
	start := w.asOf.Truncate(calendar.Day).AddDate(0, 0, -7)
	end := w.asOf.Truncate(calendar.Day)

	docs, err := w.getDocuments(*acct.ApexAccount, start, end, apex.TradeConfirmation)
	if err != nil {
		log.Error(
			"failed to pull trade confirmation list",
			"account", *acct.ApexAccount,
			"start", start,
			"end", end,
			"error", err)
	}

	for _, doc := range docs {
		var (
			code int
			body []byte
		)

		if err = try.Do(func(attempt int) (bool, error) {
			code, body, err = w.getHTTP(nil, doc.URL, timeout)
			return err != nil, err
		}); err != nil || code > fasthttp.StatusMultipleChoices {
			log.Error(
				"failed to pull trade confirmation",
				"account", *acct.ApexAccount,
				"url", doc.URL,
				"status_code", code,
				"body", string(body),
			)
			continue
		}

		t, err := time.Parse(apexFormat, doc.Date)
		if err != nil {
			log.Error(
				"failed to parse trade confirmation date",
				"account", *acct.ApexAccount,
				"date", doc.Date,
				"error", err)
			continue
		}

		filePath := fmt.Sprintf("/books_and_records/trade_confirmations/%s/%s.pdf", *acct.ApexAccount, t.Format("200601"))

		if err = w.uploadS3(bytes.NewReader(body), filePath); err != nil {
			log.Error(
				"failed to upload trade confirmation pdf to S3",
				"date", doc.Date,
				"account", *acct.ApexAccount,
				"error", err)
		}
	}
}

func (w *backupWorker) backupStatements(acct *models.Account) {
	start := w.asOf.AddDate(0, -1, 0)
	end := w.asOf

	docs, err := w.getDocuments(*acct.ApexAccount, start, end, apex.AccountStatement)
	if err != nil {
		log.Error(
			"failed to pull statement list",
			"account", *acct.ApexAccount,
			"start", start,
			"end", end,
			"error", err)
	}

	for _, doc := range docs {
		var (
			code int
			body []byte
		)

		if err = try.Do(func(attempt int) (bool, error) {
			code, body, err = w.getHTTP(nil, doc.URL, timeout)
			return err != nil, err
		}); err != nil || code > fasthttp.StatusMultipleChoices {
			log.Error(
				"failed to pull statement",
				"account", *acct.ApexAccount,
				"url", doc.URL,
				"status_code", code,
				"body", string(body),
			)
			continue
		}

		t, err := time.Parse(apexFormat, doc.Date)
		if err != nil {
			log.Error(
				"failed to parse statement date",
				"account", *acct.ApexAccount,
				"date", doc.Date,
				"error", err)
			continue
		}

		filePath := fmt.Sprintf("books_and_records/monthly_statements/%s/%s.pdf", *acct.ApexAccount, t.Format("200601"))

		if err = w.uploadS3(bytes.NewReader(body), filePath); err != nil {
			log.Error(
				"failed to upload monthly statment pdf to S3",
				"date", doc.Date,
				"account", *acct.ApexAccount,
				"error", err)
		}
	}
}

// AccountRecord defines a record in the account file to be stored in S3
type AccountRecord struct {
	AlpacaAccount               string     `csv:"alpaca_account"`
	ApexAccount                 *string    `csv:"apex_account"`
	Plan                        string     `csv:"plan"`
	LegalName                   *string    `csv:"legal_name"`
	EmailAddress                *string    `csv:"email_address"`
	LegalAddress                *string    `csv:"legal_address"`
	TelephoneNumber             *string    `csv:"telephone_number"`
	SocialSecurityNumber        *string    `csv:"social_security_number"`
	DateOfBirth                 *string    `csv:"date_of_birth"`
	VisaType                    *string    `csv:"visa_type"`
	VisaExpiration              *string    `csv:"visa_expiration"`
	PermanentResident           *bool      `csv:"permanent_resident"`
	CountryOfBirth              *string    `csv:"country_of_birth"`
	TrustedContactName          *string    `csv:"trusted_contact_name"`
	TrustedContactPhoneNumber   *string    `csv:"trusted_contact_phone_number"`
	TrustedContactEmailAddress  *string    `csv:"trusted_contact_email_address"`
	IsAffiliatedExchangeOrFINRA *bool      `csv:"is_affiliated_exchange_or_finra"`
	AffiliatedFirm              *string    `csv:"affiliated_firm"`
	IsControlPerson             *bool      `csv:"is_control_person"`
	ControllingFirms            []string   `csv:"controlling_firms"`
	EmploymentStatus            *string    `csv:"employment_status"`
	Position                    *string    `csv:"position"`
	Employer                    *string    `csv:"employer"`
	EmployerAddress             *string    `csv:"employer_address"`
	OwnerID                     string     `csv:"owner_id"`
	MarginAgreementSignedAt     *time.Time `csv:"margin_agreement_signed_at"`
	AccountAgreementSignedAt    *time.Time `csv:"account_agreement_signed_at"`
	ApprovedBy                  *string    `csv:"approved_by"`
	ApprovedAt                  *time.Time `csv:"approved_at"`
}

func genAccountRecords(acct *models.Account) ([]AccountRecord, error) {
	details := []models.OwnerDetails{}

	if err := db.DB().
		Model(acct.PrimaryOwner()).
		Related(&details, "OwnerDetails").
		Order("created_at DESC").Error; err != nil {
		return nil, err
	}

	records := make([]AccountRecord, len(details))

	for i, d := range details {
		rec := AccountRecord{
			AlpacaAccount: acct.ID,
			ApexAccount:   acct.ApexAccount,
			Plan:          string(acct.Plan),
		}

		rec.LegalName = d.LegalName
		rec.EmailAddress = &acct.PrimaryOwner().Email
		addr, err := d.FormatAddress()
		if err != nil {
			log.Error(
				"failed to format address for account record",
				"account", *acct.ApexAccount,
				"error", err)
		}
		rec.LegalAddress = &addr
		rec.TelephoneNumber = d.PhoneNumber

		// ssn
		if d.HashSSN != nil {
			if ssn, err := encryption.DecryptWithkey(*d.HashSSN, []byte(env.GetVar("BROKER_SECRET"))); err == nil {
				ssnStr := string(ssn)
				ssnMask := fmt.Sprintf("xxx-xx-%s", ssnStr[len(ssnStr)-4:])
				rec.SocialSecurityNumber = &ssnMask
			}
		}

		rec.DateOfBirth = d.DateOfBirth
		if d.VisaType != nil {
			visaStr := string(*d.VisaType)
			rec.VisaType = &visaStr
		}

		rec.VisaExpiration = d.VisaExpirationDate
		rec.PermanentResident = d.PermanentResident
		rec.CountryOfBirth = d.CountryOfBirth

		// trusted contact
		if d.IncludeTrustedContact {
			srv := trustedcontact.Service().WithTx(db.DB())
			tc, err := srv.GetByID(acct.IDAsUUID())
			if err != nil {
				log.Error(
					"failed to retrieve trusted contact for account record",
					"account", *acct.ApexAccount,
					"error", err)
			} else {
				tcName := fmt.Sprintf("%s %s", tc.GivenName, tc.FamilyName)
				rec.TrustedContactName = &tcName
				rec.TrustedContactPhoneNumber = tc.PhoneNumber
				rec.TrustedContactEmailAddress = tc.EmailAddress
			}
		}

		rec.IsAffiliatedExchangeOrFINRA = d.IsAffiliatedExchangeOrFINRA
		rec.AffiliatedFirm = d.AffiliatedFirm

		rec.IsControlPerson = d.IsControlPerson
		if d.ControllingFirms != nil {
			for _, firm := range *d.ControllingFirms {
				rec.ControllingFirms = append(rec.ControllingFirms, firm)
			}
		}

		if d.EmploymentStatus != nil {
			emplStatus := string(*d.EmploymentStatus)
			rec.EmploymentStatus = &emplStatus
			rec.Employer = d.Employer
			rec.EmployerAddress = d.EmployerAddress
			rec.Position = d.Position
		}

		rec.OwnerID = acct.PrimaryOwner().ID

		rec.MarginAgreementSignedAt = d.MarginAgreementSignedAt
		rec.AccountAgreementSignedAt = d.AccountAgreementSignedAt

		rec.ApprovedAt = d.ApprovedAt
		rec.ApprovedBy = d.ApprovedBy

		records[i] = rec
	}

	return records, nil
}

// OrderTicketRecord defines a record in the order tickets file to be stored in Egnyte
type OrderTicket struct {
	UpstreamOrderID      string           `csv:"upstream_order_id"`
	UpstreamVenue        string           `csv:"upstream_venue"`
	ExecutionID          string           `csv:"execution_id"`
	OrderStatus          string           `csv:"order_status"`
	AlpacaAccount        string           `csv:"alpaca_account"`
	ApexAccount          *string          `csv:"apex_account"`
	EnteredBy            string           `csv:"entered_by"`
	TradeDate            *string          `csv:"trade_date"`
	ReceivedTimestamp    time.Time        `csv:"received_timestamp"`
	SentTimestamp        time.Time        `csv:"sent_timestamp"`
	TransactionTimestamp time.Time        `csv:"transaction_timestamp"`
	MarketVenue          string           `csv:"market_venue"`
	Side                 string           `csv:"side"`
	OrderType            string           `csv:"order_type"`
	LimitPrice           *decimal.Decimal `csv:"limit_price"`
	CancelRebill         string           `csv:"cancel_rebill"`
	Symbol               string           `csv:"symbol"`
	CUSIP                string           `csv:"cusip"`
	Quantity             decimal.Decimal  `csv:"quantity"`
	QuantityExecuted     *decimal.Decimal `csv:"quantity_executed"`
	ExecutedPrice        *decimal.Decimal `csv:"executed_price"`
	Memo                 string           `csv:"memo"`
}

func genOrderTickets(acct *models.Account, order *models.Order) ([]OrderTicket, error) {
	if err := db.DB().
		Model(order).
		Related(&order.Executions, "Executions").
		Order("transaction_time DESC").Error; err != nil {
		return nil, err
	}

	if len(order.Executions) == 0 {
		return nil, fmt.Errorf("no executions found for order %v", order.ID)
	}

	tickets := make([]OrderTicket, len(order.Executions))

	for i, exec := range order.Executions {
		tradeDate := exec.TransactionTime.Format(dateFormat)

		ticket := OrderTicket{
			UpstreamOrderID: exec.BrokerOrderID,
			// hardcode for noew
			UpstreamVenue:        "TRAFIX",
			ExecutionID:          exec.ID,
			OrderStatus:          string(exec.OrderStatus),
			AlpacaAccount:        acct.ID,
			ApexAccount:          acct.ApexAccount,
			EnteredBy:            "customer_api",
			TradeDate:            &tradeDate,
			ReceivedTimestamp:    order.CreatedAt,
			SentTimestamp:        order.SubmittedAt,
			TransactionTimestamp: exec.TransactionTime,
			// hardcode for now
			MarketVenue: "MNGD",
			Side:        string(order.Side),
			OrderType:   string(order.Type),
			LimitPrice:  order.LimitPrice,
			// TODO: handle trades moved to error account
			CancelRebill:     "",
			Symbol:           order.GetSymbol(),
			Quantity:         order.Qty,
			QuantityExecuted: exec.Qty,
			ExecutedPrice:    exec.AvgPrice,
			Memo:             "",
		}

		if asset := assetcache.Get(order.GetSymbol()); asset != nil {
			ticket.CUSIP = asset.CUSIP
		}

		tickets[i] = ticket
	}

	return tickets, nil
}
