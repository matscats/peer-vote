package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Duration is a wrapper around time.Duration that supports JSON unmarshaling from strings
type Duration struct {
	time.Duration
}

// UnmarshalJSON implements json.Unmarshaler
func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		d.Duration = time.Duration(value)
		return nil
	case string:
		var err error
		d.Duration, err = time.ParseDuration(value)
		if err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("invalid duration")
	}
}

// Config holds the node configuration
type Config struct {
	// Validator configuration
	ValidatorKeyPath string `json:"validator_key_path"` // Path to validator private key file
	ValidatorPubKey  string `json:"validator_pub_key"`  // Base64-encoded public key

	// Network configuration
	P2PPort        int      `json:"p2p_port"`        // Port for P2P communication
	P2PAddress     string   `json:"p2p_address"`     // Full P2P address (e.g., /ip4/127.0.0.1/tcp/4001)
	BootstrapPeers []string `json:"bootstrap_peers"` // List of bootstrap peer addresses
	NetworkID      string   `json:"network_id"`      // Network identifier for isolation

	// Consensus configuration
	BlockInterval         Duration `json:"block_interval"`           // Time between block proposals (e.g., "5s")
	EmptyBlockWaitTimeout Duration `json:"empty_block_wait_timeout"` // Max time to wait for votes before proposing empty block (e.g., "1s")
	ValidatorConfig       string   `json:"validator_config"`         // Path to validator registry config file

	// Storage configuration
	DataDir string `json:"data_dir"` // Directory for blockchain data storage

	// Eligibility configuration
	EligibilityListPath string `json:"eligibility_list_path"` // Path to voter eligibility list file
}

// ValidatorInfo represents a validator's configuration
type ValidatorInfo struct {
	PublicKey string `json:"public_key"` // Base64-encoded public key
	Address   string `json:"address"`    // Network address
}

// ValidatorRegistryConfig holds the list of authorized validators
type ValidatorRegistryConfig struct {
	Validators []ValidatorInfo `json:"validators"`
}

// LoadConfig loads the node configuration from a JSON file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate required fields
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// Validate checks that all required configuration fields are set
func (c *Config) Validate() error {
	if c.ValidatorKeyPath == "" {
		return fmt.Errorf("validator_key_path is required")
	}

	if c.P2PPort <= 0 {
		return fmt.Errorf("p2p_port must be positive")
	}

	if c.BlockInterval.Duration <= 0 {
		return fmt.Errorf("block_interval must be positive")
	}

	if c.ValidatorConfig == "" {
		return fmt.Errorf("validator_config is required")
	}

	if c.DataDir == "" {
		return fmt.Errorf("data_dir is required")
	}

	if c.EligibilityListPath == "" {
		return fmt.Errorf("eligibility_list_path is required")
	}

	return nil
}

// LoadValidatorRegistry loads the validator registry from a JSON file
func LoadValidatorRegistry(path string) (*ValidatorRegistryConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read validator registry file: %w", err)
	}

	var registry ValidatorRegistryConfig
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("failed to parse validator registry file: %w", err)
	}

	if len(registry.Validators) == 0 {
		return nil, fmt.Errorf("validator registry must contain at least one validator")
	}

	return &registry, nil
}

// DefaultConfig returns a default configuration for testing/development
func DefaultConfig() *Config {
	return &Config{
		ValidatorKeyPath:      "validator.key",
		P2PPort:               4001,
		P2PAddress:            "/ip4/127.0.0.1/tcp/4001",
		BootstrapPeers:        []string{},
		NetworkID:             "peer-vote-testnet",
		BlockInterval:         Duration{5 * time.Second},
		EmptyBlockWaitTimeout: Duration{1 * time.Second},
		ValidatorConfig:       "validators.json",
		DataDir:               "./data",
		EligibilityListPath:   "eligibility.json",
	}
}
