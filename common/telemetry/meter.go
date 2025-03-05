package telemetry

import (
	"fmt"
	"runtime/debug"
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
func (i int64Counter) Add(incr int64) (err error) {
	t, err := getTelemetry()
	if err != nil {
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			t.context.Log().Errorf("Counter Add panic: %v", r)
			t.context.Log().Errorf("Stacktrace:\n%s", debug.Stack())
			err = fmt.Errorf("panic in Add %v", r)
		}
	}()

	pkgMutex.RLock()
	defer pkgMutex.RUnlock()

	return t.emitIntegerMetric(i.namespace, i.name, i.unit, metric.Sum, time.Now(), incr)
}
