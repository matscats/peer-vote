package adapters

import (
	"encoding/json"
	"fmt"
	"os"
)

// EligibilityConfig represents the JSON structure for eligibility list
type EligibilityConfig struct {
	EligibleVoters []string `json:"eligible_voters"`
}

// LoadEligibilityListFromFile loads the eligibility list from a JSON file
func LoadEligibilityListFromFile(path string) (*EligibilityList, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read eligibility file: %w", err)
	}

	var config EligibilityConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse eligibility file: %w", err)
	}

	if len(config.EligibleVoters) == 0 {
		return nil, fmt.Errorf("eligibility list must contain at least one voter")
	}

	return NewEligibilityList(config.EligibleVoters), nil
}
