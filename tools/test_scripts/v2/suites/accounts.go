package suites

// import (
// 	"fmt"
// 	"io/ioutil"
// 	"os"
// 	"time"

// 	"github.com/alpacahq/gobroker/service/admin/investigation"

// 	"github.com/alpacahq/apex"
// 	"github.com/alpacahq/gobroker/models"
// 	"github.com/alpacahq/gobroker/service/account"
// 	"github.com/alpacahq/gobroker/service/affiliate"
// 	"github.com/alpacahq/gobroker/service/docrequest"
// 	"github.com/alpacahq/gobroker/service/ownerdetails"
// 	"github.com/alpacahq/gobroker/service/trustedcontact"
// 	"github.com/alpacahq/gopaca/clock"
// 	"github.com/alpacahq/gopaca/db"
// 	"github.com/gofrs/uuid"
// 	"github.com/lib/pq"
// 	yaml "gopkg.in/yaml.v2"
// )

// type AccountLog struct {
// 	Standard        []uuid.UUID // 25
// 	TrustedContact  []uuid.UUID // 2
// 	ActionRequired  []uuid.UUID // 5
// 	Suspended       []uuid.UUID // 10
// 	FinraAffiliated []uuid.UUID // 3
// }

// func (l *AccountLog) UpdateAccounts() (err error) {
// 	fmt.Print("\nUpdating 20 accounts... ")
// 	for i := 0; i < 20; i++ {
// 		id := l.Standard[i]
// 		tx := db.Begin()
// 		srv := ownerdetails.Service().WithTx(tx)
// 		if _, err := srv.Patch(
// 			id,
// 			map[string]interface{}{
// 				"phone_number": "555-555-5555",
// 			}); err != nil {
// 			tx.Rollback()
// 			fmt.Println(err)
// 			return err
// 		}
// 		tx.Commit()
// 	}
// 	fmt.Print("Done.")
// 	return nil
// }

// func (l *AccountLog) ApproveActionRequired(adminID uuid.UUID) (err error) {
// 	fmt.Print("\nApproving 5 ACTION_REQUIRED accounts... ")
// 	for i := 0; i < len(l.ActionRequired); i++ {
// 		id := l.ActionRequired[i]
// 		tx := db.Begin()
// 		srv := investigation.Service().WithTx(tx)

// 		inv, err := srv.List(id)
// 		if err != nil {
// 			return err
// 		}
// 		if len(inv) != 1 {
// 			return fmt.Errorf("unexpected investigation count [%v] for %v", len(inv), id)
// 		}
// 		if err = srv.Accept(adminID, inv[0].ID); err != nil {
// 			return err
// 		}
// 	}
// 	fmt.Print("Done.")
// 	return
// }

// func (l *AccountLog) SnapAndAppealSuspended() (err error) {
// 	loadSnapSamples()
// 	fmt.Print("\nAppealing 10 SUSPENDED accounts... ")
// 	for i := 0; i < len(l.Suspended); i++ {
// 		id := l.Suspended[i]

// 		invSrv := investigation.Service().WithTx(db.DB())

// 		inv, err := invSrv.List(id)
// 		if err != nil {
// 			return err
// 		}

// 		if len(inv) != 1 {
// 			return fmt.Errorf("unexpected investigation count: %v", len(inv))
// 		}

// 		tx := db.Begin()
// 		docReqSrv := docrequest.Service().WithTx(tx)

// 		if err := docReqSrv.Request(
// 			inv[0].ID,
// 			[]models.DocumentCategory{models.UPIC}); err != nil {
// 			tx.Rollback()
// 			return err
// 		}

// 		fmt.Println("REQUESTED")

// 		tx.Commit()

// 		tx = db.Begin()
// 		docReqSrv = docReqSrv.WithTx(tx)

// 		// upload DL back
// 		if err := docReqSrv.Upload(
// 			id,
// 			dlBack,
// 			models.DriverLicense,
// 			models.Back,
// 			"image/png",
// 			apex.Client()); err != nil {
// 			tx.Rollback()
// 			fmt.Println(err)
// 			return err
// 		}

// 		fmt.Println("UPLOADED BACK")

// 		tx.Commit()

// 		tx = db.Begin()
// 		docReqSrv = docReqSrv.WithTx(tx)

// 		// upload DL front
// 		if err := docReqSrv.Upload(
// 			id,
// 			dlFront,
// 			models.DriverLicense,
// 			models.Front,
// 			"image/png",
// 			apex.Client()); err != nil {
// 			tx.Rollback()
// 			return err
// 		}

// 		fmt.Println("UPLOADED FRONT")

// 		tx.Commit()

// 		// appeal
// 		if err := invSrv.Appeal(Administrator.IDAsUUID(), inv[0].ID); err != nil {
// 			return err
// 		}
// 	}
// 	fmt.Print("Done.")
// 	return nil
// }

// func (l *AccountLog) Snap407Letters() (err error) {
// 	loadSnapSamples()
// 	fmt.Print("\nUploading 3 407 letters... ")
// 	for i := 0; i < len(l.FinraAffiliated); i++ {
// 		id := l.FinraAffiliated[i]

// 		invSrv := investigation.Service().WithTx(db.DB())

// 		inv, err := invSrv.List(id)
// 		if err != nil {
// 			return err
// 		}

// 		if len(inv) != 1 {
// 			return fmt.Errorf("unexpected investigation count: %v", len(inv))
// 		}

// 		tx := db.Begin()
// 		docReqSrv := docrequest.Service().WithTx(tx)

// 		if err := docReqSrv.Request(
// 			inv[0].ID,
// 			[]models.DocumentCategory{models.UPTA}); err != nil {
// 			tx.Rollback()
// 			return err
// 		}

// 		if err := docReqSrv.Upload(
// 			id,
// 			dlBack,
// 			models.Letter407,
// 			"",
// 			"image/png",
// 			apex.Client()); err != nil {
// 			tx.Rollback()
// 			return err
// 		}
// 	}
// 	fmt.Print("Done.")
// 	return nil
// }

// func (l *AccountLog) Verify() (err error) {
// 	verified := false
// 	verifiedAccountLog := AccountLog{}
// 	srv := account.Service().WithTx(db.DB())

// 	start := clock.Now()

// 	fmt.Print("\nVerifying created account states [this can take up to 10 minutes]... ")

// 	for {
// 		if time.Since(start) > 10*time.Minute {
// 			fmt.Println("account log verification timed out")
// 			break
// 		}
// 		// check standard
// 		for i := len(verifiedAccountLog.Standard); i < len(l.Standard); i++ {
// 			id := l.Standard[i]
// 			acct, err := srv.GetByID(id)
// 			if err != nil {
// 				return err
// 			}
// 			if acct.ApexApprovalStatus == models.Complete {
// 				verifiedAccountLog.Standard = append(verifiedAccountLog.Standard, id)
// 			} else {
// 				break
// 			}
// 		}
// 		// check trusted
// 		for i := len(verifiedAccountLog.TrustedContact); i < len(l.TrustedContact); i++ {
// 			id := l.TrustedContact[i]
// 			acct, err := srv.GetByID(id)
// 			if err != nil {
// 				return err
// 			}
// 			if acct.ApexApprovalStatus == models.Complete {
// 				verifiedAccountLog.TrustedContact = append(verifiedAccountLog.TrustedContact, id)
// 			} else {
// 				break
// 			}
// 		}
// 		// check action required
// 		for i := len(verifiedAccountLog.ActionRequired); i < len(l.ActionRequired); i++ {
// 			id := l.ActionRequired[i]
// 			acct, err := srv.GetByID(id)
// 			if err != nil {
// 				return err
// 			}
// 			if acct.ApexApprovalStatus == models.ActionRequired {
// 				verifiedAccountLog.ActionRequired = append(verifiedAccountLog.ActionRequired, id)
// 			} else {
// 				break
// 			}
// 		}
// 		// check suspended
// 		for i := len(verifiedAccountLog.Suspended); i < len(l.Suspended); i++ {
// 			id := l.Suspended[i]
// 			acct, err := srv.GetByID(id)
// 			if err != nil {
// 				return err
// 			}
// 			if acct.ApexApprovalStatus == models.Suspended {
// 				verifiedAccountLog.Suspended = append(verifiedAccountLog.Suspended, id)
// 			} else {
// 				break
// 			}
// 		}
// 		// check finra affiliated
// 		for i := len(verifiedAccountLog.FinraAffiliated); i < len(l.FinraAffiliated); i++ {
// 			id := l.FinraAffiliated[i]
// 			acct, err := srv.GetByID(id)
// 			if err != nil {
// 				return err
// 			}
// 			if acct.ApexApprovalStatus == models.Complete {
// 				verifiedAccountLog.FinraAffiliated = append(verifiedAccountLog.FinraAffiliated, id)
// 			} else {
// 				break
// 			}
// 		}
// 		if len(verifiedAccountLog.FinraAffiliated) == len(l.FinraAffiliated) &&
// 			len(verifiedAccountLog.Standard) == len(l.Standard) &&
// 			len(verifiedAccountLog.ActionRequired) == len(l.ActionRequired) &&
// 			len(verifiedAccountLog.Suspended) == len(l.Suspended) &&
// 			len(verifiedAccountLog.TrustedContact) == len(l.TrustedContact) {
// 			verified = true
// 			break
// 		}
// 	}
// 	if verified {
// 		fmt.Print("Done.")
// 		return nil
// 	}
// 	failureReport := `
// 		Failed to verify account log:
// 		----------------|		Required		|		Verified		|----------------
// 		Standard     	|		%v		|		%v		|----------------
// 		TrustedContact  |		%v		|		%v		|----------------
// 		ActionRequired	|		%v		|		%v		|----------------
// 		Suspended     	|		%v		|		%v		|----------------
// 		FinraAffiliated	|		%v		|		%v		|----------------`
// 	return fmt.Errorf(failureReport,
// 		len(l.Standard),
// 		len(verifiedAccountLog.Standard),
// 		len(l.TrustedContact),
// 		len(verifiedAccountLog.TrustedContact),
// 		len(l.ActionRequired),
// 		len(verifiedAccountLog.ActionRequired),
// 		len(l.Suspended),
// 		len(verifiedAccountLog.Suspended),
// 		len(l.FinraAffiliated),
// 		len(verifiedAccountLog.FinraAffiliated),
// 	)
// }

// func CreateAccounts() (acctLog *AccountLog, err error) {
// 	acctLog = &AccountLog{}

// 	fmt.Print("Creating 25 standard accounts... ")
// 	acctLog.Standard, err = createStandardAccounts()
// 	if err != nil {
// 		return nil, err
// 	}
// 	fmt.Print("Done.")

// 	fmt.Print("\nCreating 2 trustedContact=INCLUDE accounts... ")
// 	acctLog.TrustedContact, err = createTrustedContactAccounts()
// 	if err != nil {
// 		return nil, err
// 	}
// 	fmt.Print("Done.")

// 	fmt.Print("\nCreating 5 ACTION_REQUIRED accounts... ")
// 	acctLog.ActionRequired, err = createActionRequiredAccounts()
// 	if err != nil {
// 		return nil, err
// 	}
// 	fmt.Print("Done.")

// 	fmt.Print("\nCreating 10 SUSPENDED accounts... ")
// 	acctLog.Suspended, err = createSuspendedAccounts()
// 	if err != nil {
// 		return nil, err
// 	}
// 	fmt.Print("Done.")

// 	fmt.Print("\nCreating 3 isAffiliatedExchangeOrFINRA=YES accounts... ")
// 	acctLog.FinraAffiliated, err = createFinraAffiliatedAccounts()
// 	if err != nil {
// 		return nil, err
// 	}
// 	fmt.Print("Done.")

// 	return acctLog, nil
// }

// func createStandardAccounts() ([]uuid.UUID, error) {
// 	acctIDs := []uuid.UUID{}
// 	for i := 0; i < 25; i++ {
// 		// load case data
// 		email := fmt.Sprintf("%v@standard.v2", clock.Now().UnixNano())
// 		password := "apex_checklist_password"

// 		patches := map[string]interface{}{}
// 		fileName := "/go/src/github.com/alpacahq/gobroker/tools/acctloader/case1.yml"
// 		file, err := os.Open(fileName)
// 		if err != nil {
// 			return nil, err
// 		}
// 		yamlData, err := ioutil.ReadAll(file)
// 		if err != nil {
// 			return nil, err
// 		}
// 		if err := yaml.Unmarshal(yamlData, &patches); err != nil {
// 			return nil, err
// 		}

// 		// create account
// 		tx := db.Begin()
// 		acctSrv := account.Service().WithTx(tx)

// 		acct, err := acctSrv.Create(email, password)
// 		if err != nil {
// 			tx.Rollback()
// 			return nil, err
// 		}
// 		tx.Commit()

// 		// patch owner details
// 		tx = db.Begin()
// 		ownerSrv := ownerdetails.Service().WithTx(tx)

// 		if _, err = ownerSrv.Patch(acct.IDAsUUID(), patches); err != nil {
// 			tx.Rollback()
// 			return nil, err
// 		}
// 		tx.Commit()

// 		// store the ID
// 		acctIDs = append(acctIDs, acct.IDAsUUID())
// 	}
// 	return acctIDs, nil
// }

// func createTrustedContactAccounts() ([]uuid.UUID, error) {
// 	acctIDs := []uuid.UUID{}
// 	for i := 0; i < 2; i++ {
// 		// load case data
// 		email := fmt.Sprintf("%v@trustedcontact.v2", clock.Now().UnixNano())
// 		password := "apex_checklist"

// 		patches := map[string]interface{}{}
// 		fileName := "/go/src/github.com/alpacahq/gobroker/tools/acctloader/case1.yml"
// 		file, err := os.Open(fileName)
// 		if err != nil {
// 			return nil, err
// 		}
// 		yamlData, err := ioutil.ReadAll(file)
// 		if err != nil {
// 			return nil, err
// 		}
// 		if err := yaml.Unmarshal(yamlData, &patches); err != nil {
// 			return nil, err
// 		}

// 		// add trusted contact field
// 		patches["include_trusted_contact"] = true

// 		// create account
// 		tx := db.Begin()
// 		acctSrv := account.Service().WithTx(tx)

// 		acct, err := acctSrv.Create(email, password)
// 		if err != nil {
// 			tx.Rollback()
// 			return nil, err
// 		}
// 		tx.Commit()

// 		// add trusted contact to DB
// 		tx = db.Begin()
// 		tcSrv := trustedcontact.Service().WithTx(tx)

// 		trustedEmail := "trusted@contact.com"
// 		trustedPhone := "650-123-1234"

// 		tc := &models.TrustedContact{
// 			AccountID:    acct.ID,
// 			EmailAddress: &trustedEmail,
// 			PhoneNumber:  &trustedPhone,
// 			GivenName:    "Trusted",
// 			FamilyName:   "Contact",
// 		}
// 		tc, err = tcSrv.Create(tc)
// 		if err != nil {
// 			tx.Rollback()
// 			return nil, err
// 		}

// 		tx.Commit()

// 		// patch owner details
// 		tx = db.Begin()
// 		ownerSrv := ownerdetails.Service().WithTx(tx)

// 		if _, err = ownerSrv.Patch(acct.IDAsUUID(), patches); err != nil {
// 			tx.Rollback()
// 			return nil, err
// 		}
// 		tx.Commit()

// 		// store the ID
// 		acctIDs = append(acctIDs, acct.IDAsUUID())
// 	}
// 	return acctIDs, nil
// }

// func createActionRequiredAccounts() ([]uuid.UUID, error) {
// 	acctIDs := []uuid.UUID{}
// 	for i := 0; i < 5; i++ {
// 		// load case data
// 		email := fmt.Sprintf("%v@actionrequired.v2", clock.Now().UnixNano())
// 		password := "apex_checklist"

// 		patches := map[string]interface{}{}
// 		fileName := "/go/src/github.com/alpacahq/gobroker/tools/acctloader/case3.yml"
// 		file, err := os.Open(fileName)
// 		if err != nil {
// 			return nil, err
// 		}
// 		yamlData, err := ioutil.ReadAll(file)
// 		if err != nil {
// 			return nil, err
// 		}
// 		if err := yaml.Unmarshal(yamlData, &patches); err != nil {
// 			return nil, err
// 		}

// 		// create account
// 		tx := db.Begin()
// 		acctSrv := account.Service().WithTx(tx)

// 		acct, err := acctSrv.Create(email, password)
// 		if err != nil {
// 			tx.Rollback()
// 			return nil, err
// 		}
// 		tx.Commit()

// 		// patch owner details
// 		tx = db.Begin()
// 		ownerSrv := ownerdetails.Service().WithTx(tx)

// 		if _, err = ownerSrv.Patch(acct.IDAsUUID(), patches); err != nil {
// 			tx.Rollback()
// 			return nil, err
// 		}
// 		tx.Commit()

// 		// store the ID
// 		acctIDs = append(acctIDs, acct.IDAsUUID())
// 	}
// 	return acctIDs, nil
// }

// func createSuspendedAccounts() ([]uuid.UUID, error) {
// 	acctIDs := []uuid.UUID{}
// 	for i := 0; i < 10; i++ {
// 		// load case data
// 		email := fmt.Sprintf("%v@actionrequired.v2", clock.Now().UnixNano())
// 		password := "apex_checklist"

// 		patches := map[string]interface{}{}
// 		fileName := "/go/src/github.com/alpacahq/gobroker/tools/acctloader/case5.yml"
// 		file, err := os.Open(fileName)
// 		if err != nil {
// 			return nil, err
// 		}
// 		yamlData, err := ioutil.ReadAll(file)
// 		if err != nil {
// 			return nil, err
// 		}
// 		if err := yaml.Unmarshal(yamlData, &patches); err != nil {
// 			return nil, err
// 		}

// 		// create account
// 		tx := db.Begin()
// 		acctSrv := account.Service().WithTx(tx)

// 		acct, err := acctSrv.Create(email, password)
// 		if err != nil {
// 			tx.Rollback()
// 			return nil, err
// 		}
// 		tx.Commit()

// 		// patch owner details
// 		tx = db.Begin()
// 		ownerSrv := ownerdetails.Service().WithTx(tx)

// 		if _, err = ownerSrv.Patch(acct.IDAsUUID(), patches); err != nil {
// 			tx.Rollback()
// 			return nil, err
// 		}
// 		tx.Commit()

// 		// store the ID
// 		acctIDs = append(acctIDs, acct.IDAsUUID())
// 	}
// 	return acctIDs, nil
// }

// func createFinraAffiliatedAccounts() ([]uuid.UUID, error) {
// 	acctIDs := []uuid.UUID{}
// 	for i := 0; i < 3; i++ {
// 		// load case data
// 		email := fmt.Sprintf("%v@actionrequired.v2", clock.Now().UnixNano())
// 		password := "apex_checklist"

// 		patches := map[string]interface{}{}
// 		fileName := "/go/src/github.com/alpacahq/gobroker/tools/acctloader/case4.yml"
// 		file, err := os.Open(fileName)
// 		if err != nil {
// 			return nil, err
// 		}
// 		yamlData, err := ioutil.ReadAll(file)
// 		if err != nil {
// 			return nil, err
// 		}
// 		if err := yaml.Unmarshal(yamlData, &patches); err != nil {
// 			return nil, err
// 		}

// 		// create account
// 		tx := db.Begin()
// 		acctSrv := account.Service().WithTx(tx)

// 		acct, err := acctSrv.Create(email, password)
// 		if err != nil {
// 			tx.Rollback()
// 			return nil, err
// 		}

// 		tx.Commit()

// 		// create affiliate
// 		affSrv := affiliate.Service().WithTx(db.DB())

// 		aff := &models.Affiliate{
// 			AccountID:       acct.ID,
// 			StreetAddress:   pq.StringArray{"123 Somewhere Ln.", "Apt. 2"},
// 			City:            "San Mateo",
// 			State:           "CA",
// 			PostalCode:      "94402",
// 			Country:         "USA",
// 			CompanyName:     "ALPACA",
// 			ComplianceEmail: "alpaca@alpaca.com",
// 		}

// 		aff, err = affSrv.Create(aff)
// 		if err != nil {
// 			tx.Rollback()
// 			return nil, err
// 		}

// 		tx.Commit()

// 		// patch owner details
// 		tx = db.Begin()
// 		ownerSrv := ownerdetails.Service().WithTx(tx)

// 		if _, err = ownerSrv.Patch(acct.IDAsUUID(), patches); err != nil {
// 			tx.Rollback()
// 			return nil, err
// 		}
// 		tx.Commit()

// 		// store the ID
// 		acctIDs = append(acctIDs, acct.IDAsUUID())
// 	}
// 	return acctIDs, nil
// }

// var dlBack, dlFront, letter407 []byte

// func loadSnapSamples() {
// 	// back of driver's license
// 	{
// 		fileName := "/go/src/github.com/alpacahq/gobroker/tools/acctloader/sample_drivers_license_back.png"
// 		file, err := os.Open(fileName)
// 		if err != nil {
// 			panic(err)
// 		}
// 		if dlBack, err = ioutil.ReadAll(file); err != nil {
// 			panic(err)
// 		}

// 	}

// 	// front of driver's license
// 	{
// 		fileName := "/go/src/github.com/alpacahq/gobroker/tools/acctloader/sample_drivers_license_front.png"
// 		file, err := os.Open(fileName)
// 		if err != nil {
// 			panic(err)
// 		}
// 		if dlFront, err = ioutil.ReadAll(file); err != nil {
// 			panic(err)
// 		}
// 	}
// 	// 407 letter
// 	{
// 		fileName := "/go/src/github.com/alpacahq/gobroker/tools/acctloader/sample_407_letter.png"
// 		file, err := os.Open(fileName)
// 		if err != nil {
// 			panic(err)
// 		}
// 		if letter407, err = ioutil.ReadAll(file); err != nil {
// 			panic(err)
// 		}
// 	}
// }
