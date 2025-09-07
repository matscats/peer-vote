package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/matscats/peer-vote/peer-vote/domain/repositories"
	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
	"github.com/matscats/peer-vote/peer-vote/infrastructure/blockchain"
)

// BlockchainHandler gerencia endpoints relacionados à blockchain
type BlockchainHandler struct {
	blockchainRepo repositories.BlockchainRepository
	chainManager   *blockchain.ChainManager
}

// NewBlockchainHandler cria um novo handler de blockchain
func NewBlockchainHandler(blockchainRepo repositories.BlockchainRepository, chainManager *blockchain.ChainManager) *BlockchainHandler {
	return &BlockchainHandler{
		blockchainRepo: blockchainRepo,
		chainManager:   chainManager,
	}
}

// BlockResponse representa a resposta de um bloco
type BlockResponse struct {
	Index        uint64 `json:"index"`
	Hash         string `json:"hash"`
	PreviousHash string `json:"previous_hash"`
	Timestamp    int64  `json:"timestamp"`
	MerkleRoot   string `json:"merkle_root"`
	Signature    string `json:"signature"`
	Validator    string `json:"validator"`
	Transactions int    `json:"transaction_count"`
}

// ChainStatusResponse representa o status da blockchain
type ChainStatusResponse struct {
	Height      uint64 `json:"height"`
	LatestBlock string `json:"latest_block_hash"`
	IsValid     bool   `json:"is_valid"`
	TotalBlocks uint64 `json:"total_blocks"`
}

// RegisterRoutes registra as rotas do handler
func (h *BlockchainHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/blocks", h.GetBlocks).Methods("GET")
	router.HandleFunc("/blocks/{index}", h.GetBlockByIndex).Methods("GET")
	router.HandleFunc("/blocks/hash/{hash}", h.GetBlock).Methods("GET")
	router.HandleFunc("/blocks/latest", h.GetLatestBlock).Methods("GET")
	router.HandleFunc("/chain/status", h.GetChainStatus).Methods("GET")
	router.HandleFunc("/chain/validate", h.ValidateChain).Methods("GET")
}

// GetBlocks lista blocos com paginação
func (h *BlockchainHandler) GetBlocks(w http.ResponseWriter, r *http.Request) {
	// Extrair parâmetros de query para paginação
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 10 // padrão
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	offset := 0 // padrão
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Obter altura da cadeia
	height, err := h.blockchainRepo.GetBlockHeight(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Calcular range de blocos
	start := uint64(offset)
	end := start + uint64(limit)
	if end > height {
		end = height
	}

	var blocks []BlockResponse
	for i := start; i < end; i++ {
		block, err := h.blockchainRepo.GetBlockByIndex(r.Context(), i)
		if err != nil {
			continue // Pular blocos com erro
		}

		blockResp := BlockResponse{
			Index:        block.GetIndex(),
			Hash:         h.chainManager.CalculateBlockHash(r.Context(), block).String(),
			PreviousHash: block.GetPreviousHash().String(),
			Timestamp:    block.GetTimestamp().Unix(),
			MerkleRoot:   block.GetMerkleRoot().String(),
			Signature:    block.GetSignature().String(),
			Validator:    block.GetValidator().String(),
			Transactions: len(block.GetTransactions()),
		}
		blocks = append(blocks, blockResp)
	}

	// Retornar resposta
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"blocks": blocks,
		"total":  height,
		"limit":  limit,
		"offset": offset,
	})
}

// GetBlockByIndex obtém um bloco pelo índice
func (h *BlockchainHandler) GetBlockByIndex(w http.ResponseWriter, r *http.Request) {
	// Extrair índice da URL
	vars := mux.Vars(r)
	indexStr := vars["index"]

	index, err := strconv.ParseUint(indexStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid block index", http.StatusBadRequest)
		return
	}

	// Obter bloco
	block, err := h.blockchainRepo.GetBlockByIndex(r.Context(), index)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Converter para resposta
	blockResp := BlockResponse{
		Index:        block.GetIndex(),
		Hash:         h.chainManager.CalculateBlockHash(r.Context(), block).String(),
		PreviousHash: block.GetPreviousHash().String(),
		Timestamp:    block.GetTimestamp().Unix(),
		MerkleRoot:   block.GetMerkleRoot().String(),
		Signature:    block.GetSignature().String(),
		Validator:    block.GetValidator().String(),
		Transactions: len(block.GetTransactions()),
	}

	// Retornar resposta
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(blockResp)
}

// GetBlock obtém um bloco pelo hash
func (h *BlockchainHandler) GetBlock(w http.ResponseWriter, r *http.Request) {
	// Extrair hash da URL
	vars := mux.Vars(r)
	hashStr := vars["hash"]

	// Converter para Hash
	hash, err := valueobjects.NewHashFromString(hashStr)
	if err != nil {
		http.Error(w, "Invalid hash format", http.StatusBadRequest)
		return
	}

	// Obter bloco
	block, err := h.blockchainRepo.GetBlock(r.Context(), hash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Converter para resposta
	blockResp := BlockResponse{
		Index:        block.GetIndex(),
		Hash:         h.chainManager.CalculateBlockHash(r.Context(), block).String(),
		PreviousHash: block.GetPreviousHash().String(),
		Timestamp:    block.GetTimestamp().Unix(),
		MerkleRoot:   block.GetMerkleRoot().String(),
		Signature:    block.GetSignature().String(),
		Validator:    block.GetValidator().String(),
		Transactions: len(block.GetTransactions()),
	}

	// Retornar resposta
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(blockResp)
}

// GetLatestBlock obtém o último bloco
func (h *BlockchainHandler) GetLatestBlock(w http.ResponseWriter, r *http.Request) {
	// Obter último bloco
	block, err := h.blockchainRepo.GetLatestBlock(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Converter para resposta
	blockResp := BlockResponse{
		Index:        block.GetIndex(),
		Hash:         h.chainManager.CalculateBlockHash(r.Context(), block).String(),
		PreviousHash: block.GetPreviousHash().String(),
		Timestamp:    block.GetTimestamp().Unix(),
		MerkleRoot:   block.GetMerkleRoot().String(),
		Signature:    block.GetSignature().String(),
		Validator:    block.GetValidator().String(),
		Transactions: len(block.GetTransactions()),
	}

	// Retornar resposta
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(blockResp)
}

// GetChainStatus obtém o status da blockchain
func (h *BlockchainHandler) GetChainStatus(w http.ResponseWriter, r *http.Request) {
	// Obter altura da cadeia
	height, err := h.blockchainRepo.GetBlockHeight(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Obter último bloco
	latestBlock, err := h.blockchainRepo.GetLatestBlock(r.Context())
	latestHash := ""
	if err == nil {
		latestHash = h.chainManager.CalculateBlockHash(r.Context(), latestBlock).String()
	}

	// Validar cadeia
	err = h.blockchainRepo.ValidateChain(r.Context())
	isValid := err == nil

	// Criar resposta
	status := ChainStatusResponse{
		Height:      height,
		LatestBlock: latestHash,
		IsValid:     isValid,
		TotalBlocks: height,
	}

	// Retornar resposta
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// ValidateChain valida a integridade da blockchain
func (h *BlockchainHandler) ValidateChain(w http.ResponseWriter, r *http.Request) {
	// Validar cadeia
	err := h.blockchainRepo.ValidateChain(r.Context())
	isValid := err == nil
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Retornar resposta
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"is_valid": isValid,
		"message":  "Chain validation completed",
	})
}
