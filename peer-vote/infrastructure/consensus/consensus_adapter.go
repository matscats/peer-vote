package consensus

import (
	"context"

	"github.com/matscats/peer-vote/peer-vote/domain/entities"
	"github.com/matscats/peer-vote/peer-vote/domain/services"
	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
)

// ConsensusAdapter adapta PoAEngine para implementar services.ConsensusService
type ConsensusAdapter struct {
	poaEngine *PoAEngine
}

// NewConsensusAdapter cria um novo adapter para consenso
func NewConsensusAdapter(poaEngine *PoAEngine) services.ConsensusService {
	return &ConsensusAdapter{
		poaEngine: poaEngine,
	}
}

func (ca *ConsensusAdapter) StartConsensus(ctx context.Context) error {
	return ca.poaEngine.StartConsensus(ctx)
}

func (ca *ConsensusAdapter) StopConsensus(ctx context.Context) error {
	return ca.poaEngine.StopConsensus(ctx)
}

func (ca *ConsensusAdapter) AddTransaction(ctx context.Context, tx *entities.Transaction) error {
	return ca.poaEngine.AddTransaction(ctx, tx)
}

func (ca *ConsensusAdapter) ProposeBlock(ctx context.Context, block *entities.Block) error {
	return ca.poaEngine.ProposeBlock(ctx, block)
}

func (ca *ConsensusAdapter) ValidateBlock(ctx context.Context, block *entities.Block) error {
	return ca.poaEngine.ValidateBlock(ctx, block)
}

func (ca *ConsensusAdapter) GetCurrentValidator(ctx context.Context) (valueobjects.NodeID, error) {
	return ca.poaEngine.GetCurrentValidator(ctx)
}

func (ca *ConsensusAdapter) GetNextValidator(ctx context.Context) (valueobjects.NodeID, error) {
	return ca.poaEngine.GetNextValidator(ctx)
}

func (ca *ConsensusAdapter) IsValidator(ctx context.Context, nodeID valueobjects.NodeID) (bool, error) {
	return ca.poaEngine.IsValidator(ctx, nodeID)
}

// AddValidator adapta o método para a interface
func (ca *ConsensusAdapter) AddValidator(ctx context.Context, nodeID valueobjects.NodeID) error {
	// Nota: Esta implementação simplificada não tem a chave pública
	// Em um cenário real, seria necessário obter a chave pública do nodeID
	return ca.poaEngine.validatorManager.AddValidator(ctx, nodeID, nil)
}

func (ca *ConsensusAdapter) RemoveValidator(ctx context.Context, nodeID valueobjects.NodeID) error {
	return ca.poaEngine.validatorManager.RemoveValidator(ctx, nodeID)
}

func (ca *ConsensusAdapter) GetValidators(ctx context.Context) ([]valueobjects.NodeID, error) {
	return ca.poaEngine.GetValidators(ctx)
}

func (ca *ConsensusAdapter) GetValidatorCount(ctx context.Context) (int, error) {
	validators, err := ca.GetValidators(ctx)
	if err != nil {
		return 0, err
	}
	return len(validators), nil
}

func (ca *ConsensusAdapter) IsMyTurn(ctx context.Context) (bool, error) {
	return ca.poaEngine.IsMyTurn(ctx)
}

func (ca *ConsensusAdapter) GetCurrentRound(ctx context.Context) (uint64, error) {
	return ca.poaEngine.GetCurrentRound(ctx)
}

func (ca *ConsensusAdapter) AdvanceRound(ctx context.Context) error {
	return ca.poaEngine.AdvanceRound(ctx)
}

func (ca *ConsensusAdapter) HandleTimeout(ctx context.Context, validator valueobjects.NodeID) error {
	return ca.poaEngine.HandleTimeout(ctx, validator)
}

func (ca *ConsensusAdapter) GetConsensusStatus(ctx context.Context) (services.ConsensusStatus, error) {
	status, err := ca.poaEngine.GetConsensusStatus(ctx)
	if err != nil {
		return services.ConsensusStatus{}, err
	}
	
	return services.ConsensusStatus{
		IsRunning:        status.IsRunning,
		CurrentValidator: status.CurrentValidator,
		CurrentRound:     status.CurrentRound,
		ValidatorCount:   status.ValidatorCount,
		LastBlockTime:    status.LastBlockTime,
	}, nil
}
