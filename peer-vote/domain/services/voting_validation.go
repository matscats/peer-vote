package services

import (
	"context"
	"fmt"

	"github.com/matscats/peer-vote/peer-vote/domain/entities"
	"github.com/matscats/peer-vote/peer-vote/domain/repositories"
	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
)

// VotingValidationService define as operações de validação de votação
type VotingValidationService interface {
	// ValidateElection valida se uma eleição é válida
	ValidateElection(ctx context.Context, election *entities.Election) error

	// ValidateVote valida se um voto é válido
	ValidateVote(ctx context.Context, vote *entities.Vote, election *entities.Election) error

	// ValidateVoterEligibility valida se um eleitor pode votar
	ValidateVoterEligibility(ctx context.Context, voterID valueobjects.NodeID, electionID valueobjects.Hash) error

	// ValidateVoteSignature valida a assinatura de um voto
	ValidateVoteSignature(ctx context.Context, vote *entities.Vote, publicKey *PublicKey) error

	// PreventDoubleVoting previne votação dupla
	PreventDoubleVoting(ctx context.Context, voterID valueobjects.NodeID, electionID valueobjects.Hash) error

	// ValidateElectionTiming valida se a eleição está no período correto
	ValidateElectionTiming(ctx context.Context, election *entities.Election) error

	// ValidateCandidate valida se um candidato existe na eleição
	ValidateCandidate(ctx context.Context, candidateID string, election *entities.Election) error

	// ValidateVoteForAudit valida um voto para auditoria (sem regras de prevenção)
	ValidateVoteForAudit(ctx context.Context, vote *entities.Vote, election *entities.Election) error
}

// VotingValidator implementa VotingValidationService
type VotingValidator struct {
	electionRepo repositories.ElectionRepository
	voteRepo     repositories.VoteRepository
	cryptoService CryptographyService
}

// NewVotingValidator cria um novo validador de votação
func NewVotingValidator(
	electionRepo repositories.ElectionRepository,
	voteRepo repositories.VoteRepository,
	cryptoService CryptographyService,
) *VotingValidator {
	return &VotingValidator{
		electionRepo:  electionRepo,
		voteRepo:      voteRepo,
		cryptoService: cryptoService,
	}
}

// ValidateElection valida se uma eleição é válida
func (v *VotingValidator) ValidateElection(ctx context.Context, election *entities.Election) error {
	if election == nil {
		return fmt.Errorf("election is nil")
	}

	if !election.IsValid() {
		return fmt.Errorf("election is invalid")
	}

	// Verificar se já existe uma eleição com o mesmo título
	existingElection, err := v.electionRepo.GetElectionByTitle(ctx, election.GetTitle())
	if err == nil && existingElection != nil && !existingElection.GetID().Equals(election.GetID()) {
		return fmt.Errorf("election with title '%s' already exists", election.GetTitle())
	}

	return nil
}

// ValidateVote valida se um voto é válido
func (v *VotingValidator) ValidateVote(ctx context.Context, vote *entities.Vote, election *entities.Election) error {
	if vote == nil {
		return fmt.Errorf("vote is nil")
	}

	if election == nil {
		return fmt.Errorf("election is nil")
	}

	// Validações básicas do voto
	if !vote.IsValid() {
		return fmt.Errorf("vote is invalid")
	}

	// Verificar se o voto é para a eleição correta
	if !vote.GetElectionID().Equals(election.GetID()) {
		return fmt.Errorf("vote election ID does not match")
	}

	// Validar timing da eleição
	if err := v.ValidateElectionTiming(ctx, election); err != nil {
		return fmt.Errorf("election timing validation failed: %w", err)
	}

	// Validar candidato
	if err := v.ValidateCandidate(ctx, vote.GetCandidateID(), election); err != nil {
		return fmt.Errorf("candidate validation failed: %w", err)
	}

	// Validar elegibilidade do eleitor
	if !vote.IsAnonymous() {
		if err := v.ValidateVoterEligibility(ctx, vote.GetVoterID(), election.GetID()); err != nil {
			return fmt.Errorf("voter eligibility validation failed: %w", err)
		}

		// Prevenir votação dupla
		if err := v.PreventDoubleVoting(ctx, vote.GetVoterID(), election.GetID()); err != nil {
			return fmt.Errorf("double voting prevention failed: %w", err)
		}
	}

	return nil
}

// ValidateVoterEligibility valida se um eleitor pode votar
func (v *VotingValidator) ValidateVoterEligibility(ctx context.Context, voterID valueobjects.NodeID, electionID valueobjects.Hash) error {
	if voterID.IsEmpty() {
		return fmt.Errorf("voter ID is empty")
	}

	// Verificar se o eleitor não está banido ou com restrições
	// Esta lógica pode ser expandida conforme necessário
	
	return nil
}

// ValidateVoteSignature valida a assinatura de um voto
func (v *VotingValidator) ValidateVoteSignature(ctx context.Context, vote *entities.Vote, publicKey *PublicKey) error {
	if vote == nil {
		return fmt.Errorf("vote is nil")
	}

	if publicKey == nil || !publicKey.IsValid() {
		return fmt.Errorf("invalid public key")
	}

	// Serializar dados do voto para verificação
	voteData, err := vote.ToBytes()
	if err != nil {
		return fmt.Errorf("failed to serialize vote: %w", err)
	}

	// Verificar assinatura
	valid, err := v.cryptoService.Verify(ctx, voteData, vote.GetSignature(), publicKey)
	if err != nil {
		return fmt.Errorf("signature verification error: %w", err)
	}

	if !valid {
		return fmt.Errorf("invalid vote signature")
	}

	return nil
}

// PreventDoubleVoting previne votação dupla
func (v *VotingValidator) PreventDoubleVoting(ctx context.Context, voterID valueobjects.NodeID, electionID valueobjects.Hash) error {
	// Verificar se o eleitor já votou nesta eleição
	hasVoted, err := v.voteRepo.HasVoterVoted(ctx, electionID, voterID)
	if err != nil {
		return fmt.Errorf("failed to check if voter has voted: %w", err)
	}

	if hasVoted {
		// Verificar quantos votos o eleitor já fez
		voteCount, err := v.voteRepo.GetVoterVoteCount(ctx, electionID, voterID)
		if err != nil {
			return fmt.Errorf("failed to get voter vote count: %w", err)
		}

		// Obter a eleição para verificar o limite de votos
		election, err := v.electionRepo.GetElection(ctx, electionID)
		if err != nil {
			return fmt.Errorf("failed to get election: %w", err)
		}

		if voteCount >= election.GetMaxVotesPerVoter() {
			return fmt.Errorf("voter has already reached maximum votes limit (%d)", election.GetMaxVotesPerVoter())
		}
	}

	return nil
}

// ValidateElectionTiming valida se a eleição está no período correto
func (v *VotingValidator) ValidateElectionTiming(ctx context.Context, election *entities.Election) error {
	if !election.CanVote() {
		switch election.GetStatus() {
		case entities.ElectionPending:
			return fmt.Errorf("election has not started yet")
		case entities.ElectionClosed:
			return fmt.Errorf("election has ended")
		case entities.ElectionCancelled:
			return fmt.Errorf("election has been cancelled")
		default:
			return fmt.Errorf("election is not active")
		}
	}

	return nil
}

// ValidateCandidate valida se um candidato existe na eleição
func (v *VotingValidator) ValidateCandidate(ctx context.Context, candidateID string, election *entities.Election) error {
	if candidateID == "" {
		return fmt.Errorf("candidate ID is empty")
	}

	_, exists := election.GetCandidate(candidateID)
	if !exists {
		return fmt.Errorf("candidate '%s' does not exist in election", candidateID)
	}

	return nil
}

// ValidateVoteForAudit valida um voto para auditoria (sem regras de prevenção)
func (v *VotingValidator) ValidateVoteForAudit(ctx context.Context, vote *entities.Vote, election *entities.Election) error {
	if vote == nil {
		return fmt.Errorf("vote is nil")
	}

	if election == nil {
		return fmt.Errorf("election is nil")
	}

	// Validações básicas do voto
	if !vote.IsValid() {
		return fmt.Errorf("vote is invalid")
	}

	// Verificar se o voto é para a eleição correta
	if !vote.GetElectionID().Equals(election.GetID()) {
		return fmt.Errorf("vote election ID does not match")
	}

	// Validar candidato
	if err := v.ValidateCandidate(ctx, vote.GetCandidateID(), election); err != nil {
		return fmt.Errorf("candidate validation failed: %w", err)
	}

	// Validar elegibilidade básica do eleitor (sem verificar double-voting)
	if !vote.IsAnonymous() {
		if err := v.ValidateVoterEligibility(ctx, vote.GetVoterID(), election.GetID()); err != nil {
			return fmt.Errorf("voter eligibility validation failed: %w", err)
		}
	}

	// NOTA: Não validamos timing nem double-voting na auditoria
	// pois estamos auditando votos históricos já aceitos

	return nil
}
