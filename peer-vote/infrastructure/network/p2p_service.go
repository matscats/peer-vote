package network

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"

	"github.com/matscats/peer-vote/peer-vote/domain/entities"
	"github.com/matscats/peer-vote/peer-vote/domain/services"
	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
	"github.com/matscats/peer-vote/peer-vote/infrastructure/blockchain"
	"github.com/matscats/peer-vote/peer-vote/infrastructure/consensus"
)

// P2PService integra todos os componentes P2P
type P2PService struct {
	host            *LibP2PHost
	discovery       *DiscoveryService
	protocolManager *ProtocolManager
	syncService     *SyncService
	
	// Integração com outros módulos
	chainManager    *blockchain.ChainManager
	consensusEngine *consensus.PoAEngine
	cryptoService   services.CryptographyService
	
	// Estado
	isRunning       bool
	nodeID          valueobjects.NodeID
	
	// Configurações
	config          *P2PConfig
	
	// Estatísticas
	stats           *P2PStats
	
	// Mutex para operações thread-safe
	mu sync.RWMutex
	
	// Callbacks
	onPeerConnected    func(peer.ID)
	onPeerDisconnected func(peer.ID)
	onBlockReceived    func(*entities.Block)
	onTxReceived       func(*entities.Transaction)
}

// P2PConfig contém configurações para o serviço P2P
type P2PConfig struct {
	ListenAddresses []string
	BootstrapPeers  []string
	MaxConnections  int
	EnableMDNS      bool
	EnableDHT       bool
	Namespace       string
}

// P2PStats contém estatísticas do serviço P2P
type P2PStats struct {
	ConnectedPeers     int
	DiscoveredPeers    int
	BlocksReceived     int
	BlocksSent         int
	TransactionsReceived int
	TransactionsSent   int
	BytesReceived      uint64
	BytesSent          uint64
	LastSyncTime       time.Time
	Uptime             time.Duration
	StartTime          time.Time
}

// NewP2PService cria um novo serviço P2P integrado
func NewP2PService(
	chainManager *blockchain.ChainManager,
	consensusEngine *consensus.PoAEngine,
	cryptoService services.CryptographyService,
	config *P2PConfig,
) (*P2PService, error) {
	if config == nil {
		config = DefaultP2PConfig()
	}
	
	// Criar host libp2p
	hostConfig := &LibP2PConfig{
		ListenAddresses: config.ListenAddresses,
		MaxConnections:  config.MaxConnections,
	}
	
	host, err := NewLibP2PHost(hostConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}
	
	// Criar serviço de descoberta
	discoveryConfig := &DiscoveryConfig{
		Namespace:  config.Namespace,
		EnableMDNS: config.EnableMDNS,
		EnableDHT:  config.EnableDHT,
		Interval:   30 * time.Second, // Descoberta a cada 30 segundos
	}
	
	// Converter bootstrap peers (CORREÇÃO CRÍTICA)
	var bootstrapPeers []peer.AddrInfo
	for _, addr := range config.BootstrapPeers {
		// Parse do multiaddr para peer.AddrInfo
		maddr, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			continue // Pular endereços inválidos
		}
		
		// Extrair peer ID do multiaddr (se disponível) ou usar endereço sem ID
		addrInfo, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			// Se não tem peer ID, criar um AddrInfo apenas com endereço
			addrInfo = &peer.AddrInfo{
				Addrs: []multiaddr.Multiaddr{maddr},
			}
		}
		
		bootstrapPeers = append(bootstrapPeers, *addrInfo)
	}
	discoveryConfig.BootstrapPeers = bootstrapPeers
	
	discovery, err := NewDiscoveryService(host.GetHost(), discoveryConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery service: %w", err)
	}
	
	// Criar gerenciador de protocolos
	protocolManager := NewProtocolManager(host)
	
	// Criar serviço de sincronização
	syncService := NewSyncService(chainManager, protocolManager, host)
	
	p2pService := &P2PService{
		host:            host,
		discovery:       discovery,
		protocolManager: protocolManager,
		syncService:     syncService,
		chainManager:    chainManager,
		consensusEngine: consensusEngine,
		cryptoService:   cryptoService,
		nodeID:          host.GetNodeID(),
		config:          config,
		stats:           &P2PStats{StartTime: time.Now()},
	}
	
	// Configurar callbacks e integrações
	p2pService.setupIntegrations()
	
	return p2pService, nil
}

// DefaultP2PConfig retorna configuração padrão
func DefaultP2PConfig() *P2PConfig {
	return &P2PConfig{
		ListenAddresses: []string{
			"/ip4/0.0.0.0/tcp/0",
			"/ip6/::/tcp/0",
		},
		MaxConnections: 50,
		EnableMDNS:     true,
		EnableDHT:      true,
		Namespace:      "peer-vote",
	}
}

// Start inicia todos os serviços P2P
func (p2p *P2PService) Start(ctx context.Context) error {
	p2p.mu.Lock()
	defer p2p.mu.Unlock()
	
	if p2p.isRunning {
		return fmt.Errorf("P2P service already running")
	}
	
	// Iniciar host
	if err := p2p.host.Start(ctx); err != nil {
		return fmt.Errorf("failed to start host: %w", err)
	}
	
	// Iniciar descoberta
	if err := p2p.discovery.Start(ctx); err != nil {
		return fmt.Errorf("failed to start discovery: %w", err)
	}
	
	// Iniciar sincronização
	if err := p2p.syncService.Start(ctx); err != nil {
		return fmt.Errorf("failed to start sync service: %w", err)
	}
	
	p2p.isRunning = true
	p2p.stats.StartTime = time.Now()
	
	return nil
}

// Stop para todos os serviços P2P
func (p2p *P2PService) Stop(ctx context.Context) error {
	p2p.mu.Lock()
	defer p2p.mu.Unlock()
	
	if !p2p.isRunning {
		return fmt.Errorf("P2P service not running")
	}
	
	// Parar serviços na ordem inversa
	p2p.syncService.Stop(ctx)
	p2p.discovery.Stop(ctx)
	p2p.host.Stop(ctx)
	
	p2p.isRunning = false
	
	return nil
}

// setupIntegrations configura integrações entre componentes
func (p2p *P2PService) setupIntegrations() {
	// Callbacks do host
	p2p.host.SetCallbacks(
		p2p.handlePeerConnected,
		p2p.handlePeerDisconnected,
		nil, // stream handler não necessário aqui
	)
	
	// Callbacks de descoberta
	p2p.discovery.SetCallbacks(
		p2p.handlePeerDiscovered,
		p2p.handlePeerLost,
	)
	
	// Callbacks de sincronização
	p2p.syncService.SetCallbacks(
		p2p.handleSyncStart,
		p2p.handleSyncComplete,
		p2p.handleSyncError,
	)
	
	// Handlers de gossip
	p2p.protocolManager.SetTxGossipHandler(p2p.handleTransactionGossip)
	p2p.protocolManager.SetBlockGossipHandler(p2p.handleBlockGossip)
	
	// Handler de consenso
	p2p.protocolManager.SetConsensusHandler(p2p.handleConsensusMessage)
}

// Handlers de eventos

func (p2p *P2PService) handlePeerConnected(peerID peer.ID) {
	p2p.mu.Lock()
	p2p.stats.ConnectedPeers++
	p2p.mu.Unlock()
	
	if p2p.onPeerConnected != nil {
		p2p.onPeerConnected(peerID)
	}
}

func (p2p *P2PService) handlePeerDisconnected(peerID peer.ID) {
	p2p.mu.Lock()
	if p2p.stats.ConnectedPeers > 0 {
		p2p.stats.ConnectedPeers--
	}
	p2p.mu.Unlock()
	
	if p2p.onPeerDisconnected != nil {
		p2p.onPeerDisconnected(peerID)
	}
}

func (p2p *P2PService) handlePeerDiscovered(peerInfo peer.AddrInfo) {
	p2p.mu.Lock()
	p2p.stats.DiscoveredPeers++
	p2p.mu.Unlock()
}

func (p2p *P2PService) handlePeerLost(peerID peer.ID) {
	// Peer perdido - pode implementar lógica de reconexão
}

func (p2p *P2PService) handleSyncStart() {
	// Sincronização iniciada
}

func (p2p *P2PService) handleSyncComplete(blocksAdded int) {
	p2p.mu.Lock()
	p2p.stats.BlocksReceived += blocksAdded
	p2p.stats.LastSyncTime = time.Now()
	p2p.mu.Unlock()
}

func (p2p *P2PService) handleSyncError(err error) {
	// Log do erro de sincronização
	_ = err
}

func (p2p *P2PService) handleTransactionGossip(peerID peer.ID, msg *TxGossipMessage) error {
	// Deserializar transação
	tx, err := p2p.deserializeTransaction(msg.Transaction)
	if err != nil {
		return fmt.Errorf("failed to deserialize transaction: %w", err)
	}
	
	// Adicionar ao pool de consenso se temos consenso ativo
	if p2p.consensusEngine != nil {
		ctx := context.Background()
		if err := p2p.consensusEngine.AddTransaction(ctx, tx); err != nil {
			// Transação pode já existir ou ser inválida
			return nil
		}
	}
	
	p2p.mu.Lock()
	p2p.stats.TransactionsReceived++
	p2p.mu.Unlock()
	
	if p2p.onTxReceived != nil {
		p2p.onTxReceived(tx)
	}
	
	return nil
}

func (p2p *P2PService) handleBlockGossip(peerID peer.ID, msg *BlockGossipMessage) error {
	// Deserializar bloco
	block, err := p2p.deserializeBlock(msg.Block)
	if err != nil {
		return fmt.Errorf("failed to deserialize block: %w", err)
	}
	
	// Tentar adicionar à cadeia
	ctx := context.Background()
	if err := p2p.chainManager.AddBlock(ctx, block); err != nil {
		// Bloco pode já existir ou ser inválido
		return nil
	}
	
	p2p.mu.Lock()
	p2p.stats.BlocksReceived++
	p2p.mu.Unlock()
	
	if p2p.onBlockReceived != nil {
		p2p.onBlockReceived(block)
	}
	
	return nil
}

func (p2p *P2PService) handleConsensusMessage(peerID peer.ID, msgType MessageType, data json.RawMessage) error {
	// Implementação de mensagens de consenso
	// Por enquanto, apenas log
	_ = peerID
	_ = msgType
	_ = data
	
	return nil
}

// Métodos públicos para interação

// BroadcastTransaction propaga uma transação para a rede
func (p2p *P2PService) BroadcastTransaction(ctx context.Context, tx *entities.Transaction) error {
	if err := p2p.protocolManager.GossipTransaction(ctx, tx); err != nil {
		return fmt.Errorf("failed to gossip transaction: %w", err)
	}
	
	p2p.mu.Lock()
	p2p.stats.TransactionsSent++
	p2p.mu.Unlock()
	
	return nil
}

// BroadcastBlock propaga um bloco para a rede
func (p2p *P2PService) BroadcastBlock(ctx context.Context, block *entities.Block) error {
	if err := p2p.protocolManager.GossipBlock(ctx, block); err != nil {
		return fmt.Errorf("failed to gossip block: %w", err)
	}
	
	p2p.mu.Lock()
	p2p.stats.BlocksSent++
	p2p.mu.Unlock()
	
	return nil
}

// ConnectToPeer conecta a um peer específico
func (p2p *P2PService) ConnectToPeer(ctx context.Context, peerAddr string) error {
	return p2p.host.Connect(ctx, peerAddr)
}

// GetConnectedPeers retorna lista de peers conectados
func (p2p *P2PService) GetConnectedPeers() []peer.ID {
	return p2p.host.GetConnectedPeers()
}

// GetPeerCount retorna número de peers conectados
func (p2p *P2PService) GetPeerCount() (int, error) {
	return p2p.host.GetPeerCount(), nil
}

// GetNodeID retorna o ID deste nó
func (p2p *P2PService) GetNodeID() valueobjects.NodeID {
	return p2p.nodeID
}

// GetListenAddresses retorna endereços de escuta
func (p2p *P2PService) GetListenAddresses() []string {
	return p2p.host.GetListenAddresses()
}

// GetMultiAddresses retorna multiaddresses completos
func (p2p *P2PService) GetMultiAddresses() []string {
	return p2p.host.GetMultiAddresses()
}

// GetStats retorna estatísticas do P2P
func (p2p *P2PService) GetStats(ctx context.Context) (*P2PStats, error) {
	p2p.mu.RLock()
	defer p2p.mu.RUnlock()
	
	stats := *p2p.stats
	stats.Uptime = time.Since(stats.StartTime)
	stats.ConnectedPeers = p2p.host.GetPeerCount()
	stats.DiscoveredPeers = p2p.discovery.GetPeerCount()
	
	return &stats, nil
}

// IsRunning verifica se o serviço está rodando
func (p2p *P2PService) IsRunning() bool {
	p2p.mu.RLock()
	defer p2p.mu.RUnlock()
	return p2p.isRunning
}

// DiscoverPeers inicia descoberta de peers
func (p2p *P2PService) DiscoverPeers(ctx context.Context) error {
	if p2p.discovery == nil {
		return fmt.Errorf("discovery service not available")
	}
	
	return p2p.discovery.Start(ctx)
}

// SyncBlockchain sincroniza a blockchain com peers
func (p2p *P2PService) SyncBlockchain(ctx context.Context) error {
	if p2p.syncService == nil {
		return fmt.Errorf("sync service not available")
	}
	
	// Iniciar o serviço de sincronização se não estiver rodando
	if err := p2p.syncService.Start(ctx); err != nil {
		return fmt.Errorf("failed to start sync service: %w", err)
	}
	
	return nil
}

// SetOnBlockReceived define callback para blocos recebidos
func (p2p *P2PService) SetOnBlockReceived(callback func(*entities.Block)) {
	p2p.mu.Lock()
	defer p2p.mu.Unlock()
	p2p.onBlockReceived = callback
}

// SetOnTxReceived define callback para transações recebidas
func (p2p *P2PService) SetOnTxReceived(callback func(*entities.Transaction)) {
	p2p.mu.Lock()
	defer p2p.mu.Unlock()
	p2p.onTxReceived = callback
}

// Ping envia ping para um peer
func (p2p *P2PService) Ping(ctx context.Context, peerID peer.ID) (time.Duration, error) {
	return p2p.protocolManager.Ping(ctx, peerID)
}

// RequestSync solicita sincronização manual
func (p2p *P2PService) RequestSync(peerID peer.ID, startHeight, endHeight uint64) {
	p2p.syncService.RequestSync(peerID, startHeight, endHeight)
}

// SetCallbacks define callbacks para eventos
func (p2p *P2PService) SetCallbacks(
	onPeerConnected, onPeerDisconnected func(peer.ID),
	onBlockReceived func(*entities.Block),
	onTxReceived func(*entities.Transaction),
) {
	p2p.mu.Lock()
	defer p2p.mu.Unlock()
	
	p2p.onPeerConnected = onPeerConnected
	p2p.onPeerDisconnected = onPeerDisconnected
	p2p.onBlockReceived = onBlockReceived
	p2p.onTxReceived = onTxReceived
}

// Métodos auxiliares de serialização/deserialização

func (p2p *P2PService) deserializeTransaction(serialized *SerializedTransaction) (*entities.Transaction, error) {
	// Implementação simplificada - em produção seria mais robusta
	from := valueobjects.NewNodeID(serialized.From)
	to := valueobjects.NewNodeID(serialized.To)
	
	tx := entities.NewTransaction(
		entities.TransactionType(serialized.Type),
		from,
		to,
		[]byte(serialized.Data),
	)
	
	// Definir campos adicionais
	if serialized.Hash != "" {
		hash, err := valueobjects.NewHashFromString(serialized.Hash)
		if err == nil {
			tx.SetHash(hash)
		}
	}
	
	if serialized.ID != "" {
		id, err := valueobjects.NewHashFromString(serialized.ID)
		if err == nil {
			tx.SetID(id)
		}
	}
	
	if serialized.Signature != "" {
		sig, err := valueobjects.NewSignatureFromString(serialized.Signature)
		if err == nil {
			tx.SetSignature(sig)
		}
	}
	
	return tx, nil
}

func (p2p *P2PService) deserializeBlock(serialized *SerializedBlock) (*entities.Block, error) {
	if serialized == nil {
		return nil, fmt.Errorf("serialized block is nil")
	}
	
	// Converter campos básicos
	previousHash, err := valueobjects.NewHashFromString(serialized.PreviousHash)
	if err != nil {
		return nil, fmt.Errorf("invalid previous hash: %w", err)
	}
	
	merkleRoot, err := valueobjects.NewHashFromString(serialized.MerkleRoot)
	if err != nil {
		return nil, fmt.Errorf("invalid merkle root: %w", err)
	}
	
	validator := valueobjects.NewNodeID(serialized.Validator)
	
	// Deserializar transações
	transactions := make([]*entities.Transaction, len(serialized.Transactions))
	for i, serializedTx := range serialized.Transactions {
		tx, err := p2p.deserializeTransaction(serializedTx)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize transaction %d: %w", i, err)
		}
		transactions[i] = tx
	}
	
	// Criar bloco usando o BlockBuilder
	blockBuilder := blockchain.NewBlockBuilder(p2p.cryptoService)
	block, err := blockBuilder.BuildBlock(
		context.Background(),
		serialized.Index,
		previousHash,
		transactions,
		validator,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build block: %w", err)
	}
	
	// Definir merkle root (se diferente do calculado)
	block.SetMerkleRoot(merkleRoot)
	
	// Definir assinatura se presente
	if serialized.Signature != "" {
		signature, err := valueobjects.NewSignatureFromString(serialized.Signature)
		if err != nil {
			return nil, fmt.Errorf("invalid signature: %w", err)
		}
		block.SetSignature(signature)
	}
	
	return block, nil
}

// GetDiscoveryStats retorna estatísticas de descoberta
func (p2p *P2PService) GetDiscoveryStats() *DiscoveryStats {
	return p2p.discovery.GetDiscoveryStats()
}

// GetSyncStats retorna estatísticas de sincronização
func (p2p *P2PService) GetSyncStats() *SyncStats {
	return p2p.syncService.GetSyncStats()
}

