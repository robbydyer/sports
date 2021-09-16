package stockboard

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
)

func TestMinPrice(t *testing.T) {
	t.Parallel()
	now, err := time.Parse("01-02-2006 15:04:05", "01-02-2006 12:00:00")
	require.NoError(t, err)

	tests := []struct {
		name     string
		prices   []*Price
		expected *Price
	}{
		{
			name:     "none",
			prices:   []*Price{},
			expected: nil,
		},
		{
			name: "one",
			prices: []*Price{
				{
					Time:  now,
					Price: 1.0,
				},
			},
			expected: &Price{
				Time:  now,
				Price: 1.0,
			},
		},
		{
			name: "multi",
			prices: []*Price{
				{
					Time:  now.Add(10 * time.Minute),
					Price: 3.0,
				},
				{
					Time:  now.Add(20 * time.Minute),
					Price: 1.0,
				},
				{
					Time:  now.Add(5 * time.Minute),
					Price: 2.0,
				},
			},
			expected: &Price{
				Time:  now.Add(20 * time.Minute),
				Price: 1.0,
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			s := &Stock{
				Prices: test.prices,
			}
			actual := s.minPrice()
			if test.expected == nil {
				require.Nil(t, actual)
				return
			}
			require.Equal(t, test.expected.Time, actual.Time)
			require.Equal(t, test.expected.Price, actual.Price)
		})
	}
}

func TestMaxPrice(t *testing.T) {
	t.Parallel()
	now, err := time.Parse("01-02-2006 15:04:05", "01-02-2006 12:00:00")
	require.NoError(t, err)

	tests := []struct {
		name     string
		prices   []*Price
		expected *Price
	}{
		{
			name:     "none",
			prices:   []*Price{},
			expected: nil,
		},
		{
			name: "one",
			prices: []*Price{
				{
					Time:  now,
					Price: 1.0,
				},
			},
			expected: &Price{
				Time:  now,
				Price: 1.0,
			},
		},
		{
			name: "multi",
			prices: []*Price{
				{
					Time:  now.Add(20 * time.Minute),
					Price: 1.0,
				},
				{
					Time:  now.Add(10 * time.Minute),
					Price: 3.0,
				},
				{
					Time:  now.Add(5 * time.Minute),
					Price: 2.0,
				},
			},
			expected: &Price{
				Time:  now.Add(10 * time.Minute),
				Price: 3.0,
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			s := &Stock{
				Prices: test.prices,
			}
			actual := s.maxPrice()
			if test.expected == nil {
				require.Nil(t, actual)
				return
			}
			require.Equal(t, test.expected.Time, actual.Time)
			require.Equal(t, test.expected.Price, actual.Price)
		})
	}
}
func TestGetChartPrices(t *testing.T) {
	t.Parallel()
	now, err := time.Parse("01-02-2006 15:04:05", "01-02-2006 12:00:00")
	require.NoError(t, err)

	tests := []struct {
		name     string
		max      int
		stock    *Stock
		expected []*Price
	}{
		{
			name: "latest-only",
			max:  1,
			stock: &Stock{
				Prices: []*Price{
					{
						Time:  now,
						Price: 2.0,
					},
					{
						Time:  now.Add(10 * time.Minute),
						Price: 3.0,
					},
					{
						Time:  now.Add(5 * time.Minute),
						Price: 1.0,
					},
				},
			},
			expected: []*Price{
				{
					Time:  now.Add(10 * time.Minute),
					Price: 3.0,
				},
			},
		},
		{
			name: "first-latest",
			max:  2,
			stock: &Stock{
				Prices: []*Price{
					{
						Time:  now,
						Price: 2.0,
					},
					{
						Time:  now.Add(10 * time.Minute),
						Price: 3.0,
					},
					{
						Time:  now.Add(5 * time.Minute),
						Price: 1.0,
					},
				},
			},
			expected: []*Price{
				{
					Time:  now,
					Price: 2.0,
				},
				{
					Time:  now.Add(10 * time.Minute),
					Price: 3.0,
				},
			},
		},
		{
			name: "short",
			max:  3,
			stock: &Stock{
				Prices: []*Price{
					{
						Time:  now,
						Price: 2.0,
					},
					{
						Time:  now.Add(10 * time.Minute),
						Price: 3.0,
					},
					{
						Time:  now.Add(5 * time.Minute),
						Price: 1.0,
					},
				},
			},
			expected: []*Price{
				{
					Time:  now,
					Price: 2.0,
				},
				{
					Time:  now.Add(5 * time.Minute),
					Price: 1.0,
				},
				{
					Time:  now.Add(10 * time.Minute),
					Price: 3.0,
				},
			},
		},
		{
			name: "large",
			max:  5,
			stock: &Stock{
				Prices: []*Price{
					{
						Time:  now,
						Price: 2.0,
					},
					{
						Time:  now.Add(10 * time.Minute),
						Price: 3.0,
					},
					{
						Time:  now.Add(5 * time.Minute),
						Price: 0.0,
					},
					{
						Time:  now.Add(15 * time.Minute),
						Price: 1.0,
					},
					{
						Time:  now.Add(20 * time.Minute),
						Price: 1.0,
					},
				},
			},
			expected: []*Price{
				{
					Time:  now,
					Price: 2.0,
				},
				{
					Time:  now.Add(5 * time.Minute),
					Price: 0.0,
				},
				{
					Time:  now.Add(10 * time.Minute),
					Price: 3.0,
				},
				{
					Time:  now.Add(15 * time.Minute),
					Price: 1.0,
				},
				{
					Time:  now.Add(20 * time.Minute),
					Price: 1.0,
				},
			},
		},
		{
			name: "min-max-latest-first",
			max:  5,
			stock: &Stock{
				Prices: []*Price{
					{
						Time:  now,
						Price: 2.0,
					},
					{
						Time:  now.Add(5 * time.Minute),
						Price: 3.0,
					},
					{
						Time:  now.Add(10 * time.Minute),
						Price: 3.0,
					},
					{
						Time:  now.Add(15 * time.Minute),
						Price: 1.0,
					},
					{
						Time:  now.Add(20 * time.Minute),
						Price: 5.0,
					},
					{
						Time:  now.Add(25 * time.Minute),
						Price: 3.0,
					},
					{
						Time:  now.Add(30 * time.Minute),
						Price: 3.0,
					},
				},
			},
			expected: []*Price{
				{
					Time:  now,
					Price: 2.0,
				},
				{
					Time:  now.Add(5 * time.Minute),
					Price: 3.0,
				},
				{
					Time:  now.Add(15 * time.Minute),
					Price: 1.0,
				},
				{
					Time:  now.Add(20 * time.Minute),
					Price: 5.0,
				},
				{
					Time:  now.Add(30 * time.Minute),
					Price: 3.0,
				},
			},
		},
	}

	logger := zaptest.NewLogger(t, zaptest.Level(zapcore.DebugLevel))

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			s := &StockBoard{
				log: logger,
			}
			actual := s.getChartPrices(test.max, test.stock)
			require.Equal(t, len(test.expected), len(actual))

			for i, a := range actual {
				require.Greater(t, len(test.expected), i)
				require.NotNil(t, test.expected[i])
				require.NotNil(t, a)
				require.Equal(t, test.expected[i].Time, a.Time)
				require.Equal(t, test.expected[i].Price, a.Price)
			}
		})
	}
}
