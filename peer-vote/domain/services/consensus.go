package services

import (
	"context"

	"github.com/matscats/peer-vote/peer-vote/domain/entities"
	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
)

// ConsensusService define as operações do algoritmo de consenso
type ConsensusService interface {
	// StartConsensus inicia o processo de consenso
	StartConsensus(ctx context.Context) error
	
	// StopConsensus para o processo de consenso
	StopConsensus(ctx context.Context) error
	
	// ProposeBlock propõe um novo bloco para consenso
	ProposeBlock(ctx context.Context, block *entities.Block) error
	
	// ValidateBlock valida um bloco proposto
	ValidateBlock(ctx context.Context, block *entities.Block) error
	
	// GetCurrentValidator retorna o validador atual no round robin
	GetCurrentValidator(ctx context.Context) (valueobjects.NodeID, error)
	
	// GetNextValidator retorna o próximo validador na sequência
	GetNextValidator(ctx context.Context) (valueobjects.NodeID, error)
	
	// IsValidator verifica se um nó é um validador autorizado
	IsValidator(ctx context.Context, nodeID valueobjects.NodeID) (bool, error)
	
	// AddValidator adiciona um novo validador à lista
	AddValidator(ctx context.Context, nodeID valueobjects.NodeID) error
	
	// RemoveValidator remove um validador da lista
	RemoveValidator(ctx context.Context, nodeID valueobjects.NodeID) error
	
	// GetValidators retorna a lista de todos os validadores
	GetValidators(ctx context.Context) ([]valueobjects.NodeID, error)
	
	// GetValidatorCount retorna o número de validadores
	GetValidatorCount(ctx context.Context) (int, error)
	
	// IsMyTurn verifica se é a vez deste nó validar
	IsMyTurn(ctx context.Context) (bool, error)
	
	// GetCurrentRound retorna o round atual
	GetCurrentRound(ctx context.Context) (uint64, error)
	
	// AdvanceRound avança para o próximo round
	AdvanceRound(ctx context.Context) error
	
	// HandleTimeout lida com timeout de validador
	HandleTimeout(ctx context.Context, validator valueobjects.NodeID) error
	
	// GetConsensusStatus retorna o status atual do consenso
	GetConsensusStatus(ctx context.Context) (ConsensusStatus, error)
}

// ConsensusStatus representa o status do consenso
type ConsensusStatus struct {
	IsRunning        bool
	CurrentValidator valueobjects.NodeID
	CurrentRound     uint64
	ValidatorCount   int
	LastBlockTime    valueobjects.Timestamp
}
