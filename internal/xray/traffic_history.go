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
}

type ServerTraffic struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	TrafficTotals
}

type TrafficHistory struct {
	Total   TrafficTotals   `json:"total"`
	Daily   []DailyTraffic  `json:"daily"`
	Servers []ServerTraffic `json:"servers"`
}

type trafficHistoryFile struct {
	Daily   map[string]TrafficTotals `json:"daily"`
	Servers map[string]ServerTraffic `json:"servers"`
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
	return store
}

func (s *TrafficHistoryStore) Record(serverID, serverName string, uplink, downlink int64, now time.Time) {
	if uplink <= 0 && downlink <= 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	date := now.Format("2006-01-02")
	daily := s.data.Daily[date]
	daily.Uplink += uplink
	daily.Downlink += downlink
	s.data.Daily[date] = daily
	server := s.data.Servers[serverID]
	server.ID, server.Name = serverID, serverName
	server.Uplink += uplink
	server.Downlink += downlink
	s.data.Servers[serverID] = server
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
	result := TrafficHistory{Daily: make([]DailyTraffic, 0, len(s.data.Daily)), Servers: make([]ServerTraffic, 0, len(s.data.Servers))}
	for date, totals := range s.data.Daily {
		result.Daily = append(result.Daily, DailyTraffic{Date: date, TrafficTotals: totals})
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
