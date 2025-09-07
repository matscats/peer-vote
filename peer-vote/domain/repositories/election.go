package repositories

import (
	"context"

	"github.com/matscats/peer-vote/peer-vote/domain/entities"
	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
)

// ElectionRepository define as operações de persistência para eleições
type ElectionRepository interface {
	// CreateElection cria uma nova eleição
	CreateElection(ctx context.Context, election *entities.Election) error

	// GetElection obtém uma eleição pelo ID
	GetElection(ctx context.Context, electionID valueobjects.Hash) (*entities.Election, error)

	// GetElectionByTitle obtém uma eleição pelo título
	GetElectionByTitle(ctx context.Context, title string) (*entities.Election, error)

	// UpdateElection atualiza uma eleição existente
	UpdateElection(ctx context.Context, election *entities.Election) error

	// DeleteElection remove uma eleição
	DeleteElection(ctx context.Context, electionID valueobjects.Hash) error

	// ListElections lista todas as eleições
	ListElections(ctx context.Context) ([]*entities.Election, error)

	// ListActiveElections lista eleições ativas
	ListActiveElections(ctx context.Context) ([]*entities.Election, error)

	// ListElectionsByCreator lista eleições criadas por um nó específico
	ListElectionsByCreator(ctx context.Context, creatorID valueobjects.NodeID) ([]*entities.Election, error)

	// ElectionExists verifica se uma eleição existe
	ElectionExists(ctx context.Context, electionID valueobjects.Hash) (bool, error)

	// GetElectionResults obtém os resultados de uma eleição
	GetElectionResults(ctx context.Context, electionID valueobjects.Hash) (map[string]uint64, error)

	// UpdateElectionStatus atualiza o status de uma eleição
	UpdateElectionStatus(ctx context.Context, electionID valueobjects.Hash, status entities.ElectionStatus) error

	// IncrementCandidateVotes incrementa os votos de um candidato
	IncrementCandidateVotes(ctx context.Context, electionID valueobjects.Hash, candidateID string) error
}
