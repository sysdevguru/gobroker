package segment

import (
	"fmt"
	"sync"

	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/utils"
	"github.com/alpacahq/gopaca/env"
	analytics "gopkg.in/segmentio/analytics-go.v3"
)

var (
	once   sync.Once
	client analytics.Client
)

// Track an event with Segment
func Track(e Event) error {
	once.Do(func() {
		client = analytics.New(env.GetVar("SEGMENT_KEY"))
	})

	if utils.Dev() {
		return nil
	}

	return client.Enqueue(e.trackable())
}

// Identify Send Segment information about users
func Identify(account models.Account) error {
	once.Do(func() {
		client = analytics.New(env.GetVar("SEGMENT_KEY"))
	})

	accountOwner := account.PrimaryOwner()

	if accountOwner == nil {
		return fmt.Errorf("account owner is nil")
	}

	traits := analytics.NewTraits().
		SetEmail(accountOwner.Email).
		Set("plan", account.Plan).
		Set("status", account.Status).
		Set("apexApprovalStatus", account.ApexApprovalStatus).
		Set("accountBlocked", account.AccountBlocked).
		Set("tradingBlocked", account.TradingBlocked).
		Set("cash", account.Cash).
		Set("updatedAt", account.UpdatedAt)

	if accountOwner.Details.GivenName != nil {
		traits.SetFirstName(*accountOwner.Details.GivenName)
	}

	if accountOwner.Details.FamilyName != nil {
		traits.SetLastName(*accountOwner.Details.FamilyName)
	}

	if accountOwner.Details.LegalName != nil {
		traits.SetName(*accountOwner.Details.LegalName)
	}

	return client.Enqueue(analytics.Identify{
		UserId: accountOwner.ID,
		Traits: traits,
	})
}
