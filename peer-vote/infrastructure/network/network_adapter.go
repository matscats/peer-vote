package network

import (
	"context"
	"fmt"

	"github.com/matscats/peer-vote/peer-vote/domain/entities"
	"github.com/matscats/peer-vote/peer-vote/domain/services"
	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
)

// NetworkAdapter implementa a interface services.NetworkService
// Aplica o padrão Adapter para adaptar P2PService à interface do domínio
// Segue DIP: permite que o domínio dependa da abstração NetworkService
type NetworkAdapter struct {
	p2pService *P2PService
}

// NewNetworkAdapter cria um novo adapter para P2PService
func NewNetworkAdapter(p2pService *P2PService) services.NetworkService {
	return &NetworkAdapter{
		p2pService: p2pService,
	}
}

// Start inicia o serviço de rede
func (na *NetworkAdapter) Start(ctx context.Context) error {
	return na.p2pService.Start(ctx)
}

// Stop para o serviço de rede
func (na *NetworkAdapter) Stop(ctx context.Context) error {
	return na.p2pService.Stop(ctx)
}

// Connect conecta a um peer específico
func (na *NetworkAdapter) Connect(ctx context.Context, peerID valueobjects.NodeID, address string) error {
	// Implementação simplificada - P2PService usa descoberta automática
	return na.p2pService.DiscoverPeers(ctx)
}

// Disconnect desconecta de um peer
func (na *NetworkAdapter) Disconnect(ctx context.Context, peerID valueobjects.NodeID) error {
	// Implementação simplificada - libp2p gerencia conexões automaticamente
	return nil
}

// BroadcastBlock transmite um bloco para todos os peers
func (na *NetworkAdapter) BroadcastBlock(ctx context.Context, block *entities.Block) error {
	return na.p2pService.BroadcastBlock(ctx, block)
}

// BroadcastTransaction transmite uma transação para todos os peers
func (na *NetworkAdapter) BroadcastTransaction(ctx context.Context, tx *entities.Transaction) error {
	return na.p2pService.BroadcastTransaction(ctx, tx)
}

// SendBlockToPeer envia um bloco para um peer específico
func (na *NetworkAdapter) SendBlockToPeer(ctx context.Context, peerID valueobjects.NodeID, block *entities.Block) error {
	// Por enquanto usa broadcast - pode ser otimizado depois
	return na.p2pService.BroadcastBlock(ctx, block)
}

// SendTransactionToPeer envia uma transação para um peer específico
func (na *NetworkAdapter) SendTransactionToPeer(ctx context.Context, peerID valueobjects.NodeID, tx *entities.Transaction) error {
	// Por enquanto usa broadcast - pode ser otimizado depois
	return na.p2pService.BroadcastTransaction(ctx, tx)
}

// RequestBlock solicita um bloco específico de um peer
func (na *NetworkAdapter) RequestBlock(ctx context.Context, peerID valueobjects.NodeID, blockHash valueobjects.Hash) (*entities.Block, error) {
	return nil, fmt.Errorf("RequestBlock functionality removed - use broadcast mechanism instead")
}

// RequestBlockRange solicita uma faixa de blocos de um peer
func (na *NetworkAdapter) RequestBlockRange(ctx context.Context, peerID valueobjects.NodeID, startIndex, endIndex uint64) ([]*entities.Block, error) {
	return nil, fmt.Errorf("RequestBlockRange functionality removed - use broadcast mechanism instead")
}

// GetConnectedPeers retorna a lista de peers conectados
func (na *NetworkAdapter) GetConnectedPeers(ctx context.Context) ([]services.PeerInfo, error) {
	stats, err := na.p2pService.GetStats(ctx)
	if err != nil {
		return nil, err
	}
	
	// Conversão simplificada - retorna informação básica
	peers := make([]services.PeerInfo, 0, stats.ConnectedPeers)
	
	return peers, nil
}

// GetPeerCount retorna o número de peers conectados
func (na *NetworkAdapter) GetPeerCount(ctx context.Context) (int, error) {
	stats, err := na.p2pService.GetStats(ctx)
	if err != nil {
		return 0, err
	}
	
	return stats.ConnectedPeers, nil
}

// DiscoverPeers descobre novos peers na rede
func (na *NetworkAdapter) DiscoverPeers(ctx context.Context) error {
	return na.p2pService.DiscoverPeers(ctx)
}

// GetNodeID retorna o ID deste nó
func (na *NetworkAdapter) GetNodeID() valueobjects.NodeID {
	return na.p2pService.GetNodeID()
}

// GetListenAddresses retorna os endereços que este nó está escutando
func (na *NetworkAdapter) GetListenAddresses() []string {
	return na.p2pService.GetListenAddresses()
}

// SyncBlockchain sincroniza a blockchain com os peers
func (na *NetworkAdapter) SyncBlockchain(ctx context.Context) error {
	return na.p2pService.SyncBlockchain(ctx)
}

// HandleIncomingBlock processa um bloco recebido
func (na *NetworkAdapter) HandleIncomingBlock(ctx context.Context, block *entities.Block, fromPeer valueobjects.NodeID) error {
	// Esta funcionalidade é gerenciada pelos handlers registrados
	return nil
}

// HandleIncomingTransaction processa uma transação recebida
func (na *NetworkAdapter) HandleIncomingTransaction(ctx context.Context, tx *entities.Transaction, fromPeer valueobjects.NodeID) error {
	// Esta funcionalidade é gerenciada pelos handlers registrados
	return nil
}

// RegisterBlockHandler registra um handler para blocos recebidos
func (na *NetworkAdapter) RegisterBlockHandler(handler services.BlockHandler) {
	na.p2pService.SetOnBlockReceived(func(block *entities.Block) {
		// Converter callback para usar a interface correta
		// Por enquanto, usar NodeID vazio como fromPeer
		ctx := context.Background()
		handler(ctx, block, valueobjects.EmptyNodeID())
	})
}

// RegisterTransactionHandler registra um handler para transações recebidas
func (na *NetworkAdapter) RegisterTransactionHandler(handler services.TransactionHandler) {
	na.p2pService.SetOnTxReceived(func(tx *entities.Transaction) {
		// Converter callback para usar a interface correta
		// Por enquanto, usar NodeID vazio como fromPeer
		ctx := context.Background()
		handler(ctx, tx, valueobjects.EmptyNodeID())
	})
}

// GetNetworkStatus retorna o status da rede
func (na *NetworkAdapter) GetNetworkStatus(ctx context.Context) (services.NetworkStatus, error) {
	stats, err := na.p2pService.GetStats(ctx)
	if err != nil {
		return services.NetworkStatus{}, err
	}
	
	return services.NetworkStatus{
		IsRunning:      na.p2pService.IsRunning(),
		NodeID:         na.p2pService.GetNodeID(),
		PeerCount:      stats.ConnectedPeers,
		ListenAddrs:    na.p2pService.GetListenAddresses(),
		LastSync:       valueobjects.Unix(stats.LastSyncTime.Unix(), 0),
		SyncInProgress: false, // TODO: implementar status real
	}, nil
}