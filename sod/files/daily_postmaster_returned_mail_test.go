package files

import (
	"io/ioutil"

	"github.com/alpacahq/gobroker/sod/files/samples"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *FileTestSuite) TestReturnedMail() {
	f, err := samples.SamplesBundle.Open("samples/EXT986_CORR_20150921.CSV")
	require.Nil(s.T(), err)
	require.NotNil(s.T(), f)

	buf, err := ioutil.ReadAll(f)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), buf)

	sodFile := &ReturnedMailReport{}

	assert.Nil(s.T(), Parse(buf, sodFile))

	successful, errors := sodFile.Sync(s.asOf)
	assert.Equal(s.T(), successful, uint(2))
	assert.Zero(s.T(), errors)
}
