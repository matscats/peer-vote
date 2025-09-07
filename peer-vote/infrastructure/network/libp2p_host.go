package network

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/multiformats/go-multiaddr"

	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
)

// LibP2PHost encapsula um host libp2p com configurações específicas
type LibP2PHost struct {
	host         host.Host
	nodeID       valueobjects.NodeID
	listenAddrs  []multiaddr.Multiaddr
	protocols    map[protocol.ID]ProtocolHandler
	
	// Configurações
	maxConnections int
	connTimeout    time.Duration
	
	// Estado
	isRunning bool
	
	// Mutex para operações thread-safe
	mu sync.RWMutex
	
	// Callbacks
	onPeerConnect    func(peer.ID)
	onPeerDisconnect func(peer.ID)
	onStreamOpen     func(network.Stream)
}

// ProtocolHandler define um handler para protocolos
type ProtocolHandler = network.StreamHandler

// LibP2PConfig contém configurações para o host libp2p
type LibP2PConfig struct {
	ListenAddresses []string
	PrivateKey      crypto.PrivKey
	MaxConnections  int
	ConnTimeout     time.Duration
	EnableRelay     bool
	EnableAutoRelay bool
	EnableHolePunch bool
}

// NewLibP2PHost cria um novo host libp2p
func NewLibP2PHost(config *LibP2PConfig) (*LibP2PHost, error) {
	if config == nil {
		config = DefaultLibP2PConfig()
	}

	// Configurar opções do libp2p
	opts := []libp2p.Option{
		libp2p.Identity(config.PrivateKey),
		libp2p.Security(noise.ID, noise.New),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.DefaultMuxers,
	}

	// Adicionar endereços de escuta
	if len(config.ListenAddresses) > 0 {
		listenAddrs := make([]multiaddr.Multiaddr, 0, len(config.ListenAddresses))
		for _, addr := range config.ListenAddresses {
			maddr, err := multiaddr.NewMultiaddr(addr)
			if err != nil {
				return nil, fmt.Errorf("invalid listen address %s: %w", addr, err)
			}
			listenAddrs = append(listenAddrs, maddr)
		}
		opts = append(opts, libp2p.ListenAddrs(listenAddrs...))
	}

	// Configurar limites de conexão
	if config.MaxConnections > 0 {
		opts = append(opts, libp2p.ConnectionManager(NewConnectionManager(config.MaxConnections)))
	}

	// Configurar relay se habilitado
	if config.EnableRelay {
		opts = append(opts, libp2p.EnableRelay())
	}

	if config.EnableAutoRelay {
		opts = append(opts, libp2p.EnableAutoRelayWithStaticRelays(nil))
	}

	if config.EnableHolePunch {
		opts = append(opts, libp2p.EnableHolePunching())
	}

	// Criar host
	h, err := libp2p.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	// Converter peer ID para NodeID
	nodeID := valueobjects.NewNodeID(h.ID().String())

	libp2pHost := &LibP2PHost{
		host:           h,
		nodeID:         nodeID,
		listenAddrs:    h.Addrs(),
		protocols:      make(map[protocol.ID]ProtocolHandler),
		maxConnections: config.MaxConnections,
		connTimeout:    config.ConnTimeout,
		isRunning:      false,
	}

	// Configurar handlers de rede
	h.Network().Notify(libp2pHost)

	return libp2pHost, nil
}

// DefaultLibP2PConfig retorna configuração padrão
func DefaultLibP2PConfig() *LibP2PConfig {
	// Gerar chave privada
	privKey, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 2048, rand.Reader)
	if err != nil {
		panic(fmt.Sprintf("failed to generate key pair: %v", err))
	}

	return &LibP2PConfig{
		ListenAddresses: []string{
			"/ip4/0.0.0.0/tcp/0",
			"/ip6/::/tcp/0",
		},
		PrivateKey:      privKey,
		MaxConnections:  100,
		ConnTimeout:     time.Second * 30,
		EnableRelay:     false,
		EnableAutoRelay: false,
		EnableHolePunch: true,
	}
}

// Start inicia o host libp2p
func (lh *LibP2PHost) Start(ctx context.Context) error {
	lh.mu.Lock()
	defer lh.mu.Unlock()

	if lh.isRunning {
		return fmt.Errorf("host already running")
	}

	lh.isRunning = true
	return nil
}

// Stop para o host libp2p
func (lh *LibP2PHost) Stop(ctx context.Context) error {
	lh.mu.Lock()
	defer lh.mu.Unlock()

	if !lh.isRunning {
		return fmt.Errorf("host not running")
	}

	lh.isRunning = false
	return lh.host.Close()
}

// GetHost retorna o host libp2p subjacente
func (lh *LibP2PHost) GetHost() host.Host {
	return lh.host
}

// GetNodeID retorna o ID do nó
func (lh *LibP2PHost) GetNodeID() valueobjects.NodeID {
	return lh.nodeID
}

// GetPeerID retorna o peer ID do libp2p
func (lh *LibP2PHost) GetPeerID() peer.ID {
	return lh.host.ID()
}

// GetListenAddresses retorna os endereços de escuta
func (lh *LibP2PHost) GetListenAddresses() []string {
	addrs := make([]string, len(lh.listenAddrs))
	for i, addr := range lh.listenAddrs {
		addrs[i] = addr.String()
	}
	return addrs
}

// GetMultiAddresses retorna os multiaddresses completos
func (lh *LibP2PHost) GetMultiAddresses() []string {
	var addrs []string
	for _, addr := range lh.host.Addrs() {
		fullAddr := addr.Encapsulate(multiaddr.StringCast("/p2p/" + lh.host.ID().String()))
		addrs = append(addrs, fullAddr.String())
	}
	return addrs
}

// Connect conecta a um peer específico
func (lh *LibP2PHost) Connect(ctx context.Context, peerAddr string) error {
	maddr, err := multiaddr.NewMultiaddr(peerAddr)
	if err != nil {
		return fmt.Errorf("invalid peer address: %w", err)
	}

	peerInfo, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return fmt.Errorf("failed to parse peer info: %w", err)
	}

	// Configurar timeout se especificado
	connectCtx := ctx
	if lh.connTimeout > 0 {
		var cancel context.CancelFunc
		connectCtx, cancel = context.WithTimeout(ctx, lh.connTimeout)
		defer cancel()
	}

	return lh.host.Connect(connectCtx, *peerInfo)
}

// Disconnect desconecta de um peer
func (lh *LibP2PHost) Disconnect(ctx context.Context, peerID peer.ID) error {
	return lh.host.Network().ClosePeer(peerID)
}

// GetConnectedPeers retorna lista de peers conectados
func (lh *LibP2PHost) GetConnectedPeers() []peer.ID {
	return lh.host.Network().Peers()
}

// GetPeerCount retorna número de peers conectados
func (lh *LibP2PHost) GetPeerCount() int {
	return len(lh.host.Network().Peers())
}

// IsConnected verifica se está conectado a um peer
func (lh *LibP2PHost) IsConnected(peerID peer.ID) bool {
	return lh.host.Network().Connectedness(peerID) == network.Connected
}

// RegisterProtocol registra um handler para um protocolo
func (lh *LibP2PHost) RegisterProtocol(protocolID protocol.ID, handler ProtocolHandler) {
	lh.mu.Lock()
	defer lh.mu.Unlock()

	lh.protocols[protocolID] = handler
	lh.host.SetStreamHandler(protocolID, handler)
}

// UnregisterProtocol remove um handler de protocolo
func (lh *LibP2PHost) UnregisterProtocol(protocolID protocol.ID) {
	lh.mu.Lock()
	defer lh.mu.Unlock()

	delete(lh.protocols, protocolID)
	lh.host.RemoveStreamHandler(protocolID)
}

// NewStream cria um novo stream para um peer
func (lh *LibP2PHost) NewStream(ctx context.Context, peerID peer.ID, protocolID protocol.ID) (network.Stream, error) {
	return lh.host.NewStream(ctx, peerID, protocolID)
}

// SetCallbacks define callbacks para eventos de rede
func (lh *LibP2PHost) SetCallbacks(onConnect, onDisconnect func(peer.ID), onStreamOpen func(network.Stream)) {
	lh.mu.Lock()
	defer lh.mu.Unlock()

	lh.onPeerConnect = onConnect
	lh.onPeerDisconnect = onDisconnect
	lh.onStreamOpen = onStreamOpen
}

// Implementação da interface network.Notifiee

// Listen é chamado quando começamos a escutar em um endereço
func (lh *LibP2PHost) Listen(n network.Network, addr multiaddr.Multiaddr) {}

// ListenClose é chamado quando paramos de escutar em um endereço
func (lh *LibP2PHost) ListenClose(n network.Network, addr multiaddr.Multiaddr) {}

// Connected é chamado quando uma conexão é estabelecida
func (lh *LibP2PHost) Connected(n network.Network, conn network.Conn) {
	if lh.onPeerConnect != nil {
		lh.onPeerConnect(conn.RemotePeer())
	}
}

// Disconnected é chamado quando uma conexão é fechada
func (lh *LibP2PHost) Disconnected(n network.Network, conn network.Conn) {
	if lh.onPeerDisconnect != nil {
		lh.onPeerDisconnect(conn.RemotePeer())
	}
}

// OpenedStream é chamado quando um stream é aberto
func (lh *LibP2PHost) OpenedStream(n network.Network, stream network.Stream) {
	if lh.onStreamOpen != nil {
		lh.onStreamOpen(stream)
	}
}

// ClosedStream é chamado quando um stream é fechado
func (lh *LibP2PHost) ClosedStream(n network.Network, stream network.Stream) {}

// GetNetworkStats retorna estatísticas da rede
func (lh *LibP2PHost) GetNetworkStats() *NetworkStats {
	peers := lh.GetConnectedPeers()
	
	stats := &NetworkStats{
		PeerCount:     len(peers),
		ListenAddrs:   lh.GetListenAddresses(),
		MultiAddrs:    lh.GetMultiAddresses(),
		IsRunning:     lh.isRunning,
		NodeID:        lh.nodeID.String(),
		PeerID:        lh.host.ID().String(),
		ConnectedPeers: make([]string, len(peers)),
	}

	for i, peerID := range peers {
		stats.ConnectedPeers[i] = peerID.String()
	}

	return stats
}

// NetworkStats contém estatísticas da rede
type NetworkStats struct {
	PeerCount      int
	ListenAddrs    []string
	MultiAddrs     []string
	IsRunning      bool
	NodeID         string
	PeerID         string
	ConnectedPeers []string
}

// BroadcastToConnectedPeers envia dados para todos os peers conectados
func (lh *LibP2PHost) BroadcastToConnectedPeers(ctx context.Context, protocolID protocol.ID, data []byte) error {
	peers := lh.GetConnectedPeers()
	
	for _, peerID := range peers {
		go func(pid peer.ID) {
			stream, err := lh.NewStream(ctx, pid, protocolID)
			if err != nil {
				return // Ignorar erros individuais
			}
			defer stream.Close()
			
			stream.Write(data)
		}(peerID)
	}
	
	return nil
}

// SendToPeer envia dados para um peer específico
func (lh *LibP2PHost) SendToPeer(ctx context.Context, peerID peer.ID, protocolID protocol.ID, data []byte) error {
	stream, err := lh.NewStream(ctx, peerID, protocolID)
	if err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}
	defer stream.Close()
	
	_, err = stream.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}
	
	return nil
}

// GetPeerInfo retorna informações sobre um peer
func (lh *LibP2PHost) GetPeerInfo(peerID peer.ID) *PeerInfo {
	if !lh.IsConnected(peerID) {
		return nil
	}

	conns := lh.host.Network().ConnsToPeer(peerID)
	if len(conns) == 0 {
		return nil
	}

	conn := conns[0]
	
	return &PeerInfo{
		PeerID:      peerID.String(),
		RemoteAddr:  conn.RemoteMultiaddr().String(),
		LocalAddr:   conn.LocalMultiaddr().String(),
		Direction:   conn.Stat().Direction.String(),
		Opened:      conn.Stat().Opened,
		IsConnected: true,
	}
}

// PeerInfo contém informações sobre um peer
type PeerInfo struct {
	PeerID      string
	RemoteAddr  string
	LocalAddr   string
	Direction   string
	Opened      time.Time
	IsConnected bool
}
