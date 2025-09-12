package consensus

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/matscats/peer-vote/peer-vote/domain/services"
	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
)

// ValidatorStatus representa o status de um validador
type ValidatorStatus string

const (
	// ValidatorActive validador ativo e funcionando
	ValidatorActive ValidatorStatus = "ACTIVE"
	// ValidatorInactive validador temporariamente inativo
	ValidatorInactive ValidatorStatus = "INACTIVE"
	// ValidatorPenalized validador penalizado
	ValidatorPenalized ValidatorStatus = "PENALIZED"
	// ValidatorBanned validador banido permanentemente
	ValidatorBanned ValidatorStatus = "BANNED"
)

// Validator representa um validador na rede PoA
type Validator struct {
	NodeID         valueobjects.NodeID
	PublicKey      *services.PublicKey
	Status         ValidatorStatus
	AddedAt        valueobjects.Timestamp
	LastActiveAt   valueobjects.Timestamp
	MissedRounds   int
	TotalRounds    int
	PenaltyCount   int
	PenaltyExpiry  valueobjects.Timestamp
}

// ValidatorManager gerencia os validadores autorizados
type ValidatorManager struct {
	validators       map[string]*Validator // NodeID -> Validator
	validatorOrder   []valueobjects.NodeID // Ordem para Round Robin
	currentIndex     int                   // Índice atual no Round Robin
	maxMissedRounds  int                   // Máximo de rounds perdidos antes de penalidade
	penaltyDuration  time.Duration         // Duração da penalidade
	maxPenalties     int                   // Máximo de penalidades antes de ban
	
	// Mutex para operações thread-safe
	mu sync.RWMutex
}

// NewValidatorManager cria um novo gerenciador de validadores
func NewValidatorManager() *ValidatorManager {
	return &ValidatorManager{
		validators:      make(map[string]*Validator),
		validatorOrder:  make([]valueobjects.NodeID, 0),
		currentIndex:    0,
		maxMissedRounds: 3,                    // 3 rounds perdidos = penalidade
		penaltyDuration: time.Hour * 24,       // 24 horas de penalidade
		maxPenalties:    5,                    // 5 penalidades = ban permanente
	}
}

// AddValidator adiciona um novo validador à lista
func (vm *ValidatorManager) AddValidator(ctx context.Context, nodeID valueobjects.NodeID, publicKey *services.PublicKey) error {
	if nodeID.IsEmpty() {
		return errors.New("node ID cannot be empty")
	}

	if publicKey == nil || !publicKey.IsValid() {
		return errors.New("invalid public key")
	}

	vm.mu.Lock()
	defer vm.mu.Unlock()

	nodeIDStr := nodeID.String()
	
	// Verificar se já existe
	if _, exists := vm.validators[nodeIDStr]; exists {
		return fmt.Errorf("validator %s already exists", nodeID.ShortString())
	}

	// Criar novo validador
	validator := &Validator{
		NodeID:       nodeID,
		PublicKey:    publicKey,
		Status:       ValidatorActive,
		AddedAt:      valueobjects.Now(),
		LastActiveAt: valueobjects.Now(),
		MissedRounds: 0,
		TotalRounds:  0,
		PenaltyCount: 0,
	}

	// Adicionar aos mapas
	vm.validators[nodeIDStr] = validator
	vm.validatorOrder = append(vm.validatorOrder, nodeID)

	return nil
}

// RemoveValidator remove um validador da lista
func (vm *ValidatorManager) RemoveValidator(ctx context.Context, nodeID valueobjects.NodeID) error {
	if nodeID.IsEmpty() {
		return errors.New("node ID cannot be empty")
	}

	vm.mu.Lock()
	defer vm.mu.Unlock()

	nodeIDStr := nodeID.String()
	
	// Verificar se existe
	if _, exists := vm.validators[nodeIDStr]; !exists {
		return fmt.Errorf("validator %s not found", nodeID.ShortString())
	}

	// Remover do mapa
	delete(vm.validators, nodeIDStr)

	// Remover da ordem
	for i, id := range vm.validatorOrder {
		if id.Equals(nodeID) {
			vm.validatorOrder = append(vm.validatorOrder[:i], vm.validatorOrder[i+1:]...)
			
			// Ajustar índice atual se necessário
			if i < vm.currentIndex {
				vm.currentIndex--
			} else if i == vm.currentIndex && vm.currentIndex >= len(vm.validatorOrder) {
				vm.currentIndex = 0
			}
			break
		}
	}

	return nil
}

// GetValidator retorna um validador específico
func (vm *ValidatorManager) GetValidator(ctx context.Context, nodeID valueobjects.NodeID) (*Validator, error) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	validator, exists := vm.validators[nodeID.String()]
	if !exists {
		return nil, fmt.Errorf("validator %s not found", nodeID.ShortString())
	}

	return validator, nil
}

// GetAllValidators retorna todos os validadores
func (vm *ValidatorManager) GetAllValidators(ctx context.Context) ([]*Validator, error) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	validators := make([]*Validator, 0, len(vm.validators))
	for _, validator := range vm.validators {
		validators = append(validators, validator)
	}

	return validators, nil
}

// GetActiveValidators retorna apenas os validadores ativos
func (vm *ValidatorManager) GetActiveValidators(ctx context.Context) ([]*Validator, error) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	var activeValidators []*Validator
	
	for _, validator := range vm.validators {
		if vm.isValidatorActive(validator) {
			activeValidators = append(activeValidators, validator)
		}
	}

	return activeValidators, nil
}

// IsValidator verifica se um nó é um validador autorizado
func (vm *ValidatorManager) IsValidator(ctx context.Context, nodeID valueobjects.NodeID) (bool, error) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	validator, exists := vm.validators[nodeID.String()]
	if !exists {
		return false, nil
	}

	return vm.isValidatorActive(validator), nil
}

// GetValidatorCount retorna o número total de validadores
func (vm *ValidatorManager) GetValidatorCount(ctx context.Context) (int, error) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	return len(vm.validators), nil
}

// GetActiveValidatorCount retorna o número de validadores ativos
func (vm *ValidatorManager) GetActiveValidatorCount(ctx context.Context) (int, error) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	count := 0
	for _, validator := range vm.validators {
		if vm.isValidatorActive(validator) {
			count++
		}
	}

	return count, nil
}

// UpdateValidatorActivity atualiza a atividade de um validador
func (vm *ValidatorManager) UpdateValidatorActivity(ctx context.Context, nodeID valueobjects.NodeID, active bool) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	validator, exists := vm.validators[nodeID.String()]
	if !exists {
		return fmt.Errorf("validator %s not found", nodeID.ShortString())
	}

	validator.TotalRounds++
	
	if active {
		validator.LastActiveAt = valueobjects.Now()
		validator.MissedRounds = 0 // Reset contador de rounds perdidos
	} else {
		validator.MissedRounds++
		
		// Verificar se deve ser penalizado
		if validator.MissedRounds >= vm.maxMissedRounds {
			vm.penalizeValidator(validator)
		}
	}

	return nil
}

// PenalizeValidator aplica penalidade a um validador
func (vm *ValidatorManager) PenalizeValidator(ctx context.Context, nodeID valueobjects.NodeID, reason string) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	validator, exists := vm.validators[nodeID.String()]
	if !exists {
		return fmt.Errorf("validator %s not found", nodeID.ShortString())
	}

	vm.penalizeValidator(validator)
	return nil
}

// penalizeValidator aplica penalidade interna (deve ser chamado com lock)
func (vm *ValidatorManager) penalizeValidator(validator *Validator) {
	validator.PenaltyCount++
	validator.MissedRounds = 0 // Reset contador
	
	if validator.PenaltyCount >= vm.maxPenalties {
		// Ban permanente
		validator.Status = ValidatorBanned
	} else {
		// Penalidade temporária
		validator.Status = ValidatorPenalized
		validator.PenaltyExpiry = valueobjects.Now().Add(vm.penaltyDuration)
	}
}

// isValidatorActive verifica se um validador está ativo (deve ser chamado com lock)
func (vm *ValidatorManager) isValidatorActive(validator *Validator) bool {
	switch validator.Status {
	case ValidatorActive:
		return true
	case ValidatorPenalized:
		// Verificar se a penalidade expirou
		if valueobjects.Now().After(validator.PenaltyExpiry) {
			validator.Status = ValidatorActive
			return true
		}
		return false
	case ValidatorInactive, ValidatorBanned:
		return false
	default:
		return false
	}
}

// SetValidatorStatus define o status de um validador
func (vm *ValidatorManager) SetValidatorStatus(ctx context.Context, nodeID valueobjects.NodeID, status ValidatorStatus) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	validator, exists := vm.validators[nodeID.String()]
	if !exists {
		return fmt.Errorf("validator %s not found", nodeID.ShortString())
	}

	validator.Status = status
	
	// Se reativando, limpar penalidade
	if status == ValidatorActive {
		validator.PenaltyExpiry = valueobjects.Timestamp{}
	}

	return nil
}

// GetValidatorPublicKey retorna a chave pública de um validador
func (vm *ValidatorManager) GetValidatorPublicKey(ctx context.Context, nodeID valueobjects.NodeID) (*services.PublicKey, error) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	validator, exists := vm.validators[nodeID.String()]
	if !exists {
		return nil, fmt.Errorf("validator %s not found", nodeID.ShortString())
	}

	if validator.PublicKey == nil {
		return nil, fmt.Errorf("validator %s has no public key", nodeID.ShortString())
	}

	return validator.PublicKey, nil
}

// GetValidatorStats retorna estatísticas de um validador
func (vm *ValidatorManager) GetValidatorStats(ctx context.Context, nodeID valueobjects.NodeID) (*ValidatorStats, error) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	validator, exists := vm.validators[nodeID.String()]
	if !exists {
		return nil, fmt.Errorf("validator %s not found", nodeID.ShortString())
	}

	successRate := float64(0)
	if validator.TotalRounds > 0 {
		successfulRounds := validator.TotalRounds - validator.MissedRounds
		successRate = float64(successfulRounds) / float64(validator.TotalRounds) * 100
	}

	return &ValidatorStats{
		NodeID:       validator.NodeID,
		Status:       validator.Status,
		TotalRounds:  validator.TotalRounds,
		MissedRounds: validator.MissedRounds,
		SuccessRate:  successRate,
		PenaltyCount: validator.PenaltyCount,
		AddedAt:      validator.AddedAt,
		LastActiveAt: validator.LastActiveAt,
	}, nil
}

// ValidatorStats representa estatísticas de um validador
type ValidatorStats struct {
	NodeID       valueobjects.NodeID
	Status       ValidatorStatus
	TotalRounds  int
	MissedRounds int
	SuccessRate  float64
	PenaltyCount int
	AddedAt      valueobjects.Timestamp
	LastActiveAt valueobjects.Timestamp
}

// SetConfiguration define configurações do gerenciador
func (vm *ValidatorManager) SetConfiguration(maxMissedRounds int, penaltyDuration time.Duration, maxPenalties int) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if maxMissedRounds > 0 {
		vm.maxMissedRounds = maxMissedRounds
	}
	if penaltyDuration > 0 {
		vm.penaltyDuration = penaltyDuration
	}
	if maxPenalties > 0 {
		vm.maxPenalties = maxPenalties
	}
}

// GetConfiguration retorna as configurações atuais
func (vm *ValidatorManager) GetConfiguration() (int, time.Duration, int) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	return vm.maxMissedRounds, vm.penaltyDuration, vm.maxPenalties
}

// CleanupExpiredPenalties limpa penalidades expiradas
func (vm *ValidatorManager) CleanupExpiredPenalties(ctx context.Context) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	now := valueobjects.Now()
	
	for _, validator := range vm.validators {
		if validator.Status == ValidatorPenalized && now.After(validator.PenaltyExpiry) {
			validator.Status = ValidatorActive
			validator.PenaltyExpiry = valueobjects.Timestamp{}
		}
	}

	return nil
}
