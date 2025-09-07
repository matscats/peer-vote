package persistence

import (
	"context"
	"fmt"
	"sync"

	"github.com/matscats/peer-vote/peer-vote/domain/services"
	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
)

// MemoryKeyRepository implementa um repositório de chaves em memória
type MemoryKeyRepository struct {
	// Mapeamento de NodeID para chave pública
	publicKeys map[string]*services.PublicKey
	
	// Mapeamento de NodeID para chave privada (apenas para nós locais)
	privateKeys map[string]*services.PrivateKey
	
	// Mutex para operações thread-safe
	mu sync.RWMutex
}

// NewMemoryKeyRepository cria um novo repositório de chaves em memória
func NewMemoryKeyRepository() *MemoryKeyRepository {
	return &MemoryKeyRepository{
		publicKeys:  make(map[string]*services.PublicKey),
		privateKeys: make(map[string]*services.PrivateKey),
	}
}

// StoreKeyPair armazena um par de chaves para um nó
func (r *MemoryKeyRepository) StoreKeyPair(ctx context.Context, nodeID valueobjects.NodeID, keyPair *services.KeyPair) error {
	if nodeID.IsEmpty() {
		return fmt.Errorf("nodeID cannot be empty")
	}
	
	if keyPair == nil || !keyPair.IsValid() {
		return fmt.Errorf("invalid key pair")
	}
	
	r.mu.Lock()
	defer r.mu.Unlock()
	
	nodeIDStr := nodeID.String()
	r.publicKeys[nodeIDStr] = keyPair.PublicKey
	r.privateKeys[nodeIDStr] = keyPair.PrivateKey
	
	return nil
}

// StorePublicKey armazena apenas a chave pública de um nó
func (r *MemoryKeyRepository) StorePublicKey(ctx context.Context, nodeID valueobjects.NodeID, publicKey *services.PublicKey) error {
	if nodeID.IsEmpty() {
		return fmt.Errorf("nodeID cannot be empty")
	}
	
	if publicKey == nil || !publicKey.IsValid() {
		return fmt.Errorf("invalid public key")
	}
	
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.publicKeys[nodeID.String()] = publicKey
	
	return nil
}

// GetPublicKey recupera a chave pública de um nó
func (r *MemoryKeyRepository) GetPublicKey(ctx context.Context, nodeID valueobjects.NodeID) (*services.PublicKey, error) {
	if nodeID.IsEmpty() {
		return nil, fmt.Errorf("nodeID cannot be empty")
	}
	
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	publicKey, exists := r.publicKeys[nodeID.String()]
	if !exists {
		return nil, fmt.Errorf("public key not found for node %s", nodeID.String())
	}
	
	return publicKey, nil
}

// GetPrivateKey recupera a chave privada de um nó (apenas para nós locais)
func (r *MemoryKeyRepository) GetPrivateKey(ctx context.Context, nodeID valueobjects.NodeID) (*services.PrivateKey, error) {
	if nodeID.IsEmpty() {
		return nil, fmt.Errorf("nodeID cannot be empty")
	}
	
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	privateKey, exists := r.privateKeys[nodeID.String()]
	if !exists {
		return nil, fmt.Errorf("private key not found for node %s", nodeID.String())
	}
	
	return privateKey, nil
}

// HasPublicKey verifica se existe uma chave pública para o nó
func (r *MemoryKeyRepository) HasPublicKey(ctx context.Context, nodeID valueobjects.NodeID) bool {
	if nodeID.IsEmpty() {
		return false
	}
	
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	_, exists := r.publicKeys[nodeID.String()]
	return exists
}

// ListNodes retorna todos os nós que têm chaves armazenadas
func (r *MemoryKeyRepository) ListNodes(ctx context.Context) ([]valueobjects.NodeID, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	nodes := make([]valueobjects.NodeID, 0, len(r.publicKeys))
	for nodeIDStr := range r.publicKeys {
		nodes = append(nodes, valueobjects.NewNodeID(nodeIDStr))
	}
	
	return nodes, nil
}

// RemoveKeys remove as chaves de um nó
func (r *MemoryKeyRepository) RemoveKeys(ctx context.Context, nodeID valueobjects.NodeID) error {
	if nodeID.IsEmpty() {
		return fmt.Errorf("nodeID cannot be empty")
	}
	
	r.mu.Lock()
	defer r.mu.Unlock()
	
	nodeIDStr := nodeID.String()
	delete(r.publicKeys, nodeIDStr)
	delete(r.privateKeys, nodeIDStr)
	
	return nil
}

// Clear remove todas as chaves (útil para testes)
func (r *MemoryKeyRepository) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.publicKeys = make(map[string]*services.PublicKey)
	r.privateKeys = make(map[string]*services.PrivateKey)
}

// GetKeyCount retorna o número de chaves armazenadas
func (r *MemoryKeyRepository) GetKeyCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	return len(r.publicKeys)
}
