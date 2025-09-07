package network

import (
	"context"
	"fmt"
	"sync"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/libp2p/go-libp2p/p2p/discovery/util"
)

// DiscoveryService gerencia descoberta de pares usando mDNS e DHT
type DiscoveryService struct {
	host         host.Host
	dht          *dht.IpfsDHT
	mdnsService  mdns.Service
	routingDisc  *drouting.RoutingDiscovery
	
	// Configurações
	namespace    string
	mdnsEnabled  bool
	dhtEnabled   bool
	interval     time.Duration
	
	// Estado
	isRunning    bool
	discoveredPeers map[peer.ID]*DiscoveredPeer
	
	// Canais
	peerChan     chan peer.AddrInfo
	stopChan     chan struct{}
	
	// Mutex para operações thread-safe
	mu sync.RWMutex
	
	// Callbacks
	onPeerDiscovered func(peer.AddrInfo)
	onPeerLost       func(peer.ID)
}

// DiscoveredPeer contém informações sobre um peer descoberto
type DiscoveredPeer struct {
	AddrInfo     peer.AddrInfo
	DiscoveredAt time.Time
	LastSeen     time.Time
	Source       string // "mdns" ou "dht"
	Connected    bool
}

// DiscoveryConfig contém configurações para descoberta
type DiscoveryConfig struct {
	Namespace   string
	EnableMDNS  bool
	EnableDHT   bool
	Interval    time.Duration
	BootstrapPeers []peer.AddrInfo
}

// NewDiscoveryService cria um novo serviço de descoberta
func NewDiscoveryService(h host.Host, config *DiscoveryConfig) (*DiscoveryService, error) {
	if config == nil {
		config = DefaultDiscoveryConfig()
	}

	ds := &DiscoveryService{
		host:            h,
		namespace:       config.Namespace,
		mdnsEnabled:     config.EnableMDNS,
		dhtEnabled:      config.EnableDHT,
		interval:        config.Interval,
		discoveredPeers: make(map[peer.ID]*DiscoveredPeer),
		peerChan:        make(chan peer.AddrInfo, 100),
		stopChan:        make(chan struct{}),
	}

	// Configurar DHT se habilitado
	if config.EnableDHT {
		var err error
		ds.dht, err = dht.New(context.Background(), h, dht.Mode(dht.ModeAuto))
		if err != nil {
			return nil, fmt.Errorf("failed to create DHT: %w", err)
		}

		ds.routingDisc = drouting.NewRoutingDiscovery(ds.dht)

		// Bootstrap DHT se peers fornecidos
		if len(config.BootstrapPeers) > 0 {
			go ds.bootstrapDHT(config.BootstrapPeers)
		}
	}

	return ds, nil
}

// DefaultDiscoveryConfig retorna configuração padrão
func DefaultDiscoveryConfig() *DiscoveryConfig {
	return &DiscoveryConfig{
		Namespace:  "peer-vote",
		EnableMDNS: true,
		EnableDHT:  true,
		Interval:   time.Second * 30,
	}
}

// Start inicia o serviço de descoberta
func (ds *DiscoveryService) Start(ctx context.Context) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if ds.isRunning {
		return fmt.Errorf("discovery service already running")
	}

	// Iniciar mDNS se habilitado
	if ds.mdnsEnabled {
		if err := ds.startMDNS(ctx); err != nil {
			return fmt.Errorf("failed to start mDNS: %w", err)
		}
	}

	// Iniciar DHT se habilitado
	if ds.dhtEnabled && ds.dht != nil {
		if err := ds.dht.Bootstrap(ctx); err != nil {
			return fmt.Errorf("failed to bootstrap DHT: %w", err)
		}
	}

	ds.isRunning = true

	// Iniciar loops de descoberta
	go ds.discoveryLoop(ctx)
	go ds.peerHandler(ctx)

	return nil
}

// Stop para o serviço de descoberta
func (ds *DiscoveryService) Stop(ctx context.Context) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if !ds.isRunning {
		return fmt.Errorf("discovery service not running")
	}

	ds.isRunning = false
	close(ds.stopChan)

	// Parar mDNS
	if ds.mdnsService != nil {
		ds.mdnsService.Close()
	}

	// Parar DHT
	if ds.dht != nil {
		ds.dht.Close()
	}

	return nil
}

// startMDNS inicia o serviço mDNS
func (ds *DiscoveryService) startMDNS(ctx context.Context) error {
	ds.mdnsService = mdns.NewMdnsService(ds.host, ds.namespace, &mdnsNotifee{ds: ds})
	return nil
}

// mdnsNotifee implementa mdns.Notifee para receber descobertas mDNS
type mdnsNotifee struct {
	ds *DiscoveryService
}

// HandlePeerFound é chamado quando um peer é descoberto via mDNS
func (n *mdnsNotifee) HandlePeerFound(pi peer.AddrInfo) {
	n.ds.handlePeerDiscovered(pi, "mdns")
}

// handlePeerDiscovered processa um peer descoberto
func (ds *DiscoveryService) handlePeerDiscovered(pi peer.AddrInfo, source string) {
	// Ignorar nós próprios
	if pi.ID == ds.host.ID() {
		return
	}

	ds.mu.Lock()
	defer ds.mu.Unlock()

	now := time.Now()
	
	if existing, exists := ds.discoveredPeers[pi.ID]; exists {
		// Atualizar peer existente
		existing.LastSeen = now
		existing.AddrInfo = pi // Atualizar endereços
	} else {
		// Novo peer descoberto
		ds.discoveredPeers[pi.ID] = &DiscoveredPeer{
			AddrInfo:     pi,
			DiscoveredAt: now,
			LastSeen:     now,
			Source:       source,
			Connected:    false,
		}

		// Notificar via canal
		select {
		case ds.peerChan <- pi:
		default:
			// Canal cheio, pular
		}
	}
}

// discoveryLoop executa descoberta periódica via DHT
func (ds *DiscoveryService) discoveryLoop(ctx context.Context) {
	if !ds.dhtEnabled || ds.routingDisc == nil {
		return
	}

	ticker := time.NewTicker(ds.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ds.stopChan:
			return
		case <-ticker.C:
			ds.discoverPeersViaDHT(ctx)
		}
	}
}

// discoverPeersViaDHT descobre peers via DHT
func (ds *DiscoveryService) discoverPeersViaDHT(ctx context.Context) {
	// Anunciar nossa presença
	util.Advertise(ctx, ds.routingDisc, ds.namespace)

	// Descobrir outros peers
	peerChan, err := ds.routingDisc.FindPeers(ctx, ds.namespace)
	if err != nil {
		return
	}

	// Processar peers descobertos
	for peer := range peerChan {
		ds.handlePeerDiscovered(peer, "dht")
	}
}

// peerHandler processa peers descobertos
func (ds *DiscoveryService) peerHandler(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-ds.stopChan:
			return
		case pi := <-ds.peerChan:
			ds.processPeer(ctx, pi)
		}
	}
}

// processPeer processa um peer descoberto
func (ds *DiscoveryService) processPeer(ctx context.Context, pi peer.AddrInfo) {
	// Callback de peer descoberto
	if ds.onPeerDiscovered != nil {
		ds.onPeerDiscovered(pi)
	}

	// Tentar conectar automaticamente
	go ds.tryConnect(ctx, pi)
}

// tryConnect tenta conectar a um peer
func (ds *DiscoveryService) tryConnect(ctx context.Context, pi peer.AddrInfo) {
	// Timeout para conexão
	connectCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	err := ds.host.Connect(connectCtx, pi)
	
	ds.mu.Lock()
	defer ds.mu.Unlock()
	
	if peer, exists := ds.discoveredPeers[pi.ID]; exists {
		peer.Connected = (err == nil)
	}
}

// bootstrapDHT faz bootstrap do DHT com peers conhecidos
func (ds *DiscoveryService) bootstrapDHT(bootstrapPeers []peer.AddrInfo) {
	ctx := context.Background()
	
	for _, pi := range bootstrapPeers {
		if pi.ID == ds.host.ID() {
			continue
		}

		// Conectar ao peer de bootstrap
		connectCtx, cancel := context.WithTimeout(ctx, time.Second*30)
		err := ds.host.Connect(connectCtx, pi)
		cancel()
		
		if err == nil {
			ds.handlePeerDiscovered(pi, "bootstrap")
		}
	}

	// Aguardar um pouco antes de fazer bootstrap do DHT
	time.Sleep(time.Second * 2)
	
	if ds.dht != nil {
		ds.dht.Bootstrap(ctx)
	}
}

// GetDiscoveredPeers retorna lista de peers descobertos
func (ds *DiscoveryService) GetDiscoveredPeers() map[peer.ID]*DiscoveredPeer {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	result := make(map[peer.ID]*DiscoveredPeer)
	for id, peer := range ds.discoveredPeers {
		result[id] = &DiscoveredPeer{
			AddrInfo:     peer.AddrInfo,
			DiscoveredAt: peer.DiscoveredAt,
			LastSeen:     peer.LastSeen,
			Source:       peer.Source,
			Connected:    peer.Connected,
		}
	}

	return result
}

// GetPeerCount retorna número de peers descobertos
func (ds *DiscoveryService) GetPeerCount() int {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	return len(ds.discoveredPeers)
}

// GetConnectedPeerCount retorna número de peers conectados
func (ds *DiscoveryService) GetConnectedPeerCount() int {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	count := 0
	for _, peer := range ds.discoveredPeers {
		if peer.Connected {
			count++
		}
	}

	return count
}

// FindPeer procura um peer específico
func (ds *DiscoveryService) FindPeer(ctx context.Context, peerID peer.ID) (*peer.AddrInfo, error) {
	if !ds.dhtEnabled || ds.dht == nil {
		return nil, fmt.Errorf("DHT not enabled")
	}

	addrInfo, err := ds.dht.FindPeer(ctx, peerID)
	if err != nil {
		return nil, err
	}
	
	return &addrInfo, nil
}

// Provide anuncia que temos um conteúdo específico
func (ds *DiscoveryService) Provide(ctx context.Context, key string) error {
	if !ds.dhtEnabled || ds.dht == nil {
		return fmt.Errorf("DHT not enabled")
	}

	// Implementação simplificada - em produção usaria CID apropriado
	return fmt.Errorf("provide functionality not implemented yet")
}

// FindProviders encontra provedores de um conteúdo
func (ds *DiscoveryService) FindProviders(ctx context.Context, key string) ([]peer.AddrInfo, error) {
	if !ds.dhtEnabled || ds.dht == nil {
		return nil, fmt.Errorf("DHT not enabled")
	}

	// Implementação simplificada - em produção usaria CID apropriado
	return nil, fmt.Errorf("find providers functionality not implemented yet")
}

// SetCallbacks define callbacks para eventos
func (ds *DiscoveryService) SetCallbacks(onDiscovered func(peer.AddrInfo), onLost func(peer.ID)) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	ds.onPeerDiscovered = onDiscovered
	ds.onPeerLost = onLost
}

// CleanupStaleConnections remove peers que não foram vistos recentemente
func (ds *DiscoveryService) CleanupStaleConnections(maxAge time.Duration) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	now := time.Now()
	var toRemove []peer.ID

	for peerID, peer := range ds.discoveredPeers {
		if now.Sub(peer.LastSeen) > maxAge {
			toRemove = append(toRemove, peerID)
		}
	}

	for _, peerID := range toRemove {
		delete(ds.discoveredPeers, peerID)
		
		if ds.onPeerLost != nil {
			ds.onPeerLost(peerID)
		}
	}
}

// GetDiscoveryStats retorna estatísticas de descoberta
func (ds *DiscoveryService) GetDiscoveryStats() *DiscoveryStats {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	stats := &DiscoveryStats{
		TotalPeers:     len(ds.discoveredPeers),
		ConnectedPeers: 0,
		MDNSEnabled:    ds.mdnsEnabled,
		DHTEnabled:     ds.dhtEnabled,
		IsRunning:      ds.isRunning,
		Namespace:      ds.namespace,
		SourceStats:    make(map[string]int),
	}

	for _, peer := range ds.discoveredPeers {
		if peer.Connected {
			stats.ConnectedPeers++
		}
		stats.SourceStats[peer.Source]++
	}

	return stats
}

// DiscoveryStats contém estatísticas de descoberta
type DiscoveryStats struct {
	TotalPeers     int
	ConnectedPeers int
	MDNSEnabled    bool
	DHTEnabled     bool
	IsRunning      bool
	Namespace      string
	SourceStats    map[string]int
}

// RefreshPeerConnections atualiza status de conexão dos peers
func (ds *DiscoveryService) RefreshPeerConnections() {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	for peerID, peer := range ds.discoveredPeers {
		connected := ds.host.Network().Connectedness(peerID) == network.Connected
		peer.Connected = connected
		
		if connected {
			peer.LastSeen = time.Now()
		}
	}
}

// GetDHT retorna a instância DHT (para uso avançado)
func (ds *DiscoveryService) GetDHT() routing.Routing {
	return ds.dht
}

// IsRunning verifica se o serviço está rodando
func (ds *DiscoveryService) IsRunning() bool {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	return ds.isRunning
}
