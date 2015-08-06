package main

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

func TestMakeAppConfig(t *testing.T) {
	os.Setenv("SYNTHOS_MEMDB_CONN", "fake_memdb_connection")
	os.Setenv("SYNTHOS_REFRESH_INTERVAL", "12s")
	os.Setenv("SYNTHOS_PREFETCH_WINDOW", "123s")
	os.Setenv("SYNTHOS_TIME_RANGES", "1h,2h,3h")
	os.Setenv("SYNTHOS_HTTPS_REDIRECT_URL", "https://foo/bar")
	os.Setenv("SYNTHOS_DATA_DIR", "/foo/bar/baz/")

	cfg := MakeAppConfig()

	assert.NotNil(t, cfg)
	assert.Equal(t, "fake_memdb_connection", cfg.MemDbConn)
	assert.Equal(t, 12*time.Second, cfg.RefreshInterval)
	assert.Equal(t, 123*time.Second, cfg.DataPreFetchWindow)
	assert.Equal(t, []time.Duration{1 * time.Hour, 2 * time.Hour, 3 * time.Hour}, cfg.TimeRanges)
	assert.Equal(t, "https://foo/bar", cfg.HttpsRedirectUrl)
	assert.Equal(t, "/foo/bar/baz/", cfg.DataDir)
}

func TestUseMockData(t *testing.T) {
	os.Setenv("SYNTHOS_MEMDB_CONN", "")
	cfg := MakeAppConfig()
	assert.True(t, cfg.UseMockData())
}
