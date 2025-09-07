package usecases

import (
	"context"
	"fmt"

	"github.com/matscats/peer-vote/peer-vote/domain/entities"
	"github.com/matscats/peer-vote/peer-vote/domain/services"
	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
	"github.com/matscats/peer-vote/peer-vote/infrastructure/blockchain"
)

// CreateBlockUseCase representa o caso de uso para criar um novo bloco
type CreateBlockUseCase struct {
	chainManager  *blockchain.ChainManager
	cryptoService services.CryptographyService
}

// NewCreateBlockUseCase cria um novo caso de uso para criação de blocos
func NewCreateBlockUseCase(chainManager *blockchain.ChainManager, cryptoService services.CryptographyService) *CreateBlockUseCase {
	return &CreateBlockUseCase{
		chainManager:  chainManager,
		cryptoService: cryptoService,
	}
}

// CreateBlockRequest representa a requisição para criar um bloco
type CreateBlockRequest struct {
	Transactions []*entities.Transaction
	ValidatorID  valueobjects.NodeID
	PrivateKey   *services.PrivateKey
}

// CreateBlockResponse representa a resposta da criação de um bloco
type CreateBlockResponse struct {
	Block       *entities.Block
	BlockHash   valueobjects.Hash
	BlockIndex  uint64
	Success     bool
	Message     string
}

// Execute executa o caso de uso de criação de bloco
func (uc *CreateBlockUseCase) Execute(ctx context.Context, request *CreateBlockRequest) (*CreateBlockResponse, error) {
	// Validar entrada
	if request == nil {
		return &CreateBlockResponse{
			Success: false,
			Message: "request is nil",
		}, fmt.Errorf("request cannot be nil")
	}

	if len(request.Transactions) == 0 {
		return &CreateBlockResponse{
			Success: false,
			Message: "no transactions provided",
		}, fmt.Errorf("at least one transaction is required")
	}

	if request.ValidatorID.IsEmpty() {
		return &CreateBlockResponse{
			Success: false,
			Message: "validator ID is empty",
		}, fmt.Errorf("validator ID is required")
	}

	if request.PrivateKey == nil || !request.PrivateKey.IsValid() {
		return &CreateBlockResponse{
			Success: false,
			Message: "invalid private key",
		}, fmt.Errorf("valid private key is required")
	}

	// Preparar transações (calcular hashes se necessário)
	preparedTransactions, err := uc.prepareTransactions(ctx, request.Transactions)
	if err != nil {
		return &CreateBlockResponse{
			Success: false,
			Message: fmt.Sprintf("failed to prepare transactions: %v", err),
		}, fmt.Errorf("failed to prepare transactions: %w", err)
	}

	// Propor o bloco
	block, err := uc.chainManager.ProposeBlock(ctx, preparedTransactions, request.ValidatorID, request.PrivateKey)
	if err != nil {
		return &CreateBlockResponse{
			Success: false,
			Message: fmt.Sprintf("failed to propose block: %v", err),
		}, fmt.Errorf("failed to propose block: %w", err)
	}

	// Adicionar o bloco à cadeia
	err = uc.chainManager.AddBlock(ctx, block)
	if err != nil {
		return &CreateBlockResponse{
			Success: false,
			Message: fmt.Sprintf("failed to add block to chain: %v", err),
		}, fmt.Errorf("failed to add block to chain: %w", err)
	}

	// Calcular hash do bloco
	blockHash := uc.calculateBlockHash(ctx, block)

	return &CreateBlockResponse{
		Block:      block,
		BlockHash:  blockHash,
		BlockIndex: block.GetIndex(),
		Success:    true,
		Message:    "block created successfully",
	}, nil
}

// prepareTransactions prepara as transações calculando hashes se necessário
func (uc *CreateBlockUseCase) prepareTransactions(ctx context.Context, transactions []*entities.Transaction) ([]*entities.Transaction, error) {
	var preparedTransactions []*entities.Transaction

	for _, tx := range transactions {
		if tx == nil {
			continue
		}

		// Se a transação não tem hash, calcular
		if tx.GetHash().IsEmpty() {
			txData := tx.ToBytes()
			txHash := uc.cryptoService.HashTransaction(ctx, txData)
			tx.SetHash(txHash)
		}

		// Se a transação não tem ID, usar o hash como ID
		if tx.GetID().IsEmpty() {
			tx.SetID(tx.GetHash())
		}

		preparedTransactions = append(preparedTransactions, tx)
	}

	return preparedTransactions, nil
}

// calculateBlockHash calcula o hash de um bloco
func (uc *CreateBlockUseCase) calculateBlockHash(ctx context.Context, block *entities.Block) valueobjects.Hash {
	// Serializar dados do bloco
	blockData := fmt.Sprintf("%d-%s-%d-%s",
		block.GetIndex(),
		block.GetPreviousHash().String(),
		block.GetTimestamp().Unix(),
		block.GetMerkleRoot().String())

	return uc.cryptoService.Hash(ctx, []byte(blockData))
}

// ValidateBlockRequest representa a requisição para validar um bloco
type ValidateBlockRequest struct {
	Block     *entities.Block
	PublicKey *services.PublicKey
}

// ValidateBlockResponse representa a resposta da validação de um bloco
type ValidateBlockResponse struct {
	IsValid bool
	Message string
	Errors  []string
}

// ValidateBlock valida um bloco existente
func (uc *CreateBlockUseCase) ValidateBlock(ctx context.Context, request *ValidateBlockRequest) (*ValidateBlockResponse, error) {
	if request == nil || request.Block == nil {
		return &ValidateBlockResponse{
			IsValid: false,
			Message: "block is nil",
			Errors:  []string{"block cannot be nil"},
		}, nil
	}

	var errors []string

	// Validar estrutura básica do bloco
	if !request.Block.IsValid() {
		errors = append(errors, "block failed basic validation")
	}

	// Validar transações
	transactions := request.Block.GetTransactions()
	if len(transactions) == 0 {
		errors = append(errors, "block has no transactions")
	}

	for i, tx := range transactions {
		if !tx.IsValid() {
			errors = append(errors, fmt.Sprintf("transaction %d is invalid", i))
		}
	}

	// Validar Merkle Root
	if err := uc.validateMerkleRoot(ctx, request.Block); err != nil {
		errors = append(errors, fmt.Sprintf("merkle root validation failed: %v", err))
	}

	// Validar assinatura se chave pública fornecida
	if request.PublicKey != nil && request.PublicKey.IsValid() {
		if err := uc.validateBlockSignature(ctx, request.Block, request.PublicKey); err != nil {
			errors = append(errors, fmt.Sprintf("signature validation failed: %v", err))
		}
	}

	isValid := len(errors) == 0
	message := "block is valid"
	if !isValid {
		message = "block validation failed"
	}

	return &ValidateBlockResponse{
		IsValid: isValid,
		Message: message,
		Errors:  errors,
	}, nil
}

// validateMerkleRoot valida a raiz da Merkle Tree
func (uc *CreateBlockUseCase) validateMerkleRoot(ctx context.Context, block *entities.Block) error {
	transactions := block.GetTransactions()
	txData := make([][]byte, len(transactions))
	
	for i, tx := range transactions {
		txData[i] = tx.ToBytes()
	}

	merkleTree, err := blockchain.NewMerkleTree(txData)
	if err != nil {
		return fmt.Errorf("failed to create merkle tree: %w", err)
	}

	expectedRoot := merkleTree.GetRoot()
	actualRoot := block.GetMerkleRoot()

	if !expectedRoot.Equals(actualRoot) {
		return fmt.Errorf("merkle root mismatch: expected %s, got %s", 
			expectedRoot.String(), actualRoot.String())
	}

	return nil
}

// validateBlockSignature valida a assinatura de um bloco
func (uc *CreateBlockUseCase) validateBlockSignature(ctx context.Context, block *entities.Block, publicKey *services.PublicKey) error {
	signature := block.GetSignature()
	if signature.IsEmpty() {
		return fmt.Errorf("block has no signature")
	}

	// Usar o BlockBuilder para serializar consistentemente
	blockBuilder := blockchain.NewBlockBuilder(uc.cryptoService)
	return blockBuilder.ValidateBlockSignature(ctx, block, publicKey)
}
