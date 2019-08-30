package enum

import (
	"math/big"
	"strings"

	"github.com/shopspring/decimal"
)

type AccountStatus string

const (
	// account has been created, but brokerage account
	// application has not been started - only paper
	// trading is allowed for this account
	PaperOnly AccountStatus = "PAPER_ONLY"
	// account has begun creating a brokerage account
	Onboarding AccountStatus = "ONBOARDING"
	// failure on submission
	SubmissionFailed AccountStatus = "SUBMISSION_FAILED"
	// request has been submitted to Apex (for initial investigation)
	Submitted AccountStatus = "SUBMITTED"
	// modified by user
	AccountUpdated AccountStatus = "ACCOUNT_UPDATED"
	// request has been re-submitted to Apex after account update
	Resubmitted AccountStatus = "RESUBMITTED"
	// intermediate state once apex approves account
	ApprovalPending AccountStatus = "APPROVAL_PENDING"
	// indeterminate state once apex approves a re-submission
	ReapprovalPending AccountStatus = "REAPPROVAL_PENDING"
	// fully approved & enabled
	Active AccountStatus = "ACTIVE"
	// rejected - this account is dead to us, and will no
	// longer be processed by the account worker
	Rejected AccountStatus = "REJECTED"
	// if an admin updates the account and it needs to be resubmitted
	Edited AccountStatus = "EDITED"
	// brokerage account was closed by an admin
	AccountClosed AccountStatus = "ACCOUNT_CLOSED"
)

type ApexApprovalStatus string

const (
	// defined by Apex
	New                    ApexApprovalStatus = "NEW"
	Pending                ApexApprovalStatus = "PENDING"
	InvestigationSubmitted ApexApprovalStatus = "INVESTIGATION_SUBMITTED"
	ActionRequired         ApexApprovalStatus = "ACTION_REQUIRED"
	Suspended              ApexApprovalStatus = "SUSPENDED"
	ApexRejected           ApexApprovalStatus = "REJECTED"
	ReadyForBackOffice     ApexApprovalStatus = "READY_FOR_BACK_OFFICE"
	BackOffice             ApexApprovalStatus = "BACK_OFFICE"
	AccountSetup           ApexApprovalStatus = "ACCOUNT_SETUP"
	AccountCanceled        ApexApprovalStatus = "CANCELED"
	Error                  ApexApprovalStatus = "ERROR"
	Complete               ApexApprovalStatus = "COMPLETE"
)

type AccountPlan string

const (
	PremiumAccount AccountPlan = "PREMIUM"
	RegularAccount AccountPlan = "REGULAR"
)

type AccountType string

const (
	LiveAccount   AccountType = "live"
	PaperAccount  AccountType = "paper"
	CryptoAccount AccountType = "crypto"
)

type OrderType string

func (o OrderType) Readable() string {
	return strings.Replace(string(o), "_", " ", -1)
}

const (
	Market        OrderType = "market"
	Limit         OrderType = "limit"
	Stop          OrderType = "stop"
	StopLimit     OrderType = "stop_limit"
	MarketOnClose OrderType = "market_on_close"
	LimitOnClose  OrderType = "limit_on_close"
)

func ValidOrderType(oType OrderType) bool {
	return oType == Market ||
		oType == Limit ||
		oType == Stop ||
		oType == StopLimit
}

type Side string

const (
	Buy             Side = "buy"
	Sell            Side = "sell"
	BuyMinus        Side = "buy_minus"
	SellPlus        Side = "sell_plus"
	SellShort       Side = "sell_short"
	SellShortExempt Side = "sell_short_exempt"
	Undisclosed     Side = "undisclosed"
	Cross           Side = "cross"
	CrossShort      Side = "cross_short"
)

func (s *Side) Coeff() decimal.Decimal {
	i := int64(1)
	if *s == Sell {
		i = int64(-1)
	}
	return decimal.NewFromBigInt(big.NewInt(i), 0)
}

func ValidSide(side Side) bool {
	return side == Buy || side == Sell

}

type OrderStatus string

const (
	OrderAccepted           OrderStatus = "accepted" // Our own status represent we received order.
	OrderNew                OrderStatus = "new"
	OrderPartiallyFilled    OrderStatus = "partially_filled"
	OrderFilled             OrderStatus = "filled"
	OrderDoneForDay         OrderStatus = "done_for_day"
	OrderCanceled           OrderStatus = "canceled"
	OrderReplaced           OrderStatus = "replaced"
	OrderPendingCancel      OrderStatus = "pending_cancel"
	OrderStopped            OrderStatus = "stopped"
	OrderRejected           OrderStatus = "rejected"
	OrderSuspended          OrderStatus = "suspended"
	OrderPendingNew         OrderStatus = "pending_new"
	OrderCalculated         OrderStatus = "calculated"
	OrderExpired            OrderStatus = "expired"
	OrderAcceptedForBidding OrderStatus = "accepted_for_bidding"
	OrderPendingReplace     OrderStatus = "pending_replace"
)

var OrderOpen = []OrderStatus{
	OrderAccepted,
	OrderNew,
	OrderPartiallyFilled,
	OrderDoneForDay,
	OrderCalculated,
	OrderPendingNew,
	OrderPendingCancel,
	OrderPendingReplace,
	OrderAcceptedForBidding,
	OrderReplaced,
}

var OrderClosed = []OrderStatus{
	OrderFilled,
	OrderCanceled,
	OrderStopped,
	OrderRejected,
	OrderSuspended,
	OrderExpired,
}

func OrderStatusFromJSON(status string) []OrderStatus {
	switch status {
	case "open":
		return OrderOpen
	case "closed":
		return OrderClosed
	default:
		return nil
	}
}

type TimeInForce string

const (
	Day TimeInForce = "day" // good for day
	GTC TimeInForce = "gtc" // good till cancelled
	OPG TimeInForce = "opg" // at the open
	IOC TimeInForce = "ioc" // immediate or cancel (can partial fill)
	FOK TimeInForce = "fok" // fill or kill (no partial fill)
	GTX TimeInForce = "gtx" // good till crossing
	GTD TimeInForce = "gtd" // good till date
)

// ValidTimeInForce ensures the TimeInForce string is valid
func ValidTimeInForce(tif TimeInForce) bool {
	switch tif {
	case Day:
		fallthrough
	case GTC:
		fallthrough
	case OPG:
		fallthrough
	case IOC:
		fallthrough
	case FOK:
		return true
	default:
		return false
	}
}

type HandlInst string

const (
	AutoNoBroker HandlInst = "auto_no_broker"
	AutoBroker   HandlInst = "auto_broker"
	Manual       HandlInst = "manual"
)

type SettlementType string

const (
	Regular       SettlementType = "regular"
	Cash          SettlementType = "cash"
	NextDay       SettlementType = "next_day"
	TPlus3        SettlementType = "t+3"
	TPlus4        SettlementType = "t+4"
	Future        SettlementType = "future"
	WhenIssued    SettlementType = "when_issued"
	SellersOption SettlementType = "sellers_option"
	TPlus5        SettlementType = "t+5"
)

type SecurityType string

const (
	CommonStock   SecurityType = "common_stock"
	Option        SecurityType = "OPT"
	ComplexOption SecurityType = "MLEG"
)

type OrderCapacity string

const (
	Agency    OrderCapacity = "agency"
	Principal OrderCapacity = "principal"
)

type ExecInst string

const (
	Held                                         ExecInst = "held"
	NotHeld                                      ExecInst = "not_held"
	StayOnOfferSide                              ExecInst = "stay_on_offer_side"
	Work                                         ExecInst = "work"
	GoAlong                                      ExecInst = "go_along"
	OverTheDay                                   ExecInst = "over_the_day"
	ParticipantDontInitiate                      ExecInst = "participant_dont_initiate"
	StrictScale                                  ExecInst = "strict_scale"
	TryToScale                                   ExecInst = "try_to_scale"
	StayOnBidSide                                ExecInst = "stay_on_bid_side"
	NoCross                                      ExecInst = "no_cross"
	OKToCross                                    ExecInst = "ok_to_cross"
	CallFirst                                    ExecInst = "call_first"
	PercentOfVolume                              ExecInst = "percent_of_volume"
	DoNotIncrease                                ExecInst = "do_not_increase"
	DoNotReduce                                  ExecInst = "do_not_reduce"
	AllOrNone                                    ExecInst = "all_or_none"
	ReinstateOnSystemFailure                     ExecInst = "reinstate_on_system_failure"
	InstitutionsOnly                             ExecInst = "institutions_only"
	ReinstateOnTradingHalt                       ExecInst = "reinstate_on_trading_halt"
	CancelOnTradingHalt                          ExecInst = "cancel_on_trading_halt"
	LastPeg                                      ExecInst = "last_peg"
	MidPricePeg                                  ExecInst = "mid_price_peg"
	NonNegotiable                                ExecInst = "non_negotiable"
	OpeningPeg                                   ExecInst = "opening_peg"
	MarketPeg                                    ExecInst = "market_peg"
	CancelOnSystemFailure                        ExecInst = "cancel_on_system_failure"
	PrimaryPeg                                   ExecInst = "primary_peg"
	Suspend                                      ExecInst = "suspend"
	FixedPegToLocalBestBidOrOfferAtTimeOfOrder   ExecInst = "fixed_peg_to_local_best_bid_or_offer_at_time_of_order"
	CustomerDisplayInstruction                   ExecInst = "customer_display_instruction"
	Netting                                      ExecInst = "netting"
	PegToSwap                                    ExecInst = "peg_to_swap"
	TradeAlong                                   ExecInst = "trade_along"
	TryToStop                                    ExecInst = "try_to_stop"
	CancelIfNotBest                              ExecInst = "cancel_if_not_best"
	TrailingStopPeg                              ExecInst = "trailing_stop_peg"
	StrictLimit                                  ExecInst = "strict_limit"
	IgnorePriceValidityChecks                    ExecInst = "ignore_price_validity_checks"
	PegToLimitPrice                              ExecInst = "peg_to_limit_price"
	WorkToTargetStrategy                         ExecInst = "work_to_target_strategy"
	InterMarketSweep                             ExecInst = "inter_market_sweep"
	ExternalRoutingAllowed                       ExecInst = "external_routing_allowed"
	ExternalRoutingNotAllowed                    ExecInst = "external_routing_not_allowed"
	ImbalanceOnly                                ExecInst = "imbalance_only"
	SingleExecutionRequestedForBlockTrade        ExecInst = "single_execution_requested_for_block_trade"
	BestExecution                                ExecInst = "best_execution"
	SuspendOnSystemFailure                       ExecInst = "suspend_on_system_failure"
	SuspendOnTradingHalt                         ExecInst = "suspend_on_trading_halt"
	ReinstateOnConnectionLoss                    ExecInst = "reinstate_on_connection_loss"
	CancelOnConnectionLoss                       ExecInst = "cancel_on_connection_loss"
	SuspendOnConnectionLoss                      ExecInst = "suspend_on_connection_loss"
	ReleaseFromSuspension                        ExecInst = "release_from_suspension"
	ExecuteAsDeltaNeutralUsingVolatilityProvided ExecInst = "executed_as_delta_neutral_using_volatility_provided"
	ExecuteAsDurationNeutral                     ExecInst = "execute_as_duration_neutral"
	ExecuteAsFXNeutral                           ExecInst = "execute_as_fx_neutral"
)

// Executions
type ExecutionType string

const (
	ExecutionNew            ExecutionType = "new"
	ExecutionPartialFill    ExecutionType = "partial_fill"
	ExecutionFill           ExecutionType = "fill"
	ExecutionDoneForDay     ExecutionType = "done_for_day"
	ExecutionCanceled       ExecutionType = "canceled"
	ExecutionReplaced       ExecutionType = "replaced"
	ExecutionPendingCancel  ExecutionType = "pending_cancel"
	ExecutionStopped        ExecutionType = "stopped"
	ExecutionRejected       ExecutionType = "rejected"
	ExecutionSuspended      ExecutionType = "suspended"
	ExecutionPendingNew     ExecutionType = "pending_new"
	ExecutionCalculated     ExecutionType = "calculated"
	ExecutionExpired        ExecutionType = "expired"
	ExecutionRestated       ExecutionType = "restated"
	ExecutionPendingReplace ExecutionType = "pending_replace"
)

type MarginCallType string

const (
	ConcentratedMaintenance MarginCallType = "concentrated_maintenance"
	DayTrading              MarginCallType = "day_trading"
	EquityMaintenance       MarginCallType = "equity_maintenance"
	GoodFaithViolations     MarginCallType = "good_faith_violations"
	GoodFaithWarnings       MarginCallType = "good_faith_warnings"
	JBOEquity               MarginCallType = "jbo_equity"
	LeverageMaintenance     MarginCallType = "leverage_maintenance"
	MoneyDue                MarginCallType = "money_due"
	RegulationMT            MarginCallType = "regulation_mt"
	PortfolioEquity         MarginCallType = "portfolio_equity"
	PortfolioMargin         MarginCallType = "portfolio_margin"
	RequiredMaintenanceCall MarginCallType = "required_maintentance_call"
	RegulationT             MarginCallType = "regulation_t"
	Type1Shorts             MarginCallType = "type_1_shorts"
)

type DividendPositionQuantityLongOrShort string

var (
	DividendLong                  DividendPositionQuantityLongOrShort = "L"
	DividendPositionQuantityShort DividendPositionQuantityLongOrShort = "S"
)

type CorporateActionType string

const (
	SymbolChange  CorporateActionType = "symbol_change"
	Reorg         CorporateActionType = "reorg"
	StockSplit    CorporateActionType = "stock_split"
	Spinoff       CorporateActionType = "spinoff"
	MarketChange  CorporateActionType = "market_change"
	StockMerger   CorporateActionType = "stock_merger"
	ReverseSplit  CorporateActionType = "reverse_split"
	NameChange    CorporateActionType = "name_change"
	CashMerger    CorporateActionType = "cash_merger"
	CusipChange   CorporateActionType = "cusip_change"
	StockDividend CorporateActionType = "stock_dividend"
)

type SoDCorporateAction string

const (
	SoDSymbolChange  SoDCorporateAction = "Symbol Change"
	SoDReorg         SoDCorporateAction = "Reorg"
	SoDStockSplit    SoDCorporateAction = "Stock Split"
	SoDSpinoff       SoDCorporateAction = "Spinoff"
	SoDMarketChange  SoDCorporateAction = "Market Change"
	SoDStockMerger   SoDCorporateAction = "Stock Merger"
	SoDReverseSplit  SoDCorporateAction = "Reverse Split"
	SoDNameChange    SoDCorporateAction = "Name Change"
	SoDCashMerger    SoDCorporateAction = "Cash Merger"
	SoDCusipChange   SoDCorporateAction = "Cusip Change"
	SoDStockDividend SoDCorporateAction = "Stock Dividend"
)

func CorporateActionTypeFromSoD(t SoDCorporateAction) CorporateActionType {
	switch t {
	case SoDSymbolChange:
		return SymbolChange
	case SoDReorg:
		return Reorg
	case SoDStockSplit:
		return StockSplit
	case SoDSpinoff:
		return Spinoff
	case SoDMarketChange:
		return MarketChange
	case SoDStockMerger:
		return StockMerger
	case SoDReverseSplit:
		return ReverseSplit
	case SoDNameChange:
		return NameChange
	case SoDCashMerger:
		return CashMerger
	case SoDCusipChange:
		return CusipChange
	case SoDStockDividend:
		return StockDividend
	}

	return ""
}

type TransferType string

const (
	ACH  TransferType = "ach"
	Wire TransferType = "wire"
)

type RelationshipStatus string

const (
	RelationshipQueued   RelationshipStatus = "QUEUED"   // internal for queueing
	RelationshipPending  RelationshipStatus = "PENDING"  // apex
	RelationshipApproved RelationshipStatus = "APPROVED" // apex
	RelationshipCanceled RelationshipStatus = "CANCELED" // apex
)

type TransferStatus string

const (
	TransferQueued          TransferStatus = "QUEUED"           // internal for queueing
	TransferApprovalPending TransferStatus = "APPROVAL_PENDING" // internal for risky transfers
	TransferPending         TransferStatus = "PENDING"          // apex
	TransferRejected        TransferStatus = "REJECTED"         // apex
	TransferCanceled        TransferStatus = "CANCELED"         // apex
	TransferApproved        TransferStatus = "APPROVED"         // apex
	TransferFundsPosted     TransferStatus = "FUNDS_POSTED"     // apex
	TransferSentToBank      TransferStatus = "SENT_TO_BANK"     // apex
	TransferComplete        TransferStatus = "COMPLETE"         // apex
	TransferReturned        TransferStatus = "RETURNED"         // apex
	TransferPendingPrinted  TransferStatus = "PENDING_PRINTING" // apex
	TransferVoid            TransferStatus = "VOID"             // apex
	TransferStopPayment     TransferStatus = "STOP_PAYMENT"     // apex
)

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

type AffiliateType string

const (
	ControlledFirm AffiliateType = "CONTROLLED_FIRM"
	FinraFirm      AffiliateType = "FINRA_FIRM"
)

type AssetStatus string

const (
	AssetActive   AssetStatus = "active"
	AssetInactive AssetStatus = "inactive"
)

type AssetClass string

const (
	AssetClassUSEquity AssetClass = "us_equity"
)

type AccessKeyStatus string

var (
	AccessKeyActive   AccessKeyStatus = "ACTIVE"
	AccessKeyDisabled AccessKeyStatus = "DISABLED"
)
