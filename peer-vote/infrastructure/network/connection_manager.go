package network

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/connmgr"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

// ConnectionManager implementa connmgr.ConnManager para gerenciar conexões
type ConnectionManager struct {
	maxConnections int
	connections    map[peer.ID]*ConnectionInfo
	mu             sync.RWMutex
	
	// Configurações
	gracePeriod    time.Duration
	cleanupInterval time.Duration
	
	// Callbacks
	onConnectionLimit func(peer.ID)
	onConnectionTrim  func(peer.ID)
}

// ConnectionInfo armazena informações sobre uma conexão
type ConnectionInfo struct {
	PeerID      peer.ID
	ConnectedAt time.Time
	LastUsed    time.Time
	Priority    int
	Protected   bool
	Tags        map[string]int
}

// NewConnectionManager cria um novo gerenciador de conexões
func NewConnectionManager(maxConnections int) *ConnectionManager {
	cm := &ConnectionManager{
		maxConnections:  maxConnections,
		connections:     make(map[peer.ID]*ConnectionInfo),
		gracePeriod:     time.Minute * 5,
		cleanupInterval: time.Minute * 1,
	}
	
	// Iniciar limpeza periódica
	go cm.cleanupLoop()
	
	return cm
}

// Notifee implementa network.Notifiee para receber eventos de rede
func (cm *ConnectionManager) Notifee() network.Notifiee {
	return cm
}

// TagPeer adiciona uma tag a um peer
func (cm *ConnectionManager) TagPeer(p peer.ID, tag string, weight int) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	info, exists := cm.connections[p]
	if !exists {
		info = &ConnectionInfo{
			PeerID:      p,
			ConnectedAt: time.Now(),
			LastUsed:    time.Now(),
			Tags:        make(map[string]int),
		}
		cm.connections[p] = info
	}
	
	info.Tags[tag] = weight
	info.Priority += weight
}

// UntagPeer remove uma tag de um peer
func (cm *ConnectionManager) UntagPeer(p peer.ID, tag string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	info, exists := cm.connections[p]
	if !exists {
		return
	}
	
	if weight, exists := info.Tags[tag]; exists {
		delete(info.Tags, tag)
		info.Priority -= weight
	}
}

// UpsertTag atualiza ou insere uma tag
func (cm *ConnectionManager) UpsertTag(p peer.ID, tag string, upsert func(int) int) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	info, exists := cm.connections[p]
	if !exists {
		info = &ConnectionInfo{
			PeerID:      p,
			ConnectedAt: time.Now(),
			LastUsed:    time.Now(),
			Tags:        make(map[string]int),
		}
		cm.connections[p] = info
	}
	
	oldWeight := info.Tags[tag]
	newWeight := upsert(oldWeight)
	
	info.Tags[tag] = newWeight
	info.Priority = info.Priority - oldWeight + newWeight
}

// GetTagInfo retorna informações sobre tags de um peer
func (cm *ConnectionManager) GetTagInfo(p peer.ID) *connmgr.TagInfo {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	info, exists := cm.connections[p]
	if !exists {
		return nil
	}
	
	tagInfo := &connmgr.TagInfo{
		FirstSeen: info.ConnectedAt,
		Value:     info.Priority,
		Tags:      make(map[string]int),
		Conns:     make(map[string]time.Time),
	}
	
	for tag, weight := range info.Tags {
		tagInfo.Tags[tag] = weight
	}
	
	return tagInfo
}

// CheckLimit verifica se o limite de conexões foi excedido
func (cm *ConnectionManager) CheckLimit(limiter connmgr.GetConnLimiter) error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	if len(cm.connections) > cm.maxConnections {
		return fmt.Errorf("connection limit exceeded: %d/%d", len(cm.connections), cm.maxConnections)
	}
	
	return nil
}

// TrimOpenConns remove conexões desnecessárias
func (cm *ConnectionManager) TrimOpenConns(ctx context.Context) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	if len(cm.connections) <= cm.maxConnections {
		return
	}
	
	// Ordenar conexões por prioridade (menor primeiro)
	candidates := cm.getTrimmingCandidates()
	
	// Remover conexões até atingir o limite
	toRemove := len(cm.connections) - cm.maxConnections
	for i := 0; i < toRemove && i < len(candidates); i++ {
		peerID := candidates[i]
		delete(cm.connections, peerID)
		
		if cm.onConnectionTrim != nil {
			cm.onConnectionTrim(peerID)
		}
	}
}

// getTrimmingCandidates retorna peers candidatos para remoção
func (cm *ConnectionManager) getTrimmingCandidates() []peer.ID {
	var candidates []peer.ID
	
	for peerID, info := range cm.connections {
		if !info.Protected {
			candidates = append(candidates, peerID)
		}
	}
	
	// Ordenar por prioridade (menor primeiro) e tempo de última utilização
	// Implementação simplificada - em produção usaria sort.Slice
	return candidates
}

// Protect protege uma conexão de ser removida
func (cm *ConnectionManager) Protect(id peer.ID, tag string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	info, exists := cm.connections[id]
	if !exists {
		info = &ConnectionInfo{
			PeerID:      id,
			ConnectedAt: time.Now(),
			LastUsed:    time.Now(),
			Tags:        make(map[string]int),
		}
		cm.connections[id] = info
	}
	
	info.Protected = true
}

// Unprotect remove proteção de uma conexão
func (cm *ConnectionManager) Unprotect(id peer.ID, tag string) bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	info, exists := cm.connections[id]
	if !exists {
		return false
	}
	
	info.Protected = false
	return true
}

// IsProtected verifica se uma conexão está protegida
func (cm *ConnectionManager) IsProtected(id peer.ID, tag string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	info, exists := cm.connections[id]
	if !exists {
		return false
	}
	
	return info.Protected
}

// Close fecha o gerenciador de conexões
func (cm *ConnectionManager) Close() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	cm.connections = make(map[peer.ID]*ConnectionInfo)
	return nil
}

// Implementação de network.Notifiee

// Listen é chamado quando começamos a escutar
func (cm *ConnectionManager) Listen(network.Network, ma.Multiaddr) {}

// ListenClose é chamado quando paramos de escutar
func (cm *ConnectionManager) ListenClose(network.Network, ma.Multiaddr) {}

// Connected é chamado quando uma conexão é estabelecida
func (cm *ConnectionManager) Connected(n network.Network, conn network.Conn) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	peerID := conn.RemotePeer()
	
	info, exists := cm.connections[peerID]
	if !exists {
		info = &ConnectionInfo{
			PeerID:      peerID,
			ConnectedAt: time.Now(),
			Tags:        make(map[string]int),
		}
		cm.connections[peerID] = info
	}
	
	info.LastUsed = time.Now()
	
	// Verificar se excedeu o limite
	if len(cm.connections) > cm.maxConnections {
		if cm.onConnectionLimit != nil {
			cm.onConnectionLimit(peerID)
		}
		
		// Agendar limpeza
		go func() {
			time.Sleep(cm.gracePeriod)
			cm.TrimOpenConns(context.Background())
		}()
	}
}

// Disconnected é chamado quando uma conexão é fechada
func (cm *ConnectionManager) Disconnected(n network.Network, conn network.Conn) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	peerID := conn.RemotePeer()
	
	// Verificar se ainda há outras conexões para este peer
	if len(n.ConnsToPeer(peerID)) == 0 {
		delete(cm.connections, peerID)
	}
}

// OpenedStream é chamado quando um stream é aberto
func (cm *ConnectionManager) OpenedStream(n network.Network, stream network.Stream) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	peerID := stream.Conn().RemotePeer()
	if info, exists := cm.connections[peerID]; exists {
		info.LastUsed = time.Now()
	}
}

// ClosedStream é chamado quando um stream é fechado
func (cm *ConnectionManager) ClosedStream(network.Network, network.Stream) {}

// cleanupLoop executa limpeza periódica
func (cm *ConnectionManager) cleanupLoop() {
	ticker := time.NewTicker(cm.cleanupInterval)
	defer ticker.Stop()
	
	for range ticker.C {
		cm.TrimOpenConns(context.Background())
	}
}

// SetCallbacks define callbacks para eventos
func (cm *ConnectionManager) SetCallbacks(onLimit, onTrim func(peer.ID)) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	cm.onConnectionLimit = onLimit
	cm.onConnectionTrim = onTrim
}

// GetConnectionCount retorna o número de conexões
func (cm *ConnectionManager) GetConnectionCount() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	return len(cm.connections)
}

// GetConnectionInfo retorna informações sobre uma conexão
func (cm *ConnectionManager) GetConnectionInfo(peerID peer.ID) *ConnectionInfo {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	info, exists := cm.connections[peerID]
	if !exists {
		return nil
	}
	
	// Retornar cópia para evitar modificações externas
	return &ConnectionInfo{
		PeerID:      info.PeerID,
		ConnectedAt: info.ConnectedAt,
		LastUsed:    info.LastUsed,
		Priority:    info.Priority,
		Protected:   info.Protected,
		Tags:        copyTags(info.Tags),
	}
}

// copyTags cria uma cópia do mapa de tags
func copyTags(tags map[string]int) map[string]int {
	copy := make(map[string]int)
	for k, v := range tags {
		copy[k] = v
	}
	return copy
}

// GetAllConnections retorna informações sobre todas as conexões
func (cm *ConnectionManager) GetAllConnections() map[peer.ID]*ConnectionInfo {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	result := make(map[peer.ID]*ConnectionInfo)
	for peerID, info := range cm.connections {
		result[peerID] = &ConnectionInfo{
			PeerID:      info.PeerID,
			ConnectedAt: info.ConnectedAt,
			LastUsed:    info.LastUsed,
			Priority:    info.Priority,
			Protected:   info.Protected,
			Tags:        copyTags(info.Tags),
		}
	}
	
	return result
}

// SetMaxConnections atualiza o limite máximo de conexões
func (cm *ConnectionManager) SetMaxConnections(max int) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	cm.maxConnections = max
	
	// Se o novo limite é menor, agendar limpeza
	if len(cm.connections) > max {
		go cm.TrimOpenConns(context.Background())
	}
}

// GetMaxConnections retorna o limite máximo de conexões
func (cm *ConnectionManager) GetMaxConnections() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	return cm.maxConnections
}

// CheckConnections verifica e limpa conexões inativas
func (cm *ConnectionManager) CheckConnections(maxIdleTime time.Duration) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	now := time.Now()
	var toRemove []peer.ID
	
	for peerID, info := range cm.connections {
		if !info.Protected && now.Sub(info.LastUsed) > maxIdleTime {
			toRemove = append(toRemove, peerID)
		}
	}
	
	for _, peerID := range toRemove {
		delete(cm.connections, peerID)
		if cm.onConnectionTrim != nil {
			cm.onConnectionTrim(peerID)
		}
	}
}
