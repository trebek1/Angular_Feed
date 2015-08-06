package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Application configuration parameters.  This struct is passed into
// startEntityManager() in main.go.
type AppConfig struct {

	// Determines how often the app polls the registered ContentSource (MemDB) for
	// new data.
	RefreshInterval time.Duration

	// Sets the supported time ranges for the GetAllEntityInfo() endpoint.
	// Determines how long to retain data in the content buffer before expiring
	// it.  For example, setting this property to "24h,8h,1h" results in a maximum
	// 24 hour "rolling window" of data that is made available to the app, with\
	// the option of filtering on an 8 hour and 1 hour time range as well.
	TimeRanges []time.Duration

	// Determines how much data to load into the app prior to allowing public access.
	// For example, if this property is set to "10m", the app will buffer the most
	// recent 10 minutes of data from the ContentSource.
	DataPreFetchWindow time.Duration

	// MemDb connection string (e.g. "memdb01.aws.com:29932").  If empty, a mock
	// data source will be configured, which provides mock data for the app
	// (useful for local day-to-day development and testing).
	MemDbConn string

	// If running in an environment in which an HTTPS reverse proxy is forwarding
	// requests (as plain HTTP requests) to this app, set this property to the
	// HTTPS host to which non-HTTPS requests should be redirected.  For example,
	// suppose foo.bar.com resolves to our reverse proxy server (e.g. an AWS ELB
	// with HTTPS enabled), and HttpsRedirectUrl is set to 'https://foo.bar.com,
	// then the request 'http://foo.bar.com/blah' will redirect to
	// 'https://foo.bar.com/blah' with the response code http.StatusMovedPermanently.
	HttpsRedirectUrl string

	// Directory into which application state will be read/written.  The permissions
	// of this directory must allow file read/write/delete.
	DataDir string
}

// Loads application configuration parameters from shell environment variables
// into a new AppConfig object.  Each environment variable (see the 'config'
// map below) corresponds to a field within the AppConfig struct.
func MakeAppConfig() AppConfig {
	// Seed this map with default configuration values.
	config := map[string]string{
		"SYNTHOS_MEMDB_CONN":         "",
		"SYNTHOS_REFRESH_INTERVAL":   "20s",
		"SYNTHOS_PREFETCH_WINDOW":    "5m",
		"SYNTHOS_TIME_RANGES":        "1h, 2h, 8h, 24h",
		"SYNTHOS_HTTPS_REDIRECT_URL": "",
		"SYNTHOS_DATA_DIR":           "/tmp/synthos/data/",
	}

	// Load shell environment vars starting with "SYNTHOS_" into a key/value map.
	// Some of the default entries (defined above) might get overridden.
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "SYNTHOS_") {
			keyValue := strings.Split(e, "=")
			config[keyValue[0]] = keyValue[1]
		}
	}

	parseDurationOrPanic := func(s string) time.Duration {
		d, err := time.ParseDuration(strings.TrimSpace(s))
		if err != nil {
			panic(fmt.Sprintf("Error parsing duration '%v': %v", s, err))
		}
		return d
	}

	timeRangeStrings := strings.Split(config["SYNTHOS_TIME_RANGES"], ",")
	timeRanges := []time.Duration{}
	for _, timeRangeString := range timeRangeStrings {
		timeRanges = append(timeRanges, parseDurationOrPanic(timeRangeString))
	}

	return AppConfig{
		RefreshInterval:    parseDurationOrPanic(config["SYNTHOS_REFRESH_INTERVAL"]),
		TimeRanges:         timeRanges,
		DataPreFetchWindow: parseDurationOrPanic(config["SYNTHOS_PREFETCH_WINDOW"]),
		MemDbConn:          config["SYNTHOS_MEMDB_CONN"],
		HttpsRedirectUrl:   config["SYNTHOS_HTTPS_REDIRECT_URL"],
		DataDir:            config["SYNTHOS_DATA_DIR"],
	}
}

// Returns true only if the app should use mock data instead of querying the
// live content.
func (me *AppConfig) UseMockData() bool {
	return me.MemDbConn == ""
}
