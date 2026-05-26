package crypto

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"os"
)

// PublicKey represents an Ed25519 public key
type PublicKey []byte

// PrivateKey represents an Ed25519 private key
type PrivateKey []byte

// KeyPair represents a public/private key pair
type KeyPair struct {
	Public  PublicKey
	Private PrivateKey
}

// GenerateKeyPair generates a new Ed25519 key pair
func GenerateKeyPair() (*KeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	return &KeyPair{
		Public:  PublicKey(pub),
		Private: PrivateKey(priv),
	}, nil
}

// LoadKeyPair loads a key pair from a file
// The file should contain the hex-encoded private key (which includes the public key)
func LoadKeyPair(path string) (*KeyPair, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	privKeyBytes, err := hex.DecodeString(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	if len(privKeyBytes) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key size: expected %d, got %d", 
			ed25519.PrivateKeySize, len(privKeyBytes))
	}

	priv := ed25519.PrivateKey(privKeyBytes)
	pub := priv.Public().(ed25519.PublicKey)

	return &KeyPair{
		Public:  PublicKey(pub),
		Private: PrivateKey(priv),
	}, nil
}
