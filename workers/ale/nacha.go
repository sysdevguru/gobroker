package ale

import (
	"log"

	"github.com/gocarina/gocsv"
)

func init() {
	var codes []NachaCode
	if err := gocsv.UnmarshalString(nachaCsv, &codes); err != nil {
		log.Fatal("failed to unmarshal nacha codes", "error", err)
	}

	nachaCodes = make(map[string]string)
	for _, code := range codes {
		nachaCodes[code.Code] = code.NacDescription
	}
}

var nachaCodes map[string]string

type NachaCode struct {
	Code           string `csv:"Code"`
	SenDescription string `csv:"Sentinel Text Description"`
	NacDescription string `csv:"Full Description from NACHA Rules"`
}

const nachaCsv = `Code,Sentinel Text Description,Full Description from NACHA Rules,,
R01,Insufficient Funds,Insufficient Funds,,
R02,Account Closed,Account Closed,,
R03,No Account/Unable to Locate Account,"The account number structure is valid and it passes the check digit validation, but the account number does not correspond to the individual identified in the entry, or the account number designated is not anexisting account. (Note: This Return Reason Code may not be used to return ARC entries, BOC entries,or POP entries solely because they do not contain the Receiver’s name in the Individual Name/ReceivingCompany Name Field.)",,
R04,Invalid Account Number,Invalid Account Number,,
R05,Unauthorized Debit to Consumer Account,Unauthorized Debit to Consumer Account Using Corporate SEC Code (adjustment entries),,
R07,Authorization Revoked by Customer,Authorization Revoked by Customer (adjustment entries),,
R08,Payment Stopped,Payment Stopped,,
R09,Uncollected Funds,Uncollected Funds,,
R10,Customer Advises Not Authorized,"Customer Advises Not Authorized, Notice Not Provided, Improper Source Document, or Amount of Entry Not Accurately Obtained from Source Document (adjustment entries)",,
R11,Check Truncation Entry Return,Check Truncation Entry Return (Specify),,
R12,Account Sold to Another DFI,Account Sold to Another DFI,,
R13,Invalid ACH Routing Number,Invalid ACH Routing Number,,
R14,Representative Payee Deceased or Unable to Continue ,Representative Payee Deceased or Unable to Continue in that Capacity,,
R15,Beneficiary or Account Holder Deceased,Beneficiary or Account Holder (Other Than a Representative Payee) Deceased,,
R16,Account Frozen,Account Frozen,,
R17,File Record Edit Criteria,File Record Edit Criteria (Specify),,
R18,Improper Effective Entry Date,Improper Effective Entry Date,,
R19,Amount Field Error,Amount Field Error,,
R20,NonTransaction Account,NonTransaction Account,,
R21,Invalid Company Identification,Invalid Company Identification,,
R22,Invalid Individual ID Number,Invalid Individual ID Number,,
R23,Credit Entry Refused by Receiver,Credit Entry Refused by Receiver,,
R24,Duplicate Entry,Duplicate Entry,,
R25,Addenda Error,Addenda Error,,
R26,Mandatory Field Error,Mandatory Field Error,,
R27,Trace Number Error,Trace Number Error,,
R28,Routing Number Check Digit Error,Routing Number Check Digit Error,,
R29,Corporate Customer Advises Not Authorized,Corporate Customer Advises Not Authorized,,
R30,RDFI Not Participant in Check Truncation Program,RDFI Not Participant in Check Truncation Program,,
R31,Permissible Return Entry,Permissible Return Entry (CCD and CTX only),,
R32,RDFI NonSettlement,RDFI NonSettlement,,
R33,Return of XCK Entry,Return of XCK Entry,,
R34,Limited Participation DFI,Limited Participation DFI,,
R35,Return of Improper Debit Entry,Return of Improper Debit Entry,,
R36,Return of Improper Credit Entry,Return of Improper Credit Entry,,
R37,Source Document Presented for Payment,Source Document Presented for Payment,,
R38,Stop Payment on Source Document,Stop Payment on Source Document,,
R39,Improper Source Document,Improper Source Document,,
R40,Return of ENR Entry by Federal Government Agency,Return of ENR Entry by Federal Government Agency (ENR only),,
R41,Invalid Transaction Code,Invalid Transaction Code (ENR only),,
R42,Routing Number/Check Digit Error,Routing Number/Check Digit Error (ENR only),,
R43,Invalid DFI Account Number,Invalid DFI Account Number (ENR only),,
R44,Invalid Individual ID Number/ Identification Number,Invalid Individual ID Number/ Identification Number (ENR only),,
R45,Invalid Individual Name/Company Name,Invalid Individual Name/Company Name (ENR only),,
R46,Invalid Representative Payee Indicator,Invalid Representative Payee Indicator (ENR only),,
R47,Duplicate Enrollment,Duplicate Enrollment (ENR only),,
R50,State Law Affecting RCK Acceptance,State Law Affecting RCK Acceptance,,
R51,Item is Ineligible,"Item is Ineligible, Notice Not Provided, Signature Not Genuine, Item Altered, or Amount of Entry Not Accurately Obtained from Item (adjustment entries)",,
R52,Stop Payment on Item,Stop Payment on Item (adjustment entries),,
R53,Item and ACH Entry Presented for Payment,Item and ACH Entry Presented for Payment (adjustment entries),,
R61,Misrouted Return,Misrouted Return,,
R67,Duplicate Return,Duplicate Return,,
R68,Untimely Return,Untimely Return,,
R69,Field Error(s),Field Error(s),,
R70,Permissible Return Entry Not Accepted/ Return Not Requested by ODFI,Permissible Return Entry Not Accepted/ Return Not Requested by ODFI,,
R71,Misrouted Dishonored Return,Misrouted Dishonored Return,,
R72,Untimely Dishonored Return,Untimely Dishonored Return,,
R73,Timely Original Return,Timely Original Return,,
R74,Corrected Return,Corrected Return,,
R75,Original Return Not a Duplicate,Original Return Not a Duplicate,,
R76,No Errors Found,No Errors Found,,
R80,CrossBorder,CrossBorder,,
R81,NonParticipant in CrossBorder Program,NonParticipant in CrossBorder Program [Non-Partipant in IAT Program (for Gateway Operator use only)],,
R83,Foreign Receiving DFI Unable to Settle,Foreign Receiving DFI Unable to Settle [(for Gateway Operator use only)],,`
