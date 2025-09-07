package valueobjects

import (
	"encoding/hex"
	"fmt"
)

// Signature representa uma assinatura digital
type Signature struct {
	value []byte
}

// NewSignature cria uma nova assinatura a partir de bytes
func NewSignature(data []byte) Signature {
	if data == nil {
		return Signature{value: []byte{}}
	}
	
	// Copia os dados para evitar mutação externa
	sigValue := make([]byte, len(data))
	copy(sigValue, data)
	
	return Signature{value: sigValue}
}

// NewSignatureFromString cria uma assinatura a partir de uma string hexadecimal
func NewSignatureFromString(hexStr string) (Signature, error) {
	data, err := hex.DecodeString(hexStr)
	if err != nil {
		return Signature{}, fmt.Errorf("invalid hex string: %w", err)
	}
	
	return NewSignature(data), nil
}

// EmptySignature retorna uma assinatura vazia
func EmptySignature() Signature {
	return Signature{value: []byte{}}
}

// Bytes retorna os bytes da assinatura
func (s Signature) Bytes() []byte {
	// Retorna uma cópia para evitar mutação
	result := make([]byte, len(s.value))
	copy(result, s.value)
	return result
}

// String retorna a representação hexadecimal da assinatura
func (s Signature) String() string {
	return hex.EncodeToString(s.value)
}

// IsEmpty verifica se a assinatura está vazia
func (s Signature) IsEmpty() bool {
	return len(s.value) == 0
}

// Equals compara duas assinaturas
func (s Signature) Equals(other Signature) bool {
	if len(s.value) != len(other.value) {
		return false
	}
	
	for i, b := range s.value {
		if b != other.value[i] {
			return false
		}
	}
	
	return true
}

// Size retorna o tamanho da assinatura em bytes
func (s Signature) Size() int {
	return len(s.value)
}

// IsValid verifica se a assinatura é válida
func (s Signature) IsValid() bool {
	// Uma assinatura ECDSA válida deve ter entre 64 e 72 bytes
	// (dependendo da codificação DER)
	size := len(s.value)
	return size >= 64 && size <= 72
}

// Copy retorna uma cópia da assinatura
func (s Signature) Copy() Signature {
	return NewSignature(s.value)
}

// ShortString retorna uma versão abreviada da assinatura para logs
func (s Signature) ShortString() string {
	hexStr := s.String()
	if len(hexStr) < 16 {
		return hexStr
	}
	return hexStr[:16] + "..."
}
