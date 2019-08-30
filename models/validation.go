package models

import "github.com/gofrs/uuid"

var (
	GBID      string
	TestCount int
)

func init() {
	var u uuid.UUID
	var err error
	if u, err = uuid.DefaultGenerator.NewV4(); err != nil {
		panic(err)
	}
	GBID = u.String() // Instance ID for this GoBroker instance
}

type Validation struct {
	Count int
	GBID  string
}
