package storage

type FeatureFlag struct {
	Name  string `json:"flag_name"`
	State bool   `json:"state"`
}

type FeatureFlagStore interface {
	// RegisterFeatureFlags Registers featureflags for an application
	RegisterFeatureFlags(identity string, flags []*FeatureFlag) error
	// GetFeatureFlagState Gets the state of a feature flag
	GetFeatureFlagState(identity string, flag_name string) (*FeatureFlag, error)
	// GetAllFeatureFlags Gets all feature flags for a given identity
	GetAllFeatureFlags(identity string) ([]*FeatureFlag, error)
	// SetFeatureFlagState Sets the new state for a given feature flag, flag_state should either be a
	//   definitive(on/off) or a context-id and wether is should be on or off for said context
	SetFeatureFlagState(identity string, flag_name string, flag_state string) error
}
