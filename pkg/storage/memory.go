package storage

import (
	"fmt"
	"strings"
)

type IdentityStore struct {
	name          string
	currentFlags  []*FeatureFlag
	previousFlags []*FeatureFlag
}

type InMemoryStorage struct {
	storage map[string]*IdentityStore
}

// NewInMemory create new in memory storage instance
func NewInMemory() *InMemoryStorage {
	x := new(InMemoryStorage)
	x.storage = make(map[string]*IdentityStore)

	return x
}

// RegisterFeatureFlags Registers featureflags for an application
func (s *InMemoryStorage) RegisterFeatureFlags(auth string, identity string, flags []*FeatureFlag) error {
	store, ok := s.storage[identity]
	if !ok {
		s.storage[identity] = new(IdentityStore)
		store, _ = s.storage[identity]
	}

	store.previousFlags = store.currentFlags
	store.currentFlags = flags

	fmt.Printf("Storage is now: %v", s.storage)
	return nil
}

// GetFeatureFlagState Gets the state of a feature flag
func (s *InMemoryStorage) GetFeatureFlagState(auth string, identity string, flag_name string) (*FeatureFlag, error) {
	idstore, ok := s.storage[identity]
	if !ok {
		return nil, fmt.Errorf("No identity found")
	}

	flags := idstore.currentFlags
	for _, flag := range flags {
		if flag.Name == flag_name {
			return flag, nil
		}
	}
	return nil, fmt.Errorf("No feature flag found by that name")
}

// GetAllFeatureFlags Gets all feature flags for a given identity
func (s *InMemoryStorage) GetAllFeatureFlags(auth string, identity string) ([]*FeatureFlag, error) {
	return nil, nil
}

func parseFlagState(flag_state string) bool {
	flag_state_lower := strings.ToLower(flag_state)
	switch flag_state_lower {
	case "on":
		return true
	case "off":
		return false
	case "true":
		return true
	case "false":
		return false
	default:
		return false
	}
}

// SetFeatureFlagState Sets the new state for a given feature flag, flag_state should either be a
//   definitive(on/off) or a context-id and wether is should be on or off for said context
func (s *InMemoryStorage) SetFeatureFlagState(auth string, identity string, flag_name string, flag_state string) error {
	idstore, ok := s.storage[identity]
	if !ok {
		return fmt.Errorf("No configuration for given identity")
	}

	parsed_flag_state := parseFlagState(flag_state)

	flags := idstore.currentFlags
	for i, flag := range flags {
		if flag.Name == flag_name {
			flags[i].State = parsed_flag_state
		}
	}

	return nil
}
