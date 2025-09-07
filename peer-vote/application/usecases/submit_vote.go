package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/matscats/peer-vote/peer-vote/domain/entities"
	"github.com/matscats/peer-vote/peer-vote/domain/repositories"
	"github.com/matscats/peer-vote/peer-vote/domain/services"
	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
	"github.com/matscats/peer-vote/peer-vote/infrastructure/blockchain"
	"github.com/matscats/peer-vote/peer-vote/infrastructure/consensus"
)

// SubmitVoteRequest representa uma requisição para submeter um voto
type SubmitVoteRequest struct {
	ElectionID  valueobjects.Hash     `json:"election_id"`
	VoterID     valueobjects.NodeID   `json:"voter_id"`
	CandidateID string                `json:"candidate_id"`
	IsAnonymous bool                  `json:"is_anonymous"`
	PrivateKey  *services.PrivateKey  `json:"-"` // Não serializar por segurança
}

// SubmitVoteResponse representa a resposta da submissão de voto
type SubmitVoteResponse struct {
	Vote            *entities.Vote        `json:"vote"`
	VoteID          string                `json:"vote_id"`
	TransactionHash valueobjects.Hash     `json:"transaction_hash"`
	BlockHash       valueobjects.Hash     `json:"block_hash,omitempty"`
	Message         string                `json:"message"`
	Submitted       bool                  `json:"submitted"`
	InBlockchain    bool                  `json:"in_blockchain"`
}

// SubmitVoteUseCase implementa o caso de uso de submissão de votos
type SubmitVoteUseCase struct {
	electionRepo      repositories.ElectionRepository
	chainManager      *blockchain.ChainManager
	poaEngine         *consensus.PoAEngine
	cryptoService     services.CryptographyService
	validationService services.VotingValidationService
}

// NewSubmitVoteUseCase cria um novo caso de uso de submissão de votos
func NewSubmitVoteUseCase(
	electionRepo repositories.ElectionRepository,
	chainManager *blockchain.ChainManager,
	poaEngine *consensus.PoAEngine,
	cryptoService services.CryptographyService,
	validationService services.VotingValidationService,
) *SubmitVoteUseCase {
	return &SubmitVoteUseCase{
		electionRepo:      electionRepo,
		chainManager:      chainManager,
		poaEngine:         poaEngine,
		cryptoService:     cryptoService,
		validationService: validationService,
	}
}

// Execute executa o caso de uso de submissão de voto
func (uc *SubmitVoteUseCase) Execute(ctx context.Context, request *SubmitVoteRequest) (*SubmitVoteResponse, error) {
	// Validar entrada
	if err := uc.validateRequest(request); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Obter eleição da blockchain
	election, err := uc.chainManager.GetElectionFromBlockchain(ctx, request.ElectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get election from blockchain: %w", err)
	}

	// Criar voto
	vote := entities.NewVote(
		request.ElectionID,
		request.VoterID,
		request.CandidateID,
		request.IsAnonymous,
	)

	// Assinar voto primeiro
	if err := uc.signVote(ctx, vote, request.PrivateKey); err != nil {
		return nil, fmt.Errorf("failed to sign vote: %w", err)
	}

	// Validar voto após assinatura
	if err := uc.validationService.ValidateVote(ctx, vote, election); err != nil {
		return nil, fmt.Errorf("vote validation failed: %w", err)
	}

	// Gerar ID do voto
	voteData, err := vote.ToBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize vote: %w", err)
	}

	voteID := uc.cryptoService.HashTransaction(ctx, voteData)
	vote.SetID(voteID)

	// === NOVA LÓGICA BLOCKCHAIN ===
	// Criar transação blockchain com os dados do voto
	transaction, err := uc.createVoteTransaction(ctx, vote, request.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create vote transaction: %w", err)
	}

	// Adicionar transação ao pool do PoA Engine
	if err := uc.poaEngine.AddTransaction(ctx, transaction); err != nil {
		return nil, fmt.Errorf("failed to add transaction to PoA pool: %w", err)
	}

	// Aguardar confirmação da transação (opcional - pode ser assíncrono)
	blockHash, err := uc.waitForTransactionConfirmation(ctx, transaction.GetHash(), 10*time.Second)
	if err != nil {
		// Log do erro mas não falha - transação está no pool
		fmt.Printf("Warning: transaction confirmation timeout: %v\n", err)
	}

	return &SubmitVoteResponse{
		Vote:            vote,
		VoteID:          vote.GetID().String(),
		TransactionHash: transaction.GetHash(),
		BlockHash:       blockHash,
		Message:         "Vote submitted to blockchain successfully",
		Submitted:       true,
		InBlockchain:    !blockHash.IsEmpty(),
	}, nil
}

// validateRequest valida a requisição de submissão de voto
func (uc *SubmitVoteUseCase) validateRequest(request *SubmitVoteRequest) error {
	if request == nil {
		return fmt.Errorf("request is nil")
	}

	if request.ElectionID.IsEmpty() {
		return fmt.Errorf("election ID is required")
	}

	if !request.IsAnonymous && request.VoterID.IsEmpty() {
		return fmt.Errorf("voter ID is required for non-anonymous votes")
	}

	if request.CandidateID == "" {
		return fmt.Errorf("candidate ID is required")
	}

	if request.PrivateKey == nil || !request.PrivateKey.IsValid() {
		return fmt.Errorf("valid private key is required")
	}

	return nil
}

// signVote assina o voto com a chave privada
func (uc *SubmitVoteUseCase) signVote(ctx context.Context, vote *entities.Vote, privateKey *services.PrivateKey) error {
	// Serializar dados do voto para assinatura
	voteData, err := vote.ToBytes()
	if err != nil {
		return fmt.Errorf("failed to serialize vote for signing: %w", err)
	}

	// Assinar
	signature, err := uc.cryptoService.Sign(ctx, voteData, privateKey)
	if err != nil {
		return fmt.Errorf("failed to sign vote: %w", err)
	}

	vote.SetSignature(signature)
	return nil
}

// createVoteTransaction cria uma transação blockchain para o voto
func (uc *SubmitVoteUseCase) createVoteTransaction(ctx context.Context, vote *entities.Vote, privateKey *services.PrivateKey) (*entities.Transaction, error) {
	// Serializar o voto como dados da transação (incluindo ID)
	voteData, err := vote.ToBytesWithID()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize vote: %w", err)
	}

	// Criar transação com timestamp único para evitar duplicatas
	transaction := entities.NewTransaction(
		"VOTE",                     // Tipo de transação
		vote.GetVoterID(),         // Remetente (eleitor)
		valueobjects.EmptyNodeID(), // Destinatário vazio para votos
		voteData,                  // Dados do voto
	)

	// Gerar ID único baseado no conteúdo + timestamp
	txData := transaction.ToBytes()
	txHash := uc.cryptoService.HashTransaction(ctx, txData)
	transaction.SetID(txHash)
	transaction.SetHash(txHash)

	// Assinar transação
	signature, err := uc.cryptoService.Sign(ctx, txData, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}
	transaction.SetSignature(signature)

	return transaction, nil
}

// waitForTransactionConfirmation aguarda a confirmação da transação na blockchain
func (uc *SubmitVoteUseCase) waitForTransactionConfirmation(ctx context.Context, txHash valueobjects.Hash, timeout time.Duration) (valueobjects.Hash, error) {
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
func (uc *SubmitVoteUseCase) findTransactionInBlockchain(ctx context.Context, txHash valueobjects.Hash) (valueobjects.Hash, error) {
	// Obter altura atual da blockchain
	height, err := uc.chainManager.GetChainHeight(ctx)
	if err != nil {
		return valueobjects.EmptyHash(), err
	}

	// Procurar nos últimos blocos (otimização - transações recentes estão nos blocos mais novos)
	searchDepth := uint64(10) // Procurar nos últimos 10 blocos
	startIndex := uint64(0)
	if height > searchDepth {
		startIndex = height - searchDepth
	}

	for i := height; i >= startIndex && i <= height; i-- {
		block, err := uc.chainManager.GetBlockByIndex(ctx, i)
		if err != nil {
			continue // Pular blocos com erro
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

