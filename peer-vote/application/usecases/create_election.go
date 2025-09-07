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

// CreateElectionRequest representa uma requisição para criar eleição
type CreateElectionRequest struct {
	Title            string                `json:"title"`
	Description      string                `json:"description"`
	Candidates       []entities.Candidate  `json:"candidates"`
	StartTime        time.Time             `json:"start_time"`
	EndTime          time.Time             `json:"end_time"`
	CreatedBy        valueobjects.NodeID   `json:"created_by"`
	AllowAnonymous   bool                  `json:"allow_anonymous"`
	MaxVotesPerVoter int                   `json:"max_votes_per_voter"`
	PrivateKey       *services.PrivateKey  `json:"-"`
}

// CreateElectionResponse representa a resposta da criação de eleição
type CreateElectionResponse struct {
	Election        *entities.Election    `json:"election"`
	TransactionHash valueobjects.Hash     `json:"transaction_hash"`
	BlockHash       valueobjects.Hash     `json:"block_hash"`
	InBlockchain    bool                  `json:"in_blockchain"`
	Message         string                `json:"message"`
}

// CreateElectionUseCase implementa o caso de uso de criação de eleições
type CreateElectionUseCase struct {
	cryptoService     services.CryptographyService
	validationService services.VotingValidationService
	chainManager      *blockchain.ChainManager
	poaEngine         *consensus.PoAEngine
}

// NewCreateElectionUseCase cria um novo caso de uso de criação de eleições
func NewCreateElectionUseCase(
	cryptoService services.CryptographyService,
	validationService services.VotingValidationService,
	chainManager *blockchain.ChainManager,
	poaEngine *consensus.PoAEngine,
) *CreateElectionUseCase {
	return &CreateElectionUseCase{
		cryptoService:     cryptoService,
		validationService: validationService,
		chainManager:      chainManager,
		poaEngine:         poaEngine,
	}
}

// Execute executa o caso de uso de criação de eleição
func (uc *CreateElectionUseCase) Execute(ctx context.Context, request *CreateElectionRequest) (*CreateElectionResponse, error) {
	// Validar entrada
	if err := uc.validateRequest(request); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Criar eleição
	election := entities.NewElection(
		request.Title,
		request.Description,
		request.Candidates,
		request.StartTime,
		request.EndTime,
		request.CreatedBy,
	)

	// Configurar opções adicionais
	election.SetAllowAnonymous(request.AllowAnonymous)
	if request.MaxVotesPerVoter > 0 {
		election.SetMaxVotesPerVoter(request.MaxVotesPerVoter)
	}

	// Gerar ID único para a eleição
	electionData, err := election.ToBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize election: %w", err)
	}

	electionID := uc.cryptoService.HashTransaction(ctx, electionData)
	election.SetID(electionID)

	// Validar eleição
	if err := uc.validationService.ValidateElection(ctx, election); err != nil {
		return nil, fmt.Errorf("election validation failed: %w", err)
	}

	// Criar transação blockchain com os dados da eleição
	transaction, err := uc.createElectionTransaction(ctx, election, request.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create election transaction: %w", err)
	}

	// Adicionar transação ao pool do PoA Engine
	if err := uc.poaEngine.AddTransaction(ctx, transaction); err != nil {
		return nil, fmt.Errorf("failed to add election transaction to PoA pool: %w", err)
	}

	// Aguardar confirmação da transação
	blockHash, err := uc.waitForTransactionConfirmation(ctx, transaction.GetHash(), 10*time.Second)
	inBlockchain := err == nil && !blockHash.IsEmpty()
	if err != nil {
		fmt.Printf("Warning: election transaction confirmation timeout: %v\n", err)
	}

	// Eleição agora é criada apenas na blockchain

	return &CreateElectionResponse{
		Election:        election,
		TransactionHash: transaction.GetHash(),
		BlockHash:       blockHash,
		InBlockchain:    inBlockchain,
		Message:         fmt.Sprintf("Election '%s' created on blockchain", election.GetTitle()),
	}, nil
}

// validateRequest valida a requisição de criação de eleição
func (uc *CreateElectionUseCase) validateRequest(request *CreateElectionRequest) error {
	if request == nil {
		return fmt.Errorf("request is nil")
	}

	if request.Title == "" {
		return fmt.Errorf("title is required")
	}

	if len(request.Candidates) < 2 {
		return fmt.Errorf("at least 2 candidates are required")
	}

	if request.EndTime.Before(request.StartTime) {
		return fmt.Errorf("end time must be after start time")
	}

	if request.CreatedBy.IsEmpty() {
		return fmt.Errorf("creator ID is required")
	}

	if request.MaxVotesPerVoter < 0 {
		return fmt.Errorf("max votes per voter must be positive")
	}

	// Validar candidatos
	candidateIDs := make(map[string]bool)
	for i, candidate := range request.Candidates {
		if candidate.ID == "" {
			return fmt.Errorf("candidate %d: ID is required", i)
		}
		if candidate.Name == "" {
			return fmt.Errorf("candidate %d: name is required", i)
		}
		if candidateIDs[candidate.ID] {
			return fmt.Errorf("candidate %d: duplicate ID '%s'", i, candidate.ID)
		}
		candidateIDs[candidate.ID] = true
	}

	return nil
}

// createElectionTransaction cria uma transação blockchain para a eleição
func (uc *CreateElectionUseCase) createElectionTransaction(ctx context.Context, election *entities.Election, privateKey *services.PrivateKey) (*entities.Transaction, error) {
	// Serializar eleição
	electionData, err := election.ToBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize election: %w", err)
	}

	// Criar transação
	transaction := entities.NewTransaction(
		entities.ElectionTransaction,
		election.GetCreatedBy(),
		valueobjects.EmptyNodeID(),
		electionData,
	)

	// Calcular hash da transação
	txHash := uc.cryptoService.HashTransaction(ctx, electionData)
	transaction.SetHash(txHash)

	// Assinar transação
	signature, err := uc.cryptoService.Sign(ctx, electionData, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign election transaction: %w", err)
	}
	transaction.SetSignature(signature)

	return transaction, nil
}

// waitForTransactionConfirmation aguarda a confirmação da transação na blockchain
func (uc *CreateElectionUseCase) waitForTransactionConfirmation(ctx context.Context, txHash valueobjects.Hash, timeout time.Duration) (valueobjects.Hash, error) {
	// Criar contexto com timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Polling para verificar se a transação foi incluída em um bloco
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			return valueobjects.EmptyHash(), fmt.Errorf("transaction confirmation timeout")
		case <-ticker.C:
			// Verificar se a transação foi incluída na blockchain
			blockHash, err := uc.findTransactionInBlockchain(ctx, txHash)
			if err == nil && !blockHash.IsEmpty() {
				return blockHash, nil
			}
		}
	}
}

// findTransactionInBlockchain procura uma transação na blockchain
func (uc *CreateElectionUseCase) findTransactionInBlockchain(ctx context.Context, txHash valueobjects.Hash) (valueobjects.Hash, error) {
	// Obter altura atual da blockchain
	height, err := uc.chainManager.GetChainHeight(ctx)
	if err != nil {
		return valueobjects.EmptyHash(), err
	}

	// Procurar nos últimos blocos
	searchDepth := uint64(10)
	startIndex := uint64(0)
	if height > searchDepth {
		startIndex = height - searchDepth
	}

	for i := height; i >= startIndex && i <= height; i-- {
		block, err := uc.chainManager.GetBlockByIndex(ctx, i)
		if err != nil {
			continue
		}

		// Verificar se a transação está neste bloco
		transactions := block.GetTransactions()
		for _, tx := range transactions {
			if tx.GetHash().Equals(txHash) {
				// Calcular hash do bloco
				blockHash := uc.chainManager.CalculateBlockHash(ctx, block)
				return blockHash, nil
			}
		}

		// Evitar underflow
		if i == 0 {
			break
		}
	}

	return valueobjects.EmptyHash(), fmt.Errorf("transaction not found in blockchain")
}
