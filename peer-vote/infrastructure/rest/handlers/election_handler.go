package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/matscats/peer-vote/peer-vote/application/usecases"
	"github.com/matscats/peer-vote/peer-vote/domain/entities"
	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
)

// ElectionHandler gerencia endpoints relacionados a eleições
type ElectionHandler struct {
	createElectionUseCase *usecases.CreateElectionUseCase
	manageElectionUseCase *usecases.ManageElectionUseCase
}

// NewElectionHandler cria um novo handler de eleições
func NewElectionHandler(
	createElectionUseCase *usecases.CreateElectionUseCase,
	manageElectionUseCase *usecases.ManageElectionUseCase,
) *ElectionHandler {
	return &ElectionHandler{
		createElectionUseCase: createElectionUseCase,
		manageElectionUseCase: manageElectionUseCase,
	}
}

// CreateElectionRequest representa o payload para criar eleição
type CreateElectionRequest struct {
	Title            string                `json:"title"`
	Description      string                `json:"description"`
	Candidates       []entities.Candidate  `json:"candidates"`
	StartTime        string                `json:"start_time"` // RFC3339 format
	EndTime          string                `json:"end_time"`   // RFC3339 format
	CreatedBy        string                `json:"created_by"`
	AllowAnonymous   bool                  `json:"allow_anonymous"`
	MaxVotesPerVoter int                   `json:"max_votes_per_voter"`
}

// UpdateElectionStatusRequest representa o payload para atualizar status
type UpdateElectionStatusRequest struct {
	NewStatus string `json:"new_status"`
	UpdatedBy string `json:"updated_by"`
}

// RegisterRoutes registra as rotas do handler
func (h *ElectionHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/elections", h.CreateElection).Methods("POST")
	router.HandleFunc("/elections", h.ListElections).Methods("GET")
	router.HandleFunc("/elections/{id}", h.GetElection).Methods("GET")
	router.HandleFunc("/elections/{id}/status", h.UpdateElectionStatus).Methods("PUT")
	router.HandleFunc("/elections/{id}/results", h.GetElectionResults).Methods("GET")
}

// CreateElection cria uma nova eleição
func (h *ElectionHandler) CreateElection(w http.ResponseWriter, r *http.Request) {
	var req CreateElectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Validar e converter timestamps
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		http.Error(w, "Invalid start_time format (use RFC3339)", http.StatusBadRequest)
		return
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		http.Error(w, "Invalid end_time format (use RFC3339)", http.StatusBadRequest)
		return
	}

	// Converter CreatedBy para NodeID
	createdBy := valueobjects.NewNodeID(req.CreatedBy)

	// Criar request do caso de uso
	createRequest := &usecases.CreateElectionRequest{
		Title:            req.Title,
		Description:      req.Description,
		Candidates:       req.Candidates,
		StartTime:        startTime,
		EndTime:          endTime,
		CreatedBy:        createdBy,
		AllowAnonymous:   req.AllowAnonymous,
		MaxVotesPerVoter: req.MaxVotesPerVoter,
	}

	// Executar caso de uso
	response, err := h.createElectionUseCase.Execute(r.Context(), createRequest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Retornar resposta
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// ListElections lista eleições com filtros opcionais
func (h *ElectionHandler) ListElections(w http.ResponseWriter, r *http.Request) {
	// Extrair parâmetros de query
	activeOnly := r.URL.Query().Get("active") == "true"
	createdBy := r.URL.Query().Get("created_by")

	// Criar request do caso de uso
	listRequest := &usecases.ListElectionsRequest{
		ActiveOnly: activeOnly,
	}

	if createdBy != "" {
		listRequest.CreatedBy = valueobjects.NewNodeID(createdBy)
	}

	// Executar caso de uso
	response, err := h.manageElectionUseCase.ListElections(r.Context(), listRequest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Retornar resposta
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetElection obtém uma eleição específica
func (h *ElectionHandler) GetElection(w http.ResponseWriter, r *http.Request) {
	// Extrair ID da URL
	vars := mux.Vars(r)
	electionIDStr := vars["id"]

	// Converter para Hash
	electionID, err := valueobjects.NewHashFromString(electionIDStr)
	if err != nil {
		http.Error(w, "Invalid election ID format", http.StatusBadRequest)
		return
	}

	// Criar request do caso de uso
	getRequest := &usecases.GetElectionRequest{
		ElectionID: electionID,
	}

	// Executar caso de uso
	response, err := h.manageElectionUseCase.GetElection(r.Context(), getRequest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Retornar resposta
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UpdateElectionStatus atualiza o status de uma eleição
func (h *ElectionHandler) UpdateElectionStatus(w http.ResponseWriter, r *http.Request) {
	// Extrair ID da URL
	vars := mux.Vars(r)
	electionIDStr := vars["id"]

	// Converter para Hash
	electionID, err := valueobjects.NewHashFromString(electionIDStr)
	if err != nil {
		http.Error(w, "Invalid election ID format", http.StatusBadRequest)
		return
	}

	// Decodificar payload
	var req UpdateElectionStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Validar status
	var newStatus entities.ElectionStatus
	switch req.NewStatus {
	case "PENDING":
		newStatus = entities.ElectionPending
	case "ACTIVE":
		newStatus = entities.ElectionActive
	case "CLOSED":
		newStatus = entities.ElectionClosed
	case "CANCELLED":
		newStatus = entities.ElectionCancelled
	default:
		http.Error(w, "Invalid election status", http.StatusBadRequest)
		return
	}

	// Criar request do caso de uso
	updateRequest := &usecases.UpdateElectionStatusRequest{
		ElectionID: electionID,
		NewStatus:  newStatus,
		UpdatedBy:  valueobjects.NewNodeID(req.UpdatedBy),
	}

	// Executar caso de uso
	response, err := h.manageElectionUseCase.UpdateElectionStatus(r.Context(), updateRequest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Retornar resposta
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetElectionResults obtém os resultados de uma eleição
func (h *ElectionHandler) GetElectionResults(w http.ResponseWriter, r *http.Request) {
	// Extrair ID da URL
	vars := mux.Vars(r)
	electionIDStr := vars["id"]

	// Converter para Hash
	electionID, err := valueobjects.NewHashFromString(electionIDStr)
	if err != nil {
		http.Error(w, "Invalid election ID format", http.StatusBadRequest)
		return
	}

	// Criar request do caso de uso
	resultsRequest := &usecases.GetElectionResultsRequest{
		ElectionID: electionID,
	}

	// Executar caso de uso
	response, err := h.manageElectionUseCase.GetElectionResults(r.Context(), resultsRequest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Retornar resposta
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
