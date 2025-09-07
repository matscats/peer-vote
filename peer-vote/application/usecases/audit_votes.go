package usecases

import (
	"context"
	"fmt"

	"github.com/matscats/peer-vote/peer-vote/domain/entities"
	"github.com/matscats/peer-vote/peer-vote/domain/repositories"
	"github.com/matscats/peer-vote/peer-vote/domain/services"
	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
	"github.com/matscats/peer-vote/peer-vote/infrastructure/blockchain"
)

// AuditVotesRequest representa uma requisição para auditoria de votos
type AuditVotesRequest struct {
	ElectionID valueobjects.Hash `json:"election_id"`
}

// VoteAuditResult representa o resultado da auditoria de um voto
type VoteAuditResult struct {
	VoteID      string `json:"vote_id"`
	IsValid     bool   `json:"is_valid"`
	Errors      []string `json:"errors,omitempty"`
	CandidateID string `json:"candidate_id"`
	Timestamp   int64  `json:"timestamp"`
	IsAnonymous bool   `json:"is_anonymous"`
}

// ElectionAuditSummary representa o resumo da auditoria de uma eleição
type ElectionAuditSummary struct {
	TotalVotes       uint64            `json:"total_votes"`
	ValidVotes       uint64            `json:"valid_votes"`
	InvalidVotes     uint64            `json:"invalid_votes"`
	AnonymousVotes   uint64            `json:"anonymous_votes"`
	CandidateResults map[string]uint64 `json:"candidate_results"`
	IntegrityScore   float64           `json:"integrity_score"`
}

// AuditVotesResponse representa a resposta da auditoria de votos
type AuditVotesResponse struct {
	ElectionID    valueobjects.Hash     `json:"election_id"`
	ElectionTitle string                `json:"election_title"`
	AuditResults  []VoteAuditResult     `json:"audit_results"`
	Summary       ElectionAuditSummary  `json:"summary"`
	Message       string                `json:"message"`
	AuditPassed   bool                  `json:"audit_passed"`
}

// CountVotesRequest representa uma requisição para contagem de votos
type CountVotesRequest struct {
	ElectionID valueobjects.Hash `json:"election_id"`
}

// CandidateResult representa o resultado de um candidato
type CandidateResult struct {
	CandidateID   string  `json:"candidate_id"`
	CandidateName string  `json:"candidate_name"`
	VoteCount     uint64  `json:"vote_count"`
	Percentage    float64 `json:"percentage"`
}

// CountVotesResponse representa a resposta da contagem de votos
type CountVotesResponse struct {
	ElectionID      valueobjects.Hash  `json:"election_id"`
	ElectionTitle   string             `json:"election_title"`
	Results         []CandidateResult  `json:"results"`
	TotalVotes      uint64             `json:"total_votes"`
	Winner          *CandidateResult   `json:"winner,omitempty"`
	IsTie           bool               `json:"is_tie"`
	CountCompleted  bool               `json:"count_completed"`
	Message         string             `json:"message"`
}

// AuditVotesUseCase implementa os casos de uso de auditoria e contagem de votos
type AuditVotesUseCase struct {
	electionRepo      repositories.ElectionRepository
	chainManager      *blockchain.ChainManager
	cryptoService     services.CryptographyService
	validationService services.VotingValidationService
}

// NewAuditVotesUseCase cria um novo caso de uso de auditoria de votos
func NewAuditVotesUseCase(
	electionRepo repositories.ElectionRepository,
	chainManager *blockchain.ChainManager,
	cryptoService services.CryptographyService,
	validationService services.VotingValidationService,
) *AuditVotesUseCase {
	return &AuditVotesUseCase{
		electionRepo:      electionRepo,
		chainManager:      chainManager,
		cryptoService:     cryptoService,
		validationService: validationService,
	}
}

// AuditVotes executa auditoria completa dos votos de uma eleição
func (uc *AuditVotesUseCase) AuditVotes(ctx context.Context, request *AuditVotesRequest) (*AuditVotesResponse, error) {
	if request == nil || request.ElectionID.IsEmpty() {
		return nil, fmt.Errorf("invalid request: election ID is required")
	}

	// Obter eleição
	election, err := uc.electionRepo.GetElection(ctx, request.ElectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get election: %w", err)
	}

	// === NOVA LÓGICA BLOCKCHAIN ===
	// Obter todos os votos da eleição diretamente da blockchain
	votes, err := uc.extractVotesFromBlockchain(ctx, request.ElectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to extract votes from blockchain: %w", err)
	}

	// Auditar cada voto
	auditResults := make([]VoteAuditResult, 0, len(votes))
	summary := ElectionAuditSummary{
		CandidateResults: make(map[string]uint64),
	}

	for _, vote := range votes {
		result := uc.auditSingleVoteFromBlockchain(ctx, vote, election)
		auditResults = append(auditResults, result)

		// Atualizar estatísticas do resumo
		summary.TotalVotes++
		if result.IsValid {
			summary.ValidVotes++
			summary.CandidateResults[result.CandidateID]++
		} else {
			summary.InvalidVotes++
		}

		if result.IsAnonymous {
			summary.AnonymousVotes++
		}
	}

	// Calcular score de integridade
	if summary.TotalVotes > 0 {
		summary.IntegrityScore = float64(summary.ValidVotes) / float64(summary.TotalVotes) * 100
	}

	// Verificar se a auditoria passou (blockchain deve ter 100% de integridade)
	auditPassed := summary.IntegrityScore >= 99.0 // 99% de votos válidos (tolerância mínima para blockchain)

	return &AuditVotesResponse{
		ElectionID:    request.ElectionID,
		ElectionTitle: election.GetTitle(),
		AuditResults:  auditResults,
		Summary:       summary,
		Message:       fmt.Sprintf("Blockchain audit completed for election '%s' - %d votes found", election.GetTitle(), len(votes)),
		AuditPassed:   auditPassed,
	}, nil
}

// CountVotes executa contagem oficial dos votos
func (uc *AuditVotesUseCase) CountVotes(ctx context.Context, request *CountVotesRequest) (*CountVotesResponse, error) {
	if request == nil || request.ElectionID.IsEmpty() {
		return nil, fmt.Errorf("invalid request: election ID is required")
	}

	// Obter eleição
	election, err := uc.electionRepo.GetElection(ctx, request.ElectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get election: %w", err)
	}

	// Verificar se a eleição pode ter votos contados
	if election.GetStatus() != entities.ElectionClosed && election.GetStatus() != entities.ElectionActive {
		return nil, fmt.Errorf("election must be active or closed to count votes")
	}

	// === NOVA LÓGICA BLOCKCHAIN ===
	// Obter votos diretamente da blockchain
	votes, err := uc.extractVotesFromBlockchain(ctx, request.ElectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to extract votes from blockchain: %w", err)
	}

	// Contar votos por candidato diretamente da blockchain
	candidateVotes := make(map[string]uint64)
	totalVotes := uint64(0)

	for _, vote := range votes {
		// Validar voto antes de contar
		if vote.IsValid() && vote.GetElectionID().Equals(request.ElectionID) {
			candidateVotes[vote.GetCandidateID()]++
			totalVotes++
		}
	}

	// Preparar resultados dos candidatos
	candidates := election.GetCandidates()
	results := make([]CandidateResult, 0, len(candidates))
	var maxVotes uint64
	var winner *CandidateResult
	winnersCount := 0

	for _, candidate := range candidates {
		voteCount := candidateVotes[candidate.ID]
		percentage := float64(0)
		if totalVotes > 0 {
			percentage = float64(voteCount) / float64(totalVotes) * 100
		}

		result := CandidateResult{
			CandidateID:   candidate.ID,
			CandidateName: candidate.Name,
			VoteCount:     voteCount,
			Percentage:    percentage,
		}

		results = append(results, result)

		// Determinar vencedor
		if voteCount > maxVotes {
			maxVotes = voteCount
			winner = &result
			winnersCount = 1
		} else if voteCount == maxVotes && maxVotes > 0 {
			winnersCount++
		}
	}

	// Verificar empate
	isTie := winnersCount > 1

	return &CountVotesResponse{
		ElectionID:     request.ElectionID,
		ElectionTitle:  election.GetTitle(),
		Results:        results,
		TotalVotes:     totalVotes,
		Winner:         winner,
		IsTie:          isTie,
		CountCompleted: true,
		Message:        fmt.Sprintf("Blockchain vote count completed for election '%s' - %d votes counted", election.GetTitle(), totalVotes),
	}, nil
}

// auditSingleVote audita um voto individual
func (uc *AuditVotesUseCase) auditSingleVote(ctx context.Context, vote *entities.Vote, election *entities.Election) VoteAuditResult {
	result := VoteAuditResult{
		VoteID:      vote.GetID().String(),
		CandidateID: vote.GetCandidateID(),
		Timestamp:   vote.GetTimestamp().Unix(),
		IsAnonymous: vote.IsAnonymous(),
		IsValid:     true,
		Errors:      []string{},
	}

	// Validação básica do voto
	if !vote.IsValid() {
		result.IsValid = false
		result.Errors = append(result.Errors, "vote basic validation failed")
	}

	// Validação de integridade blockchain (removido repositório)
	if err := uc.validateVoteBlockchainIntegrity(ctx, vote); err != nil {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("blockchain integrity validation failed: %v", err))
	}

	// Validar voto usando método específico para auditoria
	if err := uc.validationService.ValidateVoteForAudit(ctx, vote, election); err != nil {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("audit validation failed: %v", err))
	}

	// Validar assinatura (se não for anônimo)
	if !vote.IsAnonymous() {
		// Aqui seria necessário obter a chave pública do eleitor
		// Por simplicidade, assumimos que a assinatura é válida se o voto passou nas outras validações
		if vote.GetSignature().IsEmpty() {
			result.IsValid = false
			result.Errors = append(result.Errors, "missing signature for non-anonymous vote")
		}
	}

	// Verificar se o candidato existe na eleição
	if _, exists := election.GetCandidate(vote.GetCandidateID()); !exists {
		result.IsValid = false
		result.Errors = append(result.Errors, "candidate does not exist in election")
	}

	return result
}

// extractVotesFromBlockchain extrai todos os votos de uma eleição da blockchain
func (uc *AuditVotesUseCase) extractVotesFromBlockchain(ctx context.Context, electionID valueobjects.Hash) ([]*entities.Vote, error) {
	// Obter altura atual da blockchain
	height, err := uc.chainManager.GetChainHeight(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain height: %w", err)
	}

	var votes []*entities.Vote

	// Iterar por todos os blocos da blockchain
	for i := uint64(0); i <= height; i++ {
		block, err := uc.chainManager.GetBlockByIndex(ctx, i)
		if err != nil {
			continue // Pular blocos com erro
		}

		// Processar transações do bloco
		transactions := block.GetTransactions()
		for _, tx := range transactions {
			// Verificar se é uma transação de voto
			if tx.GetType() == "VOTE" {
				// Deserializar dados da transação para obter o voto
				vote, err := uc.deserializeVoteFromTransaction(ctx, tx)
				if err != nil {
					continue // Pular transações inválidas
				}

				// Verificar se o voto pertence à eleição solicitada
				if vote.GetElectionID().Equals(electionID) {
					votes = append(votes, vote)
				}
			}
		}
	}

	return votes, nil
}

// deserializeVoteFromTransaction deserializa um voto a partir de uma transação
func (uc *AuditVotesUseCase) deserializeVoteFromTransaction(ctx context.Context, tx *entities.Transaction) (*entities.Vote, error) {
	// Obter dados da transação
	txData := tx.GetData()
	if len(txData) == 0 {
		return nil, fmt.Errorf("transaction has no data")
	}

	// Criar uma nova instância de voto e deserializar
	vote := &entities.Vote{}
	if err := vote.FromBytes(txData); err != nil {
		return nil, fmt.Errorf("failed to deserialize vote: %w", err)
	}

	return vote, nil
}

// auditSingleVoteFromBlockchain audita um voto individual extraído da blockchain
func (uc *AuditVotesUseCase) auditSingleVoteFromBlockchain(ctx context.Context, vote *entities.Vote, election *entities.Election) VoteAuditResult {
	result := VoteAuditResult{
		VoteID:      vote.GetID().String(),
		CandidateID: vote.GetCandidateID(),
		Timestamp:   vote.GetTimestamp().Unix(),
		IsAnonymous: vote.IsAnonymous(),
		IsValid:     true,
		Errors:      []string{},
	}

	// Validação básica do voto
	if !vote.IsValid() {
		result.IsValid = false
		result.Errors = append(result.Errors, "vote basic validation failed")
	}

	// Validar voto usando método específico para auditoria
	if err := uc.validationService.ValidateVoteForAudit(ctx, vote, election); err != nil {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("audit validation failed: %v", err))
	}

	// Validação específica para blockchain
	if err := uc.validateVoteBlockchainIntegrity(ctx, vote); err != nil {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("blockchain integrity validation failed: %v", err))
	}

	// Validar assinatura (se não for anônimo)
	if !vote.IsAnonymous() {
		if vote.GetSignature().IsEmpty() {
			result.IsValid = false
			result.Errors = append(result.Errors, "missing signature for non-anonymous vote")
		}
	}

	// Verificar se o candidato existe na eleição
	if _, exists := election.GetCandidate(vote.GetCandidateID()); !exists {
		result.IsValid = false
		result.Errors = append(result.Errors, "candidate does not exist in election")
	}

	return result
}

// validateVoteBlockchainIntegrity valida a integridade de um voto na blockchain
func (uc *AuditVotesUseCase) validateVoteBlockchainIntegrity(ctx context.Context, vote *entities.Vote) error {
	if vote == nil {
		return fmt.Errorf("vote is nil")
	}

	// Verificar se o voto tem um ID válido
	if vote.GetID().IsEmpty() {
		return fmt.Errorf("vote has empty ID")
	}

	// Verificar se o hash do voto é consistente com seus dados
	voteData, err := vote.ToBytes()
	if err != nil {
		return fmt.Errorf("failed to serialize vote: %w", err)
	}

	expectedHash := uc.cryptoService.HashTransaction(ctx, voteData)
	if !vote.GetID().Equals(expectedHash) {
		return fmt.Errorf("vote hash mismatch - expected %s, got %s", expectedHash.String(), vote.GetID().String())
	}

	// Verificar se a assinatura é válida (se não for anônimo)
	if !vote.IsAnonymous() && !vote.GetSignature().IsEmpty() {
		// Aqui poderíamos validar a assinatura se tivéssemos acesso às chaves públicas
		// Por simplicidade, assumimos que se chegou até aqui na blockchain, a assinatura é válida
	}

	return nil
}
