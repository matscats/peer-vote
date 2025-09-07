package persistence

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/matscats/peer-vote/peer-vote/domain/entities"
	"github.com/matscats/peer-vote/peer-vote/domain/repositories"
	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
)

// MemoryBlockchainRepository implementa BlockchainRepository em memória
type MemoryBlockchainRepository struct {
	// Armazenamento por hash
	blocksByHash map[string]*entities.Block
	
	// Armazenamento por índice
	blocksByIndex map[uint64]*entities.Block
	
	// Hash do último bloco
	latestBlockHash valueobjects.Hash
	
	// Altura da cadeia
	chainHeight uint64
	
	// Mutex para operações thread-safe
	mu sync.RWMutex
}

// NewMemoryBlockchainRepository cria um novo repositório em memória
func NewMemoryBlockchainRepository() repositories.BlockchainRepository {
	return &MemoryBlockchainRepository{
		blocksByHash:  make(map[string]*entities.Block),
		blocksByIndex: make(map[uint64]*entities.Block),
		chainHeight:   0,
	}
}

// SaveBlock salva um bloco na blockchain
func (r *MemoryBlockchainRepository) SaveBlock(ctx context.Context, block *entities.Block) error {
	if block == nil {
		return errors.New("block is nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Calcular hash do bloco
	blockHash := r.calculateBlockHash(block)
	blockHashStr := blockHash.String()

	// Verificar se o bloco já existe
	if _, exists := r.blocksByHash[blockHashStr]; exists {
		return fmt.Errorf("block with hash %s already exists", blockHashStr)
	}

	// Verificar se já existe um bloco com o mesmo índice
	if _, exists := r.blocksByIndex[block.GetIndex()]; exists {
		return fmt.Errorf("block with index %d already exists", block.GetIndex())
	}

	// Salvar o bloco
	r.blocksByHash[blockHashStr] = block
	r.blocksByIndex[block.GetIndex()] = block

	// Atualizar altura da cadeia e último bloco se necessário
	if block.GetIndex() > r.chainHeight {
		r.chainHeight = block.GetIndex()
		r.latestBlockHash = blockHash
	}

	return nil
}

// GetBlock recupera um bloco pelo seu hash
func (r *MemoryBlockchainRepository) GetBlock(ctx context.Context, hash valueobjects.Hash) (*entities.Block, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	block, exists := r.blocksByHash[hash.String()]
	if !exists {
		return nil, fmt.Errorf("block with hash %s not found", hash.String())
	}

	return block, nil
}

// GetBlockByIndex recupera um bloco pelo seu índice
func (r *MemoryBlockchainRepository) GetBlockByIndex(ctx context.Context, index uint64) (*entities.Block, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	block, exists := r.blocksByIndex[index]
	if !exists {
		return nil, fmt.Errorf("block with index %d not found", index)
	}

	return block, nil
}

// GetLatestBlock recupera o último bloco da cadeia
func (r *MemoryBlockchainRepository) GetLatestBlock(ctx context.Context) (*entities.Block, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.latestBlockHash.IsEmpty() {
		return nil, errors.New("no blocks in chain")
	}

	return r.blocksByHash[r.latestBlockHash.String()], nil
}

// GetBlockHeight retorna a altura atual da blockchain
func (r *MemoryBlockchainRepository) GetBlockHeight(ctx context.Context) (uint64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.chainHeight, nil
}

// GetBlockRange recupera uma faixa de blocos
func (r *MemoryBlockchainRepository) GetBlockRange(ctx context.Context, startIndex, endIndex uint64) ([]*entities.Block, error) {
	if startIndex > endIndex {
		return nil, errors.New("start index cannot be greater than end index")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var blocks []*entities.Block
	
	for i := startIndex; i <= endIndex; i++ {
		if block, exists := r.blocksByIndex[i]; exists {
			blocks = append(blocks, block)
		} else {
			return nil, fmt.Errorf("block with index %d not found", i)
		}
	}

	return blocks, nil
}

// BlockExists verifica se um bloco existe
func (r *MemoryBlockchainRepository) BlockExists(ctx context.Context, hash valueobjects.Hash) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.blocksByHash[hash.String()]
	return exists, nil
}

// GetBlockHash retorna o hash de um bloco pelo índice
func (r *MemoryBlockchainRepository) GetBlockHash(ctx context.Context, index uint64) (valueobjects.Hash, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	block, exists := r.blocksByIndex[index]
	if !exists {
		return valueobjects.EmptyHash(), fmt.Errorf("block with index %d not found", index)
	}

	return r.calculateBlockHash(block), nil
}

// ValidateChain valida a integridade da cadeia
func (r *MemoryBlockchainRepository) ValidateChain(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.blocksByIndex) == 0 {
		return nil // Cadeia vazia é válida
	}

	// Obter todos os índices e ordenar
	var indices []uint64
	for index := range r.blocksByIndex {
		indices = append(indices, index)
	}
	sort.Slice(indices, func(i, j int) bool {
		return indices[i] < indices[j]
	})

	// Verificar sequência contínua
	for i, index := range indices {
		if uint64(i) != index {
			return fmt.Errorf("missing block at index %d", i)
		}
	}

	// Verificar conexões entre blocos
	var previousBlock *entities.Block
	for _, index := range indices {
		block := r.blocksByIndex[index]
		
		if previousBlock != nil {
			expectedPreviousHash := r.calculateBlockHash(previousBlock)
			if !block.GetPreviousHash().Equals(expectedPreviousHash) {
				return fmt.Errorf("block at index %d has invalid previous hash", index)
			}
		} else {
			// Primeiro bloco (gênesis) deve ter hash anterior vazio
			if !block.GetPreviousHash().IsEmpty() {
				return errors.New("genesis block must have empty previous hash")
			}
		}

		previousBlock = block
	}

	return nil
}

// GetChainHead retorna o hash do último bloco
func (r *MemoryBlockchainRepository) GetChainHead(ctx context.Context) (valueobjects.Hash, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.latestBlockHash.IsEmpty() {
		return valueobjects.EmptyHash(), errors.New("no blocks in chain")
	}

	return r.latestBlockHash, nil
}

// GetGenesisBlock retorna o bloco gênesis
func (r *MemoryBlockchainRepository) GetGenesisBlock(ctx context.Context) (*entities.Block, error) {
	return r.GetBlockByIndex(ctx, 0)
}

// DeleteBlock remove um bloco (usado para reorganização)
func (r *MemoryBlockchainRepository) DeleteBlock(ctx context.Context, hash valueobjects.Hash) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	hashStr := hash.String()
	block, exists := r.blocksByHash[hashStr]
	if !exists {
		return fmt.Errorf("block with hash %s not found", hashStr)
	}

	// Remover dos mapas
	delete(r.blocksByHash, hashStr)
	delete(r.blocksByIndex, block.GetIndex())

	// Atualizar altura da cadeia se necessário
	if block.GetIndex() == r.chainHeight {
		r.recalculateChainHeight()
	}

	return nil
}

// GetBlocksAfter retorna todos os blocos após um determinado índice
func (r *MemoryBlockchainRepository) GetBlocksAfter(ctx context.Context, index uint64) ([]*entities.Block, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var blocks []*entities.Block
	
	for i := index + 1; i <= r.chainHeight; i++ {
		if block, exists := r.blocksByIndex[i]; exists {
			blocks = append(blocks, block)
		}
	}

	return blocks, nil
}

// GetBlocksBefore retorna todos os blocos antes de um determinado índice
func (r *MemoryBlockchainRepository) GetBlocksBefore(ctx context.Context, index uint64) ([]*entities.Block, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var blocks []*entities.Block
	
	for i := uint64(0); i < index && i <= r.chainHeight; i++ {
		if block, exists := r.blocksByIndex[i]; exists {
			blocks = append(blocks, block)
		}
	}

	return blocks, nil
}

// calculateBlockHash calcula o hash de um bloco
func (r *MemoryBlockchainRepository) calculateBlockHash(block *entities.Block) valueobjects.Hash {
	// Implementação simplificada - em produção usaria o serviço de criptografia
	// Por enquanto, usar uma combinação de índice e timestamp
	data := fmt.Sprintf("%d-%d-%s", 
		block.GetIndex(), 
		block.GetTimestamp().Unix(), 
		block.GetMerkleRoot().String())
	
	// Simular hash SHA-256
	hash := make([]byte, 32)
	copy(hash, []byte(data))
	
	return valueobjects.NewHash(hash)
}

// recalculateChainHeight recalcula a altura da cadeia
func (r *MemoryBlockchainRepository) recalculateChainHeight() {
	maxIndex := uint64(0)
	var latestHash valueobjects.Hash
	
	for index, block := range r.blocksByIndex {
		if index > maxIndex {
			maxIndex = index
			latestHash = r.calculateBlockHash(block)
		}
	}
	
	r.chainHeight = maxIndex
	r.latestBlockHash = latestHash
}

// GetBlockCount retorna o número total de blocos
func (r *MemoryBlockchainRepository) GetBlockCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	return len(r.blocksByIndex)
}

// Clear limpa todos os blocos (útil para testes)
func (r *MemoryBlockchainRepository) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.blocksByHash = make(map[string]*entities.Block)
	r.blocksByIndex = make(map[uint64]*entities.Block)
	r.latestBlockHash = valueobjects.EmptyHash()
	r.chainHeight = 0
}
