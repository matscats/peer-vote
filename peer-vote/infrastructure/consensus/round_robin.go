package consensus

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
)

// RoundRobinScheduler implementa a seleção Round Robin de validadores
type RoundRobinScheduler struct {
	validatorManager *ValidatorManager
	
	// Estado atual do Round Robin
	currentRound     uint64
	currentValidator valueobjects.NodeID
	roundStartTime   valueobjects.Timestamp
	
	// Configurações
	roundDuration    time.Duration // Duração de cada round
	timeoutDuration  time.Duration // Timeout para validador responder
	
	// Cache de validadores ativos (para performance)
	activeValidators []valueobjects.NodeID
	currentIndex     int
	
	// Mutex para operações thread-safe
	mu sync.RWMutex
	
	// Canal para notificações de mudança de round
	roundChangeChan chan RoundChangeEvent
}

// RoundChangeEvent representa uma mudança de round
type RoundChangeEvent struct {
	OldRound      uint64
	NewRound      uint64
	OldValidator  valueobjects.NodeID
	NewValidator  valueobjects.NodeID
	Timestamp     valueobjects.Timestamp
	Reason        string
}

// NewRoundRobinScheduler cria um novo scheduler Round Robin
func NewRoundRobinScheduler(validatorManager *ValidatorManager) *RoundRobinScheduler {
	return &RoundRobinScheduler{
		validatorManager: validatorManager,
		currentRound:     0,
		roundDuration:    time.Second * 5,  // 5 segundos por round
		timeoutDuration:  time.Second * 4,  // 4 segundos de timeout
		activeValidators: make([]valueobjects.NodeID, 0),
		currentIndex:     0,
		roundChangeChan:  make(chan RoundChangeEvent, 100),
	}
}

// Start inicia o scheduler Round Robin
func (rr *RoundRobinScheduler) Start(ctx context.Context) error {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	// Carregar validadores ativos
	if err := rr.refreshActiveValidators(ctx); err != nil {
		return fmt.Errorf("failed to refresh active validators: %w", err)
	}

	if len(rr.activeValidators) == 0 {
		return errors.New("no active validators available")
	}

	// Inicializar primeiro round
	rr.currentRound = 1
	rr.currentValidator = rr.activeValidators[0]
	rr.currentIndex = 0
	rr.roundStartTime = valueobjects.Now()

	// Iniciar goroutine para gerenciar rounds
	go rr.manageRounds(ctx)

	return nil
}

// GetCurrentValidator retorna o validador atual
func (rr *RoundRobinScheduler) GetCurrentValidator(ctx context.Context) (valueobjects.NodeID, error) {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	if rr.currentValidator.IsEmpty() {
		return valueobjects.EmptyNodeID(), errors.New("no current validator set")
	}

	return rr.currentValidator, nil
}

// GetNextValidator retorna o próximo validador na sequência
func (rr *RoundRobinScheduler) GetNextValidator(ctx context.Context) (valueobjects.NodeID, error) {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	if len(rr.activeValidators) == 0 {
		return valueobjects.EmptyNodeID(), errors.New("no active validators available")
	}

	nextIndex := (rr.currentIndex + 1) % len(rr.activeValidators)
	return rr.activeValidators[nextIndex], nil
}

// GetCurrentRound retorna o round atual
func (rr *RoundRobinScheduler) GetCurrentRound(ctx context.Context) (uint64, error) {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	return rr.currentRound, nil
}

// AdvanceRound avança para o próximo round
func (rr *RoundRobinScheduler) AdvanceRound(ctx context.Context) error {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	return rr.advanceRoundInternal(ctx, "manual advance")
}

// advanceRoundInternal avança o round internamente (deve ser chamado com lock)
func (rr *RoundRobinScheduler) advanceRoundInternal(ctx context.Context, reason string) error {
	// Atualizar atividade do validador atual
	if !rr.currentValidator.IsEmpty() {
		// Marcar como ativo se conseguiu produzir bloco, inativo caso contrário
		active := reason == "block produced" || reason == "manual advance"
		rr.validatorManager.UpdateValidatorActivity(ctx, rr.currentValidator, active)
	}

	// Refresh validadores ativos
	if err := rr.refreshActiveValidators(ctx); err != nil {
		return fmt.Errorf("failed to refresh active validators: %w", err)
	}

	if len(rr.activeValidators) == 0 {
		return errors.New("no active validators available")
	}

	// Salvar estado anterior
	oldRound := rr.currentRound
	oldValidator := rr.currentValidator

	// Avançar para próximo validador
	rr.currentIndex = (rr.currentIndex + 1) % len(rr.activeValidators)
	rr.currentValidator = rr.activeValidators[rr.currentIndex]
	rr.currentRound++
	rr.roundStartTime = valueobjects.Now()

	// Notificar mudança de round
	event := RoundChangeEvent{
		OldRound:     oldRound,
		NewRound:     rr.currentRound,
		OldValidator: oldValidator,
		NewValidator: rr.currentValidator,
		Timestamp:    rr.roundStartTime,
		Reason:       reason,
	}

	select {
	case rr.roundChangeChan <- event:
	default:
		// Canal cheio, pular notificação
	}

	return nil
}

// HandleTimeout lida com timeout de validador
func (rr *RoundRobinScheduler) HandleTimeout(ctx context.Context, validator valueobjects.NodeID) error {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	// Verificar se é o validador atual
	if !rr.currentValidator.Equals(validator) {
		return fmt.Errorf("timeout for non-current validator: %s", validator.ShortString())
	}

	// Avançar round devido a timeout
	return rr.advanceRoundInternal(ctx, "validator timeout")
}

// IsMyTurn verifica se é a vez de um validador específico
func (rr *RoundRobinScheduler) IsMyTurn(ctx context.Context, nodeID valueobjects.NodeID) (bool, error) {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	return rr.currentValidator.Equals(nodeID), nil
}

// GetRoundInfo retorna informações sobre o round atual
func (rr *RoundRobinScheduler) GetRoundInfo(ctx context.Context) (*RoundInfo, error) {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	timeElapsed := valueobjects.Now().Sub(rr.roundStartTime)
	timeRemaining := rr.roundDuration - timeElapsed
	if timeRemaining < 0 {
		timeRemaining = 0
	}

	return &RoundInfo{
		Round:           rr.currentRound,
		CurrentValidator: rr.currentValidator,
		StartTime:       rr.roundStartTime,
		Duration:        rr.roundDuration,
		TimeElapsed:     timeElapsed,
		TimeRemaining:   timeRemaining,
		ValidatorCount:  len(rr.activeValidators),
		ValidatorIndex:  rr.currentIndex,
	}, nil
}

// RoundInfo contém informações sobre um round
type RoundInfo struct {
	Round            uint64
	CurrentValidator valueobjects.NodeID
	StartTime        valueobjects.Timestamp
	Duration         time.Duration
	TimeElapsed      time.Duration
	TimeRemaining    time.Duration
	ValidatorCount   int
	ValidatorIndex   int
}

// GetRoundChangeChannel retorna o canal de notificações de mudança de round
func (rr *RoundRobinScheduler) GetRoundChangeChannel() <-chan RoundChangeEvent {
	return rr.roundChangeChan
}

// refreshActiveValidators atualiza a lista de validadores ativos
func (rr *RoundRobinScheduler) refreshActiveValidators(ctx context.Context) error {
	activeValidators, err := rr.validatorManager.GetActiveValidators(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active validators: %w", err)
	}

	// Converter para slice de NodeIDs
	newActiveValidators := make([]valueobjects.NodeID, len(activeValidators))
	for i, validator := range activeValidators {
		newActiveValidators[i] = validator.NodeID
	}

	// Verificar se o validador atual ainda está ativo
	currentStillActive := false
	newCurrentIndex := 0
	
	for i, nodeID := range newActiveValidators {
		if nodeID.Equals(rr.currentValidator) {
			currentStillActive = true
			newCurrentIndex = i
			break
		}
	}

	// Atualizar lista
	rr.activeValidators = newActiveValidators
	
	if currentStillActive {
		rr.currentIndex = newCurrentIndex
	} else if len(rr.activeValidators) > 0 {
		// Validador atual não está mais ativo, escolher o primeiro
		rr.currentIndex = 0
		rr.currentValidator = rr.activeValidators[0]
	}

	return nil
}

// manageRounds gerencia os rounds automaticamente
func (rr *RoundRobinScheduler) manageRounds(ctx context.Context) {
	ticker := time.NewTicker(time.Second) // Verificar a cada segundo
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rr.checkRoundTimeout(ctx)
		}
	}
}

// checkRoundTimeout verifica se o round atual expirou
func (rr *RoundRobinScheduler) checkRoundTimeout(ctx context.Context) {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	if rr.roundStartTime.IsZero() {
		return
	}

	timeElapsed := valueobjects.Now().Sub(rr.roundStartTime)
	
	if timeElapsed >= rr.roundDuration {
		// Round expirou, avançar automaticamente
		rr.advanceRoundInternal(ctx, "round timeout")
	}
}

// SetRoundDuration define a duração de cada round
func (rr *RoundRobinScheduler) SetRoundDuration(duration time.Duration) {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	if duration > 0 {
		rr.roundDuration = duration
		// Timeout deve ser menor que a duração do round
		if rr.timeoutDuration >= duration {
			rr.timeoutDuration = duration - time.Second
		}
	}
}

// SetTimeoutDuration define a duração do timeout
func (rr *RoundRobinScheduler) SetTimeoutDuration(duration time.Duration) {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	if duration > 0 && duration < rr.roundDuration {
		rr.timeoutDuration = duration
	}
}

// GetConfiguration retorna as configurações atuais
func (rr *RoundRobinScheduler) GetConfiguration() (time.Duration, time.Duration) {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	return rr.roundDuration, rr.timeoutDuration
}

// Reset reinicia o scheduler
func (rr *RoundRobinScheduler) Reset(ctx context.Context) error {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	rr.currentRound = 0
	rr.currentValidator = valueobjects.EmptyNodeID()
	rr.currentIndex = 0
	rr.roundStartTime = valueobjects.Timestamp{}
	rr.activeValidators = make([]valueobjects.NodeID, 0)

	// Limpar canal de notificações
	for len(rr.roundChangeChan) > 0 {
		<-rr.roundChangeChan
	}

	return nil
}

// GetValidatorPosition retorna a posição de um validador na sequência Round Robin
func (rr *RoundRobinScheduler) GetValidatorPosition(ctx context.Context, nodeID valueobjects.NodeID) (int, error) {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	for i, validator := range rr.activeValidators {
		if validator.Equals(nodeID) {
			return i, nil
		}
	}

	return -1, fmt.Errorf("validator %s not found in active list", nodeID.ShortString())
}

// GetValidatorSequence retorna a sequência completa de validadores
func (rr *RoundRobinScheduler) GetValidatorSequence(ctx context.Context) ([]valueobjects.NodeID, error) {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	// Retornar cópia para evitar modificações externas
	sequence := make([]valueobjects.NodeID, len(rr.activeValidators))
	copy(sequence, rr.activeValidators)

	return sequence, nil
}

// NotifyBlockProduced notifica que um bloco foi produzido pelo validador atual
func (rr *RoundRobinScheduler) NotifyBlockProduced(ctx context.Context, validator valueobjects.NodeID) error {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	// Verificar se é o validador atual
	if !rr.currentValidator.Equals(validator) {
		return fmt.Errorf("block produced by non-current validator: %s", validator.ShortString())
	}

	// Avançar round após produção de bloco
	return rr.advanceRoundInternal(ctx, "block produced")
}
