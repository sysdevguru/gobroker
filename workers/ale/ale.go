package ale

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/alpacahq/apex"
	"github.com/alpacahq/gobroker/external/slack"
	"github.com/alpacahq/gobroker/mailer"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/service/account"
	"github.com/alpacahq/gobroker/utils"
	"github.com/alpacahq/gobroker/workers/common"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/env"
	"github.com/alpacahq/gopaca/log"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	try "gopkg.in/matryer/try.v1"
)

type aleWorker struct {
	ale                func(topic apex.ALETopic, q apex.ALEQuery) []apex.ALEMessage
	apexGetAcctReq     func(requestId string) (*apex.AccountRequestResponse, error)
	apexGetSketch      func(id string) (*apex.GetSketchInvestigationResponse, error)
	apexGetRel         func(id string) (*apex.GetRelationshipResponse, error)
	apexTransferStatus func(id string) (*apex.TransferStatusResponse, error)
	done               chan struct{}
}

var worker *aleWorker

type topicHandler func(tx *gorm.DB, msg apex.ALEMessage) error

func Work() {
	if utils.Dev() {
		return
	}

	if worker == nil {
		worker = &aleWorker{
			ale: apex.Client().ALE,
			apexGetAcctReq: func(requestId string) (*apex.AccountRequestResponse, error) {
				log.Debug("ale worker apex atlas request", "id", requestId)
				return apex.Client().GetAccountRequest(requestId)
			},
			apexGetSketch: func(id string) (*apex.GetSketchInvestigationResponse, error) {
				log.Debug("ale worker apex sketch request", "id", id)
				return apex.Client().GetSketchInvestigation(id)
			},
			apexGetRel:         apex.Client().GetRelationship,
			apexTransferStatus: apex.Client().TransferStatus,
			done:               make(chan struct{}, 1),
		}
		worker.done <- struct{}{}
	}

	// make sure not to overlap if the work routine is taking long
	if common.WaitTimeout(worker.done, time.Second) {
		// timed out, so let's skip this round and wait until it finishes
		return
	}

	defer func() {
		worker.done <- struct{}{}
	}()

	timeout, err := time.ParseDuration(env.GetVar("ALE_TIMEOUT"))
	if err != nil {
		log.Error("invalid ale timeout", "error", err, "value", env.GetVar("ALE_TIMEOUT"))
		timeout = 10 * time.Second
	}

	// main topics
	reactors := []struct {
		topic   apex.ALETopic
		handler topicHandler
	}{
		{apex.AtlasAccountReqStatus, worker.accountUpdateHandler},
		{apex.SentinelAchXferStatus, worker.transferUpdateHandler},
		{apex.SentinelAchMicroDepXferStatus, worker.microUpdateHandler},
		{apex.SentinelAchRelationshipStatus, worker.relationshipUpdateHandler},
		{apex.SketchInvStatus, worker.sketchHandler},
		{apex.SnapDocUpload, worker.snapHandler},
		{apex.TradePostingStatus, worker.braggartHandler},
	}

	// only pull hermes in prod
	if utils.Prod() {
		reactors = append(
			reactors,
			struct {
				topic   apex.ALETopic
				handler topicHandler
			}{
				apex.EmailUpdateAleMsg,
				worker.hermesHandler,
			})
	}

	statuses := make([]interface{}, len(reactors))

	for i, reactor := range reactors {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)

		if status, err := worker.pullALE(
			ctx,
			reactor.topic,
			reactor.handler); err == nil && status.Watermark > 0 {
			statuses[i] = status
		}
		cancel()
	}

	for _, status := range statuses {
		if status != nil {
			tx := db.Begin()

			if err := tx.Save(status).Error; err != nil {
				st := status.(*models.ALEStatus)
				log.Error("database error", "action", "update", "status", *st, "error", err)
			}

			if err := tx.Commit().Error; err != nil {
				st := status.(*models.ALEStatus)
				log.Error("database error", "action", "update", "status", *st, "error", err)
			}
		}
	}
}

func (w *aleWorker) pullALE(ctx context.Context, topic apex.ALETopic, handler topicHandler) (status *models.ALEStatus, err error) {
	status = &models.ALEStatus{}

	q := db.DB().Where("topic = ?", topic).Find(&status)

	if q.Error != nil && q.Error != gorm.ErrRecordNotFound {
		return nil, q.Error
	}

	if q.RecordNotFound() {
		status.Topic = string(topic)

		if err = db.DB().Create(status).Error; err != nil {
			return nil, err
		}
	}

	msgC := make(chan []apex.ALEMessage)

	go func() {
		msgC <- w.ale(topic, apex.ALEQuery{
			HighWatermark: status.Watermark,
			Since:         clock.Now().Add(-60 * 24 * time.Hour),
		})
	}()

	select {
	case aleMsgs := <-msgC:
		for _, msg := range aleMsgs {
			if err = try.Do(func(attempt int) (bool, error) {
				tx := db.RepeatableRead()

				if err = handler(tx, msg); err != nil {
					tx.Rollback()
				} else {
					err = tx.Commit().Error
				}

				return db.IsSerializabilityError(err), err
			}); err != nil {
				log.Error(
					"ale worker update failure",
					"topic", topic,
					"payload", msg.Payload,
					"error", err)
				return nil, err
			}
			status.Watermark = msg.ID
		}

		log.Debug("ale worker processed updates", "topic", topic, "count", len(aleMsgs))

		return status, nil
	case <-ctx.Done():
		// doing this here since hermes and trade posting status topics
		// tend to be very slow and we don't want to flood the logs
		// (probably because they have very few messages)
		if isCriticalTopic(topic) {
			log.Error("ale pull took too long",
				"error", ctx.Err(),
				"timeout_duration", env.GetVar("ALE_TIMEOUT"),
				"topic", topic)
		} else {
			log.Warn("ale pull took too long",
				"error", ctx.Err(),
				"timeout_duration", env.GetVar("ALE_TIMEOUT"),
				"topic", topic)
		}

		return nil, ctx.Err()
	}
}

// handlers
func (w *aleWorker) accountUpdateHandler(tx *gorm.DB, msg apex.ALEMessage) error {
	payload, err := decodePayload(msg, apex.AtlasAccountReqStatus)

	if err != nil {
		return err
	}

	if payload == nil {
		return nil
	}

	update := payload.(apex.AccountRequestUpdate)
	acct := &models.Account{}

	q := tx.Where("apex_request_id = ?", update.RequestID).Find(acct)

	if q.RecordNotFound() {
		if utils.Prod() {
			log.Error(
				"ale worker update for unknown apex_request_id",
				"apex_request_id", update.RequestID)
		}
		return nil
	}

	if q.Error != nil {
		return q.Error
	}

	log.Debug(
		"ale worker account status update",
		"account", acct.ID,
		"status_old", acct.ApexApprovalStatus,
		"status_new", update.Status)

	if enum.ApexApprovalStatus(update.Status) == enum.ApexRejected {
		log.Info("apex rejected account", "account", acct.ID)

		if acct.ApexAccount != nil {
			log.Info(
				"updating alpaca side and removing apex account",
				"apex_account", *acct.ApexAccount)

			if err = tx.Model(acct).Update("apex_account", gorm.Expr("NULL")).Error; err != nil {
				return err
			}
		}

		if err = tx.Model(acct).Update("status", enum.Rejected).Error; err != nil {
			return err
		}
	}

	return tx.
		Model(acct).
		Update("apex_approval_status", update.Status).Error
}

func (w *aleWorker) syncInvestigation(tx *gorm.DB, sketchID string, status *string) (*models.Investigation, error) {
	var (
		err error
		inv = &models.Investigation{}
		q   *gorm.DB
	)

	// log at the end if nothing failed
	defer func() {
		if err == nil {
			log.Info(
				"ale worker sketch investigation status update",
				"investigation", inv.ID,
				"status_new", inv.Status)
		}
	}()

	// this is a new sketch investigation we've never seen - let's sync it
	if q = tx.Where("id = ?", sketchID).First(inv); q.RecordNotFound() {
		resp, err := w.apexGetSketch(sketchID)
		if err != nil {
			return nil, fmt.Errorf("failed to get sketch investigation %s (%v)", inv.ID, err)
		}

		acct := &models.Account{}

		if err = tx.Where("apex_account = ?", *resp.Request.Account).Find(acct).Error; err != nil {
			return nil, err
		}

		inv.ID = sketchID
		inv.Status = models.InvestigationStatus(*resp.Status)
		inv.AccountID = acct.ID

		return inv, tx.Save(inv).Error
	}

	if q.Error != nil {
		return nil, q.Error
	}

	// if this an existing sketch investigation, but we got a status update
	// (i.e. from ALE) let's update it now
	if status != nil {
		if err := tx.Model(inv).Update("status", *status).Error; err != nil {
			return nil, err
		}
	}

	return inv, nil
}

func (w *aleWorker) transferUpdateHandler(tx *gorm.DB, msg apex.ALEMessage) error {
	payload, err := decodePayload(msg, apex.SentinelAchXferStatus)

	if err != nil {
		return err
	}

	if payload == nil {
		return nil
	}

	update := payload.(apex.AchTransferUpdate)

	var id uuid.UUID
	id, err = uuid.FromString(update.ExternalTransferID)
	if err != nil {
		return err
	}

	transfer := &models.Transfer{}

	q := tx.Where("id = ?", id.String()).First(transfer)

	if q.Error != nil && q.Error != gorm.ErrRecordNotFound {
		return q.Error
	}

	if q.RecordNotFound() || transfer.Status == enum.TransferPending {

		resp, err := w.apexTransferStatus(update.TransferID)
		if err != nil {
			return fmt.Errorf("received sentinel ACH transfer update for unknown transfer (%v)", err)
		}

		acct := &models.Account{}
		q = tx.Where("apex_account = ?", update.Account).Find(acct)

		if q.RecordNotFound() {
			log.Error(
				"ale worker update for unknown transfer",
				"transfer_id", update.TransferID,
				"account", update.Account)
			return nil
		}

		if q.Error != nil {
			return q.Error
		}

		transfer.ID = id.String()
		transfer.AccountID = acct.ID
		transfer.ApexID = resp.TransferID
		transfer.Amount = *resp.Amount
		transfer.Direction = apex.TransferDirection(*resp.Direction)
		transfer.Status = enum.TransferStatus(*resp.State)
		transfer.EstimatedFundsAvailableDate = resp.EstimatedFundsAvailableDate

		if update.Reason != nil {
			transfer.Reason = nachaCodes[*update.Reason]
			transfer.ReasonCode = *update.Reason
		}

		return tx.Save(transfer).Error
	}

	var (
		reason     string
		reasonCode string
	)

	if update.Reason != nil {
		reason = nachaCodes[*update.Reason]
		reasonCode = *update.Reason
		transfer.Reason = reason
		if query := tx.Model(transfer).Update("reason", transfer.Reason); query.Error != nil {
			return query.Error
		}
		transfer.ReasonCode = reasonCode
		if query := tx.Model(transfer).Update("reason_code", transfer.ReasonCode); query.Error != nil {
			return query.Error
		}
	}

	status := apex.TransferStatus(update.Status)

	log.Info(
		"ale worker transfer status update",
		"transfer", transfer.ID,
		"status_old", transfer.Status,
		"status_new", status,
		"reason", reason)

	return tx.Model(transfer).Update(
		"status", status).Error
}

func (w *aleWorker) microUpdateHandler(tx *gorm.DB, msg apex.ALEMessage) error {
	payload, err := decodePayload(msg, apex.SentinelAchMicroDepXferStatus)

	if err != nil {
		return err
	}

	if payload == nil {
		return nil
	}

	update := payload.(apex.MicroDepositUpdate)

	rel := &models.ACHRelationship{}

	q := tx.Where("apex_id = ?", update.AchRelationshipID).Find(&rel)

	if q.RecordNotFound() {
		log.Error(
			"ale worker update for unknown relationship",
			"relationship_id", update.AchRelationshipID,
			"account", update.Account)
		return nil
	}

	if q.Error != nil {
		return q.Error
	}

	m := map[string]interface{}{
		"micro_deposit_id":     update.TransferID,
		"micro_deposit_status": update.Status,
	}

	if err = tx.Model(rel).Updates(m).Error; err != nil {
		return err
	}

	if rel.Status != enum.RelationshipApproved &&
		(rel.MicroDepositStatus == enum.TransferComplete || rel.MicroDepositStatus == enum.TransferRejected) {
		var success bool
		var reason string
		if update.Reason != nil {
			reason = nachaCodes[*update.Reason]
		}

		// decide what type of email to send
		if rel.MicroDepositStatus == enum.TransferComplete {
			success = true
		} else if rel.MicroDepositStatus == enum.TransferRejected {
			success = false
		}

		// grab necessary details
		svc := account.Service().WithTx(tx)
		acct, err := svc.GetByID(rel.AccountIDAsUUID())
		if err != nil {
			return err
		}
		o := acct.PrimaryOwner()

		// send email
		go mailer.SendMicroDeposit(
			success,
			rel.AccountID,
			*o.Details.GivenName,
			*rel.Nickname,
			o.Email,
			reason,
		)
	}

	return nil
}

func (w *aleWorker) relationshipUpdateHandler(tx *gorm.DB, msg apex.ALEMessage) error {
	payload, err := decodePayload(msg, apex.SentinelAchRelationshipStatus)

	if err != nil {
		return err
	}

	if payload == nil {
		return nil
	}

	update := payload.(apex.AchRelationshipUpdate)

	relationship := &models.ACHRelationship{}

	q := tx.Where("apex_id = ?", update.RelationshipID).Find(relationship)

	if q.Error != nil && q.Error != gorm.ErrRecordNotFound {
		return q.Error
	}

	var reason string

	if q.RecordNotFound() {
		resp, err := w.apexGetRel(update.RelationshipID)
		if err != nil {
			return fmt.Errorf("ale worker failed to retrieve relationship %s (%v)", update.RelationshipID, err)
		}

		acct := &models.Account{}

		q = tx.Where("apex_account = ?", *resp.Account).Find(acct)

		if q.RecordNotFound() {
			return fmt.Errorf(
				"ale worker relationship update (%s) for unknown apex account %s",
				update.RelationshipID,
				*resp.Account)
		}

		if q.Error != nil {
			return q.Error
		}

		relationship = &models.ACHRelationship{
			ApexID:         &update.RelationshipID,
			Status:         enum.RelationshipStatus(*resp.Status),
			ApprovalMethod: apex.ACHApprovalMethod(*resp.ApprovalMethod),
			AccountID:      acct.ID,
		}

		if update.Reason != nil {
			reason = *update.Reason
			relationship.Reason = reason
		}
		return tx.Create(relationship).Error
	}

	if update.Reason != nil {
		reason = *update.Reason
		relationship.Reason = reason
		if query := tx.Model(relationship).Update("reason", relationship.Reason); query.Error != nil {
			return query.Error
		}
	}

	status := apex.ACHRelationshipStatus(update.Status)

	log.Info(
		"ale worker ach relationship status update",
		"relationship", relationship.ID,
		"status_old", relationship.Status,
		"status_new", status,
		"reason", reason)

	return tx.Model(relationship).Update(
		"status", status).Error
}

func (w *aleWorker) sketchHandler(tx *gorm.DB, msg apex.ALEMessage) (err error) {
	payload, err := decodePayload(msg, apex.SketchInvStatus)

	if err != nil {
		return err
	}

	if payload == nil {
		return nil
	}

	update := payload.(apex.SketchStatusUpdate)

	_, err = w.syncInvestigation(tx, update.RequestID, &update.State)

	return
}

func (w *aleWorker) snapHandler(tx *gorm.DB, msg apex.ALEMessage) (err error) {
	payload, err := decodePayload(msg, apex.SnapDocUpload)

	if err != nil {
		return err
	}

	if payload == nil {
		return nil
	}

	update := payload.(apex.SnapStatusUpdate)

	snap := models.Snap{}

	q := tx.Where("id = ?", update.ID).Find(&snap)

	if q.RecordNotFound() {
		log.Warn(
			"ale worker snap update for unknown snap",
			"snap_id", update.ID)
		return nil
	}

	if q.Error != nil {
		return q.Error
	}

	now := clock.Now()
	snap.ALEConfirmedAt = &now
	err = tx.Save(&snap).Error

	log.Info(
		"ale worker snap upload confirmed",
		"snap", snap.ID)

	return
}

func (w *aleWorker) hermesHandler(tx *gorm.DB, msg apex.ALEMessage) error {
	if !utils.Prod() {
		return nil
	}

	payload, err := decodePayload(msg, apex.EmailUpdateAleMsg)

	if err != nil {
		return err
	}

	if payload == nil {
		return nil
	}

	update := payload.(apex.HermesStatusUpdate)

	hf := &models.HermesFailure{
		ID:                update.NotificationID,
		Status:            update.Status,
		Email:             update.Email,
		CorrespondentCode: update.CorrespondentCode,
	}

	return tx.
		Where("id = ?", update.NotificationID).
		Assign(&models.HermesFailure{Status: update.Status}).
		FirstOrCreate(hf).Error
}

// Right now, I'm not able to produce a braggart posting failure in UAT,
// and the schema is not available from Apex for the message itself, so
// for now I have decided to just alert the team via Slack in the
// #braggart-failures channel on production. Once we receive one, or
// Apex is able to give me the schema, I will then be able to handle
// it properly. Regardless, I think we will want to be notified via
// Slack when this happens because it will require manual correction,
// and shouldn't occur under normal circumstances.
func (w *aleWorker) braggartHandler(tx *gorm.DB, aleMsg apex.ALEMessage) error {
	msg := slack.NewBraggartFailure()
	msg.SetBody(aleMsg)
	slack.Notify(msg)

	return nil
}

func decodePayload(msg apex.ALEMessage, topic apex.ALETopic) (interface{}, error) {
	payload, err := msg.DecodePayload(topic)
	if err == nil {
		return payload, nil
	}

	if !strings.EqualFold(err.Error(), "ale payload is empty") {
		log.Error("failed to decode ale payload", "error", err)
		return nil, err
	}

	return nil, nil
}

func isCriticalTopic(topic apex.ALETopic) bool {
	switch topic {
	case apex.EmailUpdateAleMsg:
		fallthrough
	case apex.TradePostingStatus:
		return false
	default:
		return true
	}
}
