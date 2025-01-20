package telemetry

import (
	"time"

	"github.com/aws/amazon-ssm-agent/common/telemetry/metric"
)

// meterT is a [metric.Meter] for a specific namespace
type meterT struct {
	namespace string
}

// GetMeter gets a [metric.Meter] object for the given namespace
func GetMeter(namespace string) metric.Meter {
	// TODO: Cache it
	return meterT{namespace: namespace}
}

func (m meterT) Int64Counter(name string, unit string) metric.Int64Counter {
	return int64Counter{namespace: m.namespace, name: name, unit: unit}
}

type int64Counter struct {
	namespace string
	name      string
	unit      string
}

// Add increments the counter by the specified amount
func (i int64Counter) Add(incr int64) error {
	telemetry, err := getTelemetry()
	if err != nil {
		return err
	}

	pkgMutex.RLock()
	defer pkgMutex.RUnlock()

	return telemetry.emitIntegerMetric(i.namespace, i.name, i.unit, metric.Sum, time.Now(), incr)
}
