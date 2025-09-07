package persistence

import (
	"context"
	"fmt"
	"sync"

	"github.com/matscats/peer-vote/peer-vote/domain/entities"
	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
)

// MemoryElectionRepository implementa ElectionRepository usando memória
type MemoryElectionRepository struct {
	elections map[string]*entities.Election
	mutex     sync.RWMutex
}

// NewMemoryElectionRepository cria um novo repositório de eleições em memória
func NewMemoryElectionRepository() *MemoryElectionRepository {
	return &MemoryElectionRepository{
		elections: make(map[string]*entities.Election),
	}
}

// CreateElection cria uma nova eleição
func (r *MemoryElectionRepository) CreateElection(ctx context.Context, election *entities.Election) error {
	if election == nil {
		return fmt.Errorf("election is nil")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	electionID := election.GetID().String()
	if _, exists := r.elections[electionID]; exists {
		return fmt.Errorf("election with ID %s already exists", electionID)
	}

	r.elections[electionID] = election
	return nil
}

// GetElection obtém uma eleição pelo ID
func (r *MemoryElectionRepository) GetElection(ctx context.Context, electionID valueobjects.Hash) (*entities.Election, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	election, exists := r.elections[electionID.String()]
	if !exists {
		return nil, fmt.Errorf("election with ID %s not found", electionID.String())
	}

	return election, nil
}

// GetElectionByTitle obtém uma eleição pelo título
func (r *MemoryElectionRepository) GetElectionByTitle(ctx context.Context, title string) (*entities.Election, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for _, election := range r.elections {
		if election.GetTitle() == title {
			return election, nil
		}
	}

	return nil, fmt.Errorf("election with title '%s' not found", title)
}

// UpdateElection atualiza uma eleição existente
func (r *MemoryElectionRepository) UpdateElection(ctx context.Context, election *entities.Election) error {
	if election == nil {
		return fmt.Errorf("election is nil")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	electionID := election.GetID().String()
	if _, exists := r.elections[electionID]; !exists {
		return fmt.Errorf("election with ID %s not found", electionID)
	}

	r.elections[electionID] = election
	return nil
}

// DeleteElection remove uma eleição
func (r *MemoryElectionRepository) DeleteElection(ctx context.Context, electionID valueobjects.Hash) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	id := electionID.String()
	if _, exists := r.elections[id]; !exists {
		return fmt.Errorf("election with ID %s not found", id)
	}

	delete(r.elections, id)
	return nil
}

// ListElections lista todas as eleições
func (r *MemoryElectionRepository) ListElections(ctx context.Context) ([]*entities.Election, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	elections := make([]*entities.Election, 0, len(r.elections))
	for _, election := range r.elections {
		elections = append(elections, election)
	}

	return elections, nil
}

// ListActiveElections lista eleições ativas
func (r *MemoryElectionRepository) ListActiveElections(ctx context.Context) ([]*entities.Election, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var activeElections []*entities.Election
	for _, election := range r.elections {
		if election.IsActive() {
			activeElections = append(activeElections, election)
		}
	}

	return activeElections, nil
}

// ListElectionsByCreator lista eleições criadas por um nó específico
func (r *MemoryElectionRepository) ListElectionsByCreator(ctx context.Context, creatorID valueobjects.NodeID) ([]*entities.Election, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var creatorElections []*entities.Election
	for _, election := range r.elections {
		if election.GetCreatedBy().Equals(creatorID) {
			creatorElections = append(creatorElections, election)
		}
	}

	return creatorElections, nil
}

// ElectionExists verifica se uma eleição existe
func (r *MemoryElectionRepository) ElectionExists(ctx context.Context, electionID valueobjects.Hash) (bool, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	_, exists := r.elections[electionID.String()]
	return exists, nil
}

// GetElectionResults obtém os resultados de uma eleição
func (r *MemoryElectionRepository) GetElectionResults(ctx context.Context, electionID valueobjects.Hash) (map[string]uint64, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	election, exists := r.elections[electionID.String()]
	if !exists {
		return nil, fmt.Errorf("election with ID %s not found", electionID.String())
	}

	return election.GetResults(), nil
}

// UpdateElectionStatus atualiza o status de uma eleição
func (r *MemoryElectionRepository) UpdateElectionStatus(ctx context.Context, electionID valueobjects.Hash, status entities.ElectionStatus) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	election, exists := r.elections[electionID.String()]
	if !exists {
		return fmt.Errorf("election with ID %s not found", electionID.String())
	}

	election.SetStatus(status)
	return nil
}

// IncrementCandidateVotes incrementa os votos de um candidato
func (r *MemoryElectionRepository) IncrementCandidateVotes(ctx context.Context, electionID valueobjects.Hash, candidateID string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	election, exists := r.elections[electionID.String()]
	if !exists {
		return fmt.Errorf("election with ID %s not found", electionID.String())
	}

	if !election.IncrementVoteCount(candidateID) {
		return fmt.Errorf("candidate with ID %s not found in election", candidateID)
	}

	return nil
}
