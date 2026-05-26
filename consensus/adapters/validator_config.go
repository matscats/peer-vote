package adapters

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"peer-vote/consensus/domain"
	"peer-vote/crypto"
)

// ValidatorConfig represents the configuration structure for validators
type ValidatorConfig struct {
	Validators []ValidatorEntry `json:"validators"`
}

// ValidatorEntry represents a single validator in the configuration
type ValidatorEntry struct {
	PublicKey string `json:"publicKey"` // Hex-encoded public key
	Address   string `json:"address"`   // Network address
}

// ConfigValidatorRegistry implements ValidatorRegistry using a configuration file
// This adapter loads validators from a JSON configuration file at startup
type ConfigValidatorRegistry struct {
	validators []*domain.Validator
	keyMap     map[string]*domain.Validator // Hex-encoded public key -> validator
}

// NewConfigValidatorRegistry creates a new ConfigValidatorRegistry from a configuration file
func NewConfigValidatorRegistry(configPath string) (*ConfigValidatorRegistry, error) {
	// Read configuration file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read validator config: %w", err)
	}

	// Parse JSON configuration
	var config ValidatorConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse validator config: %w", err)
	}

	if len(config.Validators) == 0 {
		return nil, fmt.Errorf("validator config must contain at least one validator")
	}

	// Create validator entities
	validators := make([]*domain.Validator, 0, len(config.Validators))
	keyMap := make(map[string]*domain.Validator)

	for i, entry := range config.Validators {
		// Decode hex public key
		var pubKey crypto.PublicKey
		if _, err := fmt.Sscanf(entry.PublicKey, "%x", &pubKey); err != nil {
			return nil, fmt.Errorf("invalid public key format for validator %d: %w", i, err)
		}

		if entry.Address == "" {
			return nil, fmt.Errorf("validator %d has empty address", i)
		}

		// Create validator entity
		validator := domain.NewValidator(pubKey, entry.Address)
		validators = append(validators, validator)

		// Add to key map for O(1) lookup
		keyStr := fmt.Sprintf("%x", pubKey)
		keyMap[keyStr] = validator
	}

	return &ConfigValidatorRegistry{
		validators: validators,
		keyMap:     keyMap,
	}, nil
}

// GetAll returns all validators in the registry
// Validators are returned in the order they appear in the configuration file
// This ensures consistent ordering across all nodes for deterministic leader selection
func (r *ConfigValidatorRegistry) GetAll() []*domain.Validator {
	// Return a copy to prevent external modification
	validators := make([]*domain.Validator, len(r.validators))
	copy(validators, r.validators)
	return validators
}

// GetByPublicKey retrieves a validator by their public key
func (r *ConfigValidatorRegistry) GetByPublicKey(pubKey crypto.PublicKey) (*domain.Validator, error) {
	keyStr := fmt.Sprintf("%x", pubKey)
	validator, exists := r.keyMap[keyStr]
	if !exists {
		return nil, fmt.Errorf("validator with public key %s not found", keyStr)
	}
	return validator, nil
}

// IsValidator checks if a public key belongs to an authorized validator
func (r *ConfigValidatorRegistry) IsValidator(pubKey crypto.PublicKey) bool {
	// Use bytes.Equal for proper comparison
	for _, validator := range r.validators {
		if bytes.Equal(validator.PublicKey(), pubKey) {
			return true
		}
	}
	return false
}

// Count returns the total number of validators
func (r *ConfigValidatorRegistry) Count() int {
	return len(r.validators)
}
