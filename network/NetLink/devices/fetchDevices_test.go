package devices

import (
	"testing"
	"time"
)

func TestFetchDevices(t *testing.T) {
	devices, err := FetchDevices("http://localhost:4535", 20*time.Second)
	if err != nil {
		t.Fatalf("FetchDevices failed: %v", err)
	}
	t.Logf("Successfully fetched %d devices: %+v", len(devices), devices)
}
