package persistence

import (
	"context"
	"fmt"
	"sync"

	"github.com/matscats/peer-vote/peer-vote/domain/entities"
	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
)

// MemoryVoteRepository implementa VoteRepository usando memória
type MemoryVoteRepository struct {
	votes map[string]*entities.Vote
	mutex sync.RWMutex
}

// NewMemoryVoteRepository cria um novo repositório de votos em memória
func NewMemoryVoteRepository() *MemoryVoteRepository {
	return &MemoryVoteRepository{
		votes: make(map[string]*entities.Vote),
	}
}

// CreateVote cria um novo voto
func (r *MemoryVoteRepository) CreateVote(ctx context.Context, vote *entities.Vote) error {
	if vote == nil {
		return fmt.Errorf("vote is nil")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	voteID := vote.GetID().String()
	if _, exists := r.votes[voteID]; exists {
		return fmt.Errorf("vote with ID %s already exists", voteID)
	}

	r.votes[voteID] = vote
	return nil
}

// GetVote obtém um voto pelo ID
func (r *MemoryVoteRepository) GetVote(ctx context.Context, voteID valueobjects.Hash) (*entities.Vote, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	vote, exists := r.votes[voteID.String()]
	if !exists {
		return nil, fmt.Errorf("vote with ID %s not found", voteID.String())
	}

	return vote, nil
}

// GetVotesByElection obtém todos os votos de uma eleição
func (r *MemoryVoteRepository) GetVotesByElection(ctx context.Context, electionID valueobjects.Hash) ([]*entities.Vote, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var electionVotes []*entities.Vote
	for _, vote := range r.votes {
		if vote.GetElectionID().Equals(electionID) {
			electionVotes = append(electionVotes, vote)
		}
	}

	return electionVotes, nil
}

// GetVotesByVoter obtém todos os votos de um eleitor
func (r *MemoryVoteRepository) GetVotesByVoter(ctx context.Context, voterID valueobjects.NodeID) ([]*entities.Vote, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var voterVotes []*entities.Vote
	for _, vote := range r.votes {
		if !vote.IsAnonymous() && vote.GetVoterID().Equals(voterID) {
			voterVotes = append(voterVotes, vote)
		}
	}

	return voterVotes, nil
}

// GetVotesByCandidate obtém todos os votos de um candidato
func (r *MemoryVoteRepository) GetVotesByCandidate(ctx context.Context, electionID valueobjects.Hash, candidateID string) ([]*entities.Vote, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var candidateVotes []*entities.Vote
	for _, vote := range r.votes {
		if vote.GetElectionID().Equals(electionID) && vote.GetCandidateID() == candidateID {
			candidateVotes = append(candidateVotes, vote)
		}
	}

	return candidateVotes, nil
}

// VoteExists verifica se um voto existe
func (r *MemoryVoteRepository) VoteExists(ctx context.Context, voteID valueobjects.Hash) (bool, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	_, exists := r.votes[voteID.String()]
	return exists, nil
}

// HasVoterVoted verifica se um eleitor já votou em uma eleição
func (r *MemoryVoteRepository) HasVoterVoted(ctx context.Context, electionID valueobjects.Hash, voterID valueobjects.NodeID) (bool, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for _, vote := range r.votes {
		if vote.GetElectionID().Equals(electionID) && 
		   !vote.IsAnonymous() && 
		   vote.GetVoterID().Equals(voterID) {
			return true, nil
		}
	}

	return false, nil
}

// CountVotesByElection conta os votos de uma eleição
func (r *MemoryVoteRepository) CountVotesByElection(ctx context.Context, electionID valueobjects.Hash) (uint64, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var count uint64
	for _, vote := range r.votes {
		if vote.GetElectionID().Equals(electionID) {
			count++
		}
	}

	return count, nil
}

// CountVotesByCandidate conta os votos de um candidato
func (r *MemoryVoteRepository) CountVotesByCandidate(ctx context.Context, electionID valueobjects.Hash, candidateID string) (uint64, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var count uint64
	for _, vote := range r.votes {
		if vote.GetElectionID().Equals(electionID) && vote.GetCandidateID() == candidateID {
			count++
		}
	}

	return count, nil
}

// ListVotes lista todos os votos
func (r *MemoryVoteRepository) ListVotes(ctx context.Context) ([]*entities.Vote, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	votes := make([]*entities.Vote, 0, len(r.votes))
	for _, vote := range r.votes {
		votes = append(votes, vote)
	}

	return votes, nil
}

// DeleteVote remove um voto (para casos especiais de auditoria)
func (r *MemoryVoteRepository) DeleteVote(ctx context.Context, voteID valueobjects.Hash) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	id := voteID.String()
	if _, exists := r.votes[id]; !exists {
		return fmt.Errorf("vote with ID %s not found", id)
	}

	delete(r.votes, id)
	return nil
}

// GetVoterVoteCount obtém quantos votos um eleitor fez em uma eleição
func (r *MemoryVoteRepository) GetVoterVoteCount(ctx context.Context, electionID valueobjects.Hash, voterID valueobjects.NodeID) (int, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	count := 0
	for _, vote := range r.votes {
		if vote.GetElectionID().Equals(electionID) && 
		   !vote.IsAnonymous() && 
		   vote.GetVoterID().Equals(voterID) {
			count++
		}
	}

	return count, nil
}

// ValidateVoteIntegrity valida a integridade de um voto
func (r *MemoryVoteRepository) ValidateVoteIntegrity(ctx context.Context, vote *entities.Vote) error {
	if vote == nil {
		return fmt.Errorf("vote is nil")
	}

	if !vote.IsValid() {
		return fmt.Errorf("vote is invalid")
	}

	// Verificar se o voto existe no repositório
	exists, err := r.VoteExists(ctx, vote.GetID())
	if err != nil {
		return fmt.Errorf("failed to check vote existence: %w", err)
	}

	if !exists {
		return fmt.Errorf("vote does not exist in repository")
	}

	// Obter voto do repositório e comparar
	storedVote, err := r.GetVote(ctx, vote.GetID())
	if err != nil {
		return fmt.Errorf("failed to get stored vote: %w", err)
	}

	// Comparar campos críticos
	if !storedVote.GetElectionID().Equals(vote.GetElectionID()) {
		return fmt.Errorf("election ID mismatch")
	}

	if storedVote.GetCandidateID() != vote.GetCandidateID() {
		return fmt.Errorf("candidate ID mismatch")
	}

	if !storedVote.GetSignature().Equals(vote.GetSignature()) {
		return fmt.Errorf("signature mismatch")
	}

	return nil
}
