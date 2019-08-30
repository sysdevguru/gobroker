package finra

import (
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
)

var (
	url       = "http://oatsreportable.finra.org/OATSReportableSecurities-SOD.txt"
	exchanges = []string{"NYSE", "NASDAQ", "AMEX", "ARCA", "BATS", "CHX", "BX"}
)

type FinraSecurity struct {
	Symbol   string
	Name     string
	Exchange string
}

// GetSecurities retrieves the list of actively traded securities
// on the NYSE, NASDAQ, AMEX and ARCA exchanges
func GetSecurities() ([]FinraSecurity, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	buf, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	if err != nil {
		return nil, err
	}

	secs := []FinraSecurity{}

	reg, _ := regexp.Compile("^[A-Z ]+$")

	for _, line := range strings.Split(string(buf), "\n") {
		items := strings.Split(line, "|")

		if len(items) != 3 {
			continue
		}

		symbol := items[0]
		name := items[1]
		exchange := items[2]

		if !relevant(exchange) {
			continue
		}

		if !reg.Match([]byte(symbol)) {
			continue
		}

		secs = append(secs, FinraSecurity{
			Symbol:   symbol,
			Name:     name,
			Exchange: exchange,
		})
	}
	return secs, nil
}

func relevant(ex string) bool {
	for _, exchange := range exchanges {
		if strings.EqualFold(ex, exchange) {
			return true
		}
	}
	return false
}
