package profile

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/krabt/krab/internal/database"
)

type Store struct {
	mu sync.Mutex
}

func NewStore() *Store { return &Store{} }

func (s *Store) List() ([]Server, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readAll()
}

func (s *Store) Get(id string) (Server, error) {
	servers, err := s.List()
	if err != nil {
		return Server{}, err
	}
	for _, srv := range servers {
		if srv.ID == id {
			return srv, nil
		}
	}
	return Server{}, fmt.Errorf("no server with id %q", id)
}

// Add persists a new server, assigning it an ID if it doesn't already
// have one, and returns the stored copy (with that ID set) so the
// caller can hand back something actually usable by Get/Connect --
// the server passed in is a value, so its own ID field is never
// visible to the caller otherwise.
func (s *Store) Add(server Server) (Server, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	servers, err := s.readAll()
	if err != nil {
		return Server{}, err
	}

	if server.ID == "" {
		server.ID = uuid.NewString()
	}
	servers = append(servers, server)
	if err := s.writeAll(servers); err != nil {
		return Server{}, err
	}
	return server, nil
}

// ReplaceWhere removes every stored server matching drop, appends added
// (assigning IDs), and writes the file once -- so importing or refreshing
// a subscription with hundreds of servers isn't hundreds of full rewrites.
func (s *Store) ReplaceWhere(drop func(Server) bool, added []Server) ([]Server, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	servers, err := s.readAll()
	if err != nil {
		return nil, err
	}
	kept := servers[:0]
	for _, srv := range servers {
		if drop == nil || !drop(srv) {
			kept = append(kept, srv)
		}
	}
	for i := range added {
		if added[i].ID == "" {
			added[i].ID = uuid.NewString()
		}
	}
	if err := s.writeAll(append(kept, added...)); err != nil {
		return nil, err
	}
	return added, nil
}

// UpdateWhere applies fn to every stored server matching match, in place,
// and writes the file once. Returns how many servers matched.
func (s *Store) UpdateWhere(match func(Server) bool, fn func(*Server)) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	servers, err := s.readAll()
	if err != nil {
		return 0, err
	}
	n := 0
	for i := range servers {
		if match(servers[i]) {
			fn(&servers[i])
			n++
		}
	}
	if n == 0 {
		return 0, nil
	}
	return n, s.writeAll(servers)
}

// Update replaces the server with the same ID, preserving its position.
func (s *Store) Update(server Server) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	servers, err := s.readAll()
	if err != nil {
		return err
	}

	for i, srv := range servers {
		if srv.ID == server.ID {
			servers[i] = server
			return s.writeAll(servers)
		}
	}
	return fmt.Errorf("no server with id %q", server.ID)
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	servers, err := s.readAll()
	if err != nil {
		return err
	}

	filtered := servers[:0]
	for _, srv := range servers {
		if srv.ID != id {
			filtered = append(filtered, srv)
		}
	}
	return s.writeAll(filtered)
}

func (s *Store) readAll() ([]Server, error) {
	data, found, err := database.Get("profiles")
	if err != nil {
		return nil, err
	}
	if !found {
		return []Server{}, nil
	}

	var servers []Server
	if err := json.Unmarshal(data, &servers); err != nil {
		return nil, fmt.Errorf("corrupt server list: %w", err)
	}

	return servers, nil
}

func (s *Store) writeAll(servers []Server) error {
	data, err := json.Marshal(servers)
	if err != nil {
		return err
	}
	return database.Set("profiles", data)
}
