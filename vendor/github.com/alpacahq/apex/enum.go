package apex

type TransferStatus string

func (s TransferStatus) Cancelable() bool {
	switch s {
	case TransferSentToBank:
		fallthrough
	case TransferComplete:
		fallthrough
	case TransferReturned:
		fallthrough
	case TransferRejected:
		return false
	default:
		return true
	}
}

const (
	TransferPending        TransferStatus = "PENDING"
	TransferRejected       TransferStatus = "REJECTED"
	TransferCanceled       TransferStatus = "CANCELED"
	TransferApproved       TransferStatus = "APPROVED"
	TransferFundsPosted    TransferStatus = "FUNDS_POSTED"
	TransferSentToBank     TransferStatus = "SENT_TO_BANK"
	TransferComplete       TransferStatus = "COMPLETE"
	TransferReturned       TransferStatus = "RETURNED"
	TransferPendingPrinted TransferStatus = "PENDING_PRINTING"
	TransferVoid           TransferStatus = "VOID"
	TransferStopPayment    TransferStatus = "STOP_PAYMENT"
)

type TransferDirection string

const (
	Incoming TransferDirection = "INCOMING"
	Outgoing TransferDirection = "OUTGOING"
)

type ACHRelationshipStatus string

const (
	ACHPending  ACHRelationshipStatus = "PENDING"
	ACHApproved ACHRelationshipStatus = "APPROVED"
	ACHCanceled ACHRelationshipStatus = "CANCELED"
)

type ACHApprovalMethod string

const (
	Plaid        ACHApprovalMethod = "PLAID"
	MicroDeposit ACHApprovalMethod = "MICRO_DEPOSIT"
)

const (
	ReasonMicroExhausted = "EXHAUSTED_MICRO_DEPOSIT_ATTEMPTS"
)
