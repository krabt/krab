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
	store.RecordOutbound("server-b", "B", TrafficTotals{Uplink: 5, Downlink: 6}, TrafficTotals{Uplink: 7, Downlink: 8}, day.AddDate(0, 0, 1))
	snapshot := store.Snapshot()
	if snapshot.Total.Uplink != 172 || snapshot.Total.Downlink != 259 {
		t.Fatalf("unexpected totals: %#v", snapshot.Total)
	}
	if snapshot.Proxy.Uplink != 165 || snapshot.Proxy.Downlink != 251 || snapshot.Direct.Uplink != 7 || snapshot.Direct.Downlink != 8 {
		t.Fatalf("unexpected outbound totals: proxy=%#v direct=%#v", snapshot.Proxy, snapshot.Direct)
	}
	if len(snapshot.Daily) != 2 || len(snapshot.Servers) != 2 {
		t.Fatalf("unexpected history: %#v", snapshot)
	}
	if snapshot.Daily[0].Proxy.Uplink != 15 || snapshot.Daily[0].Direct.Uplink != 7 {
		t.Fatalf("unexpected daily outbound traffic: %#v", snapshot.Daily[0])
	}
}
