package consensus

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
)

// PenaltyType representa o tipo de penalidade
type PenaltyType string

const (
	// PenaltyMissedRound penalidade por perder round
	PenaltyMissedRound PenaltyType = "MISSED_ROUND"
	// PenaltyInvalidBlock penalidade por propor bloco inválido
	PenaltyInvalidBlock PenaltyType = "INVALID_BLOCK"
	// PenaltyDoubleSign penalidade por assinar múltiplos blocos no mesmo round
	PenaltyDoubleSign PenaltyType = "DOUBLE_SIGN"
	// PenaltyTimeout penalidade por timeout
	PenaltyTimeout PenaltyType = "TIMEOUT"
	// PenaltyMaliciousBehavior penalidade por comportamento malicioso
	PenaltyMaliciousBehavior PenaltyType = "MALICIOUS_BEHAVIOR"
)

// PenaltySeverity representa a severidade da penalidade
type PenaltySeverity int

const (
	// SeverityMinor penalidade menor
	SeverityMinor PenaltySeverity = 1
	// SeverityModerate penalidade moderada
	SeverityModerate PenaltySeverity = 2
	// SeverityMajor penalidade maior
	SeverityMajor PenaltySeverity = 3
	// SeverityCritical penalidade crítica
	SeverityCritical PenaltySeverity = 4
)

// PenaltyRecord representa um registro de penalidade
type PenaltyRecord struct {
	ID          string
	ValidatorID valueobjects.NodeID
	Type        PenaltyType
	Severity    PenaltySeverity
	Reason      string
	AppliedAt   valueobjects.Timestamp
	ExpiresAt   valueobjects.Timestamp
	IsActive    bool
	Evidence    map[string]interface{} // Evidências da penalidade
}

// PenaltyRule define uma regra de penalidade
type PenaltyRule struct {
	Type        PenaltyType
	Severity    PenaltySeverity
	Duration    time.Duration
	MaxCount    int           // Máximo de penalidades deste tipo antes de ban
	BanDuration time.Duration // Duração do ban se exceder MaxCount
}

// PenaltySystem gerencia o sistema de penalidades
type PenaltySystem struct {
	validatorManager *ValidatorManager
	
	// Registros de penalidades
	penalties map[string]*PenaltyRecord // ID -> PenaltyRecord
	
	// Penalidades por validador
	validatorPenalties map[string][]*PenaltyRecord // ValidatorID -> []PenaltyRecord
	
	// Regras de penalidade
	penaltyRules map[PenaltyType]*PenaltyRule
	
	// Configurações
	maxPenaltyHistory int           // Máximo de penalidades no histórico
	cleanupInterval   time.Duration // Intervalo para limpeza de penalidades expiradas
	
	// Mutex para operações thread-safe
	mu sync.RWMutex
	
	// Canal para notificações
	penaltyNotifications chan PenaltyNotification
}

// PenaltyNotification representa uma notificação de penalidade
type PenaltyNotification struct {
	ValidatorID valueobjects.NodeID
	Type        PenaltyType
	Severity    PenaltySeverity
	Reason      string
	Timestamp   valueobjects.Timestamp
	Action      string // "APPLIED", "EXPIRED", "REMOVED"
}

// NewPenaltySystem cria um novo sistema de penalidades
func NewPenaltySystem(validatorManager *ValidatorManager) *PenaltySystem {
	ps := &PenaltySystem{
		validatorManager:     validatorManager,
		penalties:            make(map[string]*PenaltyRecord),
		validatorPenalties:   make(map[string][]*PenaltyRecord),
		penaltyRules:         make(map[PenaltyType]*PenaltyRule),
		maxPenaltyHistory:    1000,
		cleanupInterval:      time.Hour,
		penaltyNotifications: make(chan PenaltyNotification, 100),
	}

	// Configurar regras padrão
	ps.setupDefaultRules()

	return ps
}

// setupDefaultRules configura as regras padrão de penalidade
func (ps *PenaltySystem) setupDefaultRules() {
	ps.penaltyRules[PenaltyMissedRound] = &PenaltyRule{
		Type:        PenaltyMissedRound,
		Severity:    SeverityMinor,
		Duration:    time.Minute * 30,
		MaxCount:    5,
		BanDuration: time.Hour * 24,
	}

	ps.penaltyRules[PenaltyInvalidBlock] = &PenaltyRule{
		Type:        PenaltyInvalidBlock,
		Severity:    SeverityModerate,
		Duration:    time.Hour * 2,
		MaxCount:    3,
		BanDuration: time.Hour * 48,
	}

	ps.penaltyRules[PenaltyDoubleSign] = &PenaltyRule{
		Type:        PenaltyDoubleSign,
		Severity:    SeverityCritical,
		Duration:    time.Hour * 24,
		MaxCount:    1,
		BanDuration: time.Hour * 24 * 7, // 1 semana
	}

	ps.penaltyRules[PenaltyTimeout] = &PenaltyRule{
		Type:        PenaltyTimeout,
		Severity:    SeverityMinor,
		Duration:    time.Minute * 15,
		MaxCount:    10,
		BanDuration: time.Hour * 12,
	}

	ps.penaltyRules[PenaltyMaliciousBehavior] = &PenaltyRule{
		Type:        PenaltyMaliciousBehavior,
		Severity:    SeverityCritical,
		Duration:    time.Hour * 24 * 7, // 1 semana
		MaxCount:    1,
		BanDuration: 0, // Ban permanente
	}
}

// ApplyPenalty aplica uma penalidade a um validador
func (ps *PenaltySystem) ApplyPenalty(ctx context.Context, validatorID valueobjects.NodeID, penaltyType PenaltyType, reason string, evidence map[string]interface{}) error {
	if validatorID.IsEmpty() {
		return fmt.Errorf("validator ID cannot be empty")
	}

	ps.mu.Lock()
	defer ps.mu.Unlock()

	// Verificar se o validador existe
	_, err := ps.validatorManager.GetValidator(ctx, validatorID)
	if err != nil {
		return fmt.Errorf("validator not found: %w", err)
	}

	// Obter regra de penalidade
	rule, exists := ps.penaltyRules[penaltyType]
	if !exists {
		return fmt.Errorf("penalty rule not found for type: %s", penaltyType)
	}

	// Criar registro de penalidade
	penaltyID := ps.generatePenaltyID(validatorID, penaltyType)
	now := valueobjects.Now()
	expiresAt := now.Add(rule.Duration)

	penalty := &PenaltyRecord{
		ID:          penaltyID,
		ValidatorID: validatorID,
		Type:        penaltyType,
		Severity:    rule.Severity,
		Reason:      reason,
		AppliedAt:   now,
		ExpiresAt:   expiresAt,
		IsActive:    true,
		Evidence:    evidence,
	}

	// Adicionar aos registros
	ps.penalties[penaltyID] = penalty

	validatorIDStr := validatorID.String()
	if ps.validatorPenalties[validatorIDStr] == nil {
		ps.validatorPenalties[validatorIDStr] = make([]*PenaltyRecord, 0)
	}
	ps.validatorPenalties[validatorIDStr] = append(ps.validatorPenalties[validatorIDStr], penalty)

	// Verificar se deve banir o validador
	if err := ps.checkForBan(ctx, validatorID, penaltyType); err != nil {
		return fmt.Errorf("failed to check for ban: %w", err)
	}

	// Aplicar penalidade no validador
	if err := ps.validatorManager.PenalizeValidator(ctx, validatorID, reason); err != nil {
		return fmt.Errorf("failed to penalize validator: %w", err)
	}

	// Notificar
	notification := PenaltyNotification{
		ValidatorID: validatorID,
		Type:        penaltyType,
		Severity:    rule.Severity,
		Reason:      reason,
		Timestamp:   now,
		Action:      "APPLIED",
	}

	select {
	case ps.penaltyNotifications <- notification:
	default:
		// Canal cheio, pular notificação
	}

	return nil
}

// checkForBan verifica se um validador deve ser banido
func (ps *PenaltySystem) checkForBan(ctx context.Context, validatorID valueobjects.NodeID, penaltyType PenaltyType) error {
	rule := ps.penaltyRules[penaltyType]
	if rule.MaxCount <= 0 {
		return nil // Sem limite
	}

	// Contar penalidades ativas deste tipo
	count := ps.countActivePenalties(validatorID, penaltyType)
	
	if count >= rule.MaxCount {
		// Banir validador
		if rule.BanDuration == 0 {
			// Ban permanente
			return ps.validatorManager.SetValidatorStatus(ctx, validatorID, ValidatorBanned)
		} else {
			// Ban temporário
			return ps.validatorManager.SetValidatorStatus(ctx, validatorID, ValidatorPenalized)
		}
	}

	return nil
}

// countActivePenalties conta penalidades ativas de um tipo específico
func (ps *PenaltySystem) countActivePenalties(validatorID valueobjects.NodeID, penaltyType PenaltyType) int {
	validatorIDStr := validatorID.String()
	penalties := ps.validatorPenalties[validatorIDStr]
	
	count := 0
	now := valueobjects.Now()
	
	for _, penalty := range penalties {
		if penalty.Type == penaltyType && penalty.IsActive && now.Before(penalty.ExpiresAt) {
			count++
		}
	}
	
	return count
}

// GetValidatorPenalties retorna as penalidades de um validador
func (ps *PenaltySystem) GetValidatorPenalties(ctx context.Context, validatorID valueobjects.NodeID) ([]*PenaltyRecord, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	validatorIDStr := validatorID.String()
	penalties := ps.validatorPenalties[validatorIDStr]
	
	// Retornar cópia para evitar modificações externas
	result := make([]*PenaltyRecord, len(penalties))
	copy(result, penalties)
	
	return result, nil
}

// GetActivePenalties retorna as penalidades ativas de um validador
func (ps *PenaltySystem) GetActivePenalties(ctx context.Context, validatorID valueobjects.NodeID) ([]*PenaltyRecord, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	validatorIDStr := validatorID.String()
	penalties := ps.validatorPenalties[validatorIDStr]
	
	var activePenalties []*PenaltyRecord
	now := valueobjects.Now()
	
	for _, penalty := range penalties {
		if penalty.IsActive && now.Before(penalty.ExpiresAt) {
			activePenalties = append(activePenalties, penalty)
		}
	}
	
	return activePenalties, nil
}

// RemovePenalty remove uma penalidade específica
func (ps *PenaltySystem) RemovePenalty(ctx context.Context, penaltyID string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	penalty, exists := ps.penalties[penaltyID]
	if !exists {
		return fmt.Errorf("penalty not found: %s", penaltyID)
	}

	// Marcar como inativa
	penalty.IsActive = false

	// Notificar
	notification := PenaltyNotification{
		ValidatorID: penalty.ValidatorID,
		Type:        penalty.Type,
		Severity:    penalty.Severity,
		Reason:      "Penalty removed",
		Timestamp:   valueobjects.Now(),
		Action:      "REMOVED",
	}

	select {
	case ps.penaltyNotifications <- notification:
	default:
		// Canal cheio, pular notificação
	}

	return nil
}

// CleanupExpiredPenalties remove penalidades expiradas
func (ps *PenaltySystem) CleanupExpiredPenalties(ctx context.Context) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	now := valueobjects.Now()
	var expiredPenalties []*PenaltyRecord

	// Encontrar penalidades expiradas
	for _, penalty := range ps.penalties {
		if penalty.IsActive && now.After(penalty.ExpiresAt) {
			penalty.IsActive = false
			expiredPenalties = append(expiredPenalties, penalty)
		}
	}

	// Notificar penalidades expiradas
	for _, penalty := range expiredPenalties {
		notification := PenaltyNotification{
			ValidatorID: penalty.ValidatorID,
			Type:        penalty.Type,
			Severity:    penalty.Severity,
			Reason:      "Penalty expired",
			Timestamp:   now,
			Action:      "EXPIRED",
		}

		select {
		case ps.penaltyNotifications <- notification:
		default:
			// Canal cheio, pular notificação
		}
	}

	return nil
}

// GetPenaltyStats retorna estatísticas de penalidades
func (ps *PenaltySystem) GetPenaltyStats(ctx context.Context) (*PenaltyStats, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	stats := &PenaltyStats{
		TotalPenalties:  len(ps.penalties),
		ActivePenalties: 0,
		PenaltiesByType: make(map[PenaltyType]int),
		PenaltiesBySeverity: make(map[PenaltySeverity]int),
	}

	now := valueobjects.Now()

	for _, penalty := range ps.penalties {
		stats.PenaltiesByType[penalty.Type]++
		stats.PenaltiesBySeverity[penalty.Severity]++
		
		if penalty.IsActive && now.Before(penalty.ExpiresAt) {
			stats.ActivePenalties++
		}
	}

	return stats, nil
}

// PenaltyStats representa estatísticas de penalidades
type PenaltyStats struct {
	TotalPenalties      int
	ActivePenalties     int
	PenaltiesByType     map[PenaltyType]int
	PenaltiesBySeverity map[PenaltySeverity]int
}

// SetPenaltyRule define uma regra de penalidade
func (ps *PenaltySystem) SetPenaltyRule(penaltyType PenaltyType, rule *PenaltyRule) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.penaltyRules[penaltyType] = rule
}

// GetPenaltyRule retorna uma regra de penalidade
func (ps *PenaltySystem) GetPenaltyRule(penaltyType PenaltyType) (*PenaltyRule, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	rule, exists := ps.penaltyRules[penaltyType]
	return rule, exists
}

// GetNotificationChannel retorna o canal de notificações
func (ps *PenaltySystem) GetNotificationChannel() <-chan PenaltyNotification {
	return ps.penaltyNotifications
}

// generatePenaltyID gera um ID único para a penalidade
func (ps *PenaltySystem) generatePenaltyID(validatorID valueobjects.NodeID, penaltyType PenaltyType) string {
	timestamp := valueobjects.Now().Unix()
	return fmt.Sprintf("%s-%s-%d", validatorID.ShortString(), penaltyType, timestamp)
}

// StartCleanupRoutine inicia a rotina de limpeza automática
func (ps *PenaltySystem) StartCleanupRoutine(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(ps.cleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				ps.CleanupExpiredPenalties(ctx)
			}
		}
	}()
}

// GetValidatorPenaltyScore calcula um score de penalidade para um validador
func (ps *PenaltySystem) GetValidatorPenaltyScore(ctx context.Context, validatorID valueobjects.NodeID) (float64, error) {
	penalties, err := ps.GetActivePenalties(ctx, validatorID)
	if err != nil {
		return 0, err
	}

	score := float64(0)
	for _, penalty := range penalties {
		score += float64(penalty.Severity)
	}

	return score, nil
}
