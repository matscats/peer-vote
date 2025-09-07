package entities

import (
	"encoding/json"
	"time"

	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
)

// ElectionStatus representa o status de uma eleição
type ElectionStatus string

const (
	// ElectionPending eleição criada mas não iniciada
	ElectionPending ElectionStatus = "PENDING"
	// ElectionActive eleição em andamento
	ElectionActive ElectionStatus = "ACTIVE"
	// ElectionClosed eleição encerrada
	ElectionClosed ElectionStatus = "CLOSED"
	// ElectionCancelled eleição cancelada
	ElectionCancelled ElectionStatus = "CANCELLED"
)

// Election representa uma eleição
type Election struct {
	id          valueobjects.Hash
	title       string
	description string
	candidates  []Candidate
	startTime   valueobjects.Timestamp
	endTime     valueobjects.Timestamp
	status      ElectionStatus
	createdBy   valueobjects.NodeID
	createdAt   valueobjects.Timestamp
	allowAnonymous bool
	maxVotesPerVoter int
}

// Candidate representa um candidato em uma eleição
type Candidate struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	VoteCount   uint64 `json:"vote_count"`
}

// ElectionData representa os dados serializáveis de uma eleição
type ElectionData struct {
	ID               string      `json:"id"`
	Title            string      `json:"title"`
	Description      string      `json:"description"`
	Candidates       []Candidate `json:"candidates"`
	StartTime        int64       `json:"start_time"`
	EndTime          int64       `json:"end_time"`
	Status           string      `json:"status"`
	CreatedBy        string      `json:"created_by"`
	CreatedAt        int64       `json:"created_at"`
	AllowAnonymous   bool        `json:"allow_anonymous"`
	MaxVotesPerVoter int         `json:"max_votes_per_voter"`
}

// NewElection cria uma nova eleição
func NewElection(title, description string, candidates []Candidate, startTime, endTime time.Time, createdBy valueobjects.NodeID) *Election {
	return &Election{
		title:            title,
		description:      description,
		candidates:       candidates,
		startTime:        valueobjects.NewTimestamp(startTime),
		endTime:          valueobjects.NewTimestamp(endTime),
		status:           ElectionPending,
		createdBy:        createdBy,
		createdAt:        valueobjects.Now(),
		allowAnonymous:   false,
		maxVotesPerVoter: 1,
	}
}

// GetID retorna o ID da eleição
func (e *Election) GetID() valueobjects.Hash {
	return e.id
}

// GetTitle retorna o título da eleição
func (e *Election) GetTitle() string {
	return e.title
}

// GetDescription retorna a descrição da eleição
func (e *Election) GetDescription() string {
	return e.description
}

// GetCandidates retorna os candidatos da eleição
func (e *Election) GetCandidates() []Candidate {
	return e.candidates
}

// GetStartTime retorna o horário de início da eleição
func (e *Election) GetStartTime() valueobjects.Timestamp {
	return e.startTime
}

// GetEndTime retorna o horário de fim da eleição
func (e *Election) GetEndTime() valueobjects.Timestamp {
	return e.endTime
}

// GetStatus retorna o status da eleição
func (e *Election) GetStatus() ElectionStatus {
	return e.status
}

// GetCreatedBy retorna quem criou a eleição
func (e *Election) GetCreatedBy() valueobjects.NodeID {
	return e.createdBy
}

// GetCreatedAt retorna quando a eleição foi criada
func (e *Election) GetCreatedAt() valueobjects.Timestamp {
	return e.createdAt
}

// AllowsAnonymousVoting verifica se permite votação anônima
func (e *Election) AllowsAnonymousVoting() bool {
	return e.allowAnonymous
}

// GetMaxVotesPerVoter retorna o máximo de votos por eleitor
func (e *Election) GetMaxVotesPerVoter() int {
	return e.maxVotesPerVoter
}

// SetID define o ID da eleição
func (e *Election) SetID(id valueobjects.Hash) {
	e.id = id
}

// SetStatus define o status da eleição
func (e *Election) SetStatus(status ElectionStatus) {
	e.status = status
}

// SetAllowAnonymous define se permite votação anônima
func (e *Election) SetAllowAnonymous(allow bool) {
	e.allowAnonymous = allow
}

// SetMaxVotesPerVoter define o máximo de votos por eleitor
func (e *Election) SetMaxVotesPerVoter(max int) {
	if max > 0 {
		e.maxVotesPerVoter = max
	}
}

// AddCandidate adiciona um candidato à eleição
func (e *Election) AddCandidate(candidate Candidate) {
	e.candidates = append(e.candidates, candidate)
}

// GetCandidate retorna um candidato pelo ID
func (e *Election) GetCandidate(candidateID string) (*Candidate, bool) {
	for i, candidate := range e.candidates {
		if candidate.ID == candidateID {
			return &e.candidates[i], true
		}
	}
	return nil, false
}

// IncrementVoteCount incrementa o contador de votos de um candidato
func (e *Election) IncrementVoteCount(candidateID string) bool {
	for i, candidate := range e.candidates {
		if candidate.ID == candidateID {
			e.candidates[i].VoteCount++
			return true
		}
	}
	return false
}

// IsActive verifica se a eleição está ativa
func (e *Election) IsActive() bool {
	now := valueobjects.Now()
	
	// Uma eleição é ativa se está no período correto E não foi cancelada
	isInTimePeriod := now.After(e.startTime) && now.Before(e.endTime)
	isNotCancelled := e.status != ElectionCancelled
	
	return isInTimePeriod && isNotCancelled
}

// CanVote verifica se é possível votar nesta eleição
func (e *Election) CanVote() bool {
	return e.IsActive()
}

// IsValid verifica se a eleição é válida
func (e *Election) IsValid() bool {
	// Validações básicas
	if e.title == "" {
		return false
	}

	if len(e.candidates) < 2 {
		return false
	}

	if e.endTime.Before(e.startTime) {
		return false
	}

	if e.createdBy.IsEmpty() {
		return false
	}

	if e.maxVotesPerVoter <= 0 {
		return false
	}

	// Verifica se todos os candidatos têm IDs únicos
	candidateIDs := make(map[string]bool)
	for _, candidate := range e.candidates {
		if candidate.ID == "" || candidate.Name == "" {
			return false
		}
		if candidateIDs[candidate.ID] {
			return false // ID duplicado
		}
		candidateIDs[candidate.ID] = true
	}

	return true
}

// ToBytes serializa a eleição para bytes
func (e *Election) ToBytes() ([]byte, error) {
	data := ElectionData{
		ID:               e.id.String(),
		Title:            e.title,
		Description:      e.description,
		Candidates:       e.candidates,
		StartTime:        e.startTime.Unix(),
		EndTime:          e.endTime.Unix(),
		Status:           string(e.status),
		CreatedBy:        e.createdBy.String(),
		CreatedAt:        e.createdAt.Unix(),
		AllowAnonymous:   e.allowAnonymous,
		MaxVotesPerVoter: e.maxVotesPerVoter,
	}

	return json.Marshal(data)
}

// FromBytes deserializa uma eleição de bytes
func (e *Election) FromBytes(data []byte) error {
	var electionData ElectionData
	if err := json.Unmarshal(data, &electionData); err != nil {
		return err
	}

	id, err := valueobjects.NewHashFromString(electionData.ID)
	if err != nil {
		return err
	}

	e.id = id
	e.title = electionData.Title
	e.description = electionData.Description
	e.candidates = electionData.Candidates
	e.startTime = valueobjects.Unix(electionData.StartTime, 0)
	e.endTime = valueobjects.Unix(electionData.EndTime, 0)
	e.status = ElectionStatus(electionData.Status)
	e.createdBy = valueobjects.NewNodeID(electionData.CreatedBy)
	e.createdAt = valueobjects.Unix(electionData.CreatedAt, 0)
	e.allowAnonymous = electionData.AllowAnonymous
	e.maxVotesPerVoter = electionData.MaxVotesPerVoter

	return nil
}

// GetResults retorna os resultados da eleição
func (e *Election) GetResults() map[string]uint64 {
	results := make(map[string]uint64)
	for _, candidate := range e.candidates {
		results[candidate.ID] = candidate.VoteCount
	}
	return results
}

// GetTotalVotes retorna o total de votos na eleição
func (e *Election) GetTotalVotes() uint64 {
	var total uint64
	for _, candidate := range e.candidates {
		total += candidate.VoteCount
	}
	return total
}
