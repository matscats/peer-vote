package repositories

import (
	"context"

	"github.com/matscats/peer-vote/peer-vote/domain/entities"
	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
)

// BlockchainRepository define as operações de persistência da blockchain
type BlockchainRepository interface {
	// SaveBlock salva um bloco na blockchain
	SaveBlock(ctx context.Context, block *entities.Block) error
	
	// GetBlock recupera um bloco pelo seu hash
	GetBlock(ctx context.Context, hash valueobjects.Hash) (*entities.Block, error)
	
	// GetBlockByIndex recupera um bloco pelo seu índice
	GetBlockByIndex(ctx context.Context, index uint64) (*entities.Block, error)
	
	// GetLatestBlock recupera o último bloco da cadeia
	GetLatestBlock(ctx context.Context) (*entities.Block, error)
	
	// GetBlockHeight retorna a altura atual da blockchain
	GetBlockHeight(ctx context.Context) (uint64, error)
	
	// GetBlockRange recupera uma faixa de blocos
	GetBlockRange(ctx context.Context, startIndex, endIndex uint64) ([]*entities.Block, error)
	
	// BlockExists verifica se um bloco existe
	BlockExists(ctx context.Context, hash valueobjects.Hash) (bool, error)
	
	// GetBlockHash retorna o hash de um bloco pelo índice
	GetBlockHash(ctx context.Context, index uint64) (valueobjects.Hash, error)
	
	// ValidateChain valida a integridade da cadeia
	ValidateChain(ctx context.Context) error
	
	// GetChainHead retorna o hash do último bloco
	GetChainHead(ctx context.Context) (valueobjects.Hash, error)
	
	// GetGenesisBlock retorna o bloco gênesis
	GetGenesisBlock(ctx context.Context) (*entities.Block, error)
	
	// DeleteBlock remove um bloco (usado para reorganização)
	DeleteBlock(ctx context.Context, hash valueobjects.Hash) error
	
	// GetBlocksAfter retorna todos os blocos após um determinado índice
	GetBlocksAfter(ctx context.Context, index uint64) ([]*entities.Block, error)
	
	// GetBlocksBefore retorna todos os blocos antes de um determinado índice
	GetBlocksBefore(ctx context.Context, index uint64) ([]*entities.Block, error)
}
