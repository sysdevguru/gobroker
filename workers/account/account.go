package account

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"time"

	"github.com/alpacahq/apex"
	"github.com/alpacahq/apex/forms"
	"github.com/alpacahq/gobroker/external/slack"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/service/account"
	"github.com/alpacahq/gobroker/service/ownerdetails"
	"github.com/alpacahq/gobroker/stream"
	"github.com/alpacahq/gobroker/utils"
	"github.com/alpacahq/gobroker/workers/account/form"
	"github.com/alpacahq/gobroker/workers/common"
	"github.com/alpacahq/gopaca/calendar"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/log"
	"github.com/alpacahq/gopaca/rmq/pubsub"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

type accountWorker struct {
	stream        chan<- pubsub.Message
	cancel        context.CancelFunc
	apexPostAcct  func(sub forms.FormSubmission) (arr *apex.AccountRequestResponse, body []byte, err error)
	apexGetAcct   func(requestId string) (*apex.AccountRequestResponse, error)
	apexGetSketch func(id string) (*apex.GetSketchInvestigationResponse, error)
	done          chan struct{}
}

var worker *accountWorker

// Stop disconnects the RMQ connection and prepares the routine
// for graceful shutdown
func Stop() {
	worker.cancel()
}

// Work manages the upkeep of accounts, including their corresponding
// sketch investigations, and submits new accounts for creation
// by Apex.
func Work() {
	if worker == nil {
		worker = &accountWorker{
			apexPostAcct: apex.Client().PostAccountRequest,
			apexGetAcct: func(requestId string) (*apex.AccountRequestResponse, error) {
				log.Debug("account worker apex atlas request", "id", requestId)
				return apex.Client().GetAccountRequest(requestId)
			},
			apexGetSketch: func(id string) (*apex.GetSketchInvestigationResponse, error) {
				log.Debug("account worker apex sketch request", "id", id)
				return apex.Client().GetSketchInvestigation(id)
			},
			done: make(chan struct{}, 1),
		}
		worker.done <- struct{}{}
		worker.stream, worker.cancel = pubsub.NewPubSub("stream").Publish()
	}

	// make sure not to overlap if the work routine is taking long
	if common.WaitTimeout(worker.done, time.Second) {
		// timed out, so let's skip this round and wait until it finishes
		return
	}

	defer func() {
		worker.done <- struct{}{}
	}()

	tx := db.Begin()

	accounts := []models.Account{}

	err := tx.
		Where(
			"NOT (status = ? OR (status = ? AND apex_approval_status = ?))",
			enum.Rejected, enum.Active, enum.Complete).
		Find(&accounts).Error

	if err != nil {
		tx.Rollback()
		log.Error("account worker database error", "error", err)
		return
	}

	if err = tx.Commit().Error; err != nil {
		tx.Rollback()
		log.Error("account worker database error", "error", err)
		return
	}

	log.Debug("account worker processing accounts", "count", len(accounts))

	for _, acct := range accounts {
		tx = db.Begin()
		// pass acct ID & use service
		if err = worker.processAccount(tx, acct.IDAsUUID()); err != nil {
			tx.Rollback()
			log.Error(
				"account worker processing failure",
				"account", acct.ID,
				"error", err)
			continue
		}
		if err = tx.Commit().Error; err != nil {
			log.Error(
				"account worker processing failure",
				"account", acct.ID,
				"error", err)
			continue
		}
	}
}

func (w *accountWorker) processAccount(tx *gorm.DB, id uuid.UUID) (err error) {
	srv := account.Service().WithTx(tx)
	srv.SetForUpdate()

	acct, err := srv.GetByID(id)
	if err != nil {
		return err
	}

	odSrv := ownerdetails.Service().WithTx(tx)

	details, err := odSrv.GetPrimaryByAccountID(id)
	if err != nil {
		return err
	}

	// make a copy for deep equal comparision later to determine
	// if a stream message is warranted
	m := acct.ForJSON()
	defer func() {
		if sM := acct.ForJSON(); !reflect.DeepEqual(m, sM) {
			msg := stream.OutboundMessage{
				Stream: stream.AccountUpdatesStream(id),
				Data:   sM,
			}
			buf, _ := json.Marshal(msg)
			w.stream <- pubsub.Message(buf)
		}
	}()

	switch acct.Status {
	case enum.Onboarding:
		err = w.handleOnboarding(tx, acct, details)
	case enum.AccountUpdated:
		log.Debug("account worker handling account updated")
		err = w.handleAccountUpdated(tx, acct, details)
	}

	if err != nil {
		return err
	}

	if acct.MarkedRiskyTransfersAt != nil {
		markedAt := acct.MarkedRiskyTransfersAt.In(calendar.NY)
		// if markedAt is later than 90 days, de-flag the account
		if (clock.Now().Sub(markedAt).Hours() / 24) > 90 {
			updates := map[string]interface{}{
				"risky_transfers":           false,
				"marked_risky_transfers_at": nil,
			}
			if err = tx.Model(acct).Updates(updates).Error; err != nil {
				tx.Rollback()
				log.Error("error updating transfers blocked to false", "error", err, "acct", acct.ID)
				return err
			}
		}
	}

	switch acct.ApexApprovalStatus {
	case enum.ActionRequired:
	case enum.Suspended:
	case enum.ApexRejected:
	case enum.ReadyForBackOffice:
	case enum.BackOffice:
	case enum.AccountSetup:
	case enum.AccountCanceled:
	case enum.Error:
	case enum.Complete:
		// set to approval pending
		err = w.handleComplete(tx, acct)
	}
	return
}

func (w *accountWorker) handleOnboarding(tx *gorm.DB, a *models.Account, od *models.OwnerDetails) error {
	if od.AccountAgreementSigned == nil || od.MarginAgreementSigned == nil {
		return nil
	}
	if !*od.AccountAgreementSigned || !*od.MarginAgreementSigned {
		return nil
	}

	q := tx.Model(a).Related(&a.Owners, "Owners")

	if q.Error != nil {
		return q.Error
	}

	formSubmission := form.PrepareSubmission(a, od, "CREATE")

	if formSubmission == nil {
		log.Error("account worker form fill failure", "account", a.ID)
		a.Status = enum.SubmissionFailed
		return tx.Save(a).Error
	}

	resp, body, err := w.apexPostAcct(*formSubmission)

	a.AccReqResult = string(body)

	if err != nil || resp == nil || strings.Contains(a.AccReqResult, "errorCode") {
		log.Error(
			"account worker create submission failure",
			"error", err,
			"account", a.ID,
			"result", a.AccReqResult)

		a.Status = enum.SubmissionFailed

		return tx.Save(a).Error
	}

	if (a.ApexAccount == nil ||
		a.ApexApprovalStatus == enum.Suspended ||
		a.ApexApprovalStatus == enum.ActionRequired) &&
		len(*resp.Account) > 0 {
		a.ApexAccount = resp.Account
	}

	a.ApexApprovalStatus = enum.ApexApprovalStatus(*resp.Status)
	a.Status = enum.Submitted
	a.ApexRequestID = *resp.ID

	if len(resp.SketchIDs) > 0 {
		for _, sid := range resp.SketchIDs {
			if err := tx.Create(&models.Investigation{
				ID:        sid,
				AccountID: a.ID,
				Status:    models.SketchPending,
			}).Error; err != nil {
				log.Error("account worker sketch storage failure", "sketch_id", sid, "error", err)
				return err
			}
		}
	}

	log.Info("account request submitted", "status", a.ApexApprovalStatus)

	// notify via slack
	{
		msg := slack.NewAccountUpdate()

		msg.SetBody(struct {
			Type        string `json:"type"`
			ApexAccount string `json:"apex_account"`
			Name        string `json:"name"`
			Email       string `json:"email"`
		}{
			"account_submitted",
			*a.ApexAccount,
			*od.LegalName,
			a.Email,
		})

		slack.Notify(msg)
	}

	return tx.Save(a).Error
}

func (w *accountWorker) handleAccountUpdated(tx *gorm.DB, a *models.Account, od *models.OwnerDetails) error {
	if od.AccountAgreementSigned == nil || od.MarginAgreementSigned == nil {
		return nil
	}
	if !*od.AccountAgreementSigned || !*od.MarginAgreementSigned {
		return nil
	}

	q := tx.Model(a).Related(&a.Owners, "Owners")

	if q.Error != nil {
		return q.Error
	}

	formSubmission := form.PrepareSubmission(a, od, "UPDATE")
	if formSubmission == nil {
		log.Error("account worker form fill failure", "account", a.ID)
		a.Status = enum.SubmissionFailed
		return tx.Save(a).Error
	}

	resp, body, err := w.apexPostAcct(*formSubmission)
	if err != nil {
		return errors.Wrap(err, string(body))
	}

	if resp == nil {
		log.Error("account worker updated submission failure", "account", a.ID)
		a.Status = enum.SubmissionFailed
		a.AccReqResult = string(body)
		return tx.Save(a).Error
	}

	log.Info("account worker forms resubmitted", "account", a.ID)

	a.ApexAccount = resp.Account
	a.ApexApprovalStatus = enum.ApexApprovalStatus(*resp.Status)
	a.Status = enum.Resubmitted
	a.ApexRequestID = *resp.ID
	a.AccReqResult = string(body)

	if len(resp.SketchIDs) > 0 {
		for _, sid := range resp.SketchIDs {
			if err := tx.Create(&models.Investigation{
				ID:        sid,
				AccountID: a.ID,
				Status:    models.SketchPending,
			}).Error; err != nil {
				log.Error("account worker sketch storage failure", "sketch_id", sid, "error", err)
			}
		}
	}

	// notify via slack
	{
		msg := slack.NewAccountUpdate()

		msg.SetBody(struct {
			Type        string `json:"type"`
			ApexAccount string `json:"apex_account"`
			Name        string `json:"name"`
			Email       string `json:"email"`
		}{
			"account_updated",
			*a.ApexAccount,
			*od.LegalName,
			a.Email,
		})

		slack.Notify(msg)
	}

	return tx.Save(a).Error
}

// func (w *accountWorker) handleActionRequired(tx *gorm.DB, a *models.Account) error {
// 	q := tx.Model(&a).Related(&a.Investigations)

// 	if q.Error != nil {
// 		return q.Error
// 	}

// 	arr, err := w.apexGetAcct(a.ApexRequestID)
// 	if err != nil {
// 		return fmt.Errorf("failed to retrieve apex account request %v (%v)", a.ApexRequestID, err)
// 	}

// 	for _, sid := range arr.SketchIDs {
// 		i := sort.Search(len(a.Investigations), func(i int) bool {
// 			return a.Investigations[i].ID == sid
// 		})
// 		// this is an investigation we don't know about yet
// 		if i >= len(a.Investigations) {
// 			inv, err := w.apexGetSketch(sid)
// 			if err != nil {
// 				return fmt.Errorf("failed to retrieve apex sketch investigation %v (%v)", sid, err)
// 			}

// 			investigation := &models.Investigation{
// 				ID:        sid,
// 				AccountID: a.ID,
// 			}

// 			if err := tx.FirstOrCreate(investigation, "id = ?", investigation.ID).Error; err != nil {
// 				log.Error(
// 					"account worker investigation create failure",
// 					"account", a.ID,
// 					"investigation", investigation.ID,
// 					"error", err,
// 				)
// 				return err
// 			}

// 			investigation.Status = models.InvestigationStatus(*inv.Status)

// 			if err := tx.Save(investigation).Error; err != nil {
// 				log.Error(
// 					"account worker investigation update failure",
// 					"account", a.ID,
// 					"investigation", investigation.ID,
// 					"error", err,
// 				)
// 				return err
// 			}
// 		}
// 	}
// 	// process investigations
// 	for _, investigation := range a.Investigations {
// 		if !investigation.Status.Closed() {
// 			inv, err := w.apexGetSketch(investigation.ID)
// 			if err != nil {
// 				return fmt.Errorf("failed to retrieve sketch investigation %v (%v)", investigation.ID, err)
// 			}

// 			if arr != nil {
// 				investigation.Status = models.InvestigationStatus(*inv.Status)
// 				if err := tx.Save(&investigation).Error; err != nil {
// 					log.Error("account worker investigation status update failure", "account", a.ID, "error", err)
// 					return err
// 				}
// 			} else {
// 				log.Error("account worker sketch status request failure", "account", a.ID)
// 			}
// 		}
// 	}

// 	return nil
// }

func (w *accountWorker) handleComplete(tx *gorm.DB, a *models.Account) error {
	switch a.Status {
	case enum.Submitted:
		switch {
		case utils.Dev():
			// automatically approve in DEV mode
			a.Status = enum.Active
		case utils.Stg():
			fallthrough
		case utils.Prod():
			// require manual approval in staging/prod
			a.Status = enum.ApprovalPending
		}
	case enum.Resubmitted:
		switch {
		case utils.Dev():
			// automatically re-approve in DEV mode
			a.Status = enum.Active
		case utils.Stg():
			fallthrough
		case utils.Prod():
			// require manual re-approval in staging/prod
			a.Status = enum.ReapprovalPending
		}
	}

	return tx.Save(a).Error
}
