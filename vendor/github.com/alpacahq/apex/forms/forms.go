package forms

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"runtime"

	"github.com/alpacahq/apex/forms/v1"
)

var (
	accountForm map[string]interface{}
	marginForm  map[string]interface{}
)

func GetForms() map[string]interface{} {
	if accountForm == nil {
		accountForm = make(map[string]interface{})
		readForm("/v1/new_account_form.json", &accountForm)
	}
	if marginForm == nil {
		marginForm = make(map[string]interface{})
		readForm("/v1/margin_agreement_form.json", &marginForm)
	}
	return map[string]interface{}{
		"new_account": accountForm,
		"margin":      marginForm,
	}
}

func readForm(path string, form *map[string]interface{}) error {
	_, f, _, _ := runtime.Caller(0)
	dir, err := filepath.Abs(filepath.Dir(f))
	if err != nil {
		return err
	}
	data, err := ioutil.ReadFile(dir + path)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, form)
	if err != nil {
		return err
	}
	return nil
}

type FormSubmission struct {
	ModifyType string    `json:"modifyType"`
	RepCode    string    `json:"repCode"`
	Branch     string    `json:"branch"`
	Forms      []v1.Form `json:"forms"`
	Account    string    `json:"account,omitempty"`
}
