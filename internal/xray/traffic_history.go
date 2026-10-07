package xray

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/krabt/krab/internal/database"
)

type TrafficTotals struct {
	Uplink   int64 `json:"uplink"`
	Downlink int64 `json:"downlink"`
}

type DailyTraffic struct {
	Date string `json:"date"`
	TrafficTotals
	Proxy  TrafficTotals `json:"proxy"`
	Direct TrafficTotals `json:"direct"`
}

type ServerTraffic struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	TrafficTotals
}

type TrafficHistory struct {
	Total   TrafficTotals   `json:"total"`
	Proxy   TrafficTotals   `json:"proxy"`
	Direct  TrafficTotals   `json:"direct"`
	Daily   []DailyTraffic  `json:"daily"`
	Servers []ServerTraffic `json:"servers"`
}

type trafficHistoryFile struct {
	Daily       map[string]TrafficTotals `json:"daily"`
	DailyProxy  map[string]TrafficTotals `json:"dailyProxy,omitempty"`
	DailyDirect map[string]TrafficTotals `json:"dailyDirect,omitempty"`
	Servers     map[string]ServerTraffic `json:"servers"`
	Proxy       TrafficTotals            `json:"proxy"`
	Direct      TrafficTotals            `json:"direct"`
}

type TrafficHistoryStore struct {
	mu   sync.Mutex
	path string
	data trafficHistoryFile
}

func NewTrafficHistoryStore() *TrafficHistoryStore {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	store := &TrafficHistoryStore{path: filepath.Join(dir, "krab", "traffic-history.json")}
	store.data.Daily = map[string]TrafficTotals{}
	store.data.DailyProxy = map[string]TrafficTotals{}
	store.data.DailyDirect = map[string]TrafficTotals{}
	store.data.Servers = map[string]ServerTraffic{}
	if data, found, err := database.Get("traffic_history"); err == nil && found {
		_ = json.Unmarshal(data, &store.data)
	} else if data, err := os.ReadFile(store.path); err == nil {
		_ = json.Unmarshal(data, &store.data)
		_ = database.Set("traffic_history", data)
	}
	if store.data.Daily == nil {
		store.data.Daily = map[string]TrafficTotals{}
	}
	if store.data.Servers == nil {
		store.data.Servers = map[string]ServerTraffic{}
	}
	if store.data.DailyProxy == nil {
		store.data.DailyProxy = map[string]TrafficTotals{}
	}
	if store.data.DailyDirect == nil {
		store.data.DailyDirect = map[string]TrafficTotals{}
	}
	if len(store.data.DailyProxy) == 0 && len(store.data.DailyDirect) == 0 {
		for date, totals := range store.data.Daily {
			store.data.DailyProxy[date] = totals
		}
	}
	// Traffic recorded before outbound categories were introduced only
	// contained the proxy outbound counters. Preserve it as proxy traffic.
	if store.data.Proxy == (TrafficTotals{}) && store.data.Direct == (TrafficTotals{}) {
		for _, totals := range store.data.Daily {
			store.data.Proxy.Uplink += totals.Uplink
			store.data.Proxy.Downlink += totals.Downlink
		}
	}
	return store
}

func (s *TrafficHistoryStore) Record(serverID, serverName string, uplink, downlink int64, now time.Time) {
	s.RecordOutbound(serverID, serverName, TrafficTotals{Uplink: uplink, Downlink: downlink}, TrafficTotals{}, now)
}

func (s *TrafficHistoryStore) RecordOutbound(serverID, serverName string, proxy, direct TrafficTotals, now time.Time) {
	uplink, downlink := proxy.Uplink+direct.Uplink, proxy.Downlink+direct.Downlink
	if uplink <= 0 && downlink <= 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.DailyProxy == nil {
		s.data.DailyProxy = map[string]TrafficTotals{}
	}
	if s.data.DailyDirect == nil {
		s.data.DailyDirect = map[string]TrafficTotals{}
	}
	date := now.Format("2006-01-02")
	daily := s.data.Daily[date]
	daily.Uplink += uplink
	daily.Downlink += downlink
	s.data.Daily[date] = daily
	dailyProxy := s.data.DailyProxy[date]
	dailyProxy.Uplink += proxy.Uplink
	dailyProxy.Downlink += proxy.Downlink
	s.data.DailyProxy[date] = dailyProxy
	dailyDirect := s.data.DailyDirect[date]
	dailyDirect.Uplink += direct.Uplink
	dailyDirect.Downlink += direct.Downlink
	s.data.DailyDirect[date] = dailyDirect
	server := s.data.Servers[serverID]
	server.ID, server.Name = serverID, serverName
	server.Uplink += uplink
	server.Downlink += downlink
	s.data.Servers[serverID] = server
	s.data.Proxy.Uplink += proxy.Uplink
	s.data.Proxy.Downlink += proxy.Downlink
	s.data.Direct.Uplink += direct.Uplink
	s.data.Direct.Downlink += direct.Downlink
}

func (s *TrafficHistoryStore) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.Marshal(s.data)
	if err != nil {
		return err
	}
	return database.Set("traffic_history", data)
}

func (s *TrafficHistoryStore) Snapshot() TrafficHistory {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := TrafficHistory{Proxy: s.data.Proxy, Direct: s.data.Direct, Daily: make([]DailyTraffic, 0, len(s.data.Daily)), Servers: make([]ServerTraffic, 0, len(s.data.Servers))}
	for date, totals := range s.data.Daily {
		result.Daily = append(result.Daily, DailyTraffic{Date: date, TrafficTotals: totals, Proxy: s.data.DailyProxy[date], Direct: s.data.DailyDirect[date]})
		result.Total.Uplink += totals.Uplink
		result.Total.Downlink += totals.Downlink
	}
	for _, server := range s.data.Servers {
		result.Servers = append(result.Servers, server)
	}
	sort.Slice(result.Daily, func(i, j int) bool { return result.Daily[i].Date > result.Daily[j].Date })
	sort.Slice(result.Servers, func(i, j int) bool {
		return result.Servers[i].Uplink+result.Servers[i].Downlink > result.Servers[j].Uplink+result.Servers[j].Downlink
	})
	return result
}
