package valueobjects

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// NodeID representa o identificador único de um nó na rede
type NodeID struct {
	value string
}

// NewNodeID cria um novo NodeID a partir de uma string
func NewNodeID(id string) NodeID {
	return NodeID{value: id}
}

// GenerateNodeID gera um novo NodeID aleatório
func GenerateNodeID() (NodeID, error) {
	// Gera 16 bytes aleatórios (128 bits)
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		return NodeID{}, fmt.Errorf("failed to generate random node ID: %w", err)
	}
	
	// Converte para string hexadecimal
	id := hex.EncodeToString(bytes)
	return NodeID{value: id}, nil
}

// EmptyNodeID retorna um NodeID vazio
func EmptyNodeID() NodeID {
	return NodeID{value: ""}
}

// String retorna a representação string do NodeID
func (n NodeID) String() string {
	return n.value
}

// Bytes retorna os bytes do NodeID
func (n NodeID) Bytes() []byte {
	data, _ := hex.DecodeString(n.value)
	return data
}

// IsEmpty verifica se o NodeID está vazio
func (n NodeID) IsEmpty() bool {
	return n.value == ""
}

// Equals compara dois NodeIDs
func (n NodeID) Equals(other NodeID) bool {
	return n.value == other.value
}

// IsValid verifica se o NodeID é válido
func (n NodeID) IsValid() bool {
	if n.IsEmpty() {
		return false
	}
	
	// Deve ser uma string hexadecimal válida de 32 caracteres (16 bytes)
	if len(n.value) != 32 {
		return false
	}
	
	// Verifica se é hexadecimal válido
	_, err := hex.DecodeString(n.value)
	return err == nil
}

// ShortString retorna uma versão abreviada do NodeID para logs
func (n NodeID) ShortString() string {
	if len(n.value) < 8 {
		return n.value
	}
	return n.value[:8] + "..."
}

// Copy retorna uma cópia do NodeID
func (n NodeID) Copy() NodeID {
	return NodeID{value: n.value}
}
