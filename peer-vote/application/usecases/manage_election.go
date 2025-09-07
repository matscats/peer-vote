package usecases

import (
	"context"
	"fmt"

	"github.com/matscats/peer-vote/peer-vote/domain/entities"
	"github.com/matscats/peer-vote/peer-vote/domain/repositories"
	"github.com/matscats/peer-vote/peer-vote/domain/services"
	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
)

// GetElectionRequest representa uma requisição para obter eleição
type GetElectionRequest struct {
	ElectionID valueobjects.Hash `json:"election_id"`
}

// GetElectionResponse representa a resposta de obter eleição
type GetElectionResponse struct {
	Election *entities.Election `json:"election"`
	Results  map[string]uint64  `json:"results"`
	Message  string             `json:"message"`
}

// ListElectionsRequest representa uma requisição para listar eleições
type ListElectionsRequest struct {
	ActiveOnly bool                  `json:"active_only"`
	CreatedBy  valueobjects.NodeID   `json:"created_by,omitempty"`
}

// ListElectionsResponse representa a resposta de listar eleições
type ListElectionsResponse struct {
	Elections []*entities.Election `json:"elections"`
	Count     int                  `json:"count"`
	Message   string               `json:"message"`
}

// UpdateElectionStatusRequest representa uma requisição para atualizar status
type UpdateElectionStatusRequest struct {
	ElectionID valueobjects.Hash         `json:"election_id"`
	NewStatus  entities.ElectionStatus   `json:"new_status"`
	UpdatedBy  valueobjects.NodeID       `json:"updated_by"`
}

// UpdateElectionStatusResponse representa a resposta de atualizar status
type UpdateElectionStatusResponse struct {
	Election *entities.Election `json:"election"`
	Message  string             `json:"message"`
	Updated  bool               `json:"updated"`
}

// GetElectionResultsRequest representa uma requisição para obter resultados
type GetElectionResultsRequest struct {
	ElectionID valueobjects.Hash `json:"election_id"`
}

// GetElectionResultsResponse representa a resposta de obter resultados
type GetElectionResultsResponse struct {
	ElectionID   valueobjects.Hash  `json:"election_id"`
	Results      map[string]uint64  `json:"results"`
	TotalVotes   uint64             `json:"total_votes"`
	Candidates   []entities.Candidate `json:"candidates"`
	ElectionInfo *entities.Election `json:"election_info"`
	Message      string             `json:"message"`
}

// ManageElectionUseCase implementa os casos de uso de gerenciamento de eleições
type ManageElectionUseCase struct {
	electionRepo      repositories.ElectionRepository
	voteRepo          repositories.VoteRepository
	validationService services.VotingValidationService
}

// NewManageElectionUseCase cria um novo caso de uso de gerenciamento de eleições
func NewManageElectionUseCase(
	electionRepo repositories.ElectionRepository,
	voteRepo repositories.VoteRepository,
	validationService services.VotingValidationService,
) *ManageElectionUseCase {
	return &ManageElectionUseCase{
		electionRepo:      electionRepo,
		voteRepo:          voteRepo,
		validationService: validationService,
	}
}

// GetElection obtém uma eleição específica
func (uc *ManageElectionUseCase) GetElection(ctx context.Context, request *GetElectionRequest) (*GetElectionResponse, error) {
	if request == nil || request.ElectionID.IsEmpty() {
		return nil, fmt.Errorf("invalid request: election ID is required")
	}

	// Obter eleição
	election, err := uc.electionRepo.GetElection(ctx, request.ElectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get election: %w", err)
	}

	// Obter resultados
	results, err := uc.electionRepo.GetElectionResults(ctx, request.ElectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get election results: %w", err)
	}

	return &GetElectionResponse{
		Election: election,
		Results:  results,
		Message:  fmt.Sprintf("Election '%s' retrieved successfully", election.GetTitle()),
	}, nil
}

// ListElections lista eleições
func (uc *ManageElectionUseCase) ListElections(ctx context.Context, request *ListElectionsRequest) (*ListElectionsResponse, error) {
	var elections []*entities.Election
	var err error

	if request == nil {
		request = &ListElectionsRequest{}
	}

	// Escolher método de listagem baseado nos filtros
	if !request.CreatedBy.IsEmpty() {
		elections, err = uc.electionRepo.ListElectionsByCreator(ctx, request.CreatedBy)
	} else if request.ActiveOnly {
		elections, err = uc.electionRepo.ListActiveElections(ctx)
	} else {
		elections, err = uc.electionRepo.ListElections(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list elections: %w", err)
	}

	return &ListElectionsResponse{
		Elections: elections,
		Count:     len(elections),
		Message:   fmt.Sprintf("Found %d elections", len(elections)),
	}, nil
}

// UpdateElectionStatus atualiza o status de uma eleição
func (uc *ManageElectionUseCase) UpdateElectionStatus(ctx context.Context, request *UpdateElectionStatusRequest) (*UpdateElectionStatusResponse, error) {
	if err := uc.validateUpdateStatusRequest(request); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Obter eleição atual
	election, err := uc.electionRepo.GetElection(ctx, request.ElectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get election: %w", err)
	}

	// Validar transição de status
	if err := uc.validateStatusTransition(election.GetStatus(), request.NewStatus); err != nil {
		return nil, fmt.Errorf("invalid status transition: %w", err)
	}

	// Verificar permissões (apenas o criador pode alterar)
	if !election.GetCreatedBy().Equals(request.UpdatedBy) {
		return nil, fmt.Errorf("only the election creator can update its status")
	}

	// Atualizar status
	if err := uc.electionRepo.UpdateElectionStatus(ctx, request.ElectionID, request.NewStatus); err != nil {
		return nil, fmt.Errorf("failed to update election status: %w", err)
	}

	// Obter eleição atualizada
	updatedElection, err := uc.electionRepo.GetElection(ctx, request.ElectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated election: %w", err)
	}

	return &UpdateElectionStatusResponse{
		Election: updatedElection,
		Message:  fmt.Sprintf("Election status updated to %s", request.NewStatus),
		Updated:  true,
	}, nil
}

// GetElectionResults obtém os resultados detalhados de uma eleição
func (uc *ManageElectionUseCase) GetElectionResults(ctx context.Context, request *GetElectionResultsRequest) (*GetElectionResultsResponse, error) {
	if request == nil || request.ElectionID.IsEmpty() {
		return nil, fmt.Errorf("invalid request: election ID is required")
	}

	// Obter eleição
	election, err := uc.electionRepo.GetElection(ctx, request.ElectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get election: %w", err)
	}

	// Obter resultados do repositório
	results, err := uc.electionRepo.GetElectionResults(ctx, request.ElectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get election results: %w", err)
	}

	// Contar total de votos
	totalVotes, err := uc.voteRepo.CountVotesByElection(ctx, request.ElectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to count votes: %w", err)
	}

	// Atualizar contadores dos candidatos com dados dos votos
	candidates := election.GetCandidates()
	for i, candidate := range candidates {
		if count, exists := results[candidate.ID]; exists {
			candidates[i].VoteCount = count
		}
	}

	return &GetElectionResultsResponse{
		ElectionID:   request.ElectionID,
		Results:      results,
		TotalVotes:   totalVotes,
		Candidates:   candidates,
		ElectionInfo: election,
		Message:      fmt.Sprintf("Results for election '%s'", election.GetTitle()),
	}, nil
}

// validateUpdateStatusRequest valida requisição de atualização de status
func (uc *ManageElectionUseCase) validateUpdateStatusRequest(request *UpdateElectionStatusRequest) error {
	if request == nil {
		return fmt.Errorf("request is nil")
	}

	if request.ElectionID.IsEmpty() {
		return fmt.Errorf("election ID is required")
	}

	if request.UpdatedBy.IsEmpty() {
		return fmt.Errorf("updater ID is required")
	}

	validStatuses := []entities.ElectionStatus{
		entities.ElectionPending,
		entities.ElectionActive,
		entities.ElectionClosed,
		entities.ElectionCancelled,
	}

	for _, status := range validStatuses {
		if request.NewStatus == status {
			return nil
		}
	}

	return fmt.Errorf("invalid election status: %s", request.NewStatus)
}

// validateStatusTransition valida se a transição de status é permitida
func (uc *ManageElectionUseCase) validateStatusTransition(currentStatus, newStatus entities.ElectionStatus) error {
	// Definir transições válidas
	validTransitions := map[entities.ElectionStatus][]entities.ElectionStatus{
		entities.ElectionPending: {
			entities.ElectionActive,
			entities.ElectionCancelled,
		},
		entities.ElectionActive: {
			entities.ElectionClosed,
			entities.ElectionCancelled,
		},
		entities.ElectionClosed: {
			// Eleições fechadas não podem mudar de status
		},
		entities.ElectionCancelled: {
			// Eleições canceladas não podem mudar de status
		},
	}

	allowedTransitions, exists := validTransitions[currentStatus]
	if !exists {
		return fmt.Errorf("unknown current status: %s", currentStatus)
	}

	for _, allowedStatus := range allowedTransitions {
		if newStatus == allowedStatus {
			return nil
		}
	}

	return fmt.Errorf("cannot transition from %s to %s", currentStatus, newStatus)
}
