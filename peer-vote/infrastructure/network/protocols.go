package network

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"

	"github.com/matscats/peer-vote/peer-vote/domain/entities"
)

// Definições de protocolos
const (
	// ProtocolBlockSync protocolo para sincronização de blocos
	ProtocolBlockSync protocol.ID = "/peer-vote/block-sync/1.0.0"
	// ProtocolTxGossip protocolo para propagação de transações
	ProtocolTxGossip protocol.ID = "/peer-vote/tx-gossip/1.0.0"
	// ProtocolConsensus protocolo para consenso
	ProtocolConsensus protocol.ID = "/peer-vote/consensus/1.0.0"
	// ProtocolPing protocolo para ping/pong
	ProtocolPing protocol.ID = "/peer-vote/ping/1.0.0"
)

// MessageType define tipos de mensagens
type MessageType string

const (
	// Mensagens de sincronização
	MsgBlockRequest    MessageType = "BLOCK_REQUEST"
	MsgBlockResponse   MessageType = "BLOCK_RESPONSE"
	MsgBlockRangeReq   MessageType = "BLOCK_RANGE_REQUEST"
	MsgBlockRangeResp  MessageType = "BLOCK_RANGE_RESPONSE"
	MsgChainStatusReq  MessageType = "CHAIN_STATUS_REQUEST"
	MsgChainStatusResp MessageType = "CHAIN_STATUS_RESPONSE"
	
	// Mensagens de gossip
	MsgTxGossip        MessageType = "TX_GOSSIP"
	MsgBlockGossip     MessageType = "BLOCK_GOSSIP"
	
	// Mensagens de consenso
	MsgConsensusProposal MessageType = "CONSENSUS_PROPOSAL"
	MsgConsensusVote     MessageType = "CONSENSUS_VOTE"
	
	// Mensagens de controle
	MsgPing            MessageType = "PING"
	MsgPong            MessageType = "PONG"
	MsgError           MessageType = "ERROR"
)

// Message representa uma mensagem P2P
type Message struct {
	Type      MessageType     `json:"type"`
	Data      json.RawMessage `json:"data"`
	Timestamp int64           `json:"timestamp"`
	From      string          `json:"from"`
	RequestID string          `json:"request_id,omitempty"`
}

// BlockRequest requisição de bloco específico
type BlockRequest struct {
	BlockHash   string `json:"block_hash"`
	BlockIndex  uint64 `json:"block_index,omitempty"`
}

// BlockResponse resposta com bloco
type BlockResponse struct {
	Block     *SerializedBlock `json:"block"`
	Found     bool             `json:"found"`
	ErrorMsg  string           `json:"error_msg,omitempty"`
}

// BlockRangeRequest requisição de faixa de blocos
type BlockRangeRequest struct {
	StartIndex uint64 `json:"start_index"`
	EndIndex   uint64 `json:"end_index"`
	MaxBlocks  int    `json:"max_blocks"`
}

// BlockRangeResponse resposta com faixa de blocos
type BlockRangeResponse struct {
	Blocks   []*SerializedBlock `json:"blocks"`
	HasMore  bool               `json:"has_more"`
	ErrorMsg string             `json:"error_msg,omitempty"`
}

// ChainStatusRequest requisição de status da cadeia
type ChainStatusRequest struct {
	// Vazio por enquanto
}

// ChainStatusResponse resposta com status da cadeia
type ChainStatusResponse struct {
	Height        uint64 `json:"height"`
	LatestHash    string `json:"latest_hash"`
	GenesisHash   string `json:"genesis_hash"`
	PeerCount     int    `json:"peer_count"`
	LastBlockTime int64  `json:"last_block_time"`
}

// TxGossipMessage mensagem de gossip de transação
type TxGossipMessage struct {
	Transaction *SerializedTransaction `json:"transaction"`
	TTL         int                    `json:"ttl"`
	SeenBy      []string               `json:"seen_by"`
}

// BlockGossipMessage mensagem de gossip de bloco
type BlockGossipMessage struct {
	Block  *SerializedBlock `json:"block"`
	TTL    int              `json:"ttl"`
	SeenBy []string         `json:"seen_by"`
}

// SerializedBlock representa um bloco serializado
type SerializedBlock struct {
	Index        uint64                    `json:"index"`
	PreviousHash string                    `json:"previous_hash"`
	Timestamp    int64                     `json:"timestamp"`
	MerkleRoot   string                    `json:"merkle_root"`
	Validator    string                    `json:"validator"`
	Signature    string                    `json:"signature"`
	Transactions []*SerializedTransaction  `json:"transactions"`
}

// SerializedTransaction representa uma transação serializada
type SerializedTransaction struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	From      string `json:"from"`
	To        string `json:"to"`
	Data      string `json:"data"` // Base64 encoded
	Timestamp int64  `json:"timestamp"`
	Signature string `json:"signature"`
	Hash      string `json:"hash"`
}

// PingMessage mensagem de ping
type PingMessage struct {
	Timestamp int64  `json:"timestamp"`
	Message   string `json:"message"`
}

// PongMessage mensagem de pong
type PongMessage struct {
	Timestamp     int64  `json:"timestamp"`
	OriginalTime  int64  `json:"original_time"`
	Message       string `json:"message"`
}

// ErrorMessage mensagem de erro
type ErrorMessage struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// ProtocolManager gerencia protocolos P2P
type ProtocolManager struct {
	host *LibP2PHost
	
	// Handlers para diferentes tipos de mensagem
	blockRequestHandler    func(peer.ID, *BlockRequest) (*BlockResponse, error)
	blockRangeHandler      func(peer.ID, *BlockRangeRequest) (*BlockRangeResponse, error)
	chainStatusHandler     func(peer.ID, *ChainStatusRequest) (*ChainStatusResponse, error)
	txGossipHandler        func(peer.ID, *TxGossipMessage) error
	blockGossipHandler     func(peer.ID, *BlockGossipMessage) error
	consensusHandler       func(peer.ID, MessageType, json.RawMessage) error
	
	// Cache de mensagens vistas (para evitar loops)
	seenMessages map[string]time.Time
	seenMutex    sync.RWMutex
	
	// Configurações
	maxMessageSize int
	messageTimeout time.Duration
	gossipTTL      int
}

// NewProtocolManager cria um novo gerenciador de protocolos
func NewProtocolManager(host *LibP2PHost) *ProtocolManager {
	pm := &ProtocolManager{
		host:           host,
		seenMessages:   make(map[string]time.Time),
		maxMessageSize: 1024 * 1024, // 1MB
		messageTimeout: time.Second * 30,
		gossipTTL:      5,
	}
	
	// Registrar handlers de protocolo
	pm.registerProtocolHandlers()
	
	// Iniciar limpeza de mensagens vistas
	go pm.cleanupSeenMessages()
	
	return pm
}

// registerProtocolHandlers registra handlers para todos os protocolos
func (pm *ProtocolManager) registerProtocolHandlers() {
	pm.host.RegisterProtocol(ProtocolBlockSync, pm.handleBlockSync)
	pm.host.RegisterProtocol(ProtocolTxGossip, pm.handleTxGossip)
	pm.host.RegisterProtocol(ProtocolConsensus, pm.handleConsensus)
	pm.host.RegisterProtocol(ProtocolPing, pm.handlePing)
}

// handleBlockSync lida com protocolo de sincronização de blocos
func (pm *ProtocolManager) handleBlockSync(stream network.Stream) {
	defer stream.Close()
	
	reader := bufio.NewReader(stream)
	writer := bufio.NewWriter(stream)
	
	// Ler mensagem
	msg, err := pm.readMessage(reader)
	if err != nil {
		pm.sendError(writer, 400, "Failed to read message", err.Error())
		return
	}
	
	peerID := stream.Conn().RemotePeer()
	
	switch msg.Type {
	case MsgBlockRequest:
		pm.handleBlockRequest(peerID, msg, writer)
	case MsgBlockRangeReq:
		pm.handleBlockRangeRequest(peerID, msg, writer)
	case MsgChainStatusReq:
		pm.handleChainStatusRequest(peerID, msg, writer)
	case MsgBlockGossip:
		// CORREÇÃO: Tratar gossip de blocos no protocolo BlockSync
		pm.handleBlockGossipMessage(peerID, msg)
	default:
		pm.sendError(writer, 400, "Unknown message type", string(msg.Type))
	}
}

// handleBlockGossipMessage processa mensagem de gossip de bloco
// Aplica SRP: responsabilidade única de processar gossip de blocos
func (pm *ProtocolManager) handleBlockGossipMessage(peerID peer.ID, msg *Message) {
	// Verificar se já vimos esta mensagem
	if pm.hasSeenMessage(msg.RequestID) {
		return
	}
	
	pm.markMessageSeen(msg.RequestID)
	
	// Processar gossip de bloco
	var gossipMsg BlockGossipMessage
	if err := json.Unmarshal(msg.Data, &gossipMsg); err != nil {
		return
	}
	
	// Chamar handler se disponível
	if pm.blockGossipHandler != nil {
		pm.blockGossipHandler(peerID, &gossipMsg)
	}
	
	// Propagar para outros peers se TTL > 0
	if gossipMsg.TTL > 0 {
		pm.propagateGossip(msg, peerID)
	}
}

// handleTxGossip lida com protocolo de gossip de transações
func (pm *ProtocolManager) handleTxGossip(stream network.Stream) {
	defer stream.Close()
	
	reader := bufio.NewReader(stream)
	
	// Ler mensagem
	msg, err := pm.readMessage(reader)
	if err != nil {
		return
	}
	
	if msg.Type != MsgTxGossip {
		return
	}
	
	// Verificar se já vimos esta mensagem
	if pm.hasSeenMessage(msg.RequestID) {
		return
	}
	
	pm.markMessageSeen(msg.RequestID)
	
	// Processar gossip
	var gossipMsg TxGossipMessage
	if err := json.Unmarshal(msg.Data, &gossipMsg); err != nil {
		return
	}
	
	peerID := stream.Conn().RemotePeer()
	
	if pm.txGossipHandler != nil {
		pm.txGossipHandler(peerID, &gossipMsg)
	}
	
	// Propagar para outros peers se TTL > 0
	if gossipMsg.TTL > 0 {
		pm.propagateGossip(msg, peerID)
	}
}

// handleConsensus lida com protocolo de consenso
func (pm *ProtocolManager) handleConsensus(stream network.Stream) {
	defer stream.Close()
	
	reader := bufio.NewReader(stream)
	
	// Ler mensagem
	msg, err := pm.readMessage(reader)
	if err != nil {
		return
	}
	
	peerID := stream.Conn().RemotePeer()
	
	if pm.consensusHandler != nil {
		pm.consensusHandler(peerID, msg.Type, msg.Data)
	}
}

// handlePing lida com protocolo de ping
func (pm *ProtocolManager) handlePing(stream network.Stream) {
	defer stream.Close()
	
	reader := bufio.NewReader(stream)
	writer := bufio.NewWriter(stream)
	
	// Ler mensagem
	msg, err := pm.readMessage(reader)
	if err != nil {
		return
	}
	
	if msg.Type == MsgPing {
		var pingMsg PingMessage
		if err := json.Unmarshal(msg.Data, &pingMsg); err != nil {
			return
		}
		
		// Responder com pong
		pongMsg := PongMessage{
			Timestamp:    time.Now().Unix(),
			OriginalTime: pingMsg.Timestamp,
			Message:      "pong",
		}
		
		pm.sendMessage(writer, MsgPong, pongMsg, "")
	}
}

// SendBlockRequest envia requisição de bloco
func (pm *ProtocolManager) SendBlockRequest(ctx context.Context, peerID peer.ID, blockHash string) (*BlockResponse, error) {
	stream, err := pm.host.NewStream(ctx, peerID, ProtocolBlockSync)
	if err != nil {
		return nil, fmt.Errorf("failed to create stream: %w", err)
	}
	defer stream.Close()
	
	writer := bufio.NewWriter(stream)
	reader := bufio.NewReader(stream)
	
	// Enviar requisição
	req := BlockRequest{BlockHash: blockHash}
	if err := pm.sendMessage(writer, MsgBlockRequest, req, ""); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	
	// Ler resposta
	msg, err := pm.readMessage(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	if msg.Type == MsgError {
		var errMsg ErrorMessage
		json.Unmarshal(msg.Data, &errMsg)
		return nil, fmt.Errorf("peer error: %s", errMsg.Message)
	}
	
	if msg.Type != MsgBlockResponse {
		return nil, fmt.Errorf("unexpected response type: %s", msg.Type)
	}
	
	var response BlockResponse
	if err := json.Unmarshal(msg.Data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	
	return &response, nil
}

// SendChainStatusRequest envia requisição de status da cadeia
func (pm *ProtocolManager) SendChainStatusRequest(ctx context.Context, peerID peer.ID) (*ChainStatusResponse, error) {
	stream, err := pm.host.NewStream(ctx, peerID, ProtocolBlockSync)
	if err != nil {
		return nil, fmt.Errorf("failed to create stream: %w", err)
	}
	defer stream.Close()
	
	writer := bufio.NewWriter(stream)
	reader := bufio.NewReader(stream)
	
	// Enviar requisição
	req := ChainStatusRequest{}
	if err := pm.sendMessage(writer, MsgChainStatusReq, req, ""); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	
	// Ler resposta
	msg, err := pm.readMessage(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	if msg.Type != MsgChainStatusResp {
		return nil, fmt.Errorf("unexpected response type: %s", msg.Type)
	}
	
	var response ChainStatusResponse
	if err := json.Unmarshal(msg.Data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	
	return &response, nil
}

// GossipTransaction propaga uma transação via gossip
func (pm *ProtocolManager) GossipTransaction(ctx context.Context, tx *entities.Transaction) error {
	serializedTx := pm.serializeTransaction(tx)
	
	gossipMsg := TxGossipMessage{
		Transaction: serializedTx,
		TTL:         pm.gossipTTL,
		SeenBy:      []string{pm.host.GetPeerID().String()},
	}
	
	return pm.broadcastGossip(ctx, MsgTxGossip, gossipMsg)
}

// GossipBlock propaga um bloco via gossip
func (pm *ProtocolManager) GossipBlock(ctx context.Context, block *entities.Block) error {
	serializedBlock := pm.serializeBlock(block)
	
	gossipMsg := BlockGossipMessage{
		Block:  serializedBlock,
		TTL:    pm.gossipTTL,
		SeenBy: []string{pm.host.GetPeerID().String()},
	}
	
	return pm.broadcastGossip(ctx, MsgBlockGossip, gossipMsg)
}

// Ping envia ping para um peer
func (pm *ProtocolManager) Ping(ctx context.Context, peerID peer.ID) (time.Duration, error) {
	stream, err := pm.host.NewStream(ctx, peerID, ProtocolPing)
	if err != nil {
		return 0, fmt.Errorf("failed to create stream: %w", err)
	}
	defer stream.Close()
	
	writer := bufio.NewWriter(stream)
	reader := bufio.NewReader(stream)
	
	start := time.Now()
	
	// Enviar ping
	pingMsg := PingMessage{
		Timestamp: start.Unix(),
		Message:   "ping",
	}
	
	if err := pm.sendMessage(writer, MsgPing, pingMsg, ""); err != nil {
		return 0, fmt.Errorf("failed to send ping: %w", err)
	}
	
	// Ler pong
	msg, err := pm.readMessage(reader)
	if err != nil {
		return 0, fmt.Errorf("failed to read pong: %w", err)
	}
	
	if msg.Type != MsgPong {
		return 0, fmt.Errorf("unexpected response type: %s", msg.Type)
	}
	
	return time.Since(start), nil
}

// Métodos auxiliares

func (pm *ProtocolManager) readMessage(reader *bufio.Reader) (*Message, error) {
	// Implementação simplificada - em produção usaria um protocolo mais robusto
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	
	var msg Message
	if err := json.Unmarshal(line, &msg); err != nil {
		return nil, err
	}
	
	return &msg, nil
}

func (pm *ProtocolManager) sendMessage(writer *bufio.Writer, msgType MessageType, data interface{}, requestID string) error {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	
	msg := Message{
		Type:      msgType,
		Data:      dataBytes,
		Timestamp: time.Now().Unix(),
		From:      pm.host.GetPeerID().String(),
		RequestID: requestID,
	}
	
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	
	msgBytes = append(msgBytes, '\n')
	
	_, err = writer.Write(msgBytes)
	if err != nil {
		return err
	}
	
	return writer.Flush()
}

func (pm *ProtocolManager) sendError(writer *bufio.Writer, code int, message, details string) {
	errMsg := ErrorMessage{
		Code:    code,
		Message: message,
		Details: details,
	}
	
	pm.sendMessage(writer, MsgError, errMsg, "")
}

func (pm *ProtocolManager) hasSeenMessage(messageID string) bool {
	pm.seenMutex.RLock()
	defer pm.seenMutex.RUnlock()
	
	_, seen := pm.seenMessages[messageID]
	return seen
}

func (pm *ProtocolManager) markMessageSeen(messageID string) {
	pm.seenMutex.Lock()
	defer pm.seenMutex.Unlock()
	
	pm.seenMessages[messageID] = time.Now()
}

func (pm *ProtocolManager) cleanupSeenMessages() {
	ticker := time.NewTicker(time.Minute * 5)
	defer ticker.Stop()
	
	for range ticker.C {
		pm.seenMutex.Lock()
		now := time.Now()
		for id, timestamp := range pm.seenMessages {
			if now.Sub(timestamp) > time.Hour {
				delete(pm.seenMessages, id)
			}
		}
		pm.seenMutex.Unlock()
	}
}

// Métodos de serialização

func (pm *ProtocolManager) serializeBlock(block *entities.Block) *SerializedBlock {
	transactions := make([]*SerializedTransaction, len(block.GetTransactions()))
	for i, tx := range block.GetTransactions() {
		transactions[i] = pm.serializeTransaction(tx)
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

func (pm *ProtocolManager) serializeTransaction(tx *entities.Transaction) *SerializedTransaction {
	return &SerializedTransaction{
		ID:        tx.GetID().String(),
		Type:      string(tx.GetType()),
		From:      tx.GetFrom().String(),
		To:        tx.GetTo().String(),
		Data:      string(tx.GetData()), // Simplificado
		Timestamp: tx.GetTimestamp().Unix(),
		Signature: tx.GetSignature().String(),
		Hash:      tx.GetHash().String(),
	}
}

// Setters para handlers

func (pm *ProtocolManager) SetBlockRequestHandler(handler func(peer.ID, *BlockRequest) (*BlockResponse, error)) {
	pm.blockRequestHandler = handler
}

func (pm *ProtocolManager) SetBlockRangeHandler(handler func(peer.ID, *BlockRangeRequest) (*BlockRangeResponse, error)) {
	pm.blockRangeHandler = handler
}

func (pm *ProtocolManager) SetChainStatusHandler(handler func(peer.ID, *ChainStatusRequest) (*ChainStatusResponse, error)) {
	pm.chainStatusHandler = handler
}

func (pm *ProtocolManager) SetTxGossipHandler(handler func(peer.ID, *TxGossipMessage) error) {
	pm.txGossipHandler = handler
}

func (pm *ProtocolManager) SetBlockGossipHandler(handler func(peer.ID, *BlockGossipMessage) error) {
	pm.blockGossipHandler = handler
}

func (pm *ProtocolManager) SetConsensusHandler(handler func(peer.ID, MessageType, json.RawMessage) error) {
	pm.consensusHandler = handler
}

// Métodos de handler internos

func (pm *ProtocolManager) handleBlockRequest(peerID peer.ID, msg *Message, writer *bufio.Writer) {
	if pm.blockRequestHandler == nil {
		pm.sendError(writer, 501, "Block request handler not implemented", "")
		return
	}
	
	var req BlockRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		pm.sendError(writer, 400, "Invalid request format", err.Error())
		return
	}
	
	response, err := pm.blockRequestHandler(peerID, &req)
	if err != nil {
		pm.sendError(writer, 500, "Handler error", err.Error())
		return
	}
	
	pm.sendMessage(writer, MsgBlockResponse, response, msg.RequestID)
}

func (pm *ProtocolManager) handleBlockRangeRequest(peerID peer.ID, msg *Message, writer *bufio.Writer) {
	if pm.blockRangeHandler == nil {
		pm.sendError(writer, 501, "Block range handler not implemented", "")
		return
	}
	
	var req BlockRangeRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		pm.sendError(writer, 400, "Invalid request format", err.Error())
		return
	}
	
	response, err := pm.blockRangeHandler(peerID, &req)
	if err != nil {
		pm.sendError(writer, 500, "Handler error", err.Error())
		return
	}
	
	pm.sendMessage(writer, MsgBlockRangeResp, response, msg.RequestID)
}

func (pm *ProtocolManager) handleChainStatusRequest(peerID peer.ID, msg *Message, writer *bufio.Writer) {
	if pm.chainStatusHandler == nil {
		pm.sendError(writer, 501, "Chain status handler not implemented", "")
		return
	}
	
	var req ChainStatusRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		pm.sendError(writer, 400, "Invalid request format", err.Error())
		return
	}
	
	response, err := pm.chainStatusHandler(peerID, &req)
	if err != nil {
		pm.sendError(writer, 500, "Handler error", err.Error())
		return
	}
	
	pm.sendMessage(writer, MsgChainStatusResp, response, msg.RequestID)
}

func (pm *ProtocolManager) broadcastGossip(ctx context.Context, msgType MessageType, data interface{}) error {
	peers := pm.host.GetConnectedPeers()
	
	// Escolher protocolo baseado no tipo de mensagem (CORREÇÃO CRÍTICA)
	var protocolID protocol.ID
	switch msgType {
	case MsgBlockGossip:
		protocolID = ProtocolBlockSync // Usar protocolo de blocos para gossip de blocos
	case MsgTxGossip:
		protocolID = ProtocolTxGossip  // Usar protocolo de transações para gossip de transações
	default:
		protocolID = ProtocolTxGossip  // Fallback para protocolo de transações
	}
	
	for _, peerID := range peers {
		go func(pid peer.ID) {
			stream, err := pm.host.NewStream(ctx, pid, protocolID)
			if err != nil {
				return
			}
			defer stream.Close()
			
			writer := bufio.NewWriter(stream)
			pm.sendMessage(writer, msgType, data, "")
		}(peerID)
	}
	
	return nil
}

func (pm *ProtocolManager) propagateGossip(msg *Message, excludePeer peer.ID) {
	// Decrementar TTL
	var gossipData map[string]interface{}
	json.Unmarshal(msg.Data, &gossipData)
	
	if ttl, ok := gossipData["ttl"].(float64); ok && ttl > 1 {
		gossipData["ttl"] = ttl - 1
		
		// Adicionar este peer à lista de "seen_by"
		if seenBy, ok := gossipData["seen_by"].([]interface{}); ok {
			seenBy = append(seenBy, pm.host.GetPeerID().String())
			gossipData["seen_by"] = seenBy
		}
		
		// Reserializar
		newData, _ := json.Marshal(gossipData)
		msg.Data = newData
		
		// Propagar para outros peers
		peers := pm.host.GetConnectedPeers()
		for _, peerID := range peers {
			if peerID == excludePeer {
				continue
			}
			
			go func(pid peer.ID) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
				defer cancel()
				
				stream, err := pm.host.NewStream(ctx, pid, ProtocolTxGossip)
				if err != nil {
					return
				}
				defer stream.Close()
				
				writer := bufio.NewWriter(stream)
				msgBytes, _ := json.Marshal(msg)
				msgBytes = append(msgBytes, '\n')
				writer.Write(msgBytes)
				writer.Flush()
			}(peerID)
		}
	}
}
