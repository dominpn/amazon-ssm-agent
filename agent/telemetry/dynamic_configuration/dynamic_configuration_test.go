package dynamicconfiguration

import (
	"os"
	"path/filepath"
	"time"

	"testing"

	"github.com/aws/amazon-ssm-agent/agent/fileutil"
	logmocks "github.com/aws/amazon-ssm-agent/agent/mocks/log"
	"github.com/stretchr/testify/assert"
)

const testingDir = "./testingvar"
const fakeConfigFilePath = ("./testingvar/dynamic_config.json")

func Refresh() {
	EvictCache()
	testingDir := filepath.Clean(testingDir)
	fileutil.DeleteDirectory(testingDir)
}
func Setup() {
	Refresh()
}

func Teardown() {
	Refresh()
	// Needed to make sure watchers left behind by previous tests do not react to stale events and pollute cache
	// TODO: Find better way to do this. Maybe add explicit stopper on watcher, to be called only during tests
	time.Sleep(5 * time.Second)
}

func TestNewTelemetryDynamicConfigurationPopulatesDefaults(t *testing.T) {
	// Act
	Setup()
	defer Teardown()

	log := logmocks.NewMockLog()

	NewTelemetryDynamicConfiguration(log, false, fakeConfigFilePath)
	dynamicConfig := GetCachedDynamicConfiguration()

	assert.Equal(t, dynamicConfig["default"].TelemetryDisabledTill, int64(0))
	assert.Equal(t, 0, dynamicConfig["default"].PercentageLimit)
	assert.Equal(t, 0, dynamicConfig["default"].MaxRolls)
	assert.Equal(t, int64(0), dynamicConfig["default"].MaxRollSize)
	assert.Equal(t, 0, dynamicConfig["default"].ExportPeriod)
}

func TestNewTelemetryDynamicConfigurationReadsPreExistingConfigFile(t *testing.T) {
	// Act
	Setup()
	defer Teardown()

	log := logmocks.NewMockLog()
	os.MkdirAll(filepath.Clean(testingDir), 0755)
	os.WriteFile(
		filepath.Clean(fakeConfigFilePath),
		[]byte(`{
	    "default": {
	        "telemetryDisabledTillEpoch": 10,
	        "percentageLimit": 50,
	        "maxRolls": 15,
	        "maxRollSize": 500,
	        "exportPeriodMinutes": 15
	    }
	}`),
		0644,
	)

	NewTelemetryDynamicConfiguration(log, false, filepath.Clean(fakeConfigFilePath))
	dynamicConfig := GetCachedDynamicConfiguration()
	assert.Equal(t, int64(10), dynamicConfig["default"].TelemetryDisabledTill)
	assert.Equal(t, 50, dynamicConfig["default"].PercentageLimit)
	assert.Equal(t, 15, dynamicConfig["default"].MaxRolls, 15)
	assert.Equal(t, int64(500), dynamicConfig["default"].MaxRollSize)
	assert.Equal(t, 15, dynamicConfig["default"].ExportPeriod)
}

func TestNewTelemetryDynamicConfigurationWatchesSimpleConfigFileUpdates(t *testing.T) {
	// Act
	Setup()
	defer Teardown()

	log := logmocks.NewMockLog()
	os.MkdirAll(filepath.Clean(testingDir), 0755)
	os.WriteFile(filepath.Clean(fakeConfigFilePath), []byte(`{"default":{"telemetryDisabledTillEpoch":10,"percentageLimit":50,"maxRolls":15,"maxRollSize":500,"exportPeriodMinutes":15}}`), 0644)

	NewTelemetryDynamicConfiguration(log, true, filepath.Clean(fakeConfigFilePath))

	os.WriteFile(fakeConfigFilePath, []byte(`{"default":{"telemetryDisabledTillEpoch":55,"percentageLimit":12,"maxRolls":17,"maxRollSize":502,"exportPeriodMinutes":20}}`), 0644)

	time.Sleep(5 * time.Second)

	dynamicConfig := GetCachedDynamicConfiguration()
	assert.Equal(t, int64(55), dynamicConfig["default"].TelemetryDisabledTill)
	assert.Equal(t, 12, dynamicConfig["default"].PercentageLimit)
	assert.Equal(t, 17, dynamicConfig["default"].MaxRolls)
	assert.Equal(t, int64(502), dynamicConfig["default"].MaxRollSize)
	assert.Equal(t, 20, dynamicConfig["default"].ExportPeriod)
}

func TestNewTelemetryDynamicConfigurationWatchesConfigFileUpdatesWithNewNamespaces(t *testing.T) {
	// Act
	Setup()
	defer Teardown()

	log := logmocks.NewMockLog()
	os.MkdirAll(filepath.Clean(testingDir), 0755)
	err := os.WriteFile(filepath.Clean(fakeConfigFilePath), []byte(`{"default":{"telemetryDisabledTillEpoch":10,"percentageLimit":50,"maxRolls":15,"maxRollSize":500,"exportPeriodMinutes":15}}`), 0644)

	if err != nil {
		log.Errorf("Error while opening config file: %v", err)
	}

	NewTelemetryDynamicConfiguration(log, true, filepath.Clean(fakeConfigFilePath))

	os.WriteFile(filepath.Clean(fakeConfigFilePath), []byte(`{
     "default": {
         "telemetryDisabledTillEpoch": 55,
         "percentageLimit": 12,
         "maxRolls": 17,
         "maxRollSize": 502,
         "exportPeriodMinutes": 20
     },
     "distributor": {
         "telemetryDisabledTillEpoch": 89,
         "percentageLimit": 82,
         "maxRolls": 87,
         "maxRollSize": 902,
         "exportPeriodMinutes": 20
     }
 }`), 0644)

	time.Sleep(5 * time.Second)

	dynamicConfig := GetCachedDynamicConfiguration()

	assert.Equal(t, int64(55), dynamicConfig["default"].TelemetryDisabledTill)
	assert.Equal(t, 12, dynamicConfig["default"].PercentageLimit)
	assert.Equal(t, 17, dynamicConfig["default"].MaxRolls)
	assert.Equal(t, int64(502), dynamicConfig["default"].MaxRollSize)
	assert.Equal(t, 20, dynamicConfig["default"].ExportPeriod)

	assert.Equal(t, int64(89), dynamicConfig["distributor"].TelemetryDisabledTill)
	assert.Equal(t, 82, dynamicConfig["distributor"].PercentageLimit)
	assert.Equal(t, 87, dynamicConfig["distributor"].MaxRolls)
	assert.Equal(t, int64(902), dynamicConfig["distributor"].MaxRollSize)
	assert.Equal(t, 20, dynamicConfig["distributor"].ExportPeriod)
}
