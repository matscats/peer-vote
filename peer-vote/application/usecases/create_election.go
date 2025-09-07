package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/matscats/peer-vote/peer-vote/domain/entities"
	"github.com/matscats/peer-vote/peer-vote/domain/repositories"
	"github.com/matscats/peer-vote/peer-vote/domain/services"
	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
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
}

// CreateElectionResponse representa a resposta da criação de eleição
type CreateElectionResponse struct {
	Election *entities.Election `json:"election"`
	Message  string             `json:"message"`
}

// CreateElectionUseCase implementa o caso de uso de criação de eleições
type CreateElectionUseCase struct {
	electionRepo      repositories.ElectionRepository
	cryptoService     services.CryptographyService
	validationService services.VotingValidationService
}

// NewCreateElectionUseCase cria um novo caso de uso de criação de eleições
func NewCreateElectionUseCase(
	electionRepo repositories.ElectionRepository,
	cryptoService services.CryptographyService,
	validationService services.VotingValidationService,
) *CreateElectionUseCase {
	return &CreateElectionUseCase{
		electionRepo:      electionRepo,
		cryptoService:     cryptoService,
		validationService: validationService,
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

	// Persistir eleição
	if err := uc.electionRepo.CreateElection(ctx, election); err != nil {
		return nil, fmt.Errorf("failed to create election: %w", err)
	}

	return &CreateElectionResponse{
		Election: election,
		Message:  fmt.Sprintf("Election '%s' created successfully", election.GetTitle()),
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

	if request.StartTime.Before(time.Now()) {
		return fmt.Errorf("start time must be in the future")
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
