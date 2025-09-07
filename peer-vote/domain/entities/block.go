package entities

import (
	"crypto/rand"
	"encoding/binary"
	"time"

	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
)

// Block representa um bloco na blockchain
type Block struct {
	header       *BlockHeader
	transactions []*Transaction
	merkleRoot   valueobjects.Hash
}

// BlockHeader contém os metadados do bloco
type BlockHeader struct {
	index        uint64
	previousHash valueobjects.Hash
	timestamp    valueobjects.Timestamp
	merkleRoot   valueobjects.Hash
	nonce        uint64
	validator    valueobjects.NodeID
	signature    valueobjects.Signature
}

// generateBlockNonce gera um nonce aleatório para o bloco
func generateBlockNonce() uint64 {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return binary.BigEndian.Uint64(bytes)
}

// NewBlock cria um novo bloco
func NewBlock(index uint64, previousHash valueobjects.Hash, transactions []*Transaction, validator valueobjects.NodeID) *Block {
	block := &Block{
		transactions: transactions,
	}

	block.header = &BlockHeader{
		index:        index,
		previousHash: previousHash,
		timestamp:    valueobjects.NewTimestamp(time.Now()),
		nonce:        generateBlockNonce(),
		validator:    validator,
	}

	// O Merkle Root será calculado pela infraestrutura
	return block
}

// GetIndex retorna o índice do bloco
func (b *Block) GetIndex() uint64 {
	return b.header.index
}

// GetPreviousHash retorna o hash do bloco anterior
func (b *Block) GetPreviousHash() valueobjects.Hash {
	return b.header.previousHash
}

// GetTimestamp retorna o timestamp do bloco
func (b *Block) GetTimestamp() valueobjects.Timestamp {
	return b.header.timestamp
}

// GetTransactions retorna as transações do bloco
func (b *Block) GetTransactions() []*Transaction {
	return b.transactions
}

// GetMerkleRoot retorna a raiz da Merkle Tree
func (b *Block) GetMerkleRoot() valueobjects.Hash {
	return b.header.merkleRoot
}

// GetValidator retorna o validador do bloco
func (b *Block) GetValidator() valueobjects.NodeID {
	return b.header.validator
}

// GetNonce retorna o nonce do bloco
func (b *Block) GetNonce() uint64 {
	return b.header.nonce
}

// GetSignature retorna a assinatura do bloco
func (b *Block) GetSignature() valueobjects.Signature {
	return b.header.signature
}

// SetMerkleRoot define a raiz da Merkle Tree
func (b *Block) SetMerkleRoot(root valueobjects.Hash) {
	b.header.merkleRoot = root
	b.merkleRoot = root
}

// SetSignature define a assinatura do bloco
func (b *Block) SetSignature(signature valueobjects.Signature) {
	b.header.signature = signature
}

// GetHeader retorna o header do bloco
func (b *Block) GetHeader() *BlockHeader {
	return b.header
}

// IsValid verifica se o bloco é válido
func (b *Block) IsValid() bool {
	// Validações básicas
	if b.header == nil {
		return false
	}

	if len(b.transactions) == 0 {
		return false
	}

	if b.header.timestamp.IsZero() {
		return false
	}

	return true
}

// AddTransaction adiciona uma transação ao bloco
func (b *Block) AddTransaction(tx *Transaction) {
	if tx != nil {
		b.transactions = append(b.transactions, tx)
	}
}
