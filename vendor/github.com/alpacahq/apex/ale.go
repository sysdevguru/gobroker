package apex

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

var alePath = "/ale/api/v1/read/"

type ALEMessage struct {
	ID       uint64    `json:"id"`
	DateTime time.Time `json:"dateTime"`
	Payload  string    `json:"payload"`
}

func (a *ALEMessage) DecodePayload(topic ALETopic) (interface{}, error) {
	if a.Payload == "" {
		return nil, fmt.Errorf("ale payload is empty")
	}

	var (
		err error
		pl  = []byte(a.Payload)
	)

	switch topic {
	case AtlasAccountReqStatus:
		accReqUpdate := AccountRequestUpdate{}
		err = json.Unmarshal(pl, &accReqUpdate)
		return accReqUpdate, err
	case EmailUpdateAleMsg:
		hermesUpdate := HermesStatusUpdate{}
		err = json.Unmarshal(pl, &hermesUpdate)
		return hermesUpdate, err
	case SentinelAchRelationshipStatus:
		relUpdate := AchRelationshipUpdate{}
		err = json.Unmarshal(pl, &relUpdate)
		return relUpdate, err
	case SentinelAchXferStatus:
		transferUpdate := AchTransferUpdate{}
		err = json.Unmarshal(pl, &transferUpdate)
		return transferUpdate, err
	case SentinelAchMicroDepXferStatus:
		microUpdate := MicroDepositUpdate{}
		err = json.Unmarshal(pl, &microUpdate)
		return microUpdate, err
	case SketchInvStatus:
		sketchUpdate := SketchStatusUpdate{}
		err = json.Unmarshal(pl, &sketchUpdate)
		return sketchUpdate, err
	case SnapDocUpload:
		snapUpdate := SnapStatusUpdate{}
		err = json.Unmarshal(pl, &snapUpdate)
		return snapUpdate, err
	case TradePostingStatus:
		// no schema available yet
		tradePostUpdate := TradePostingStatusUpdate{}
		err = json.Unmarshal(pl, &tradePostUpdate)
		return tradePostUpdate, err
	default:
		return nil, fmt.Errorf("unsupported ale topic (%v)", topic)
	}
}

type ALEQuery struct {
	HighWatermark uint64
	Since         time.Time
	Limit         uint
	StreamType    string
	Timeout       uint
}

func (aq *ALEQuery) URLEncode() string {
	q := fmt.Sprintf("highWaterMark=%v", aq.HighWatermark)
	if !aq.Since.IsZero() {
		q += fmt.Sprintf("&sinceDateTime=%v", aq.Since.Format("2006-01-02"))
	}
	if aq.Limit > 0 {
		q += fmt.Sprintf("&limit=%v", aq.Limit)
	}
	if aq.StreamType != "" {
		q += fmt.Sprintf("&streamType=%v", aq.StreamType)
	}
	if aq.Timeout > 0 {
		q += fmt.Sprintf("&timeoutSeconds=%v", aq.Timeout)
	}
	return q
}

type ALETopic string

const (
	AlpsAcatStatus                ALETopic = "alps-acat-status"
	AtlasAccountReqStatus         ALETopic = "atlas-account_request-status"
	EquilibriumBasketSummary      ALETopic = "equilibrium-basket-summary"
	EmailUpdateAleMsg             ALETopic = "hermes-email-update"
	SentinelAchMicroDepXferStatus ALETopic = "sentinel-ach-micro-deposit-transfer-status"
	SentinelAchRelationshipStatus ALETopic = "sentinel-ach-relationship-status"
	SentinelAchXferStatus         ALETopic = "sentinel-ach-transfer-status"
	SentinelWireXferStatus        ALETopic = "sentinel-wire-transfer-status"
	SketchInvStatus               ALETopic = "sketch-investigation-status"
	SnapDocUpload                 ALETopic = "snap-document-upload"
	TradePostingStatus            ALETopic = "trade-posting-status"
)

var ALETopics = []ALETopic{
	AlpsAcatStatus,
	AtlasAccountReqStatus,
	EquilibriumBasketSummary,
	EmailUpdateAleMsg,
	SentinelAchMicroDepXferStatus,
	SentinelAchRelationshipStatus,
	SentinelAchXferStatus,
	SentinelWireXferStatus,
	SketchInvStatus,
	SnapDocUpload,
	TradePostingStatus,
}

func (a *Apex) ALE(topic ALETopic, q ALEQuery) []ALEMessage {
	uri := fmt.Sprintf(
		"%v%v%v/%v?%v",
		os.Getenv("APEX_URL"),
		alePath,
		topic,
		os.Getenv("APEX_CORRESPONDENT_CODE"),
		q.URLEncode(),
	)
	msgs := []ALEMessage{}
	if _, err := a.getJSON(uri, &msgs); err != nil {
		return nil
	}
	return msgs
}

type AccountRequestUpdate struct {
	Status    string `json:"status"`
	RequestID string `json:"requestId"`
}

type AchRelationshipUpdate struct {
	Timestamp         time.Time `json:"timestamp"`
	RelationshipID    string    `json:"relationshipId"`
	Status            string    `json:"status"`
	CorrespondentCode string    `json:"correspondentCode"`
	ApprovalMethod    string    `json:"approvalMethod"`
	Reason            *string   `json:"reason"`
}

type AchTransferUpdate struct {
	Timestamp          time.Time `json:"timestamp"`
	TransferID         string    `json:"transferId"`
	ExternalTransferID string    `json:"externalTransferId"`
	Direction          string    `json:"direction"`
	TransferMechanism  string    `json:"transferMechanism"`
	Account            string    `json:"account"`
	CorrespondentCode  string    `json:"correspondentCode"`
	Amount             float64   `json:"amount"`
	Status             string    `json:"status"`
	Reason             *string   `json:"reason"`
}

type MicroDepositUpdate struct {
	Timestamp          time.Time `json:"timestamp"`
	TransferID         string    `json:"transferId"`
	ExternalTransferID *string   `json:"externalTransferId"`
	TransferMechanism  string    `json:"transferMechanism"`
	Direction          string    `json:"direction"`
	Account            string    `json:"account"`
	CorrespondentCode  string    `json:"correspondentCode"`
	Status             string    `json:"status"`
	Reason             *string   `json:"reason"`
	AchRelationshipID  string    `json:"achRelationshipId"`
}

type SketchStatusUpdate struct {
	State     string `json:"state"`
	RequestID string `json:"requestId"`
	Source    string `json:"source"`
	SourceID  string `json:"sourceId"`
}

type SnapStatusUpdate struct {
	ID                string   `json:"id"`
	CorrespondentCode string   `json:"correspondentCode"`
	Account           *string  `json:"account"`
	Tags              []string `json:"tags"`
}

type TradePostingStatusUpdate struct {
	ID         string `json:"id"`
	ExternalID string `json:"externalId"`
	Status     string `json:"status"`
	Message    string `json:"message"`
}

type HermesStatus string

const (
	HermesHardBounce HermesStatus = "hard_bounce"
	HermesSoftBounce HermesStatus = "soft_bounce"
	HermesResend     HermesStatus = "resend"
	HermesReject     HermesStatus = "reject"
	HermesError      HermesStatus = "error"
)

type HermesStatusUpdate struct {
	NotificationID    string       `json:"notificationId"`
	CorrespondentCode string       `json:"correspondentCode"`
	Email             string       `json:"email"`
	Status            HermesStatus `json:"status"`
}
