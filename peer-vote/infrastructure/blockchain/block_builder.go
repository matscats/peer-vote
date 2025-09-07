package blockchain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/matscats/peer-vote/peer-vote/domain/entities"
	"github.com/matscats/peer-vote/peer-vote/domain/services"
	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
)

// BlockBuilder constrói blocos com validação e Merkle Tree
type BlockBuilder struct {
	cryptoService services.CryptographyService
	maxTxPerBlock int
	maxBlockSize  int64
}

// NewBlockBuilder cria um novo construtor de blocos
func NewBlockBuilder(cryptoService services.CryptographyService) *BlockBuilder {
	return &BlockBuilder{
		cryptoService: cryptoService,
		maxTxPerBlock: 1000,    // Máximo de transações por bloco
		maxBlockSize:  1048576, // 1MB máximo por bloco
	}
}

// SetMaxTransactionsPerBlock define o máximo de transações por bloco
func (bb *BlockBuilder) SetMaxTransactionsPerBlock(max int) {
	if max > 0 {
		bb.maxTxPerBlock = max
	}
}

// SetMaxBlockSize define o tamanho máximo do bloco em bytes
func (bb *BlockBuilder) SetMaxBlockSize(size int64) {
	if size > 0 {
		bb.maxBlockSize = size
	}
}

// BuildBlock constrói um novo bloco com as transações fornecidas
func (bb *BlockBuilder) BuildBlock(ctx context.Context, index uint64, previousHash valueobjects.Hash, transactions []*entities.Transaction, validator valueobjects.NodeID) (*entities.Block, error) {
	if len(transactions) == 0 {
		return nil, errors.New("cannot build block with no transactions")
	}

	if len(transactions) > bb.maxTxPerBlock {
		return nil, fmt.Errorf("too many transactions: %d, max allowed: %d", len(transactions), bb.maxTxPerBlock)
	}

	// Validar todas as transações
	validTransactions, err := bb.validateTransactions(ctx, transactions)
	if err != nil {
		return nil, fmt.Errorf("transaction validation failed: %w", err)
	}

	if len(validTransactions) == 0 {
		return nil, errors.New("no valid transactions to include in block")
	}

	// Criar bloco básico
	block := entities.NewBlock(index, previousHash, validTransactions, validator)

	// Calcular Merkle Root
	merkleRoot, err := bb.calculateMerkleRoot(ctx, validTransactions)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate merkle root: %w", err)
	}

	block.SetMerkleRoot(merkleRoot)

	// Verificar tamanho do bloco
	blockSize, err := bb.calculateBlockSize(ctx, block)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate block size: %w", err)
	}

	if blockSize > bb.maxBlockSize {
		return nil, fmt.Errorf("block size %d exceeds maximum %d", blockSize, bb.maxBlockSize)
	}

	return block, nil
}

// SignBlock assina um bloco com a chave privada do validador
func (bb *BlockBuilder) SignBlock(ctx context.Context, block *entities.Block, privateKey *services.PrivateKey) error {
	if block == nil {
		return errors.New("block is nil")
	}

	if privateKey == nil || !privateKey.IsValid() {
		return errors.New("invalid private key")
	}

	// Serializar dados do bloco para assinatura
	blockData, err := bb.serializeBlockForSigning(ctx, block)
	if err != nil {
		return fmt.Errorf("failed to serialize block for signing: %w", err)
	}

	// Assinar os dados
	signature, err := bb.cryptoService.Sign(ctx, blockData, privateKey)
	if err != nil {
		return fmt.Errorf("failed to sign block: %w", err)
	}

	// Definir assinatura no bloco
	block.SetSignature(signature)

	return nil
}

// ValidateBlock valida um bloco completo
func (bb *BlockBuilder) ValidateBlock(ctx context.Context, block *entities.Block) error {
	if block == nil {
		return errors.New("block is nil")
	}

	// Validação básica da entidade
	if !block.IsValid() {
		return errors.New("block failed basic validation")
	}

	// Validar transações
	transactions := block.GetTransactions()
	_, err := bb.validateTransactions(ctx, transactions)
	if err != nil {
		return fmt.Errorf("block contains invalid transactions: %w", err)
	}

	// Validar Merkle Root
	expectedMerkleRoot, err := bb.calculateMerkleRoot(ctx, transactions)
	if err != nil {
		return fmt.Errorf("failed to calculate expected merkle root: %w", err)
	}

	if !block.GetMerkleRoot().Equals(expectedMerkleRoot) {
		return errors.New("merkle root mismatch")
	}

	// Validar timestamp (não deve ser muito no futuro)
	now := valueobjects.Now()
	blockTime := block.GetTimestamp()
	
	if blockTime.After(now.Add(time.Minute * 5)) {
		return errors.New("block timestamp too far in the future")
	}

	// Validar tamanho do bloco
	blockSize, err := bb.calculateBlockSize(ctx, block)
	if err != nil {
		return fmt.Errorf("failed to calculate block size: %w", err)
	}

	if blockSize > bb.maxBlockSize {
		return fmt.Errorf("block size %d exceeds maximum %d", blockSize, bb.maxBlockSize)
	}

	return nil
}

// ValidateBlockSignature valida a assinatura de um bloco
func (bb *BlockBuilder) ValidateBlockSignature(ctx context.Context, block *entities.Block, publicKey *services.PublicKey) error {
	if block == nil {
		return errors.New("block is nil")
	}

	if publicKey == nil || !publicKey.IsValid() {
		return errors.New("invalid public key")
	}

	signature := block.GetSignature()
	if signature.IsEmpty() {
		return errors.New("block has no signature")
	}

	// Serializar dados do bloco (sem a assinatura)
	blockData, err := bb.serializeBlockForSigning(ctx, block)
	if err != nil {
		return fmt.Errorf("failed to serialize block for signature validation: %w", err)
	}

	// Verificar assinatura
	valid, err := bb.cryptoService.Verify(ctx, blockData, signature, publicKey)
	if err != nil {
		return fmt.Errorf("failed to verify block signature: %w", err)
	}

	if !valid {
		return errors.New("invalid block signature")
	}

	return nil
}

// validateTransactions valida uma lista de transações
func (bb *BlockBuilder) validateTransactions(ctx context.Context, transactions []*entities.Transaction) ([]*entities.Transaction, error) {
	var validTransactions []*entities.Transaction
	seenHashes := make(map[string]bool)

	for _, tx := range transactions {
		if tx == nil {
			continue // Pular transações nulas
		}

		// Validação básica da transação
		if !tx.IsValid() {
			return nil, fmt.Errorf("invalid transaction: %s", tx.GetID().String())
		}

		// Verificar duplicatas
		txHash := tx.GetHash().String()
		if seenHashes[txHash] {
			return nil, fmt.Errorf("duplicate transaction: %s", txHash)
		}
		seenHashes[txHash] = true

		// Calcular e verificar hash da transação
		txData := tx.ToBytes()
		expectedHash := bb.cryptoService.HashTransaction(ctx, txData)
		
		if !tx.GetHash().Equals(expectedHash) {
			return nil, fmt.Errorf("transaction hash mismatch: %s", tx.GetID().String())
		}

		validTransactions = append(validTransactions, tx)
	}

	return validTransactions, nil
}

// calculateMerkleRoot calcula a raiz da Merkle Tree para as transações
func (bb *BlockBuilder) calculateMerkleRoot(ctx context.Context, transactions []*entities.Transaction) (valueobjects.Hash, error) {
	if len(transactions) == 0 {
		return valueobjects.EmptyHash(), errors.New("no transactions to calculate merkle root")
	}

	// Preparar dados das transações
	txData := make([][]byte, len(transactions))
	for i, tx := range transactions {
		txData[i] = tx.ToBytes()
	}

	// Criar Merkle Tree
	merkleTree, err := NewMerkleTree(txData)
	if err != nil {
		return valueobjects.EmptyHash(), fmt.Errorf("failed to create merkle tree: %w", err)
	}

	return merkleTree.GetRoot(), nil
}

// calculateBlockSize calcula o tamanho do bloco em bytes
func (bb *BlockBuilder) calculateBlockSize(ctx context.Context, block *entities.Block) (int64, error) {
	// Serializar o bloco completo
	blockData, err := bb.serializeBlock(ctx, block)
	if err != nil {
		return 0, fmt.Errorf("failed to serialize block: %w", err)
	}

	return int64(len(blockData)), nil
}

// serializeBlockForSigning serializa um bloco para assinatura (sem incluir a assinatura)
func (bb *BlockBuilder) serializeBlockForSigning(ctx context.Context, block *entities.Block) ([]byte, error) {
	blockData := struct {
		Index        uint64                    `json:"index"`
		PreviousHash string                    `json:"previous_hash"`
		Timestamp    int64                     `json:"timestamp"`
		MerkleRoot   string                    `json:"merkle_root"`
		Validator    string                    `json:"validator"`
		Nonce        uint64                    `json:"nonce"`
		Transactions []map[string]interface{}  `json:"transactions"`
	}{
		Index:        block.GetIndex(),
		PreviousHash: block.GetPreviousHash().String(),
		Timestamp:    block.GetTimestamp().Unix(),
		MerkleRoot:   block.GetMerkleRoot().String(),
		Validator:    block.GetValidator().String(),
		Nonce:        block.GetNonce(),
	}

	// Serializar transações
	transactions := block.GetTransactions()
	blockData.Transactions = make([]map[string]interface{}, len(transactions))
	
	for i, tx := range transactions {
		blockData.Transactions[i] = map[string]interface{}{
			"id":        tx.GetID().String(),
			"type":      string(tx.GetType()),
			"from":      tx.GetFrom().String(),
			"to":        tx.GetTo().String(),
			"timestamp": tx.GetTimestamp().Unix(),
			"hash":      tx.GetHash().String(),
		}
	}

	return json.Marshal(blockData)
}

// serializeBlock serializa um bloco completo (incluindo assinatura)
func (bb *BlockBuilder) serializeBlock(ctx context.Context, block *entities.Block) ([]byte, error) {
	// Primeiro, obter dados para assinatura
	blockData, err := bb.serializeBlockForSigning(ctx, block)
	if err != nil {
		return nil, err
	}

	// Deserializar para adicionar assinatura
	var blockMap map[string]interface{}
	if err := json.Unmarshal(blockData, &blockMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal block data: %w", err)
	}

	// Adicionar assinatura
	blockMap["signature"] = block.GetSignature().String()

	return json.Marshal(blockMap)
}

// CreateGenesisBlock cria o bloco gênesis
func (bb *BlockBuilder) CreateGenesisBlock(ctx context.Context, genesisTransactions []*entities.Transaction, validator valueobjects.NodeID) (*entities.Block, error) {
	// Bloco gênesis tem índice 0 e hash anterior vazio
	genesisBlock, err := bb.BuildBlock(ctx, 0, valueobjects.EmptyHash(), genesisTransactions, validator)
	if err != nil {
		return nil, fmt.Errorf("failed to build genesis block: %w", err)
	}

	return genesisBlock, nil
}
