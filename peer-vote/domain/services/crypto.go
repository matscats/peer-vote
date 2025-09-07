package services

import (
	"context"
	"errors"

	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
)

// CryptographyService define as operações criptográficas
type CryptographyService interface {
	// GenerateKeyPair gera um novo par de chaves ECDSA
	GenerateKeyPair(ctx context.Context) (*KeyPair, error)
	
	// LoadKeyPair carrega um par de chaves de um arquivo
	LoadKeyPair(ctx context.Context, privateKeyPath string) (*KeyPair, error)
	
	// SaveKeyPair salva um par de chaves em um arquivo
	SaveKeyPair(ctx context.Context, keyPair *KeyPair, privateKeyPath string) error
	
	// Sign assina dados com uma chave privada
	Sign(ctx context.Context, data []byte, privateKey *PrivateKey) (valueobjects.Signature, error)
	
	// Verify verifica uma assinatura com uma chave pública
	Verify(ctx context.Context, data []byte, signature valueobjects.Signature, publicKey *PublicKey) (bool, error)
	
	// Hash calcula o hash SHA-256 de dados
	Hash(ctx context.Context, data []byte) valueobjects.Hash
	
	// HashTransaction calcula o hash de uma transação
	HashTransaction(ctx context.Context, txData []byte) valueobjects.Hash
	
	// HashBlock calcula o hash de um bloco
	HashBlock(ctx context.Context, blockData []byte) valueobjects.Hash
	
	// GenerateNodeID gera um ID de nó baseado na chave pública
	GenerateNodeID(ctx context.Context, publicKey *PublicKey) valueobjects.NodeID
	
	// ValidateSignature valida se uma assinatura é válida para os dados
	ValidateSignature(ctx context.Context, data []byte, signature valueobjects.Signature, nodeID valueobjects.NodeID) (bool, error)
}

// KeyPair representa um par de chaves pública e privada
type KeyPair struct {
	PrivateKey *PrivateKey
	PublicKey  *PublicKey
}

// PrivateKey representa uma chave privada ECDSA
type PrivateKey struct {
	D     []byte // Valor da chave privada
	Curve string // Nome da curva (ex: "P-256")
}

// PublicKey representa uma chave pública ECDSA
type PublicKey struct {
	X     []byte // Coordenada X do ponto
	Y     []byte // Coordenada Y do ponto
	Curve string // Nome da curva (ex: "P-256")
}

// ToBytes serializa a chave pública para bytes
func (pk *PublicKey) ToBytes() []byte {
	// Concatena X + Y
	result := make([]byte, len(pk.X)+len(pk.Y))
	copy(result, pk.X)
	copy(result[len(pk.X):], pk.Y)
	return result
}

// FromBytes deserializa uma chave pública de bytes
func (pk *PublicKey) FromBytes(data []byte, curve string) error {
	if len(data)%2 != 0 {
		return errors.New("invalid public key data length")
	}
	
	half := len(data) / 2
	pk.X = make([]byte, half)
	pk.Y = make([]byte, half)
	pk.Curve = curve
	
	copy(pk.X, data[:half])
	copy(pk.Y, data[half:])
	
	return nil
}

// IsValid verifica se a chave pública é válida
func (pk *PublicKey) IsValid() bool {
	return len(pk.X) > 0 && len(pk.Y) > 0 && pk.Curve != ""
}

// ToBytes serializa a chave privada para bytes
func (priv *PrivateKey) ToBytes() []byte {
	return priv.D
}

// FromBytes deserializa uma chave privada de bytes
func (priv *PrivateKey) FromBytes(data []byte, curve string) {
	priv.D = make([]byte, len(data))
	copy(priv.D, data)
	priv.Curve = curve
}

// IsValid verifica se a chave privada é válida
func (priv *PrivateKey) IsValid() bool {
	return len(priv.D) > 0 && priv.Curve != ""
}

// GetPublicKey deriva a chave pública da chave privada
func (kp *KeyPair) GetPublicKey() *PublicKey {
	return kp.PublicKey
}

// GetPrivateKey retorna a chave privada
func (kp *KeyPair) GetPrivateKey() *PrivateKey {
	return kp.PrivateKey
}

// IsValid verifica se o par de chaves é válido
func (kp *KeyPair) IsValid() bool {
	return kp.PrivateKey != nil && kp.PublicKey != nil &&
		kp.PrivateKey.IsValid() && kp.PublicKey.IsValid()
}
