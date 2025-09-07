package repositories

import (
	"context"

	"github.com/matscats/peer-vote/peer-vote/domain/entities"
	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
)

// VoteRepository define as operações de persistência para votos
type VoteRepository interface {
	// CreateVote cria um novo voto
	CreateVote(ctx context.Context, vote *entities.Vote) error

	// GetVote obtém um voto pelo ID
	GetVote(ctx context.Context, voteID valueobjects.Hash) (*entities.Vote, error)

	// GetVotesByElection obtém todos os votos de uma eleição
	GetVotesByElection(ctx context.Context, electionID valueobjects.Hash) ([]*entities.Vote, error)

	// GetVotesByVoter obtém todos os votos de um eleitor
	GetVotesByVoter(ctx context.Context, voterID valueobjects.NodeID) ([]*entities.Vote, error)

	// GetVotesByCandidate obtém todos os votos de um candidato
	GetVotesByCandidate(ctx context.Context, electionID valueobjects.Hash, candidateID string) ([]*entities.Vote, error)

	// VoteExists verifica se um voto existe
	VoteExists(ctx context.Context, voteID valueobjects.Hash) (bool, error)

	// HasVoterVoted verifica se um eleitor já votou em uma eleição
	HasVoterVoted(ctx context.Context, electionID valueobjects.Hash, voterID valueobjects.NodeID) (bool, error)

	// CountVotesByElection conta os votos de uma eleição
	CountVotesByElection(ctx context.Context, electionID valueobjects.Hash) (uint64, error)

	// CountVotesByCandidate conta os votos de um candidato
	CountVotesByCandidate(ctx context.Context, electionID valueobjects.Hash, candidateID string) (uint64, error)

	// ListVotes lista todos os votos
	ListVotes(ctx context.Context) ([]*entities.Vote, error)

	// DeleteVote remove um voto (para casos especiais de auditoria)
	DeleteVote(ctx context.Context, voteID valueobjects.Hash) error

	// GetVoterVoteCount obtém quantos votos um eleitor fez em uma eleição
	GetVoterVoteCount(ctx context.Context, electionID valueobjects.Hash, voterID valueobjects.NodeID) (int, error)

	// ValidateVoteIntegrity valida a integridade de um voto
	ValidateVoteIntegrity(ctx context.Context, vote *entities.Vote) error
}
