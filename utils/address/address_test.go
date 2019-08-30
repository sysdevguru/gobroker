package address

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type AddressTestSuite struct {
	suite.Suite
}

func TestAddressTestSuite(t *testing.T) {
	suite.Run(t, new(AddressTestSuite))
}

func (s *AddressTestSuite) TestAddressUtils() {
	expected := Address([]string{"123456789012345678901234567890"})
	result, err := ToApexFormat("123456789012345678901234567890")
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), expected, result)

	expected = Address([]string{"123 Areallyreallyreallyreallyr", "eallyreallylongaddress"})
	result, err = ToApexFormat("123 Areallyreallyreallyreallyreallyreallylongaddress")
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), expected, result)

	// an address with whitespace to the left of the 30 char mark
	expected = Address([]string{"12345 Absolutely super long", "crazy address"})
	result, err = ToApexFormat("12345 Absolutely super long crazy address")
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), expected, result)

	// off by 1 errors?
	expected = Address([]string{"1234567890", "veryveryveryveryveryverylooong"})
	result, err = ToApexFormat("1234567890 veryveryveryveryveryverylooong")
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), expected, result)

	expected = Address([]string{"123456789012345678901234567890", "veryverylooong"})
	result, err = ToApexFormat("123456789012345678901234567890 veryverylooong")
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), expected, result)

	expected = Address([]string{"123456789012345678901234567890", "1 too long"})
	result, err = ToApexFormat("1234567890123456789012345678901 too long")
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), expected, result)

	expected = Address([]string{"1 too long 1234567890123456789", "012345678901"})
	result, err = ToApexFormat("1 too long 1234567890123456789012345678901")
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), expected, result)

	_, err = ToApexFormat("waywaywaywaywaywaywaywaywaywaywaywaywaywaywaywaywaywaytoolong")
	assert.NotNil(s.T(), err)
}
