package blockchain

import (
	"context"

	"github.com/matscats/peer-vote/peer-vote/domain/entities"
	"github.com/matscats/peer-vote/peer-vote/domain/services"
	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
)

// BlockchainAdapter adapta ChainManager para implementar services.BlockchainService
type BlockchainAdapter struct {
	chainManager *ChainManager
}

// NewBlockchainAdapter cria um novo adapter para blockchain
func NewBlockchainAdapter(chainManager *ChainManager) services.BlockchainService {
	return &BlockchainAdapter{
		chainManager: chainManager,
	}
}

func (ba *BlockchainAdapter) CreateGenesisBlock(ctx context.Context, transactions []*entities.Transaction, validatorID valueobjects.NodeID, privateKey *services.PrivateKey) error {
	return ba.chainManager.CreateGenesisBlock(ctx, transactions, validatorID, privateKey)
}

func (ba *BlockchainAdapter) AddBlock(ctx context.Context, block *entities.Block) error {
	return ba.chainManager.AddBlock(ctx, block)
}

func (ba *BlockchainAdapter) GetBlockByIndex(ctx context.Context, index uint64) (*entities.Block, error) {
	return ba.chainManager.GetBlockByIndex(ctx, index)
}

func (ba *BlockchainAdapter) GetChainHeight(ctx context.Context) (uint64, error) {
	return ba.chainManager.GetChainHeight(ctx)
}

func (ba *BlockchainAdapter) CalculateBlockHash(ctx context.Context, block *entities.Block) valueobjects.Hash {
	return ba.chainManager.CalculateBlockHash(ctx, block)
}

func (ba *BlockchainAdapter) GetElectionFromBlockchain(ctx context.Context, electionID valueobjects.Hash) (*entities.Election, error) {
	return ba.chainManager.GetElectionFromBlockchain(ctx, electionID)
}

func (ba *BlockchainAdapter) ValidateChain(ctx context.Context) error {
	return ba.chainManager.ValidateChain(ctx)
}

func (ba *BlockchainAdapter) GetLatestBlock(ctx context.Context) (*entities.Block, error) {
	return ba.chainManager.GetLatestBlock(ctx)
}
