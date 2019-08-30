package main

import (
	"os"

	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/encryption"
	"github.com/alpacahq/gopaca/env"
)

func main() {
	// SSN is in the form of xxx-xx-xxxx
	ssn := os.Args[1]
	// ownerID is owner_details.owner_id
	ownerID := os.Args[2]

	var hash []byte
	hash, err := encryption.EncryptWithKey([]byte(ssn), []byte(env.GetVar("BROKER_SECRET")))
	if err != nil {
		panic(err)
	}

	q := db.DB().Exec(`UPDATE owner_details SET hash_ssn = ? WHERE owner_id = ? AND replaced_by IS NULL`, hash, ownerID)

	if q.Error != nil {
		panic(q.Error)
	}
}
