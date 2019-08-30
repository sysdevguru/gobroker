package entities

type PolyAuthRequest struct {
	APIKeyID string `json:"api_key_id"`
}

type PolySubscriberEntity struct {
	Email        string `json:"email"`
	UserID       string `json:"user_id"`
	FullName     string `json:"full_name"`
	Professional bool   `json:"professional"`
	Address      string `json:"address"`
}

type PolySubscriberRequest struct {
	APIKeyIDs []string `json:"api_key_ids"`
}
