package network

import (
	"context"

	"github.com/matscats/peer-vote/peer-vote/domain/entities"
	"github.com/matscats/peer-vote/peer-vote/domain/services"
	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
)

// NetworkAdapter adapta P2PService para a interface NetworkService
type NetworkAdapter struct {
	p2pService *P2PService
}

// NewNetworkAdapter cria um novo adapter
func NewNetworkAdapter(p2pService *P2PService) *NetworkAdapter {
	return &NetworkAdapter{
		p2pService: p2pService,
	}
}

// Start inicia o serviço de rede
func (n *NetworkAdapter) Start(ctx context.Context) error {
	return n.p2pService.Start(ctx)
}

// Stop para o serviço de rede
func (n *NetworkAdapter) Stop(ctx context.Context) error {
	return n.p2pService.Stop(ctx)
}

// Connect conecta a um peer específico
func (n *NetworkAdapter) Connect(ctx context.Context, peerID valueobjects.NodeID, address string) error {
	// TODO: Implementar quando P2PService tiver método ConnectToPeer
	return nil
}

// Disconnect desconecta de um peer
func (n *NetworkAdapter) Disconnect(ctx context.Context, peerID valueobjects.NodeID) error {
	// TODO: Implementar quando P2PService tiver método DisconnectFromPeer
	return nil
}

// BroadcastBlock transmite um bloco para todos os peers
func (n *NetworkAdapter) BroadcastBlock(ctx context.Context, block *entities.Block) error {
	return n.p2pService.BroadcastBlock(ctx, block)
}

// BroadcastTransaction transmite uma transação para todos os peers
func (n *NetworkAdapter) BroadcastTransaction(ctx context.Context, tx *entities.Transaction) error {
	return n.p2pService.BroadcastTransaction(ctx, tx)
}

// SendBlockToPeer envia um bloco para um peer específico
func (n *NetworkAdapter) SendBlockToPeer(ctx context.Context, peerID valueobjects.NodeID, block *entities.Block) error {
	// P2PService não tem método específico, usar broadcast como fallback
	return n.p2pService.BroadcastBlock(ctx, block)
}

// SendTransactionToPeer envia uma transação para um peer específico
func (n *NetworkAdapter) SendTransactionToPeer(ctx context.Context, peerID valueobjects.NodeID, tx *entities.Transaction) error {
	// P2PService não tem método específico, usar broadcast como fallback
	return n.p2pService.BroadcastTransaction(ctx, tx)
}

// RequestBlock solicita um bloco específico de um peer
func (n *NetworkAdapter) RequestBlock(ctx context.Context, peerID valueobjects.NodeID, blockHash valueobjects.Hash) (*entities.Block, error) {
	// TODO: Implementar quando P2PService tiver método RequestBlock
	return nil, nil
}

// RequestBlockRange solicita uma faixa de blocos de um peer
func (n *NetworkAdapter) RequestBlockRange(ctx context.Context, peerID valueobjects.NodeID, startIndex, endIndex uint64) ([]*entities.Block, error) {
	// TODO: Implementar quando P2PService tiver método RequestBlockRange
	return nil, nil
}

// GetConnectedPeers retorna a lista de peers conectados
func (n *NetworkAdapter) GetConnectedPeers(ctx context.Context) ([]services.PeerInfo, error) {
	connectedPeers := n.p2pService.GetConnectedPeers()
	
	peers := make([]services.PeerInfo, len(connectedPeers))
	for i, peerID := range connectedPeers {
		peers[i] = services.PeerInfo{
			ID:        valueobjects.NewNodeID(string(peerID)),
			Address:   "", // TODO: Obter endereço real
			Connected: true,
			LastSeen:  valueobjects.Now(),
		}
	}
	
	return peers, nil
}

// GetPeerCount retorna o número de peers conectados
func (n *NetworkAdapter) GetPeerCount(ctx context.Context) (int, error) {
	return len(n.p2pService.GetConnectedPeers()), nil
}

// DiscoverPeers descobre novos peers na rede
func (n *NetworkAdapter) DiscoverPeers(ctx context.Context) error {
	// P2PService faz descoberta automaticamente
	return nil
}

// GetNodeID retorna o ID deste nó
func (n *NetworkAdapter) GetNodeID() valueobjects.NodeID {
	return n.p2pService.GetNodeID()
}

// GetListenAddresses retorna os endereços que este nó está escutando
func (n *NetworkAdapter) GetListenAddresses() []string {
	return n.p2pService.GetListenAddresses()
}

// SyncBlockchain sincroniza a blockchain com os peers
func (n *NetworkAdapter) SyncBlockchain(ctx context.Context) error {
	// TODO: Implementar quando P2PService tiver método SyncWithPeers
	return nil
}

// HandleIncomingBlock processa um bloco recebido
func (n *NetworkAdapter) HandleIncomingBlock(ctx context.Context, block *entities.Block, fromPeer valueobjects.NodeID) error {
	// P2PService já tem handlers internos
	return nil
}

// HandleIncomingTransaction processa uma transação recebida
func (n *NetworkAdapter) HandleIncomingTransaction(ctx context.Context, tx *entities.Transaction, fromPeer valueobjects.NodeID) error {
	// P2PService já tem handlers internos
	return nil
}

// RegisterBlockHandler registra um handler para blocos recebidos
func (n *NetworkAdapter) RegisterBlockHandler(handler services.BlockHandler) {
	// TODO: Implementar se necessário
}

// RegisterTransactionHandler registra um handler para transações recebidas
func (n *NetworkAdapter) RegisterTransactionHandler(handler services.TransactionHandler) {
	// TODO: Implementar se necessário
}

// GetNetworkStatus retorna o status da rede
func (n *NetworkAdapter) GetNetworkStatus(ctx context.Context) (services.NetworkStatus, error) {
	stats := n.p2pService.GetStats()
	
	return services.NetworkStatus{
		IsRunning:      n.p2pService.IsRunning(),
		NodeID:         n.p2pService.GetNodeID(),
		PeerCount:      stats.ConnectedPeers,
		ListenAddrs:    n.p2pService.GetListenAddresses(),
		LastSync:       valueobjects.NewTimestamp(stats.LastSyncTime),
		SyncInProgress: false, // TODO: Implementar se necessário
	}, nil
}
