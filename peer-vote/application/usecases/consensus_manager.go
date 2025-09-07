package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/matscats/peer-vote/peer-vote/domain/entities"
	"github.com/matscats/peer-vote/peer-vote/domain/services"
	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
	"github.com/matscats/peer-vote/peer-vote/infrastructure/blockchain"
	"github.com/matscats/peer-vote/peer-vote/infrastructure/consensus"
)

// ConsensusManagerUseCase gerencia o consenso PoA
type ConsensusManagerUseCase struct {
	poaEngine        *consensus.PoAEngine
	validatorManager *consensus.ValidatorManager
	penaltySystem    *consensus.PenaltySystem
	chainManager     *blockchain.ChainManager
	cryptoService    services.CryptographyService
}

// NewConsensusManagerUseCase cria um novo gerenciador de consenso
func NewConsensusManagerUseCase(
	poaEngine *consensus.PoAEngine,
	validatorManager *consensus.ValidatorManager,
	penaltySystem *consensus.PenaltySystem,
	chainManager *blockchain.ChainManager,
	cryptoService services.CryptographyService,
) *ConsensusManagerUseCase {
	return &ConsensusManagerUseCase{
		poaEngine:        poaEngine,
		validatorManager: validatorManager,
		penaltySystem:    penaltySystem,
		chainManager:     chainManager,
		cryptoService:    cryptoService,
	}
}

// StartConsensusRequest representa a requisição para iniciar consenso
type StartConsensusRequest struct {
	InitialValidators []ValidatorInfo
	GenesisTransactions []*entities.Transaction
}

// ValidatorInfo contém informações de um validador
type ValidatorInfo struct {
	NodeID    valueobjects.NodeID
	PublicKey *services.PublicKey
}

// StartConsensusResponse representa a resposta do início do consenso
type StartConsensusResponse struct {
	Success          bool
	Message          string
	GenesisBlockHash valueobjects.Hash
	ValidatorCount   int
	CurrentRound     uint64
}

// StartConsensus inicia o processo de consenso
func (uc *ConsensusManagerUseCase) StartConsensus(ctx context.Context, request *StartConsensusRequest) (*StartConsensusResponse, error) {
	if request == nil {
		return &StartConsensusResponse{
			Success: false,
			Message: "request is nil",
		}, fmt.Errorf("request cannot be nil")
	}

	if len(request.InitialValidators) == 0 {
		return &StartConsensusResponse{
			Success: false,
			Message: "no initial validators provided",
		}, fmt.Errorf("at least one validator is required")
	}

	// Adicionar validadores iniciais
	for _, validatorInfo := range request.InitialValidators {
		err := uc.validatorManager.AddValidator(ctx, validatorInfo.NodeID, validatorInfo.PublicKey)
		if err != nil {
			return &StartConsensusResponse{
				Success: false,
				Message: fmt.Sprintf("failed to add validator %s: %v", validatorInfo.NodeID.ShortString(), err),
			}, fmt.Errorf("failed to add validator: %w", err)
		}
	}

	// Criar bloco gênesis se necessário
	var genesisBlockHash valueobjects.Hash
	if len(request.GenesisTransactions) > 0 {
		// Para o gênesis, precisamos de uma chave privada - isso seria fornecido na configuração real
		// Por enquanto, vamos assumir que o bloco gênesis já foi criado
		genesisBlockHash = valueobjects.EmptyHash() // Placeholder
	}

	// Iniciar consenso
	err := uc.poaEngine.StartConsensus(ctx)
	if err != nil {
		return &StartConsensusResponse{
			Success: false,
			Message: fmt.Sprintf("failed to start consensus: %v", err),
		}, fmt.Errorf("failed to start consensus: %w", err)
	}

	// Obter informações atuais
	currentRound, _ := uc.poaEngine.GetCurrentRound(ctx)
	validatorCount, _ := uc.poaEngine.GetValidatorCount(ctx)

	return &StartConsensusResponse{
		Success:          true,
		Message:          "consensus started successfully",
		GenesisBlockHash: genesisBlockHash,
		ValidatorCount:   validatorCount,
		CurrentRound:     currentRound,
	}, nil
}

// AddValidatorRequest representa a requisição para adicionar validador
type AddValidatorRequest struct {
	NodeID    valueobjects.NodeID
	PublicKey *services.PublicKey
}

// AddValidatorResponse representa a resposta da adição de validador
type AddValidatorResponse struct {
	Success bool
	Message string
}

// AddValidator adiciona um novo validador
func (uc *ConsensusManagerUseCase) AddValidator(ctx context.Context, request *AddValidatorRequest) (*AddValidatorResponse, error) {
	if request == nil {
		return &AddValidatorResponse{
			Success: false,
			Message: "request is nil",
		}, fmt.Errorf("request cannot be nil")
	}

	if request.NodeID.IsEmpty() {
		return &AddValidatorResponse{
			Success: false,
			Message: "node ID is empty",
		}, fmt.Errorf("node ID cannot be empty")
	}

	if request.PublicKey == nil || !request.PublicKey.IsValid() {
		return &AddValidatorResponse{
			Success: false,
			Message: "invalid public key",
		}, fmt.Errorf("public key is invalid")
	}

	err := uc.poaEngine.AddValidator(ctx, request.NodeID, request.PublicKey)
	if err != nil {
		return &AddValidatorResponse{
			Success: false,
			Message: fmt.Sprintf("failed to add validator: %v", err),
		}, fmt.Errorf("failed to add validator: %w", err)
	}

	return &AddValidatorResponse{
		Success: true,
		Message: fmt.Sprintf("validator %s added successfully", request.NodeID.ShortString()),
	}, nil
}

// SubmitTransactionRequest representa a requisição para submeter transação
type SubmitTransactionRequest struct {
	Transaction *entities.Transaction
}

// SubmitTransactionResponse representa a resposta da submissão de transação
type SubmitTransactionResponse struct {
	Success       bool
	Message       string
	TransactionID valueobjects.Hash
}

// SubmitTransaction submete uma transação para o pool de consenso
func (uc *ConsensusManagerUseCase) SubmitTransaction(ctx context.Context, request *SubmitTransactionRequest) (*SubmitTransactionResponse, error) {
	if request == nil || request.Transaction == nil {
		return &SubmitTransactionResponse{
			Success: false,
			Message: "transaction is nil",
		}, fmt.Errorf("transaction cannot be nil")
	}

	// Preparar transação (calcular hash se necessário)
	tx := request.Transaction
	if tx.GetHash().IsEmpty() {
		txData := tx.ToBytes()
		txHash := uc.cryptoService.HashTransaction(ctx, txData)
		tx.SetHash(txHash)
	}

	if tx.GetID().IsEmpty() {
		tx.SetID(tx.GetHash())
	}

	// Submeter para o consenso
	err := uc.poaEngine.AddTransaction(ctx, tx)
	if err != nil {
		return &SubmitTransactionResponse{
			Success: false,
			Message: fmt.Sprintf("failed to submit transaction: %v", err),
		}, fmt.Errorf("failed to submit transaction: %w", err)
	}

	return &SubmitTransactionResponse{
		Success:       true,
		Message:       "transaction submitted successfully",
		TransactionID: tx.GetHash(),
	}, nil
}

// GetConsensusStatusResponse representa o status do consenso
type GetConsensusStatusResponse struct {
	IsRunning         bool
	CurrentValidator  valueobjects.NodeID
	CurrentRound      uint64
	ValidatorCount    int
	PendingTxCount    int
	LastBlockTime     valueobjects.Timestamp
	ValidatorStats    []ValidatorStatusInfo
}

// ValidatorStatusInfo contém informações de status de um validador
type ValidatorStatusInfo struct {
	NodeID       valueobjects.NodeID
	Status       consensus.ValidatorStatus
	TotalRounds  int
	MissedRounds int
	SuccessRate  float64
	PenaltyCount int
	LastActiveAt valueobjects.Timestamp
}

// GetConsensusStatus retorna o status atual do consenso
func (uc *ConsensusManagerUseCase) GetConsensusStatus(ctx context.Context) (*GetConsensusStatusResponse, error) {
	// Obter status do consenso
	consensusStatus, err := uc.poaEngine.GetConsensusStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get consensus status: %w", err)
	}

	// Obter informações dos validadores
	validators, err := uc.validatorManager.GetAllValidators(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get validators: %w", err)
	}

	validatorStats := make([]ValidatorStatusInfo, len(validators))
	for i, validator := range validators {
		stats, err := uc.validatorManager.GetValidatorStats(ctx, validator.NodeID)
		if err != nil {
			continue // Pular validador com erro
		}

		validatorStats[i] = ValidatorStatusInfo{
			NodeID:       stats.NodeID,
			Status:       stats.Status,
			TotalRounds:  stats.TotalRounds,
			MissedRounds: stats.MissedRounds,
			SuccessRate:  stats.SuccessRate,
			PenaltyCount: stats.PenaltyCount,
			LastActiveAt: stats.LastActiveAt,
		}
	}

	return &GetConsensusStatusResponse{
		IsRunning:        consensusStatus.IsRunning,
		CurrentValidator: consensusStatus.CurrentValidator,
		CurrentRound:     consensusStatus.CurrentRound,
		ValidatorCount:   consensusStatus.ValidatorCount,
		PendingTxCount:   uc.poaEngine.GetPendingTransactionCount(),
		LastBlockTime:    consensusStatus.LastBlockTime,
		ValidatorStats:   validatorStats,
	}, nil
}

// ApplyPenaltyRequest representa a requisição para aplicar penalidade
type ApplyPenaltyRequest struct {
	ValidatorID valueobjects.NodeID
	PenaltyType consensus.PenaltyType
	Reason      string
	Evidence    map[string]interface{}
}

// ApplyPenaltyResponse representa a resposta da aplicação de penalidade
type ApplyPenaltyResponse struct {
	Success bool
	Message string
}

// ApplyPenalty aplica uma penalidade a um validador
func (uc *ConsensusManagerUseCase) ApplyPenalty(ctx context.Context, request *ApplyPenaltyRequest) (*ApplyPenaltyResponse, error) {
	if request == nil {
		return &ApplyPenaltyResponse{
			Success: false,
			Message: "request is nil",
		}, fmt.Errorf("request cannot be nil")
	}

	if request.ValidatorID.IsEmpty() {
		return &ApplyPenaltyResponse{
			Success: false,
			Message: "validator ID is empty",
		}, fmt.Errorf("validator ID cannot be empty")
	}

	err := uc.penaltySystem.ApplyPenalty(ctx, request.ValidatorID, request.PenaltyType, request.Reason, request.Evidence)
	if err != nil {
		return &ApplyPenaltyResponse{
			Success: false,
			Message: fmt.Sprintf("failed to apply penalty: %v", err),
		}, fmt.Errorf("failed to apply penalty: %w", err)
	}

	return &ApplyPenaltyResponse{
		Success: true,
		Message: fmt.Sprintf("penalty applied to validator %s", request.ValidatorID.ShortString()),
	}, nil
}

// GetValidatorPenaltiesRequest representa a requisição para obter penalidades
type GetValidatorPenaltiesRequest struct {
	ValidatorID valueobjects.NodeID
	ActiveOnly  bool
}

// GetValidatorPenaltiesResponse representa a resposta das penalidades
type GetValidatorPenaltiesResponse struct {
	ValidatorID valueobjects.NodeID
	Penalties   []PenaltyInfo
	TotalCount  int
	ActiveCount int
}

// PenaltyInfo contém informações de uma penalidade
type PenaltyInfo struct {
	ID        string
	Type      consensus.PenaltyType
	Severity  consensus.PenaltySeverity
	Reason    string
	AppliedAt valueobjects.Timestamp
	ExpiresAt valueobjects.Timestamp
	IsActive  bool
}

// GetValidatorPenalties retorna as penalidades de um validador
func (uc *ConsensusManagerUseCase) GetValidatorPenalties(ctx context.Context, request *GetValidatorPenaltiesRequest) (*GetValidatorPenaltiesResponse, error) {
	if request == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	var penalties []*consensus.PenaltyRecord
	var err error

	if request.ActiveOnly {
		penalties, err = uc.penaltySystem.GetActivePenalties(ctx, request.ValidatorID)
	} else {
		penalties, err = uc.penaltySystem.GetValidatorPenalties(ctx, request.ValidatorID)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get penalties: %w", err)
	}

	penaltyInfos := make([]PenaltyInfo, len(penalties))
	activeCount := 0

	for i, penalty := range penalties {
		penaltyInfos[i] = PenaltyInfo{
			ID:        penalty.ID,
			Type:      penalty.Type,
			Severity:  penalty.Severity,
			Reason:    penalty.Reason,
			AppliedAt: penalty.AppliedAt,
			ExpiresAt: penalty.ExpiresAt,
			IsActive:  penalty.IsActive,
		}

		if penalty.IsActive && valueobjects.Now().Before(penalty.ExpiresAt) {
			activeCount++
		}
	}

	return &GetValidatorPenaltiesResponse{
		ValidatorID: request.ValidatorID,
		Penalties:   penaltyInfos,
		TotalCount:  len(penalties),
		ActiveCount: activeCount,
	}, nil
}

// ConfigureConsensusRequest representa a requisição para configurar consenso
type ConfigureConsensusRequest struct {
	BlockInterval   time.Duration
	MinTxPerBlock   int
	MaxTxPerBlock   int
	RoundDuration   time.Duration
	TimeoutDuration time.Duration
}

// ConfigureConsensusResponse representa a resposta da configuração
type ConfigureConsensusResponse struct {
	Success bool
	Message string
}

// ConfigureConsensus configura parâmetros do consenso
func (uc *ConsensusManagerUseCase) ConfigureConsensus(ctx context.Context, request *ConfigureConsensusRequest) (*ConfigureConsensusResponse, error) {
	if request == nil {
		return &ConfigureConsensusResponse{
			Success: false,
			Message: "request is nil",
		}, fmt.Errorf("request cannot be nil")
	}

	// Configurar motor PoA
	if request.BlockInterval > 0 || request.MinTxPerBlock > 0 || request.MaxTxPerBlock > 0 {
		uc.poaEngine.SetConfiguration(request.BlockInterval, request.MinTxPerBlock, request.MaxTxPerBlock)
	}

	return &ConfigureConsensusResponse{
		Success: true,
		Message: "consensus configured successfully",
	}, nil
}

// StopConsensus para o processo de consenso
func (uc *ConsensusManagerUseCase) StopConsensus(ctx context.Context) error {
	return uc.poaEngine.StopConsensus(ctx)
}
