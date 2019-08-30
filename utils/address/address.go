package address

import (
	"database/sql/driver"
	"encoding/json"
	"errors"

	"github.com/lib/pq"
)

// apex's max array element length
const maxLength = 30

type Address pq.StringArray

// ToApexFormat formats addresses in Apex's
// ridiculous array format, where each component
// has <= 30 characters. Since we want our system
// to be sane, we allow plain strings in the api calls.
// This package is dedicated to reconciling the two schemas.
// Apex actually allows up to 3 address components,
// but we have reserved the third for the unit number.
// If it turns out there exist street addresses longer than
// 60 characters we may need to make this more flexible.
// But then it starts to resemble the packing problem,
// and I am really not down to solve the packing problem today.
func ToApexFormat(streetAddress string) (Address, error) {
	length := len(streetAddress)
	if length <= maxLength {
		return Address([]string{streetAddress}), nil
	}
	if length > maxLength*2 {
		return nil, errors.New("Street address too long")
	}

	// Each element of the array must have length <= 30,
	// so the earliest we could split the array is (len-1)-30,
	// and the latest is exactly at 30. We will look for
	// whitespace somewhere between those two values.
	soonestSplit := length - maxLength - 1
	for split := maxLength; split >= soonestSplit; split-- {
		if streetAddress[split] == ' ' {
			return Address([]string{
				streetAddress[:split],
				streetAddress[split+1 : length], // +1 to remove the space from the splitted string
			}), nil
		}
	}
	// If we can't find a good place to split, throw up our hands,
	// split at some arbitrary point and hope apex likes us.
	return Address([]string{
		streetAddress[:maxLength],
		streetAddress[maxLength:length],
	}), nil
}

func HandleApiAddress(addr interface{}) (Address, error) {
	switch a := addr.(type) {
	case string:
		return ToApexFormat(a)
	case []interface{}:
		var apexAddr Address
		for _, val := range a {
			apexAddr = append(apexAddr, val.(string))
		}
		return apexAddr, nil
	case []string:
		var apexAddr Address
		for _, val := range a {
			apexAddr = append(apexAddr, val)
		}
		return apexAddr, nil
	}
	return nil, errors.New("Invalid street address format")
}

func (addr *Address) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		// json parameter is an array (deprecated format), unmarshal it
		var arr pq.StringArray
		if err2 := json.Unmarshal(data, &arr); err2 != nil {
			// json param is invalid, return the string unmarshalling error
			// because that's the new and un-deprecated api format
			return err
		}
		*addr = Address(arr)
	} else {
		// json parameter is a string, convert to array for our internal use
		a, err := ToApexFormat(str)
		if err != nil {
			return err
		}
		*addr = a
	}
	return nil
}

func (a *Address) Scan(src interface{}) error {
	strArr := pq.StringArray(*a)
	err := strArr.Scan(src)
	if err != nil {
		return err
	}
	*a = Address(strArr)
	return nil
}

func (a Address) Value() (driver.Value, error) {
	return pq.StringArray(a).Value()
}
