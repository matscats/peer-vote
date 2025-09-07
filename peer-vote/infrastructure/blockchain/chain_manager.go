package blockchain

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/matscats/peer-vote/peer-vote/domain/entities"
	"github.com/matscats/peer-vote/peer-vote/domain/repositories"
	"github.com/matscats/peer-vote/peer-vote/domain/services"
	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
)

// ChainManager gerencia a cadeia de blocos
type ChainManager struct {
	repository    repositories.BlockchainRepository
	blockBuilder  *BlockBuilder
	cryptoService services.CryptographyService
	
	// Cache para otimização
	latestBlock   *entities.Block
	chainHeight   uint64
	
	// Mutex para operações thread-safe
	mu sync.RWMutex
	
	// Configurações
	maxReorgDepth int // Profundidade máxima para reorganização
}

// NewChainManager cria um novo gerenciador de cadeia
func NewChainManager(repository repositories.BlockchainRepository, cryptoService services.CryptographyService) *ChainManager {
	blockBuilder := NewBlockBuilder(cryptoService)
	
	return &ChainManager{
		repository:    repository,
		blockBuilder:  blockBuilder,
		cryptoService: cryptoService,
		maxReorgDepth: 100, // Máximo de 100 blocos para reorganização
	}
}

// Initialize inicializa o gerenciador de cadeia
func (cm *ChainManager) Initialize(ctx context.Context) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Tentar carregar o último bloco
	latestBlock, err := cm.repository.GetLatestBlock(ctx)
	if err != nil {
		// Se não há blocos, a cadeia está vazia (será criado o gênesis depois)
		cm.latestBlock = nil
		cm.chainHeight = 0
		return nil
	}

	cm.latestBlock = latestBlock
	cm.chainHeight = latestBlock.GetIndex()

	// Validar integridade da cadeia
	return cm.validateChainIntegrity(ctx)
}

// AddBlock adiciona um novo bloco à cadeia
func (cm *ChainManager) AddBlock(ctx context.Context, block *entities.Block) error {
	if block == nil {
		return errors.New("block is nil")
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Validar o bloco
	if err := cm.blockBuilder.ValidateBlock(ctx, block); err != nil {
		return fmt.Errorf("block validation failed: %w", err)
	}

	// Verificar se o bloco se conecta à cadeia atual
	if err := cm.validateBlockConnection(ctx, block); err != nil {
		return fmt.Errorf("block connection validation failed: %w", err)
	}

	// Salvar o bloco
	if err := cm.repository.SaveBlock(ctx, block); err != nil {
		return fmt.Errorf("failed to save block: %w", err)
	}

	// Atualizar cache
	cm.latestBlock = block
	cm.chainHeight = block.GetIndex()

	return nil
}

// GetLatestBlock retorna o último bloco da cadeia
func (cm *ChainManager) GetLatestBlock(ctx context.Context) (*entities.Block, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.latestBlock != nil {
		return cm.latestBlock, nil
	}

	return cm.repository.GetLatestBlock(ctx)
}

// GetBlock retorna um bloco pelo hash
func (cm *ChainManager) GetBlock(ctx context.Context, hash valueobjects.Hash) (*entities.Block, error) {
	return cm.repository.GetBlock(ctx, hash)
}

// GetBlockByIndex retorna um bloco pelo índice
func (cm *ChainManager) GetBlockByIndex(ctx context.Context, index uint64) (*entities.Block, error) {
	return cm.repository.GetBlockByIndex(ctx, index)
}

// GetChainHeight retorna a altura atual da cadeia
func (cm *ChainManager) GetChainHeight(ctx context.Context) (uint64, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.latestBlock != nil {
		return cm.chainHeight, nil
	}

	return cm.repository.GetBlockHeight(ctx)
}

// ValidateChain valida toda a cadeia de blocos
func (cm *ChainManager) ValidateChain(ctx context.Context) error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.validateChainIntegrity(ctx)
}

// CreateGenesisBlock cria e adiciona o bloco gênesis
func (cm *ChainManager) CreateGenesisBlock(ctx context.Context, genesisTransactions []*entities.Transaction, validator valueobjects.NodeID, privateKey *services.PrivateKey) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Verificar se já existe um bloco gênesis
	exists, err := cm.repository.BlockExists(ctx, valueobjects.EmptyHash())
	if err != nil {
		return fmt.Errorf("failed to check genesis block existence: %w", err)
	}

	if exists {
		return errors.New("genesis block already exists")
	}

	// Criar bloco gênesis
	genesisBlock, err := cm.blockBuilder.CreateGenesisBlock(ctx, genesisTransactions, validator)
	if err != nil {
		return fmt.Errorf("failed to create genesis block: %w", err)
	}

	// Assinar o bloco gênesis
	if err := cm.blockBuilder.SignBlock(ctx, genesisBlock, privateKey); err != nil {
		return fmt.Errorf("failed to sign genesis block: %w", err)
	}

	// Salvar o bloco gênesis
	if err := cm.repository.SaveBlock(ctx, genesisBlock); err != nil {
		return fmt.Errorf("failed to save genesis block: %w", err)
	}

	// Atualizar cache
	cm.latestBlock = genesisBlock
	cm.chainHeight = 0

	return nil
}

// ProposeBlock propõe um novo bloco com as transações fornecidas
func (cm *ChainManager) ProposeBlock(ctx context.Context, transactions []*entities.Transaction, validator valueobjects.NodeID, privateKey *services.PrivateKey) (*entities.Block, error) {
	cm.mu.RLock()
	latestBlock := cm.latestBlock
	cm.mu.RUnlock()

	if latestBlock == nil {
		return nil, errors.New("no latest block found, genesis block may be missing")
	}

	// Calcular hash do bloco anterior
	previousHash := cm.calculateBlockHash(ctx, latestBlock)
	nextIndex := latestBlock.GetIndex() + 1

	// Construir o bloco
	block, err := cm.blockBuilder.BuildBlock(ctx, nextIndex, previousHash, transactions, validator)
	if err != nil {
		return nil, fmt.Errorf("failed to build block: %w", err)
	}

	// Assinar o bloco
	if err := cm.blockBuilder.SignBlock(ctx, block, privateKey); err != nil {
		return nil, fmt.Errorf("failed to sign block: %w", err)
	}

	return block, nil
}

// HandleFork lida com situações de fork na cadeia
func (cm *ChainManager) HandleFork(ctx context.Context, alternativeBlock *entities.Block) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if alternativeBlock == nil {
		return errors.New("alternative block is nil")
	}

	// Validar o bloco alternativo
	if err := cm.blockBuilder.ValidateBlock(ctx, alternativeBlock); err != nil {
		return fmt.Errorf("alternative block validation failed: %w", err)
	}

	// Determinar se devemos reorganizar a cadeia
	shouldReorg, err := cm.shouldReorganize(ctx, alternativeBlock)
	if err != nil {
		return fmt.Errorf("failed to determine reorganization: %w", err)
	}

	if shouldReorg {
		return cm.reorganizeChain(ctx, alternativeBlock)
	}

	return nil
}

// validateBlockConnection verifica se um bloco se conecta corretamente à cadeia
func (cm *ChainManager) validateBlockConnection(ctx context.Context, block *entities.Block) error {
	if cm.latestBlock == nil {
		// Se não há bloco anterior, este deve ser o gênesis
		if block.GetIndex() != 0 {
			return errors.New("first block must be genesis block with index 0")
		}
		if !block.GetPreviousHash().IsEmpty() {
			return errors.New("genesis block must have empty previous hash")
		}
		return nil
	}

	// Verificar índice sequencial
	expectedIndex := cm.latestBlock.GetIndex() + 1
	if block.GetIndex() != expectedIndex {
		return fmt.Errorf("block index %d is not sequential, expected %d", block.GetIndex(), expectedIndex)
	}

	// Verificar hash do bloco anterior
	expectedPreviousHash := cm.calculateBlockHash(ctx, cm.latestBlock)
	if !block.GetPreviousHash().Equals(expectedPreviousHash) {
		return errors.New("block previous hash does not match latest block hash")
	}

	return nil
}

// validateChainIntegrity valida a integridade de toda a cadeia
func (cm *ChainManager) validateChainIntegrity(ctx context.Context) error {
	height, err := cm.repository.GetBlockHeight(ctx)
	if err != nil {
		return fmt.Errorf("failed to get chain height: %w", err)
	}

	if height == 0 {
		return nil // Cadeia vazia é válida
	}

	// Validar do gênesis até o último bloco
	var previousBlock *entities.Block
	
	for i := uint64(0); i <= height; i++ {
		block, err := cm.repository.GetBlockByIndex(ctx, i)
		if err != nil {
			return fmt.Errorf("failed to get block at index %d: %w", i, err)
		}

		// Validar o bloco
		if err := cm.blockBuilder.ValidateBlock(ctx, block); err != nil {
			return fmt.Errorf("block at index %d is invalid: %w", i, err)
		}

		// Validar conexão com o bloco anterior
		if previousBlock != nil {
			expectedPreviousHash := cm.calculateBlockHash(ctx, previousBlock)
			if !block.GetPreviousHash().Equals(expectedPreviousHash) {
				return fmt.Errorf("block at index %d has invalid previous hash", i)
			}
		}

		previousBlock = block
	}

	return nil
}

// CalculateBlockHash calcula o hash de um bloco (método público)
func (cm *ChainManager) CalculateBlockHash(ctx context.Context, block *entities.Block) valueobjects.Hash {
	return cm.calculateBlockHash(ctx, block)
}

// calculateBlockHash calcula o hash de um bloco
func (cm *ChainManager) calculateBlockHash(ctx context.Context, block *entities.Block) valueobjects.Hash {
	// Serializar dados do bloco
	blockData, err := cm.serializeBlockForHashing(ctx, block)
	if err != nil {
		// Em caso de erro, retornar hash vazio
		return valueobjects.EmptyHash()
	}

	return cm.cryptoService.HashBlock(ctx, blockData)
}

// serializeBlockForHashing serializa um bloco para cálculo de hash
func (cm *ChainManager) serializeBlockForHashing(ctx context.Context, block *entities.Block) ([]byte, error) {
	// Usar o mesmo método do block builder, mas incluindo a assinatura
	return cm.blockBuilder.serializeBlock(ctx, block)
}

// shouldReorganize determina se a cadeia deve ser reorganizada
func (cm *ChainManager) shouldReorganize(ctx context.Context, alternativeBlock *entities.Block) (bool, error) {
	if alternativeBlock == nil {
		return false, errors.New("alternative block is nil")
	}
	
	// Verificar se o bloco alternativo é válido
	if err := cm.blockBuilder.ValidateBlock(ctx, alternativeBlock); err != nil {
		return false, fmt.Errorf("alternative block is invalid: %w", err)
	}
	
	// Verificar se temos um bloco atual para comparar
	if cm.latestBlock == nil {
		return true, nil // Se não temos bloco, aceitar qualquer bloco válido
	}
	
	// Verificar se o bloco alternativo tem o mesmo índice
	if alternativeBlock.GetIndex() != cm.latestBlock.GetIndex() {
		return false, nil // Índices diferentes, não reorganizar
	}
	
	// Verificar se o bloco alternativo tem o mesmo hash anterior
	if !alternativeBlock.GetPreviousHash().Equals(cm.latestBlock.GetPreviousHash()) {
		return false, nil // Hash anterior diferente, não reorganizar
	}
	
	// Critério de reorganização: bloco mais antigo (produzido primeiro)
	// Em PoA, o primeiro bloco válido produzido deve ser aceito
	if alternativeBlock.GetTimestamp().Before(cm.latestBlock.GetTimestamp()) {
		return true, nil
	}
	
	// Se os timestamps são iguais, usar o hash como critério de desempate
	if alternativeBlock.GetTimestamp().Equal(cm.latestBlock.GetTimestamp()) {
		altHash := cm.calculateBlockHash(ctx, alternativeBlock)
		currentHash := cm.calculateBlockHash(ctx, cm.latestBlock)
		
		// Usar comparação lexicográfica dos hashes
		return altHash.String() < currentHash.String(), nil
	}
	
	return false, nil
}

// reorganizeChain reorganiza a cadeia para uma alternativa
func (cm *ChainManager) reorganizeChain(ctx context.Context, alternativeBlock *entities.Block) error {
	if alternativeBlock == nil {
		return errors.New("alternative block is nil")
	}
	
	// Implementação básica de reorganização
	// Esta é uma versão simplificada que apenas substitui o último bloco
	
	// Verificar se o bloco alternativo é válido
	if err := cm.blockBuilder.ValidateBlock(ctx, alternativeBlock); err != nil {
		return fmt.Errorf("alternative block is invalid: %w", err)
	}
	
	// Verificar se o bloco alternativo tem o mesmo índice que o atual
	currentHeight := cm.chainHeight
	if alternativeBlock.GetIndex() != currentHeight {
		return fmt.Errorf("alternative block index %d does not match current height %d", 
			alternativeBlock.GetIndex(), currentHeight)
	}
	
	// Obter o bloco atual
	currentBlock, err := cm.repository.GetBlockByIndex(ctx, currentHeight)
	if err != nil {
		return fmt.Errorf("failed to get current block: %w", err)
	}
	
	// Verificar se o bloco alternativo tem o mesmo hash anterior
	if !alternativeBlock.GetPreviousHash().Equals(currentBlock.GetPreviousHash()) {
		return fmt.Errorf("alternative block has different previous hash")
	}
	
	// Comparar "peso" dos blocos (por simplicidade, usar timestamp)
	// Em uma implementação real, seria baseado em dificuldade acumulada
	if alternativeBlock.GetTimestamp().Before(currentBlock.GetTimestamp()) {
		// Bloco alternativo é "melhor" (mais antigo = produzido primeiro)
		
		// Remover o bloco atual
		currentBlockHash := cm.calculateBlockHash(ctx, currentBlock)
		if err := cm.repository.DeleteBlock(ctx, currentBlockHash); err != nil {
			return fmt.Errorf("failed to remove current block: %w", err)
		}
		
		// Adicionar o bloco alternativo
		if err := cm.repository.SaveBlock(ctx, alternativeBlock); err != nil {
			// Tentar restaurar o bloco original em caso de erro
			cm.repository.SaveBlock(ctx, currentBlock)
			return fmt.Errorf("failed to save alternative block: %w", err)
		}
		
		// Atualizar cache
		cm.latestBlock = alternativeBlock
		
		return nil
	}
	
	// Bloco atual é melhor, não reorganizar
	return fmt.Errorf("current block is better than alternative, no reorganization needed")
}

// GetBlockRange retorna uma faixa de blocos
func (cm *ChainManager) GetBlockRange(ctx context.Context, startIndex, endIndex uint64) ([]*entities.Block, error) {
	return cm.repository.GetBlockRange(ctx, startIndex, endIndex)
}

// BlockExists verifica se um bloco existe
func (cm *ChainManager) BlockExists(ctx context.Context, hash valueobjects.Hash) (bool, error) {
	return cm.repository.BlockExists(ctx, hash)
}

// GetTransactionFromBlock recupera uma transação específica de um bloco
func (cm *ChainManager) GetTransactionFromBlock(ctx context.Context, blockHash valueobjects.Hash, txHash valueobjects.Hash) (*entities.Transaction, error) {
	block, err := cm.repository.GetBlock(ctx, blockHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get block: %w", err)
	}

	transactions := block.GetTransactions()
	for _, tx := range transactions {
		if tx.GetHash().Equals(txHash) {
			return tx, nil
		}
	}

	return nil, errors.New("transaction not found in block")
}

// VerifyTransactionInclusion verifica se uma transação está incluída em um bloco usando Merkle Proof
func (cm *ChainManager) VerifyTransactionInclusion(ctx context.Context, blockHash valueobjects.Hash, txData []byte) (bool, error) {
	block, err := cm.repository.GetBlock(ctx, blockHash)
	if err != nil {
		return false, fmt.Errorf("failed to get block: %w", err)
	}

	// Preparar dados das transações para Merkle Tree
	transactions := block.GetTransactions()
	txDataList := make([][]byte, len(transactions))
	
	for i, tx := range transactions {
		txDataList[i] = tx.ToBytes()
	}

	// Criar Merkle Tree
	merkleTree, err := NewMerkleTree(txDataList)
	if err != nil {
		return false, fmt.Errorf("failed to create merkle tree: %w", err)
	}

	// Gerar prova de inclusão
	proof, err := merkleTree.GenerateProof(txData)
	if err != nil {
		return false, fmt.Errorf("failed to generate merkle proof: %w", err)
	}

	// Verificar prova contra o Merkle Root do bloco
	return VerifyProof(proof, block.GetMerkleRoot()), nil
}
