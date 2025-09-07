package network

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/matscats/peer-vote/peer-vote/domain/entities"
	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
	"github.com/matscats/peer-vote/peer-vote/infrastructure/blockchain"
)

// SyncService gerencia sincronização de blockchain entre peers
type SyncService struct {
	chainManager    *blockchain.ChainManager
	protocolManager *ProtocolManager
	host            *LibP2PHost
	
	// Estado da sincronização
	isSyncing       bool
	syncPeers       map[peer.ID]*SyncPeerInfo
	lastSyncTime    time.Time
	
	// Configurações
	syncInterval    time.Duration
	maxSyncPeers    int
	blockBatchSize  int
	syncTimeout     time.Duration
	
	// Canais
	syncRequestChan chan SyncRequest
	stopChan        chan struct{}
	
	// Mutex para operações thread-safe
	mu sync.RWMutex
	
	// Callbacks
	onSyncStart     func()
	onSyncComplete  func(blocksAdded int)
	onSyncError     func(error)
}

// SyncPeerInfo contém informações sobre um peer de sincronização
type SyncPeerInfo struct {
	PeerID        peer.ID
	ChainHeight   uint64
	LatestHash    string
	LastContact   time.Time
	IsReliable    bool
	SyncAttempts  int
	FailureCount  int
}

// SyncRequest representa uma requisição de sincronização
type SyncRequest struct {
	PeerID      peer.ID
	StartHeight uint64
	EndHeight   uint64
	Priority    int
}

// SyncStats contém estatísticas de sincronização
type SyncStats struct {
	IsSyncing       bool
	LastSyncTime    time.Time
	SyncPeerCount   int
	BlocksReceived  int
	SyncAttempts    int
	FailureCount    int
	AverageLatency  time.Duration
}

// NewSyncService cria um novo serviço de sincronização
func NewSyncService(chainManager *blockchain.ChainManager, protocolManager *ProtocolManager, host *LibP2PHost) *SyncService {
	ss := &SyncService{
		chainManager:    chainManager,
		protocolManager: protocolManager,
		host:            host,
		syncPeers:       make(map[peer.ID]*SyncPeerInfo),
		syncInterval:    time.Second * 30,
		maxSyncPeers:    5,
		blockBatchSize:  50,
		syncTimeout:     time.Second * 60,
		syncRequestChan: make(chan SyncRequest, 100),
		stopChan:        make(chan struct{}),
	}
	
	// Configurar handlers de protocolo
	ss.setupProtocolHandlers()
	
	return ss
}

// Start inicia o serviço de sincronização
func (ss *SyncService) Start(ctx context.Context) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	
	if ss.isSyncing {
		return fmt.Errorf("sync service already running")
	}
	
	// Iniciar loops de sincronização
	go ss.syncLoop(ctx)
	go ss.requestHandler(ctx)
	go ss.peerMonitor(ctx)
	
	return nil
}

// Stop para o serviço de sincronização
func (ss *SyncService) Stop(ctx context.Context) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	
	close(ss.stopChan)
	ss.isSyncing = false
	
	return nil
}

// setupProtocolHandlers configura handlers para protocolos de sincronização
func (ss *SyncService) setupProtocolHandlers() {
	// Handler para requisições de bloco
	ss.protocolManager.SetBlockRequestHandler(ss.handleBlockRequest)
	
	// Handler para requisições de faixa de blocos
	ss.protocolManager.SetBlockRangeHandler(ss.handleBlockRangeRequest)
	
	// Handler para requisições de status da cadeia
	ss.protocolManager.SetChainStatusHandler(ss.handleChainStatusRequest)
	
	// Handler para gossip de blocos
	ss.protocolManager.SetBlockGossipHandler(ss.handleBlockGossip)
}

// syncLoop executa sincronização periódica
func (ss *SyncService) syncLoop(ctx context.Context) {
	ticker := time.NewTicker(ss.syncInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ss.stopChan:
			return
		case <-ticker.C:
			ss.performSync(ctx)
		}
	}
}

// performSync executa uma rodada de sincronização
func (ss *SyncService) performSync(ctx context.Context) {
	ss.mu.Lock()
	if ss.isSyncing {
		ss.mu.Unlock()
		return
	}
	ss.isSyncing = true
	ss.mu.Unlock()
	
	defer func() {
		ss.mu.Lock()
		ss.isSyncing = false
		ss.lastSyncTime = time.Now()
		ss.mu.Unlock()
	}()
	
	if ss.onSyncStart != nil {
		ss.onSyncStart()
	}
	
	// Descobrir peers e obter status das cadeias
	if err := ss.discoverSyncPeers(ctx); err != nil {
		if ss.onSyncError != nil {
			ss.onSyncError(fmt.Errorf("failed to discover sync peers: %w", err))
		}
		return
	}
	
	// Determinar se precisamos sincronizar
	needsSync, bestPeer := ss.needsSynchronization(ctx)
	if !needsSync {
		return
	}
	
	// Executar sincronização
	blocksAdded, err := ss.synchronizeWithPeer(ctx, bestPeer)
	if err != nil {
		if ss.onSyncError != nil {
			ss.onSyncError(fmt.Errorf("synchronization failed: %w", err))
		}
		return
	}
	
	if ss.onSyncComplete != nil {
		ss.onSyncComplete(blocksAdded)
	}
}

// discoverSyncPeers descobre peers e obtém status das cadeias
func (ss *SyncService) discoverSyncPeers(ctx context.Context) error {
	connectedPeers := ss.host.GetConnectedPeers()
	
	for _, peerID := range connectedPeers {
		go ss.updatePeerInfo(ctx, peerID)
	}
	
	// Aguardar um pouco para respostas
	time.Sleep(time.Second * 2)
	
	return nil
}

// updatePeerInfo atualiza informações sobre um peer
func (ss *SyncService) updatePeerInfo(ctx context.Context, peerID peer.ID) {
	// Solicitar status da cadeia
	status, err := ss.protocolManager.SendChainStatusRequest(ctx, peerID)
	if err != nil {
		ss.markPeerUnreliable(peerID)
		return
	}
	
	ss.mu.Lock()
	defer ss.mu.Unlock()
	
	peerInfo, exists := ss.syncPeers[peerID]
	if !exists {
		peerInfo = &SyncPeerInfo{
			PeerID: peerID,
		}
		ss.syncPeers[peerID] = peerInfo
	}
	
	peerInfo.ChainHeight = status.Height
	peerInfo.LatestHash = status.LatestHash
	peerInfo.LastContact = time.Now()
	peerInfo.IsReliable = true
}

// needsSynchronization determina se precisamos sincronizar
func (ss *SyncService) needsSynchronization(ctx context.Context) (bool, peer.ID) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	
	// Obter altura atual da nossa cadeia
	ourHeight, err := ss.chainManager.GetChainHeight(ctx)
	if err != nil {
		return false, ""
	}
	
	// Encontrar o melhor peer (maior altura)
	var bestPeer peer.ID
	var bestHeight uint64
	
	for peerID, peerInfo := range ss.syncPeers {
		if peerInfo.IsReliable && peerInfo.ChainHeight > bestHeight {
			bestPeer = peerID
			bestHeight = peerInfo.ChainHeight
		}
	}
	
	// Sincronizar se o peer tem mais blocos que nós
	return bestHeight > ourHeight, bestPeer
}

// synchronizeWithPeer sincroniza com um peer específico
func (ss *SyncService) synchronizeWithPeer(ctx context.Context, peerID peer.ID) (int, error) {
	if peerID == "" {
		return 0, fmt.Errorf("no peer specified")
	}
	
	// Obter altura atual
	ourHeight, err := ss.chainManager.GetChainHeight(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get chain height: %w", err)
	}
	
	peerInfo := ss.syncPeers[peerID]
	if peerInfo == nil {
		return 0, fmt.Errorf("peer info not found")
	}
	
	blocksAdded := 0
	currentHeight := ourHeight
	
	// Sincronizar em lotes
	for currentHeight < peerInfo.ChainHeight {
		endHeight := currentHeight + uint64(ss.blockBatchSize)
		if endHeight > peerInfo.ChainHeight {
			endHeight = peerInfo.ChainHeight
		}
		
		// Solicitar faixa de blocos
		blocks, err := ss.requestBlockRange(ctx, peerID, currentHeight+1, endHeight)
		if err != nil {
			ss.markPeerUnreliable(peerID)
			return blocksAdded, fmt.Errorf("failed to request block range: %w", err)
		}
		
		// Adicionar blocos à cadeia
		for _, block := range blocks {
			if err := ss.chainManager.AddBlock(ctx, block); err != nil {
				return blocksAdded, fmt.Errorf("failed to add block %d: %w", block.GetIndex(), err)
			}
			blocksAdded++
			currentHeight = block.GetIndex()
		}
		
		// Verificar se devemos parar
		select {
		case <-ctx.Done():
			return blocksAdded, ctx.Err()
		case <-ss.stopChan:
			return blocksAdded, fmt.Errorf("sync service stopped")
		default:
		}
	}
	
	return blocksAdded, nil
}

// requestBlockRange solicita uma faixa de blocos de um peer
func (ss *SyncService) requestBlockRange(ctx context.Context, peerID peer.ID, startHeight, endHeight uint64) ([]*entities.Block, error) {
	// Criar stream para requisição
	stream, err := ss.host.NewStream(ctx, peerID, ProtocolBlockSync)
	if err != nil {
		return nil, fmt.Errorf("failed to create stream: %w", err)
	}
	defer stream.Close()
	
	// Implementação simplificada - em produção usaria o ProtocolManager
	// Por enquanto, retornar erro indicando que precisa ser implementado
	return nil, fmt.Errorf("block range request not fully implemented yet")
}

// markPeerUnreliable marca um peer como não confiável
func (ss *SyncService) markPeerUnreliable(peerID peer.ID) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	
	if peerInfo, exists := ss.syncPeers[peerID]; exists {
		peerInfo.IsReliable = false
		peerInfo.FailureCount++
	}
}

// requestHandler processa requisições de sincronização
func (ss *SyncService) requestHandler(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-ss.stopChan:
			return
		case req := <-ss.syncRequestChan:
			ss.processRequest(ctx, req)
		}
	}
}

// processRequest processa uma requisição de sincronização
func (ss *SyncService) processRequest(ctx context.Context, req SyncRequest) {
	// Implementação de processamento de requisições
	// Por enquanto, apenas log da requisição
}

// peerMonitor monitora peers e remove os inativos
func (ss *SyncService) peerMonitor(ctx context.Context) {
	ticker := time.NewTicker(time.Minute * 5)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ss.stopChan:
			return
		case <-ticker.C:
			ss.cleanupInactivePeers()
		}
	}
}

// cleanupInactivePeers remove peers inativos
func (ss *SyncService) cleanupInactivePeers() {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	
	now := time.Now()
	for peerID, peerInfo := range ss.syncPeers {
		// Remover peers que não respondem há muito tempo
		if now.Sub(peerInfo.LastContact) > time.Minute*10 {
			delete(ss.syncPeers, peerID)
		}
	}
}

// Handlers para protocolos

// handleBlockRequest lida com requisições de bloco individual
func (ss *SyncService) handleBlockRequest(peerID peer.ID, req *BlockRequest) (*BlockResponse, error) {
	ctx := context.Background()
	
	// Buscar bloco por hash se fornecido
	if req.BlockHash != "" {
		hash, err := valueobjects.NewHashFromString(req.BlockHash)
		if err != nil {
			return &BlockResponse{
				Found:    false,
				ErrorMsg: "Invalid block hash format",
			}, nil
		}
		
		block, err := ss.chainManager.GetBlock(ctx, hash)
		if err != nil {
			return &BlockResponse{
				Found:    false,
				ErrorMsg: "Block not found",
			}, nil
		}
		
		serializedBlock := ss.serializeBlock(block)
		return &BlockResponse{
			Block: serializedBlock,
			Found: true,
		}, nil
	}
	
	// Buscar bloco por índice se fornecido
	if req.BlockIndex > 0 {
		block, err := ss.chainManager.GetBlockByIndex(ctx, req.BlockIndex)
		if err != nil {
			return &BlockResponse{
				Found:    false,
				ErrorMsg: "Block not found",
			}, nil
		}
		
		serializedBlock := ss.serializeBlock(block)
		return &BlockResponse{
			Block: serializedBlock,
			Found: true,
		}, nil
	}
	
	return &BlockResponse{
		Found:    false,
		ErrorMsg: "No block hash or index provided",
	}, nil
}

// handleBlockRangeRequest lida com requisições de faixa de blocos
func (ss *SyncService) handleBlockRangeRequest(peerID peer.ID, req *BlockRangeRequest) (*BlockRangeResponse, error) {
	ctx := context.Background()
	
	// Validar requisição
	if req.StartIndex > req.EndIndex {
		return &BlockRangeResponse{
			ErrorMsg: "Invalid range: start index greater than end index",
		}, nil
	}
	
	maxBlocks := req.MaxBlocks
	if maxBlocks <= 0 || maxBlocks > ss.blockBatchSize {
		maxBlocks = ss.blockBatchSize
	}
	
	// Ajustar range se necessário
	endIndex := req.EndIndex
	if endIndex > req.StartIndex+uint64(maxBlocks)-1 {
		endIndex = req.StartIndex + uint64(maxBlocks) - 1
	}
	
	// Buscar blocos
	blocks, err := ss.chainManager.GetBlockRange(ctx, req.StartIndex, endIndex)
	if err != nil {
		return &BlockRangeResponse{
			ErrorMsg: fmt.Sprintf("Failed to get block range: %v", err),
		}, nil
	}
	
	// Serializar blocos
	serializedBlocks := make([]*SerializedBlock, len(blocks))
	for i, block := range blocks {
		serializedBlocks[i] = ss.serializeBlock(block)
	}
	
	hasMore := req.EndIndex > endIndex
	
	return &BlockRangeResponse{
		Blocks:  serializedBlocks,
		HasMore: hasMore,
	}, nil
}

// handleChainStatusRequest lida com requisições de status da cadeia
func (ss *SyncService) handleChainStatusRequest(peerID peer.ID, req *ChainStatusRequest) (*ChainStatusResponse, error) {
	ctx := context.Background()
	
	// Obter altura da cadeia
	height, err := ss.chainManager.GetChainHeight(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain height: %w", err)
	}
	
	// Obter último bloco
	latestBlock, err := ss.chainManager.GetLatestBlock(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest block: %w", err)
	}
	
	// Obter bloco gênesis
	genesisBlock, err := ss.chainManager.GetBlockByIndex(ctx, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get genesis block: %w", err)
	}
	
	latestHash := ss.calculateBlockHash(latestBlock)
	genesisHash := ss.calculateBlockHash(genesisBlock)
	
	return &ChainStatusResponse{
		Height:        height,
		LatestHash:    latestHash.String(),
		GenesisHash:   genesisHash.String(),
		PeerCount:     ss.host.GetPeerCount(),
		LastBlockTime: latestBlock.GetTimestamp().Unix(),
	}, nil
}

// handleBlockGossip lida com gossip de blocos
func (ss *SyncService) handleBlockGossip(peerID peer.ID, msg *BlockGossipMessage) error {
	ctx := context.Background()
	
	// Deserializar bloco
	block, err := ss.deserializeBlock(msg.Block)
	if err != nil {
		return fmt.Errorf("failed to deserialize block: %w", err)
	}
	
	// Tentar adicionar bloco à cadeia
	if err := ss.chainManager.AddBlock(ctx, block); err != nil {
		// Bloco pode já existir ou ser inválido - não é necessariamente um erro
		return nil
	}
	
	return nil
}

// Métodos auxiliares

func (ss *SyncService) serializeBlock(block *entities.Block) *SerializedBlock {
	transactions := make([]*SerializedTransaction, len(block.GetTransactions()))
	for i, tx := range block.GetTransactions() {
		transactions[i] = ss.serializeTransaction(tx)
	}
	
	return &SerializedBlock{
		Index:        block.GetIndex(),
		PreviousHash: block.GetPreviousHash().String(),
		Timestamp:    block.GetTimestamp().Unix(),
		MerkleRoot:   block.GetMerkleRoot().String(),
		Validator:    block.GetValidator().String(),
		Signature:    block.GetSignature().String(),
		Transactions: transactions,
	}
}

func (ss *SyncService) serializeTransaction(tx *entities.Transaction) *SerializedTransaction {
	return &SerializedTransaction{
		ID:        tx.GetID().String(),
		Type:      string(tx.GetType()),
		From:      tx.GetFrom().String(),
		To:        tx.GetTo().String(),
		Data:      string(tx.GetData()),
		Timestamp: tx.GetTimestamp().Unix(),
		Signature: tx.GetSignature().String(),
		Hash:      tx.GetHash().String(),
	}
}

func (ss *SyncService) deserializeBlock(serialized *SerializedBlock) (*entities.Block, error) {
	// Implementação simplificada - em produção seria mais robusta
	// Por enquanto, retornar erro indicando que precisa ser implementado
	return nil, fmt.Errorf("block deserialization not fully implemented yet")
}

func (ss *SyncService) calculateBlockHash(block *entities.Block) valueobjects.Hash {
	// Implementação simplificada
	data := fmt.Sprintf("%d-%s-%d", 
		block.GetIndex(), 
		block.GetMerkleRoot().String(), 
		block.GetTimestamp().Unix())
	
	hash := make([]byte, 32)
	copy(hash, []byte(data))
	
	return valueobjects.NewHash(hash)
}

// Métodos públicos para controle

// RequestSync solicita sincronização com um peer específico
func (ss *SyncService) RequestSync(peerID peer.ID, startHeight, endHeight uint64) {
	req := SyncRequest{
		PeerID:      peerID,
		StartHeight: startHeight,
		EndHeight:   endHeight,
		Priority:    1,
	}
	
	select {
	case ss.syncRequestChan <- req:
	default:
		// Canal cheio, pular requisição
	}
}

// GetSyncStats retorna estatísticas de sincronização
func (ss *SyncService) GetSyncStats() *SyncStats {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	
	return &SyncStats{
		IsSyncing:     ss.isSyncing,
		LastSyncTime:  ss.lastSyncTime,
		SyncPeerCount: len(ss.syncPeers),
	}
}

// GetSyncPeers retorna informações sobre peers de sincronização
func (ss *SyncService) GetSyncPeers() map[peer.ID]*SyncPeerInfo {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	
	result := make(map[peer.ID]*SyncPeerInfo)
	for peerID, peerInfo := range ss.syncPeers {
		result[peerID] = &SyncPeerInfo{
			PeerID:       peerInfo.PeerID,
			ChainHeight:  peerInfo.ChainHeight,
			LatestHash:   peerInfo.LatestHash,
			LastContact:  peerInfo.LastContact,
			IsReliable:   peerInfo.IsReliable,
			SyncAttempts: peerInfo.SyncAttempts,
			FailureCount: peerInfo.FailureCount,
		}
	}
	
	return result
}

// SetCallbacks define callbacks para eventos de sincronização
func (ss *SyncService) SetCallbacks(onStart func(), onComplete func(int), onError func(error)) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	
	ss.onSyncStart = onStart
	ss.onSyncComplete = onComplete
	ss.onSyncError = onError
}

// IsSyncing verifica se está sincronizando
func (ss *SyncService) IsSyncing() bool {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	
	return ss.isSyncing
}
