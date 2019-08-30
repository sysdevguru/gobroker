package files

import (
	"io/ioutil"

	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/sod/files/samples"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *FileTestSuite) TestPositions() {
	f, err := samples.SamplesBundle.Open("samples/EXT871_CORR_20150925.csv")
	require.Nil(s.T(), err)
	require.NotNil(s.T(), f)

	buf, err := ioutil.ReadAll(f)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), buf)

	sodFile := &PositionReport{}

	assert.Nil(s.T(), Parse(buf, sodFile))
	assert.NotPanics(s.T(), func() { sodFile.Sync(s.asOf) })
}

func (s *FileTestSuite) TestSplit() {
	// Positions: (10, 10, 5)
	// Split: 1:3
	// Apex Qty: 8
	// Expected: (4, 4, closed)
	// Cost Basis: $250
	{
		positions, err := handleSplit(genPositions([]decimal.Decimal{
			decimal.New(10, 0),
			decimal.New(10, 0),
			decimal.New(5, 0),
		}), decimal.New(8, 0))

		require.Nil(s.T(), err)
		assert.Len(s.T(), positions, 3)
		assert.Equal(s.T(), countStatus(positions, models.Closed), 1)
		assert.Equal(s.T(), countStatus(positions, models.Open), 2)
		assert.Equal(s.T(), countQty(positions, decimal.New(4, 0)), 2)
		assert.Equal(s.T(), countQty(positions, decimal.Zero), 1)

		costBasis := decimal.Zero

		for _, position := range positions {
			if position.Status != models.Closed {
				costBasis = costBasis.Add(position.Qty.Mul(position.EntryPrice))
			}
		}

		assert.True(s.T(), decimal.New(250, 0).Equal(costBasis))
	}

	// Positions: (5, 5, 5, 5)
	// Split: 4:1
	// Apex Qty: 80
	// Expected: (20, 20, 20, 20)
	// Cost Basis: $200
	{
		positions, err := handleSplit(genPositions([]decimal.Decimal{
			decimal.New(5, 0),
			decimal.New(5, 0),
			decimal.New(5, 0),
			decimal.New(5, 0),
		}), decimal.New(80, 0))

		require.Nil(s.T(), err)
		assert.Len(s.T(), positions, 4)
		assert.Equal(s.T(), countStatus(positions, models.Closed), 0)
		assert.Equal(s.T(), countStatus(positions, models.Open), 4)
		assert.Equal(s.T(), countQty(positions, decimal.New(20, 0)), 4)

		costBasis := decimal.Zero

		for _, position := range positions {
			if position.Status != models.Closed {
				costBasis = costBasis.Add(position.Qty.Mul(position.EntryPrice))
			}
		}

		assert.True(s.T(), decimal.New(200, 0).Equal(costBasis))
	}

	// Positions: (1, 1)
	// Split: 1:2
	// Apex Qty: 1
	// Expected: (1, closed)
	// Cost Basis: $20
	{
		positions, err := handleSplit(genPositions([]decimal.Decimal{
			decimal.New(1, 0),
			decimal.New(1, 0),
		}), decimal.New(1, 0))

		require.Nil(s.T(), err)
		assert.Len(s.T(), positions, 2)
		assert.Equal(s.T(), countStatus(positions, models.Closed), 1)
		assert.Equal(s.T(), countStatus(positions, models.Open), 1)
		assert.Equal(s.T(), countQty(positions, decimal.New(1, 0)), 1)
		assert.Equal(s.T(), countQty(positions, decimal.Zero), 1)

		costBasis := decimal.Zero

		for _, position := range positions {
			if position.Status != models.Closed {
				costBasis = costBasis.Add(position.Qty.Mul(position.EntryPrice))
			}
		}

		assert.True(s.T(), decimal.New(20, 0).Equal(costBasis))
	}

	// Positions: (1)
	// Split: 10:1
	// Apex Qty: 10
	// Expected: (10)
	// Cost Basis: $10
	{
		positions, err := handleSplit(genPositions([]decimal.Decimal{
			decimal.New(1, 0),
		}), decimal.New(10, 0))

		require.Nil(s.T(), err)
		assert.Len(s.T(), positions, 1)
		assert.Equal(s.T(), countStatus(positions, models.Open), 1)
		assert.Equal(s.T(), countQty(positions, decimal.New(10, 0)), 1)

		costBasis := decimal.Zero

		for _, position := range positions {
			if position.Status != models.Closed {
				costBasis = costBasis.Add(position.Qty.Mul(position.EntryPrice))
			}
		}

		assert.True(s.T(), decimal.New(10, 0).Equal(costBasis))
	}

	// Positions: (1, 2, 3, 4, 5)
	// Split: 1:4
	// Apex Qty: 3
	// Expected: (closed, closed, closed, 1, 2)
	// Cost Basis: $140
	{
		positions, err := handleSplit(genPositions([]decimal.Decimal{
			decimal.New(1, 0),
			decimal.New(2, 0),
			decimal.New(3, 0),
			decimal.New(4, 0),
			decimal.New(5, 0),
		}), decimal.New(3, 0))

		require.Nil(s.T(), err)
		assert.Len(s.T(), positions, 5)
		assert.Equal(s.T(), countStatus(positions, models.Closed), 3)
		assert.Equal(s.T(), countStatus(positions, models.Open), 2)
		assert.Equal(s.T(), countQty(positions, decimal.New(1, 0)), 1)
		assert.Equal(s.T(), countQty(positions, decimal.New(2, 0)), 1)
		assert.Equal(s.T(), countQty(positions, decimal.Zero), 3)

		costBasis := decimal.Zero

		for _, position := range positions {
			if position.Status != models.Closed {
				costBasis = costBasis.Add(position.Qty.Mul(position.EntryPrice))
			}
		}

		assert.True(s.T(), decimal.New(150, 0).Equal(costBasis))
	}

	// error - no split
	{
		positions, err := handleSplit(genPositions([]decimal.Decimal{
			decimal.New(1, 0),
		}), decimal.New(1, 0))

		require.NotNil(s.T(), err)
		require.Nil(s.T(), positions)
	}
}

func genPositions(qties []decimal.Decimal) []*models.Position {
	positions := make([]*models.Position, len(qties))

	for i := range qties {
		positions[i] = &models.Position{
			Qty:        qties[i],
			Status:     models.Open,
			EntryPrice: decimal.New(10, 0),
		}
	}

	return positions
}

func countStatus(positions []*models.Position, status models.PositionStatus) (count int) {
	for _, p := range positions {
		if p.Status == status {
			count++
		}
	}
	return
}

func countQty(positions []*models.Position, qty decimal.Decimal) (count int) {
	for _, p := range positions {
		if p.Qty.Equal(qty) {
			count++
		}
	}
	return
}
