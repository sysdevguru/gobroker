package slack

import (
	"encoding/json"
	"fmt"

	"github.com/alpacahq/gopaca/log"
	sl "github.com/ashwanthkumar/slack-go-webhook"
)

type Message struct {
	prod *channel
	stg  *channel
	body interface{}
}

func (m *Message) SetBody(body interface{}) {
	m.body = body
}

func (m *Message) FormatBody() string {
	switch v := m.body.(type) {
	case string:
		return v
	default:
		buf, _ := json.MarshalIndent(v, "", "\t")
		return fmt.Sprintf("```%s```", string(buf))
	}
}

func (m *Message) SendStaging() {
	if m.stg == nil {
		return
	}

	errors := sl.Send(
		m.stg.webhook,
		"", sl.Payload{
			Text:     m.FormatBody(),
			Channel:  m.stg.name,
			Username: m.stg.user,
		})

	if len(errors) > 0 {
		log.Error("slack send errors", "errors", errors)
	}
}

func (m *Message) SendProduction() {
	if m.prod == nil {
		return
	}

	errors := sl.Send(
		m.prod.webhook,
		"", sl.Payload{
			Text:     m.FormatBody(),
			Channel:  m.prod.name,
			Username: m.prod.user,
		})

	if len(errors) > 0 {
		log.Error("slack send errors", "errors", errors)
	}
}

type channel struct {
	name    string
	user    string
	webhook string
}

func NewBatchStatus() Message {
	return Message{
		prod: &channel{
			webhook: "https://hooks.slack.com/services/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",
			name:    "#batch-proc-status",
			user:    "Production Batch",
		},
		stg: &channel{
			webhook: "https://hooks.slack.com/services/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",
			name:    "#batch-proc-status-stg",
			user:    "Staging Batch",
		},
	}
}

func NewMailFailure() Message {
	return Message{
		prod: &channel{
			webhook: "https://hooks.slack.com/services/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",
			name:    "#mail-failures",
			user:    "Mail Failures",
		},
		stg: &channel{
			webhook: "https://hooks.slack.com/services/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",
			name:    "#mail-failures-stg",
			user:    "Staging Mail Failures",
		},
	}
}

func NewBraggartFailure() Message {
	return Message{
		prod: &channel{
			webhook: "https://hooks.slack.com/services/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",
			name:    "#braggart-failures",
			user:    "Braggart Failures",
		},
		stg: &channel{
			webhook: "https://hooks.slack.com/services/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",
			name:    "#braggart-failures-stg",
			user:    "Staging Braggart Failures",
		},
	}
}

func NewServerError() Message {
	return Message{
		prod: &channel{
			webhook: "https://hooks.slack.com/services/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",
			name:    "#server-errors",
			user:    "Server Errors",
		},
		stg: &channel{
			webhook: "https://hooks.slack.com/services/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",
			name:    "#server-errors-stg",
			user:    "Staging Server Errors",
		},
	}
}

func NewAccountUpdate() Message {
	return Message{
		prod: &channel{
			webhook: "https://hooks.slack.com/services/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",
			name:    "#brokerage-accounts",
			user:    "Broker Accounts",
		},
		stg: nil,
	}
}

func NewFundingActivity() Message {
	return Message{
		prod: &channel{
			webhook: "https://hooks.slack.com/services/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",
			name:    "#funding-activity",
			user:    "Funding Activity",
		},
		stg: nil,
	}
}
