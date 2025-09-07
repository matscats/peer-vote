package consensus

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/matscats/peer-vote/peer-vote/domain/entities"
	"github.com/matscats/peer-vote/peer-vote/domain/services"
	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
	"github.com/matscats/peer-vote/peer-vote/infrastructure/blockchain"
	"github.com/matscats/peer-vote/peer-vote/infrastructure/crypto"
)

// PoAEngine implementa o algoritmo de consenso Proof of Authority
type PoAEngine struct {
	// Componentes principais
	validatorManager *ValidatorManager
	roundRobin       *RoundRobinScheduler
	chainManager     *blockchain.ChainManager
	cryptoService    services.CryptographyService
	keyRepository    crypto.KeyRepository // Repositório de chaves para validação
	
	// Estado do consenso
	isRunning        bool
	currentValidator valueobjects.NodeID
	myNodeID         valueobjects.NodeID
	myPrivateKey     *services.PrivateKey
	
	// Pool de transações pendentes
	pendingTxs       []*entities.Transaction
	maxPendingTxs    int
	
	// Configurações
	blockInterval    time.Duration // Intervalo entre blocos
	minTxPerBlock    int          // Mínimo de transações por bloco
	maxTxPerBlock    int          // Máximo de transações por bloco
	
	// Canais para comunicação
	newTxChan        chan *entities.Transaction
	stopChan         chan struct{}
	
	// Mutex para operações thread-safe
	mu sync.RWMutex
	
	// Callbacks para eventos
	onBlockProduced  func(*entities.Block)
	onConsensusError func(error)
}

// NewPoAEngine cria um novo motor de consenso PoA
func NewPoAEngine(
	validatorManager *ValidatorManager,
	chainManager *blockchain.ChainManager,
	cryptoService services.CryptographyService,
	myNodeID valueobjects.NodeID,
	myPrivateKey *services.PrivateKey,
) *PoAEngine {
	roundRobin := NewRoundRobinScheduler(validatorManager)
	
	return &PoAEngine{
		validatorManager: validatorManager,
		roundRobin:       roundRobin,
		chainManager:     chainManager,
		cryptoService:    cryptoService,
		keyRepository:    nil, // Será definido depois
		myNodeID:         myNodeID,
		myPrivateKey:     myPrivateKey,
		pendingTxs:       make([]*entities.Transaction, 0),
		maxPendingTxs:    10000,
		blockInterval:    time.Second * 2,
		minTxPerBlock:    1,
		maxTxPerBlock:    1000,
		newTxChan:        make(chan *entities.Transaction, 1000),
		stopChan:         make(chan struct{}),
	}
}

// SetKeyRepository define o repositório de chaves para validação
func (poa *PoAEngine) SetKeyRepository(keyRepo crypto.KeyRepository) {
	poa.mu.Lock()
	defer poa.mu.Unlock()
	poa.keyRepository = keyRepo
}

// StartConsensus inicia o processo de consenso
func (poa *PoAEngine) StartConsensus(ctx context.Context) error {
	poa.mu.Lock()
	defer poa.mu.Unlock()

	if poa.isRunning {
		return errors.New("consensus already running")
	}

	// Verificar se este nó é um validador
	isValidator, err := poa.validatorManager.IsValidator(ctx, poa.myNodeID)
	if err != nil {
		return fmt.Errorf("failed to check validator status: %w", err)
	}

	if !isValidator {
		return fmt.Errorf("node %s is not an authorized validator", poa.myNodeID.ShortString())
	}

	// Iniciar Round Robin scheduler
	if err := poa.roundRobin.Start(ctx); err != nil {
		return fmt.Errorf("failed to start round robin scheduler: %w", err)
	}

	poa.isRunning = true

	// Iniciar goroutines
	go poa.consensusLoop(ctx)
	go poa.transactionProcessor(ctx)
	go poa.roundMonitor(ctx)

	return nil
}

// StopConsensus para o processo de consenso
func (poa *PoAEngine) StopConsensus(ctx context.Context) error {
	poa.mu.Lock()
	defer poa.mu.Unlock()

	if !poa.isRunning {
		return errors.New("consensus not running")
	}

	poa.isRunning = false
	close(poa.stopChan)

	return nil
}

// ProposeBlock propõe um novo bloco para consenso
func (poa *PoAEngine) ProposeBlock(ctx context.Context, block *entities.Block) error {
	poa.mu.RLock()
	defer poa.mu.RUnlock()

	if !poa.isRunning {
		return errors.New("consensus not running")
	}

	// Verificar se é nossa vez de validar
	isMyTurn, err := poa.roundRobin.IsMyTurn(ctx, poa.myNodeID)
	if err != nil {
		return fmt.Errorf("failed to check turn: %w", err)
	}

	if !isMyTurn {
		return errors.New("not my turn to propose block")
	}

	// Validar o bloco
	if err := poa.validateProposedBlock(ctx, block); err != nil {
		return fmt.Errorf("block validation failed: %w", err)
	}

	// Adicionar bloco à cadeia
	if err := poa.chainManager.AddBlock(ctx, block); err != nil {
		return fmt.Errorf("failed to add block to chain: %w", err)
	}

	// Notificar produção de bloco
	if err := poa.roundRobin.NotifyBlockProduced(ctx, poa.myNodeID); err != nil {
		return fmt.Errorf("failed to notify block production: %w", err)
	}

	// Callback de bloco produzido
	if poa.onBlockProduced != nil {
		poa.onBlockProduced(block)
	}

	return nil
}

// ValidateBlock valida um bloco proposto
func (poa *PoAEngine) ValidateBlock(ctx context.Context, block *entities.Block) error {
	if block == nil {
		return errors.New("block is nil")
	}

	// Validar estrutura básica do bloco
	if !block.IsValid() {
		return errors.New("block failed basic validation")
	}

	// Verificar se o validador é autorizado
	validator := block.GetValidator()
	isValidator, err := poa.validatorManager.IsValidator(ctx, validator)
	if err != nil {
		return fmt.Errorf("failed to check validator authorization: %w", err)
	}

	if !isValidator {
		return fmt.Errorf("block validator %s is not authorized", validator.ShortString())
	}

	// Verificar assinatura do bloco usando repositório de chaves se disponível
	if poa.keyRepository != nil {
		// Usar repositório de chaves para validação mais robusta
		blockData, err := poa.serializeBlockForValidation(ctx, block)
		if err != nil {
			return fmt.Errorf("failed to serialize block for validation: %w", err)
		}
		
		ecdsaService, ok := poa.cryptoService.(*crypto.ECDSAService)
		if ok {
			isValid, err := ecdsaService.ValidateSignatureWithKeyRepo(
				ctx, 
				blockData, 
				block.GetSignature(), 
				validator, 
				poa.keyRepository,
			)
			if err != nil {
				return fmt.Errorf("signature validation failed: %w", err)
			}
			if !isValid {
				return fmt.Errorf("block signature is invalid")
			}
		} else {
			// Fallback para validação tradicional
			publicKey, err := poa.validatorManager.GetValidatorPublicKey(ctx, validator)
			if err != nil {
				return fmt.Errorf("failed to get validator public key: %w", err)
			}

			blockBuilder := blockchain.NewBlockBuilder(poa.cryptoService)
			if err := blockBuilder.ValidateBlockSignature(ctx, block, publicKey); err != nil {
				return fmt.Errorf("block signature validation failed: %w", err)
			}
		}
	} else {
		// Validação tradicional sem repositório de chaves
		publicKey, err := poa.validatorManager.GetValidatorPublicKey(ctx, validator)
		if err != nil {
			return fmt.Errorf("failed to get validator public key: %w", err)
		}

		blockBuilder := blockchain.NewBlockBuilder(poa.cryptoService)
		if err := blockBuilder.ValidateBlockSignature(ctx, block, publicKey); err != nil {
			return fmt.Errorf("block signature validation failed: %w", err)
		}
	}

	// Verificar se é o validador correto para este round
	currentValidator, err := poa.roundRobin.GetCurrentValidator(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current validator: %w", err)
	}

	if !validator.Equals(currentValidator) {
		return fmt.Errorf("block validator %s is not the current validator %s", 
			validator.ShortString(), currentValidator.ShortString())
	}

	return nil
}

// AddTransaction adiciona uma transação ao pool de pendentes
func (poa *PoAEngine) AddTransaction(ctx context.Context, tx *entities.Transaction) error {
	if tx == nil {
		return errors.New("transaction is nil")
	}

	if !tx.IsValid() {
		return errors.New("invalid transaction")
	}

	poa.mu.Lock()
	defer poa.mu.Unlock()

	// Verificar se o pool não está cheio
	if len(poa.pendingTxs) >= poa.maxPendingTxs {
		return errors.New("transaction pool is full")
	}

	// Verificar duplicatas
	for _, pendingTx := range poa.pendingTxs {
		if pendingTx.GetHash().Equals(tx.GetHash()) {
			return errors.New("transaction already in pool")
		}
	}

	// Adicionar ao pool
	poa.pendingTxs = append(poa.pendingTxs, tx)

	// Notificar processador de transações
	select {
	case poa.newTxChan <- tx:
	default:
		// Canal cheio, transação já está no pool
	}

	return nil
}

// GetCurrentValidator retorna o validador atual no round robin
func (poa *PoAEngine) GetCurrentValidator(ctx context.Context) (valueobjects.NodeID, error) {
	return poa.roundRobin.GetCurrentValidator(ctx)
}

// GetNextValidator retorna o próximo validador na sequência
func (poa *PoAEngine) GetNextValidator(ctx context.Context) (valueobjects.NodeID, error) {
	return poa.roundRobin.GetNextValidator(ctx)
}

// IsValidator verifica se um nó é um validador autorizado
func (poa *PoAEngine) IsValidator(ctx context.Context, nodeID valueobjects.NodeID) (bool, error) {
	return poa.validatorManager.IsValidator(ctx, nodeID)
}

// AddValidator adiciona um novo validador à lista
func (poa *PoAEngine) AddValidator(ctx context.Context, nodeID valueobjects.NodeID, publicKey *services.PublicKey) error {
	return poa.validatorManager.AddValidator(ctx, nodeID, publicKey)
}

// RemoveValidator remove um validador da lista
func (poa *PoAEngine) RemoveValidator(ctx context.Context, nodeID valueobjects.NodeID) error {
	return poa.validatorManager.RemoveValidator(ctx, nodeID)
}

// GetValidators retorna a lista de todos os validadores
func (poa *PoAEngine) GetValidators(ctx context.Context) ([]valueobjects.NodeID, error) {
	validators, err := poa.validatorManager.GetAllValidators(ctx)
	if err != nil {
		return nil, err
	}

	nodeIDs := make([]valueobjects.NodeID, len(validators))
	for i, validator := range validators {
		nodeIDs[i] = validator.NodeID
	}

	return nodeIDs, nil
}

// GetValidatorCount retorna o número de validadores
func (poa *PoAEngine) GetValidatorCount(ctx context.Context) (int, error) {
	return poa.validatorManager.GetValidatorCount(ctx)
}

// IsMyTurn verifica se é a vez deste nó validar
func (poa *PoAEngine) IsMyTurn(ctx context.Context) (bool, error) {
	return poa.roundRobin.IsMyTurn(ctx, poa.myNodeID)
}

// GetCurrentRound retorna o round atual
func (poa *PoAEngine) GetCurrentRound(ctx context.Context) (uint64, error) {
	return poa.roundRobin.GetCurrentRound(ctx)
}

// AdvanceRound avança para o próximo round
func (poa *PoAEngine) AdvanceRound(ctx context.Context) error {
	return poa.roundRobin.AdvanceRound(ctx)
}

// HandleTimeout lida com timeout de validador
func (poa *PoAEngine) HandleTimeout(ctx context.Context, validator valueobjects.NodeID) error {
	return poa.roundRobin.HandleTimeout(ctx, validator)
}

// GetConsensusStatus retorna o status atual do consenso
func (poa *PoAEngine) GetConsensusStatus(ctx context.Context) (*services.ConsensusStatus, error) {
	poa.mu.RLock()
	defer poa.mu.RUnlock()

	currentValidator, _ := poa.roundRobin.GetCurrentValidator(ctx)
	currentRound, _ := poa.roundRobin.GetCurrentRound(ctx)
	validatorCount, _ := poa.validatorManager.GetValidatorCount(ctx)

	latestBlock, err := poa.chainManager.GetLatestBlock(ctx)
	var lastBlockTime valueobjects.Timestamp
	if err == nil && latestBlock != nil {
		lastBlockTime = latestBlock.GetTimestamp()
	}

	return &services.ConsensusStatus{
		IsRunning:        poa.isRunning,
		CurrentValidator: currentValidator,
		CurrentRound:     currentRound,
		ValidatorCount:   validatorCount,
		LastBlockTime:    lastBlockTime,
	}, nil
}

// consensusLoop é o loop principal do consenso
func (poa *PoAEngine) consensusLoop(ctx context.Context) {
	ticker := time.NewTicker(poa.blockInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-poa.stopChan:
			return
		case <-ticker.C:
			poa.tryProduceBlock(ctx)
		}
	}
}

// tryProduceBlock tenta produzir um bloco se for nossa vez
func (poa *PoAEngine) tryProduceBlock(ctx context.Context) {
	// Verificar se é nossa vez
	isMyTurn, err := poa.roundRobin.IsMyTurn(ctx, poa.myNodeID)
	if err != nil || !isMyTurn {
		return
	}

	poa.mu.Lock()
	defer poa.mu.Unlock()

	// Verificar se temos transações suficientes
	if len(poa.pendingTxs) < poa.minTxPerBlock {
		return
	}

	// Selecionar transações para o bloco
	txCount := len(poa.pendingTxs)
	if txCount > poa.maxTxPerBlock {
		txCount = poa.maxTxPerBlock
	}

	selectedTxs := make([]*entities.Transaction, txCount)
	copy(selectedTxs, poa.pendingTxs[:txCount])

	// Propor bloco
	block, err := poa.chainManager.ProposeBlock(ctx, selectedTxs, poa.myNodeID, poa.myPrivateKey)
	if err != nil {
		if poa.onConsensusError != nil {
			poa.onConsensusError(fmt.Errorf("failed to propose block: %w", err))
		}
		return
	}

	// Adicionar bloco à cadeia
	if err := poa.chainManager.AddBlock(ctx, block); err != nil {
		if poa.onConsensusError != nil {
			poa.onConsensusError(fmt.Errorf("failed to add block: %w", err))
		}
		return
	}

	// Remover transações processadas do pool
	poa.pendingTxs = poa.pendingTxs[txCount:]

	// Notificar produção de bloco
	poa.roundRobin.NotifyBlockProduced(ctx, poa.myNodeID)

	// Callback
	if poa.onBlockProduced != nil {
		poa.onBlockProduced(block)
	}
}

// transactionProcessor processa transações recebidas
func (poa *PoAEngine) transactionProcessor(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-poa.stopChan:
			return
		case tx := <-poa.newTxChan:
			// Transação já foi adicionada ao pool, apenas processa se necessário
			_ = tx
		}
	}
}

// roundMonitor monitora mudanças de round
func (poa *PoAEngine) roundMonitor(ctx context.Context) {
	roundChangeChan := poa.roundRobin.GetRoundChangeChannel()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-poa.stopChan:
			return
		case event := <-roundChangeChan:
			poa.handleRoundChange(ctx, event)
		}
	}
}

// handleRoundChange lida com mudanças de round
func (poa *PoAEngine) handleRoundChange(ctx context.Context, event RoundChangeEvent) {
	poa.mu.Lock()
	poa.currentValidator = event.NewValidator
	poa.mu.Unlock()

	// Limpar penalidades expiradas
	poa.validatorManager.CleanupExpiredPenalties(ctx)
}

// validateProposedBlock valida um bloco proposto internamente
func (poa *PoAEngine) validateProposedBlock(ctx context.Context, block *entities.Block) error {
	// Usar validação padrão
	return poa.ValidateBlock(ctx, block)
}

// SetCallbacks define callbacks para eventos
func (poa *PoAEngine) SetCallbacks(onBlockProduced func(*entities.Block), onConsensusError func(error)) {
	poa.mu.Lock()
	defer poa.mu.Unlock()

	poa.onBlockProduced = onBlockProduced
	poa.onConsensusError = onConsensusError
}

// SetConfiguration define configurações do consenso
func (poa *PoAEngine) SetConfiguration(blockInterval time.Duration, minTxPerBlock, maxTxPerBlock int) {
	poa.mu.Lock()
	defer poa.mu.Unlock()

	if blockInterval > 0 {
		poa.blockInterval = blockInterval
	}
	if minTxPerBlock > 0 {
		poa.minTxPerBlock = minTxPerBlock
	}
	if maxTxPerBlock > 0 {
		poa.maxTxPerBlock = maxTxPerBlock
	}
}

// GetPendingTransactionCount retorna o número de transações pendentes
func (poa *PoAEngine) GetPendingTransactionCount() int {
	poa.mu.RLock()
	defer poa.mu.RUnlock()

	return len(poa.pendingTxs)
}

// ClearPendingTransactions limpa o pool de transações pendentes
func (poa *PoAEngine) ClearPendingTransactions() {
	poa.mu.Lock()
	defer poa.mu.Unlock()

	poa.pendingTxs = make([]*entities.Transaction, 0)
}

// serializeBlockForValidation serializa um bloco para validação de assinatura
func (poa *PoAEngine) serializeBlockForValidation(ctx context.Context, block *entities.Block) ([]byte, error) {
	// Criar estrutura determinística para validação
	blockData := struct {
		Index        uint64
		PreviousHash string
		MerkleRoot   string
		Timestamp    int64
		Validator    string
	}{
		Index:        block.GetIndex(),
		PreviousHash: block.GetPreviousHash().String(),
		MerkleRoot:   block.GetMerkleRoot().String(),
		Timestamp:    block.GetTimestamp().Unix(),
		Validator:    block.GetValidator().String(),
	}
	
	// Serializar como string determinística
	data := fmt.Sprintf("%d|%s|%s|%d|%s",
		blockData.Index,
		blockData.PreviousHash,
		blockData.MerkleRoot,
		blockData.Timestamp,
		blockData.Validator,
	)
	
	return []byte(data), nil
}
