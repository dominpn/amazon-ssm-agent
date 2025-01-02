package metric

import (
	"time"
)

type Metric[N int64 | float64] struct {
	// Name is the name of the Instrument that created this data.
	Name       string
	Unit       string
	DataPoints []DataPoint[N]
}

// DataPoint is a single data point in a timeseries.
type DataPoint[N int64 | float64] struct {
	// StartTime is when the timeseries was started. (optional)
	StartTime time.Time `json:",omitempty"`
	// Time is the time when the timeseries was recorded. (optional)
	EndTime time.Time `json:",omitempty"`
	// Value is the value of this data point.
	Value N
}

type Int64Counter Counter[int64]

type Counter[N int64 | float64] interface {
	// Add increments the counter by the specified amount
	Add(incr N) error
}

// Meter interface provides different "instruments" like counter, gauge, histogram etc.
type Meter interface {
	Int64Counter(name string, unit string) Int64Counter
}
