package models

import (
	"testing"

	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/utils/address"
	"github.com/lib/pq"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type OwnersSuite struct {
	suite.Suite
}

func TestOwnersSuite(t *testing.T) {
	suite.Run(t, new(OwnersSuite))
}

func (s *OwnersSuite) TestOwnerDetailsValidation() {
	dob := "1991-08-30"
	od := models.OwnerDetails{DateOfBirth: &dob}
	assert.Nil(s.T(), od.Validate())

	dob = "30-08-1991"
	od = models.OwnerDetails{DateOfBirth: &dob}
	assert.NotNil(s.T(), od.Validate())

	emplStatus := models.Student
	od = models.OwnerDetails{EmploymentStatus: &emplStatus}
	assert.Nil(s.T(), od.Validate())

	emplStatus = ""
	od = models.OwnerDetails{EmploymentStatus: &emplStatus}
	assert.NotNil(s.T(), od.Validate())

	maritalStatus := models.Married
	od = models.OwnerDetails{MaritalStatus: &maritalStatus}
	assert.Nil(s.T(), od.Validate())

	maritalStatus = ""
	od = models.OwnerDetails{MaritalStatus: &maritalStatus}
	assert.NotNil(s.T(), od.Validate())

	yearsEmpl := uint(20)
	od = models.OwnerDetails{YearsEmployed: &yearsEmpl}
	assert.Nil(s.T(), od.Validate())

	yearsEmpl = uint(999)
	od = models.OwnerDetails{YearsEmployed: &yearsEmpl}
	assert.NotNil(s.T(), od.Validate())

	prefix := "Mr."
	od = models.OwnerDetails{Prefix: &prefix}
	assert.Nil(s.T(), od.Validate())

	prefix = ""
	od = models.OwnerDetails{Prefix: &prefix}
	assert.NotNil(s.T(), od.Validate())

	suffix := "Sr."
	od = models.OwnerDetails{Suffix: &suffix}
	assert.Nil(s.T(), od.Validate())

	suffix = ""
	od = models.OwnerDetails{Suffix: &suffix}
	assert.NotNil(s.T(), od.Validate())

	visa := models.Visa("H1B")
	od = models.OwnerDetails{VisaType: &visa}
	assert.Nil(s.T(), od.Validate())

	visa = models.Visa("some_random_visa_string")
	od = models.OwnerDetails{VisaType: &visa}
	assert.NotNil(s.T(), od.Validate())

	phone := "650-111-2234"
	od = models.OwnerDetails{PhoneNumber: &phone}
	assert.Nil(s.T(), od.Validate())

	phone = "j0892fj39"
	od = models.OwnerDetails{PhoneNumber: &phone}
	assert.NotNil(s.T(), od.Validate())

	street := pq.StringArray{"123 Somewhere Ln", "Apt. 3"}
	od = models.OwnerDetails{StreetAddress: address.Address(street)}
	assert.Nil(s.T(), od.Validate())

	street = pq.StringArray{"123 Somewhere Ln", "Apt. 3", "Somewhere", "Too Far"}
	od = models.OwnerDetails{StreetAddress: address.Address(street)}
	assert.NotNil(s.T(), od.Validate())

	city := "San Mateo"
	od = models.OwnerDetails{City: &city}
	assert.Nil(s.T(), od.Validate())

	city = "1"
	od = models.OwnerDetails{City: &city}
	assert.NotNil(s.T(), od.Validate())

	state := "CA"
	od = models.OwnerDetails{State: &state}
	assert.Nil(s.T(), od.Validate())

	state = "california"
	od = models.OwnerDetails{State: &state}
	assert.NotNil(s.T(), od.Validate())

	postal := "94402"
	od = models.OwnerDetails{PostalCode: &postal}
	assert.Nil(s.T(), od.Validate())

	postal = "hello"
	od = models.OwnerDetails{PostalCode: &postal}
	assert.NotNil(s.T(), od.Validate())
}
