package apex

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/alpacahq/apex/encryption"
	"github.com/shopspring/decimal"
)

var sentinelPath = "/sentinel/api/v1/"

type ACHRelationship struct {
	Account              string `json:"account"`
	BankRoutingNumber    string `json:"bankRoutingNumber"`
	BankAccount          string `json:"bankAccount"`
	BankAccountOwnerName string `json:"bankAccountOwnerName"`
	BankAccountType      string `json:"bankAccountType"`
	Nickname             string `json:"nickname"`
	ApprovalMethod       string `json:"approvalMethod"`
}

type GetRelationshipResponse struct {
	Account              *string `json:"account"`
	BankRoutingNumber    *string `json:"bankRoutingNumber"`
	BankAccount          *string `json:"bankAccount"`
	BankAccountOwnerName *string `json:"bankAccountOwnerName"`
	BankAccountType      *string `json:"bankAccountType"`
	Nickname             *string `json:"nickname"`
	ApprovalMethod       *string `json:"approvalMethod"`
	ID                   *string `json:"id"`
	Status               *string `json:"status"`
	Cancellation         *struct {
		Comment          *string `json:"comment"`
		Reason           *string `json:"reason"`
		CancellationTime *string `json:"cancellationTime"`
	} `json:"cancellation"`
	Approval *struct {
		ApprovalTime *string `json:"approvalTime"`
		ApprovedBy   *struct {
			UserName   *string `json:"userName"`
			UserEntity *string `json:"userEntity"`
			UserClass  *string `json:"userClass"`
		} `json:"approvedBy"`
	} `json:"approval"`
	CreationTime *string `json:"creationTime"`
}

type CreateRelationshipResponse struct {
	Account              *string `json:"account"`
	BankRoutingNumber    *string `json:"bankRoutingNumber"`
	BankAccount          *string `json:"bankAccount"`
	BankAccountOwnerName *string `json:"bankAccountOwnerName"`
	BankAccountType      *string `json:"bankAccountType"`
	Nickname             *string `json:"nickname"`
	ApprovalMethod       *string `json:"approvalMethod"`
	ID                   *string `json:"id"`
	Status               *string `json:"status"`
	Cancellation         *struct {
		Comment          *string `json:"comment"`
		Reason           *string `json:"reason"`
		CancellationTime *string `json:"cancellationTime"`
	} `json:"cancellation"`
	Approval *struct {
		ApprovalTime *string `json:"approvalTime"`
		ApprovedBy   *struct {
			UserName   *string `json:"userName"`
			UserEntity *string `json:"userEntity"`
			UserClass  *string `json:"userClass"`
		} `json:"approvedBy"`
	} `json:"approval"`
	CreationTime *string `json:"creationTime"`
}

type CancelRelationshipParams struct {
	Comment string `json:"comment"`
}

type CancelRelationshipResponse struct {
	Account              *string `json:"account"`
	BankRoutingNumber    *string `json:"bankRoutingNumber"`
	BankAccount          *string `json:"bankAccount"`
	BankAccountOwnerName *string `json:"bankAccountOwnerName"`
	BankAccountType      *string `json:"bankAccountType"`
	Nickname             *string `json:"nickname"`
	ApprovalMethod       *string `json:"approvalMethod"`
	ID                   *string `json:"id"`
	Status               *string `json:"status"`
	Cancellation         *struct {
		Comment          *string `json:"comment"`
		Reason           *string `json:"reason"`
		CancellationTime *string `json:"cancellationTime"`
	} `json:"cancellation"`
	Approval *struct {
		ApprovalTime *string `json:"approvalTime"`
		ApprovedBy   *struct {
			UserName   *string `json:"userName"`
			UserEntity *string `json:"userEntity"`
			UserClass  *string `json:"userClass"`
		} `json:"approvedBy"`
	} `json:"approval"`
	CreationTime *string `json:"creationTime"`
}

func (a *Apex) GetRelationship(id string) (*GetRelationshipResponse, error) {
	m := GetRelationshipResponse{}
	uri := fmt.Sprintf(
		"%v%vach_relationships/%v",
		os.Getenv("APEX_URL"),
		sentinelPath,
		id,
	)
	if _, err := a.getJSON(uri, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (a *Apex) CreateRelationship(relationship ACHRelationship) (*CreateRelationshipResponse, error) {
	m := CreateRelationshipResponse{}
	if !a.Dev {
		uri := fmt.Sprintf(
			"%v%vach_relationships",
			os.Getenv("APEX_URL"),
			sentinelPath,
		)
		if _, err := a.call(uri, "POST", relationship, &m); err != nil {
			return nil, err
		}
	} else {
		id := encryption.GenRandomKey(20)
		m.ID = &id
		pending := string(ACHPending)
		m.Status = &pending
	}
	return &m, nil
}

func (a *Apex) CancelRelationship(id string, reason string) (*CancelRelationshipResponse, error) {
	m := CancelRelationshipResponse{}
	if !a.Dev {
		uri := fmt.Sprintf(
			"%v%vach_relationships/%v/cancel",
			os.Getenv("APEX_URL"),
			sentinelPath,
			id,
		)
		body := CancelRelationshipParams{Comment: reason}
		if _, err := a.call(uri, "POST", body, &m); err != nil {
			return nil, err
		}
	} else {
		canceled := string(ACHCanceled)
		m.Status = &canceled
	}

	return &m, nil
}

type ApproveRelationshipResponse struct {
	Account              *string `json:"account"`
	BankRoutingNumber    *string `json:"bankRoutingNumber"`
	BankAccount          *string `json:"bankAccount"`
	BankAccountOwnerName *string `json:"bankAccountOwnerName"`
	BankAccountType      *string `json:"bankAccountType"`
	Nickname             *string `json:"nickname"`
	ApprovalMethod       *string `json:"approvalMethod"`
	ID                   *string `json:"id"`
	Status               *string `json:"status"`
	Cancellation         *struct {
		Comment          *string `json:"comment"`
		Reason           *string `json:"reason"`
		CancellationTime *string `json:"cancellationTime"`
	} `json:"cancellation"`
	Approval *struct {
		ApprovalTime *string `json:"approvalTime"`
		ApprovedBy   *struct {
			UserName   *string `json:"userName"`
			UserEntity *string `json:"userEntity"`
			UserClass  *string `json:"userClass"`
		} `json:"approvedBy"`
	} `json:"approval"`
	CreationTime *string `json:"creationTime"`
}

type ApproveRelationshipParams struct {
	Method    string          `json:"method"`
	AmountOne decimal.Decimal `json:"amount1"`
	AmountTwo decimal.Decimal `json:"amount2"`
}

var ErrInvalidAmounts = fmt.Errorf("micro deposit amounts do not match")
var ErrExhausted = fmt.Errorf("bank relationship exhausted")

type MicroDepositAmounts []decimal.Decimal

// Confirm the Micro Deposits
func (a *Apex) ApproveRelationship(id string, amounts MicroDepositAmounts) (*ApproveRelationshipResponse, error) {
	if len(amounts) != 2 {
		return nil, fmt.Errorf("not the correct amount of arguments")
	}
	m := ApproveRelationshipResponse{}
	if !a.Dev {
		uri := fmt.Sprintf(
			"%v%vach_relationships/%v/approve",
			os.Getenv("APEX_URL"),
			sentinelPath,
			id,
		)
		body := ApproveRelationshipParams{
			Method:    "MICRO_DEPOSIT",
			AmountOne: amounts[0],
			AmountTwo: amounts[1],
		}
		if resp, err := a.call(uri, "POST", body, &m); err != nil {
			if resp.StatusCode() == http.StatusBadRequest {
				return nil, ErrInvalidAmounts
			}
			return nil, err
		}
	} else {
		m.ID = &id
		pending := string(ACHApproved)
		m.Status = &pending
	}

	return &m, nil
}

// Reissue micro deposits
func (a *Apex) ReissueMicroDeposits(id string) error {
	if !a.Dev {
		uri := fmt.Sprintf(
			"%v%vach_relationships/%v/reissue",
			os.Getenv("APEX_URL"),
			sentinelPath,
			id,
		)
		_, err := a.call(uri, "POST", nil, nil)
		return err
	}

	return nil
}

type ACHTransfer struct {
	ID               string          `json:"externalTransferId"`
	Amount           decimal.Decimal `json:"amount"`
	RelationshipID   string          `json:"achRelationshipId"`
	DisbursementType Disbursement    `json:"disbursementType,omitempty"`
}

type TransferResponse struct {
	TransferID                  *string          `json:"transferId"`
	ExternalTransferID          *string          `json:"externalTransferId"`
	Mechanism                   *string          `json:"mechanism"`
	Direction                   *string          `json:"direction"`
	State                       *string          `json:"state"`
	RejectReason                *string          `json:"rejectReason"`
	DisbursementType            *string          `json:"disbursementType,omitempty"`
	RequestedAmount             *decimal.Decimal `json:"requestedAmount,omitempty"`
	Amount                      *decimal.Decimal `json:"amount"`
	EstimatedFundsAvailableDate *string          `json:"estimatedFundsAvailableDate,omitempty"`
	AchRelationshipID           *string          `json:"achRelationshipId"`
	IraDetails                  *struct {
		ContributionType      *string `json:"contributionType,omitempty"`
		ContributionYear      *int    `json:"contributionYear,omitempty"`
		DistributionReason    *string `json:"distributionReason,omitempty"`
		FederalTaxWithholding *struct {
			ValueType *string `json:"valueType,omitempty"`
			Value     *int    `json:"value,omitempty"`
		} `json:"federalTaxWithholding,omitempty"`
		StateTaxWithholding *struct {
			ValueType *string `json:"valueType,omitempty"`
			Value     *int    `json:"value,omitempty"`
		} `json:"stateTaxWithholding,omitempty"`
		ReceivingInstitutionName *string `json:"receivingInstitutionName,omitempty"`
	} `json:"iraDetails"`
	Fees []struct {
		Type                   *string          `json:"type,omitempty"`
		CustomerDebitAmount    *decimal.Decimal `json:"customerDebitAmount,omitempty"`
		FirmCreditAmount       *decimal.Decimal `json:"firmCreditAmount,omitempty"`
		CorrespondentNetAmount *decimal.Decimal `json:"correspondentNetAmount,omitempty"`
	} `json:"fees,omitempty"`
}

type CancelTransferRequest struct {
	Comment string `json:"comment"`
}

type CancelTransferResponse struct {
	TransferID         *string `json:"transferId"`
	ExternalTransferID *string `json:"externalTransferId"`
	Mechanism          *string `json:"mechanism"`
	Direction          *string `json:"direction"`
	State              *string `json:"state"`
	RejectReason       *string `json:"rejectReason"`
}

type TransferStatusResponse struct {
	TransferID                  *string          `json:"transferId"`
	ExternalTransferID          *string          `json:"externalTransferId"`
	Mechanism                   *string          `json:"mechanism"`
	Direction                   *string          `json:"direction"`
	State                       *string          `json:"state"`
	Amount                      *decimal.Decimal `json:"amount"`
	EstimatedFundsAvailableDate *string          `json:"estimatedFundsAvailableDate"`
	AchRelationshipID           *string          `json:"achRelationshipId"`
	Fees                        []struct {
		Type                   *string          `json:"type,omitempty"`
		CustomerDebitAmount    *decimal.Decimal `json:"customerDebitAmount,omitempty"`
		FirmCreditAmount       *decimal.Decimal `json:"firmCreditAmount,omitempty"`
		CorrespondentNetAmount *decimal.Decimal `json:"correspondentNetAmount,omitempty"`
	}
}

type AmountAvailableResponse struct {
	Total                          *int             `json:"total"`
	UnAdjustedTotal                *decimal.Decimal `json:"unAdjustedTotal"`
	StartDayCashAvailable          *decimal.Decimal `json:"startDayCashAvailable"`
	PendingDebitInterest           *decimal.Decimal `json:"pendingDebitInterest"`
	PendingDebitDividends          *decimal.Decimal `json:"pendingDebitDividends"`
	IncludeFullyPaidUnsettledFunds *bool            `json:"includeFullyPaidUnsettledFunds"`
	FullyPaidUnsettledFunds        *decimal.Decimal `json:"fullyPaidUnsettledFunds"`
	TotalDisbursements             *decimal.Decimal `json:"totalDisbursements"`
	TotalDeposits                  *decimal.Decimal `json:"totalDeposits"`
	Disbursements                  []struct {
		TransferID               *int             `json:"transferId"`
		Mechanism                *string          `json:"mechanism"`
		State                    *string          `json:"state"`
		RequestedAmount          *decimal.Decimal `json:"requestedAmount"`
		FeeAmount                *decimal.Decimal `json:"feeAmount"`
		TotalCustomerDebitAmount *decimal.Decimal `json:"totalCustomerDebitAmount"`
		TransferTime             *string          `json:"transferTime"`
		FundsPostedDate          *string          `json:"fundsPostedDate"`
	} `json:"disbursements"`
	RecentDeposits []struct {
		Account     *string          `json:"account"`
		Amount      *decimal.Decimal `json:"amount"`
		Mechanism   *string          `json:"mechanism"`
		Description *string          `json:"description"`
		ProcessDate *string          `json:"processDate"`
	} `json:"recentDeposits"`
}

type Disbursement string

const (
	DisbursementPartial Disbursement = "PARTIAL_BALANCE"
	DisbursementFull    Disbursement = "FULL_BALANCE"
)

func (a *Apex) Transfer(direction TransferDirection, transfer ACHTransfer) (*TransferResponse, error) {
	m := TransferResponse{}
	if !a.Dev {
		uri := fmt.Sprintf(
			"%v%vtransfers/achs/%v",
			os.Getenv("APEX_URL"),
			sentinelPath,
			strings.ToLower(string(direction)),
		)
		if direction == Outgoing {
			// may be others, but nothing else is listed in docs
			transfer.DisbursementType = "PARTIAL_BALANCE"
		}
		if _, err := a.call(uri, "POST", transfer, &m); err != nil {
			return nil, err
		}
	} else {
		key := encryption.GenRandomKey(20)
		fundDate := time.Now().Add(36 * time.Hour).Format("2006-01-02")
		transPending := "PENDING"
		m = TransferResponse{
			TransferID:                  &key,
			EstimatedFundsAvailableDate: &fundDate,
			State:                       &transPending}
	}
	return &m, nil
}

func (a *Apex) CancelTransfer(id string, comment string) (*CancelTransferResponse, error) {
	uri := fmt.Sprintf(
		"%v%vtransfers/achs/%v/cancel",
		os.Getenv("APEX_URL"),
		sentinelPath,
		id,
	)
	reqBody := CancelTransferRequest{Comment: comment}
	m := CancelTransferResponse{}
	if _, err := a.call(uri, "POST", reqBody, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (a *Apex) TransferStatus(id string) (*TransferStatusResponse, error) {
	uri := fmt.Sprintf(
		"%v%vtransfers/achs/%v",
		os.Getenv("APEX_URL"),
		sentinelPath,
		id,
	)
	m := TransferStatusResponse{}
	if _, err := a.getJSON(uri, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (a *Apex) AmountAvailable(accountId string) (*AmountAvailableResponse, error) {
	uri := fmt.Sprintf(
		"%v%vamount_available/%v",
		os.Getenv("APEX_URL"),
		sentinelPath,
		accountId,
	)
	m := AmountAvailableResponse{}
	if _, err := a.getJSON(uri, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (a *Apex) SimulateTransferApproval(transferId string) error {
	uri := fmt.Sprintf(
		"%v%vtest/simulation/achs/%v/approve",
		os.Getenv("APEX_URL"),
		sentinelPath,
		transferId,
	)
	if _, err := a.call(uri, "POST", nil, nil); err != nil {
		return err
	}
	return nil
}

func (a *Apex) SimulateTransferRejection(transferId string) error {
	uri := fmt.Sprintf(
		"%v%vtest/simulation/achs/%v/approve",
		os.Getenv("APEX_URL"),
		sentinelPath,
		transferId,
	)
	if _, err := a.call(uri, "POST", nil, nil); err != nil {
		return err
	}
	return nil
}

type NOCRequest struct {
	NewAccountNumber   string `json:"newAccountNumber"`
	NewRoutingNumber   string `json:"newRoutingNumber"`
	NewBankAccountType string `json:"newBankAccountType"`
}

func (a *Apex) SimulateAchNOC(relationshipId string, noc NOCRequest) error {
	uri := fmt.Sprintf(
		"%v%vtest/simulation/achs/%v/noc",
		os.Getenv("APEX_URL"),
		sentinelPath,
		relationshipId,
	)
	if _, err := a.call(uri, "POST", noc, nil); err != nil {
		return err
	}
	return nil
}

func (a *Apex) SimulateAchReturn(relationshipId string, cancelRelationship bool) error {
	uri := fmt.Sprintf(
		"%v%vtest/simulation/achs/%v/return",
		os.Getenv("APEX_URL"),
		sentinelPath,
		relationshipId,
	)
	m := map[string]interface{}{"cancelRelationship": cancelRelationship}
	if _, err := a.call(uri, "POST", m, nil); err != nil {
		return err
	}
	return nil
}

func (a *Apex) SimulateMicroDepositAmount(relationshipID string) (MicroDepositAmounts, error) {
	m := MicroDepositAmounts{}
	uri := fmt.Sprintf(
		"%v%vtest/simulation/micro_deposits/%v",
		os.Getenv("APEX_URL"),
		sentinelPath,
		relationshipID,
	)
	_, err := a.getJSON(uri, &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}
