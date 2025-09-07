package usecases

import (
	"context"
	"fmt"

	"github.com/matscats/peer-vote/peer-vote/domain/entities"
	"github.com/matscats/peer-vote/peer-vote/domain/repositories"
	"github.com/matscats/peer-vote/peer-vote/domain/services"
	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
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
	Vote      *entities.Vote `json:"vote"`
	VoteID    string         `json:"vote_id"`
	Message   string         `json:"message"`
	Submitted bool           `json:"submitted"`
}

// SubmitVoteUseCase implementa o caso de uso de submissão de votos
type SubmitVoteUseCase struct {
	electionRepo      repositories.ElectionRepository
	voteRepo          repositories.VoteRepository
	cryptoService     services.CryptographyService
	validationService services.VotingValidationService
}

// NewSubmitVoteUseCase cria um novo caso de uso de submissão de votos
func NewSubmitVoteUseCase(
	electionRepo repositories.ElectionRepository,
	voteRepo repositories.VoteRepository,
	cryptoService services.CryptographyService,
	validationService services.VotingValidationService,
) *SubmitVoteUseCase {
	return &SubmitVoteUseCase{
		electionRepo:      electionRepo,
		voteRepo:          voteRepo,
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

	// Obter eleição
	election, err := uc.electionRepo.GetElection(ctx, request.ElectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get election: %w", err)
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

	// Persistir voto
	if err := uc.voteRepo.CreateVote(ctx, vote); err != nil {
		return nil, fmt.Errorf("failed to create vote: %w", err)
	}

	// Incrementar contador de votos do candidato
	if err := uc.electionRepo.IncrementCandidateVotes(ctx, request.ElectionID, request.CandidateID); err != nil {
		return nil, fmt.Errorf("failed to increment candidate votes: %w", err)
	}

	return &SubmitVoteResponse{
		Vote:      vote,
		VoteID:    vote.GetID().String(),
		Message:   "Vote submitted successfully",
		Submitted: true,
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

