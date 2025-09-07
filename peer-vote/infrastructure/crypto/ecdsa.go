package crypto

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"strings"

	"github.com/matscats/peer-vote/peer-vote/domain/services"
	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
)

// ECDSAService implementa CryptographyService usando ECDSA
type ECDSAService struct {
	curve elliptic.Curve
}

// NewECDSAService cria um novo serviço de criptografia ECDSA
func NewECDSAService() *ECDSAService {
	return &ECDSAService{
		curve: elliptic.P256(), // Usando curva P-256
	}
}

// GenerateKeyPair gera um novo par de chaves ECDSA
func (e *ECDSAService) GenerateKeyPair(ctx context.Context) (*services.KeyPair, error) {
	privateKey, err := ecdsa.GenerateKey(e.curve, rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ECDSA key pair: %w", err)
	}

	// Converter para o formato do domínio
	domainPrivateKey := &services.PrivateKey{
		D:     privateKey.D.Bytes(),
		Curve: "P-256",
	}

	domainPublicKey := &services.PublicKey{
		X:     privateKey.PublicKey.X.Bytes(),
		Y:     privateKey.PublicKey.Y.Bytes(),
		Curve: "P-256",
	}

	return &services.KeyPair{
		PrivateKey: domainPrivateKey,
		PublicKey:  domainPublicKey,
	}, nil
}

// LoadKeyPair carrega um par de chaves de um arquivo
func (e *ECDSAService) LoadKeyPair(ctx context.Context, privateKeyPath string) (*services.KeyPair, error) {
	// Ler arquivo da chave privada
	keyData, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	// Decodificar PEM
	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	// Parsear chave privada
	privateKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse EC private key: %w", err)
	}

	// Converter para o formato do domínio
	domainPrivateKey := &services.PrivateKey{
		D:     privateKey.D.Bytes(),
		Curve: "P-256",
	}

	domainPublicKey := &services.PublicKey{
		X:     privateKey.PublicKey.X.Bytes(),
		Y:     privateKey.PublicKey.Y.Bytes(),
		Curve: "P-256",
	}

	return &services.KeyPair{
		PrivateKey: domainPrivateKey,
		PublicKey:  domainPublicKey,
	}, nil
}

// SaveKeyPair salva um par de chaves em um arquivo
func (e *ECDSAService) SaveKeyPair(ctx context.Context, keyPair *services.KeyPair, privateKeyPath string) error {
	if !keyPair.IsValid() {
		return errors.New("invalid key pair")
	}

	// Converter de volta para ecdsa.PrivateKey
	privateKey := &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: e.curve,
			X:     new(big.Int).SetBytes(keyPair.PublicKey.X),
			Y:     new(big.Int).SetBytes(keyPair.PublicKey.Y),
		},
		D: new(big.Int).SetBytes(keyPair.PrivateKey.D),
	}

	// Serializar chave privada
	keyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}

	// Criar bloco PEM
	pemBlock := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyBytes,
	}

	// Criar arquivo
	file, err := os.Create(privateKeyPath)
	if err != nil {
		return fmt.Errorf("failed to create private key file: %w", err)
	}
	defer file.Close()

	// Escrever PEM
	err = pem.Encode(file, pemBlock)
	if err != nil {
		return fmt.Errorf("failed to encode PEM: %w", err)
	}

	return nil
}

// Sign assina dados com uma chave privada
func (e *ECDSAService) Sign(ctx context.Context, data []byte, privateKey *services.PrivateKey) (valueobjects.Signature, error) {
	if !privateKey.IsValid() {
		return valueobjects.EmptySignature(), errors.New("invalid private key")
	}

	// Converter para ecdsa.PrivateKey
	d := new(big.Int).SetBytes(privateKey.D)
	
	// Gerar a chave pública a partir da privada
	ecdsaPrivateKey := &ecdsa.PrivateKey{
		D: d,
	}
	ecdsaPrivateKey.PublicKey.Curve = e.curve
	ecdsaPrivateKey.PublicKey.X, ecdsaPrivateKey.PublicKey.Y = e.curve.ScalarBaseMult(d.Bytes())

	// Calcular hash dos dados
	hash := sha256.Sum256(data)

	// Assinar
	r, s, err := ecdsa.Sign(rand.Reader, ecdsaPrivateKey, hash[:])
	if err != nil {
		return valueobjects.EmptySignature(), fmt.Errorf("failed to sign data: %w", err)
	}

	// Serializar assinatura (r || s)
	rBytes := r.Bytes()
	sBytes := s.Bytes()
	
	// Garantir que r e s tenham 32 bytes cada
	signature := make([]byte, 64)
	copy(signature[32-len(rBytes):32], rBytes)
	copy(signature[64-len(sBytes):], sBytes)

	return valueobjects.NewSignature(signature), nil
}

// Verify verifica uma assinatura com uma chave pública
func (e *ECDSAService) Verify(ctx context.Context, data []byte, signature valueobjects.Signature, publicKey *services.PublicKey) (bool, error) {
	if !publicKey.IsValid() {
		return false, errors.New("invalid public key")
	}

	if signature.IsEmpty() {
		return false, errors.New("empty signature")
	}

	// Converter para ecdsa.PublicKey
	ecdsaPublicKey := &ecdsa.PublicKey{
		Curve: e.curve,
		X:     new(big.Int).SetBytes(publicKey.X),
		Y:     new(big.Int).SetBytes(publicKey.Y),
	}

	// Calcular hash dos dados
	hash := sha256.Sum256(data)

	// Deserializar assinatura
	sigBytes := signature.Bytes()
	if len(sigBytes) != 64 {
		return false, errors.New("invalid signature length")
	}

	r := new(big.Int).SetBytes(sigBytes[:32])
	s := new(big.Int).SetBytes(sigBytes[32:])

	// Verificar assinatura
	return ecdsa.Verify(ecdsaPublicKey, hash[:], r, s), nil
}

// Hash calcula o hash SHA-256 de dados
func (e *ECDSAService) Hash(ctx context.Context, data []byte) valueobjects.Hash {
	hash := sha256.Sum256(data)
	return valueobjects.NewHash(hash[:])
}

// HashTransaction calcula o hash de uma transação
func (e *ECDSAService) HashTransaction(ctx context.Context, txData []byte) valueobjects.Hash {
	return e.Hash(ctx, txData)
}

// HashBlock calcula o hash de um bloco
func (e *ECDSAService) HashBlock(ctx context.Context, blockData []byte) valueobjects.Hash {
	return e.Hash(ctx, blockData)
}

// GenerateNodeID gera um ID de nó baseado na chave pública
func (e *ECDSAService) GenerateNodeID(ctx context.Context, publicKey *services.PublicKey) valueobjects.NodeID {
	if !publicKey.IsValid() {
		return valueobjects.EmptyNodeID()
	}

	// Hash da chave pública
	pubKeyBytes := publicKey.ToBytes()
	hash := sha256.Sum256(pubKeyBytes)
	
	// Usar os primeiros 16 bytes como ID
	nodeIDBytes := hash[:16]
	nodeIDHex := hex.EncodeToString(nodeIDBytes)
	
	return valueobjects.NewNodeID(nodeIDHex)
}

// ValidateSignature valida se uma assinatura é válida para os dados
func (e *ECDSAService) ValidateSignature(ctx context.Context, data []byte, signature valueobjects.Signature, nodeID valueobjects.NodeID) (bool, error) {
	// Esta implementação requer que tenhamos uma forma de recuperar a chave pública do nodeID
	// Por enquanto, retornamos erro indicando que precisa ser implementado com um repositório de chaves
	return false, errors.New("ValidateSignature requires a key repository - not implemented yet")
}

// RecoverPublicKey recupera a chave pública de uma assinatura (se possível)
func (e *ECDSAService) RecoverPublicKey(ctx context.Context, data []byte, signature valueobjects.Signature) (*services.PublicKey, error) {
	// ECDSA padrão não permite recuperação de chave pública
	// Isso seria possível com ECDSA recuperável (como usado no Ethereum)
	return nil, errors.New("public key recovery not supported with standard ECDSA")
}

// GetCurveName retorna o nome da curva utilizada
func (e *ECDSAService) GetCurveName() string {
	return "P-256"
}

// GetKeySize retorna o tamanho da chave em bits
func (e *ECDSAService) GetKeySize() int {
	return 256
}

// ParsePrivateKeyFromString converte uma string em PrivateKey
func (e *ECDSAService) ParsePrivateKeyFromString(keyStr string) (*services.PrivateKey, error) {
	// Tentar primeiro como PEM
	if strings.HasPrefix(keyStr, "-----BEGIN") {
		return e.parsePrivateKeyFromPEM(keyStr)
	}
	
	// Tentar como hex
	return e.parsePrivateKeyFromHex(keyStr)
}

// parsePrivateKeyFromPEM converte PEM string em PrivateKey
func (e *ECDSAService) parsePrivateKeyFromPEM(pemStr string) (*services.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	
	if block.Type != "EC PRIVATE KEY" {
		return nil, fmt.Errorf("invalid PEM type: %s", block.Type)
	}
	
	ecdsaKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse EC private key: %w", err)
	}
	
	return &services.PrivateKey{
		D: ecdsaKey.D.Bytes(),
	}, nil
}

// parsePrivateKeyFromHex converte hex string em PrivateKey
func (e *ECDSAService) parsePrivateKeyFromHex(hexStr string) (*services.PrivateKey, error) {
	// Remover prefixo 0x se presente
	hexStr = strings.TrimPrefix(hexStr, "0x")
	
	keyBytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex string: %w", err)
	}
	
	// Verificar tamanho da chave (32 bytes para P-256)
	if len(keyBytes) != 32 {
		return nil, fmt.Errorf("invalid key size: expected 32 bytes, got %d", len(keyBytes))
	}
	
	return &services.PrivateKey{
		D: keyBytes,
	}, nil
}
