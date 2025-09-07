package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/matscats/peer-vote/peer-vote/application/usecases"
	"github.com/matscats/peer-vote/peer-vote/domain/services"
	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
	"github.com/matscats/peer-vote/peer-vote/infrastructure/crypto"
)

// VoteHandler gerencia endpoints relacionados a votos
type VoteHandler struct {
	submitVoteUseCase *usecases.SubmitVoteUseCase
	auditVotesUseCase *usecases.AuditVotesUseCase
	cryptoService     services.CryptographyService
}

// NewVoteHandler cria um novo handler de votos
func NewVoteHandler(
	submitVoteUseCase *usecases.SubmitVoteUseCase,
	auditVotesUseCase *usecases.AuditVotesUseCase,
	cryptoService services.CryptographyService,
) *VoteHandler {
	return &VoteHandler{
		submitVoteUseCase: submitVoteUseCase,
		auditVotesUseCase: auditVotesUseCase,
		cryptoService:     cryptoService,
	}
}

// SubmitVoteRequest representa o payload para submeter voto
type SubmitVoteRequest struct {
	ElectionID  string `json:"election_id"`
	VoterID     string `json:"voter_id"`
	CandidateID string `json:"candidate_id"`
	IsAnonymous bool   `json:"is_anonymous"`
	PrivateKey  string `json:"private_key"` // Base64 encoded
}

// RegisterRoutes registra as rotas do handler
func (h *VoteHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/votes", h.SubmitVote).Methods("POST")
	router.HandleFunc("/votes/audit/{election_id}", h.AuditVotes).Methods("GET")
	router.HandleFunc("/votes/count/{election_id}", h.CountVotes).Methods("GET")
}

// SubmitVote submete um novo voto
func (h *VoteHandler) SubmitVote(w http.ResponseWriter, r *http.Request) {
	var req SubmitVoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Converter ElectionID para Hash
	electionID, err := valueobjects.NewHashFromString(req.ElectionID)
	if err != nil {
		http.Error(w, "Invalid election ID format", http.StatusBadRequest)
		return
	}

	// Converter VoterID para NodeID
	voterID := valueobjects.NewNodeID(req.VoterID)

	// Converter PrivateKey de string para *services.PrivateKey
	var privateKey *services.PrivateKey
	if req.PrivateKey != "" {
		// Usar o ECDSAService para fazer o parsing
		if ecdsaService, ok := h.cryptoService.(*crypto.ECDSAService); ok {
			var err error
			privateKey, err = ecdsaService.ParsePrivateKeyFromString(req.PrivateKey)
			if err != nil {
				http.Error(w, "Invalid private key format", http.StatusBadRequest)
				return
			}
		} else {
			http.Error(w, "Crypto service does not support key parsing", http.StatusInternalServerError)
			return
		}
	} else {
		// Gerar chave privada se não fornecida
		keyPair, err := h.cryptoService.GenerateKeyPair(r.Context())
		if err != nil {
			http.Error(w, "Failed to generate key pair", http.StatusInternalServerError)
			return
		}
		privateKey = keyPair.PrivateKey
	}

	// Criar request do caso de uso
	submitRequest := &usecases.SubmitVoteRequest{
		ElectionID:  electionID,
		VoterID:     voterID,
		CandidateID: req.CandidateID,
		IsAnonymous: req.IsAnonymous,
		PrivateKey:  privateKey,
	}

	// Executar caso de uso
	response, err := h.submitVoteUseCase.Execute(r.Context(), submitRequest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Retornar resposta
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// AuditVotes executa auditoria de votos de uma eleição
func (h *VoteHandler) AuditVotes(w http.ResponseWriter, r *http.Request) {
	// Extrair ID da URL
	vars := mux.Vars(r)
	electionIDStr := vars["election_id"]

	// Converter para Hash
	electionID, err := valueobjects.NewHashFromString(electionIDStr)
	if err != nil {
		http.Error(w, "Invalid election ID format", http.StatusBadRequest)
		return
	}

	// Criar request do caso de uso
	auditRequest := &usecases.AuditVotesRequest{
		ElectionID: electionID,
	}

	// Executar caso de uso
	response, err := h.auditVotesUseCase.AuditVotes(r.Context(), auditRequest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Retornar resposta
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CountVotes executa contagem de votos de uma eleição
func (h *VoteHandler) CountVotes(w http.ResponseWriter, r *http.Request) {
	// Extrair ID da URL
	vars := mux.Vars(r)
	electionIDStr := vars["election_id"]

	// Converter para Hash
	electionID, err := valueobjects.NewHashFromString(electionIDStr)
	if err != nil {
		http.Error(w, "Invalid election ID format", http.StatusBadRequest)
		return
	}

	// Criar request do caso de uso
	countRequest := &usecases.CountVotesRequest{
		ElectionID: electionID,
	}

	// Executar caso de uso
	response, err := h.auditVotesUseCase.CountVotes(r.Context(), countRequest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Retornar resposta
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
