package valueobjects

import (
	"bytes"
	"encoding/hex"
	"fmt"
)

// Hash representa um hash criptográfico
type Hash struct {
	value []byte
}

// NewHash cria um novo hash a partir de bytes
func NewHash(data []byte) Hash {
	if data == nil {
		return Hash{value: make([]byte, 32)} // SHA-256 hash size
	}
	
	// Copia os dados para evitar mutação externa
	hashValue := make([]byte, len(data))
	copy(hashValue, data)
	
	return Hash{value: hashValue}
}

// NewHashFromString cria um hash a partir de uma string hexadecimal
func NewHashFromString(hexStr string) (Hash, error) {
	data, err := hex.DecodeString(hexStr)
	if err != nil {
		return Hash{}, fmt.Errorf("invalid hex string: %w", err)
	}
	
	return NewHash(data), nil
}

// EmptyHash retorna um hash vazio
func EmptyHash() Hash {
	return Hash{value: make([]byte, 32)}
}

// Bytes retorna os bytes do hash
func (h Hash) Bytes() []byte {
	// Retorna uma cópia para evitar mutação
	result := make([]byte, len(h.value))
	copy(result, h.value)
	return result
}

// String retorna a representação hexadecimal do hash
func (h Hash) String() string {
	return hex.EncodeToString(h.value)
}

// IsEmpty verifica se o hash está vazio
func (h Hash) IsEmpty() bool {
	for _, b := range h.value {
		if b != 0 {
			return false
		}
	}
	return true
}

// Equals compara dois hashes
func (h Hash) Equals(other Hash) bool {
	return bytes.Equal(h.value, other.value)
}

// Size retorna o tamanho do hash em bytes
func (h Hash) Size() int {
	return len(h.value)
}

// IsValid verifica se o hash é válido
func (h Hash) IsValid() bool {
	// Um hash válido deve ter exatamente 32 bytes (SHA-256)
	return len(h.value) == 32
}

// Copy retorna uma cópia do hash
func (h Hash) Copy() Hash {
	return NewHash(h.value)
}
