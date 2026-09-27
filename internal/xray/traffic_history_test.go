package xray

import (
	"testing"
	"time"
)

func TestTrafficHistoryPersistsAndGroupsData(t *testing.T) {
	store := &TrafficHistoryStore{
		data: trafficHistoryFile{Daily: map[string]TrafficTotals{}, Servers: map[string]ServerTraffic{}},
	}
	day := time.Date(2026, 9, 28, 12, 0, 0, 0, time.Local)
	store.Record("server-a", "A", 100, 200, day)
	store.Record("server-a", "A", 50, 25, day)
	store.Record("server-b", "B", 10, 20, day.AddDate(0, 0, 1))
	snapshot := store.Snapshot()
	if snapshot.Total.Uplink != 160 || snapshot.Total.Downlink != 245 {
		t.Fatalf("unexpected totals: %#v", snapshot.Total)
	}
	if len(snapshot.Daily) != 2 || len(snapshot.Servers) != 2 {
		t.Fatalf("unexpected history: %#v", snapshot)
	}
}
