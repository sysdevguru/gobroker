package segment

import (
	"encoding/json"

	"github.com/gofrs/uuid"
	analytics "gopkg.in/segmentio/analytics-go.v3"
)

type Event struct {
	name       string
	accountID  uuid.UUID
	properties map[string]interface{}
}

func (e *Event) trackable() analytics.Track {
	return analytics.Track{
		Event:      e.name,
		UserId:     e.accountID.String(),
		Properties: e.properties,
	}
}

func (e *Event) String() string {
	buf, _ := json.Marshal(*e)
	return string(buf)
}

func (e *Event) SetAccountID(id uuid.UUID) {
	e.accountID = id
}

func (e *Event) SetProperty(key string, value interface{}) {
	if e.properties == nil {
		e.properties = map[string]interface{}{}
	}
	e.properties[key] = value
}

func NewAccountCreatedEvent() Event {
	return Event{
		name:       "Account Created",
		properties: map[string]interface{}{},
	}
}

func NewAccountUpdatedEvent() Event {
	return Event{
		name:       "Account Updated",
		properties: map[string]interface{}{},
	}
}

func NewBankLinkCanceledEvent() Event {
	return Event{
		name:       "Banking Link Canceled",
		properties: map[string]interface{}{},
	}
}

func NewBankLinkCreatedEvent() Event {
	return Event{
		name:       "Banking Link Created",
		properties: map[string]interface{}{},
	}
}

func NewTransferCreatedEvent() Event {
	return Event{
		name:       "Transfer Created",
		properties: map[string]interface{}{},
	}
}
