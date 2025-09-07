package services

import (
	"context"

	"github.com/matscats/peer-vote/peer-vote/domain/entities"
	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
)

// NetworkService define as operações de rede P2P
type NetworkService interface {
	// Start inicia o serviço de rede
	Start(ctx context.Context) error
	
	// Stop para o serviço de rede
	Stop(ctx context.Context) error
	
	// Connect conecta a um peer específico
	Connect(ctx context.Context, peerID valueobjects.NodeID, address string) error
	
	// Disconnect desconecta de um peer
	Disconnect(ctx context.Context, peerID valueobjects.NodeID) error
	
	// BroadcastBlock transmite um bloco para todos os peers
	BroadcastBlock(ctx context.Context, block *entities.Block) error
	
	// BroadcastTransaction transmite uma transação para todos os peers
	BroadcastTransaction(ctx context.Context, tx *entities.Transaction) error
	
	// SendBlockToPeer envia um bloco para um peer específico
	SendBlockToPeer(ctx context.Context, peerID valueobjects.NodeID, block *entities.Block) error
	
	// SendTransactionToPeer envia uma transação para um peer específico
	SendTransactionToPeer(ctx context.Context, peerID valueobjects.NodeID, tx *entities.Transaction) error
	
	// RequestBlock solicita um bloco específico de um peer
	RequestBlock(ctx context.Context, peerID valueobjects.NodeID, blockHash valueobjects.Hash) (*entities.Block, error)
	
	// RequestBlockRange solicita uma faixa de blocos de um peer
	RequestBlockRange(ctx context.Context, peerID valueobjects.NodeID, startIndex, endIndex uint64) ([]*entities.Block, error)
	
	// GetConnectedPeers retorna a lista de peers conectados
	GetConnectedPeers(ctx context.Context) ([]PeerInfo, error)
	
	// GetPeerCount retorna o número de peers conectados
	GetPeerCount(ctx context.Context) (int, error)
	
	// DiscoverPeers descobre novos peers na rede
	DiscoverPeers(ctx context.Context) error
	
	// GetNodeID retorna o ID deste nó
	GetNodeID() valueobjects.NodeID
	
	// GetListenAddresses retorna os endereços que este nó está escutando
	GetListenAddresses() []string
	
	// SyncBlockchain sincroniza a blockchain com os peers
	SyncBlockchain(ctx context.Context) error
	
	// HandleIncomingBlock processa um bloco recebido
	HandleIncomingBlock(ctx context.Context, block *entities.Block, fromPeer valueobjects.NodeID) error
	
	// HandleIncomingTransaction processa uma transação recebida
	HandleIncomingTransaction(ctx context.Context, tx *entities.Transaction, fromPeer valueobjects.NodeID) error
	
	// RegisterBlockHandler registra um handler para blocos recebidos
	RegisterBlockHandler(handler BlockHandler)
	
	// RegisterTransactionHandler registra um handler para transações recebidas
	RegisterTransactionHandler(handler TransactionHandler)
	
	// GetNetworkStatus retorna o status da rede
	GetNetworkStatus(ctx context.Context) (NetworkStatus, error)
}

// PeerInfo contém informações sobre um peer
type PeerInfo struct {
	ID        valueobjects.NodeID
	Address   string
	Connected bool
	LastSeen  valueobjects.Timestamp
}

// NetworkStatus representa o status da rede
type NetworkStatus struct {
	IsRunning     bool
	NodeID        valueobjects.NodeID
	PeerCount     int
	ListenAddrs   []string
	LastSync      valueobjects.Timestamp
	SyncInProgress bool
}

// BlockHandler é chamado quando um bloco é recebido
type BlockHandler func(ctx context.Context, block *entities.Block, fromPeer valueobjects.NodeID) error

// TransactionHandler é chamado quando uma transação é recebida
type TransactionHandler func(ctx context.Context, tx *entities.Transaction, fromPeer valueobjects.NodeID) error
