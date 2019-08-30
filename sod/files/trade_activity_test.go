package files

import (
	"io/ioutil"

	"github.com/alpacahq/gobroker/sod/files/samples"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *FileTestSuite) TestTradeActivities() {
	f, err := samples.SamplesBundle.Open("samples/EXT872_APXD.CSV")
	require.Nil(s.T(), err)
	require.NotNil(s.T(), f)

	buf, err := ioutil.ReadAll(f)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), buf)

	sodFile := &TradeActivityReport{}

	assert.Nil(s.T(), Parse(buf, sodFile))
	assert.NotPanics(s.T(), func() { sodFile.Sync(s.asOf) })
}
