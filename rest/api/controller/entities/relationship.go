package entities

// Last 4 are for Micro Deposits, Method is used to determine
// how to build the BankAcctInfo struct
type CreateRelationshipRequest struct {
	PublicToken   string `json:"public_token"`
	AccountID     string `json:"account_id"`
	Method        string `json:"method"`
	NickName      string `json:"nickname"`
	AccountNumber string `json:"account_number"`
	RoutingNumber string `json:"routing_number"`
	AccountType   string `json:"account_type"`
}
