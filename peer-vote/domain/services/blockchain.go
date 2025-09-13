package services

import (
	"context"

	"github.com/matscats/peer-vote/peer-vote/domain/entities"
	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
)

// BlockchainService define as operações de gerenciamento da blockchain
type BlockchainService interface {
	// CreateGenesisBlock cria o bloco gênesis
	CreateGenesisBlock(ctx context.Context, transactions []*entities.Transaction, validatorID valueobjects.NodeID, privateKey *PrivateKey) error
	
	// AddBlock adiciona um novo bloco à cadeia
	AddBlock(ctx context.Context, block *entities.Block) error
	
	// GetBlockByIndex recupera um bloco pelo índice
	GetBlockByIndex(ctx context.Context, index uint64) (*entities.Block, error)
	
	// GetChainHeight retorna a altura atual da cadeia
	GetChainHeight(ctx context.Context) (uint64, error)
	
	// CalculateBlockHash calcula o hash de um bloco
	CalculateBlockHash(ctx context.Context, block *entities.Block) valueobjects.Hash
	
	// GetElectionFromBlockchain recupera uma eleição da blockchain
	GetElectionFromBlockchain(ctx context.Context, electionID valueobjects.Hash) (*entities.Election, error)
	
	// ValidateChain valida a integridade da cadeia
	ValidateChain(ctx context.Context) error
	
	// GetLatestBlock retorna o último bloco da cadeia
	GetLatestBlock(ctx context.Context) (*entities.Block, error)
}
