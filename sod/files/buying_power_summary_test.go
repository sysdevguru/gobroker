package files

import (
	"io/ioutil"

	"github.com/alpacahq/gobroker/sod/files/samples"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *FileTestSuite) TestBuyingPowerSummary() {
	f, err := samples.SamplesBundle.Open("samples/EXT981_XXXX.CSV")
	require.Nil(s.T(), err)
	require.NotNil(s.T(), f)

	buf, err := ioutil.ReadAll(f)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), buf)

	sodFile := &BuyingPowerSummaryReport{}

	assert.Nil(s.T(), Parse(buf, sodFile))
	assert.NotPanics(s.T(), func() { sodFile.Sync(s.asOf) })
}
