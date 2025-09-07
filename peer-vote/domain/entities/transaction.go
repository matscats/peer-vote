package entities

import (
	"time"

	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
)

// TransactionType define o tipo de transação
type TransactionType string

const (
	// VoteTransaction representa uma transação de voto
	VoteTransaction TransactionType = "VOTE"
	// ElectionTransaction representa uma transação de criação de eleição
	ElectionTransaction TransactionType = "ELECTION"
	// ValidatorTransaction representa uma transação de validador
	ValidatorTransaction TransactionType = "VALIDATOR"
)

// Transaction representa uma transação na blockchain
type Transaction struct {
	id        valueobjects.Hash
	txType    TransactionType
	from      valueobjects.NodeID
	to        valueobjects.NodeID
	data      []byte
	timestamp valueobjects.Timestamp
	signature valueobjects.Signature
	hash      valueobjects.Hash
}

// NewTransaction cria uma nova transação
func NewTransaction(txType TransactionType, from, to valueobjects.NodeID, data []byte) *Transaction {
	return &Transaction{
		txType:    txType,
		from:      from,
		to:        to,
		data:      data,
		timestamp: valueobjects.NewTimestamp(time.Now()),
	}
}

// GetID retorna o ID da transação
func (t *Transaction) GetID() valueobjects.Hash {
	return t.id
}

// GetType retorna o tipo da transação
func (t *Transaction) GetType() TransactionType {
	return t.txType
}

// GetFrom retorna o remetente da transação
func (t *Transaction) GetFrom() valueobjects.NodeID {
	return t.from
}

// GetTo retorna o destinatário da transação
func (t *Transaction) GetTo() valueobjects.NodeID {
	return t.to
}

// GetData retorna os dados da transação
func (t *Transaction) GetData() []byte {
	return t.data
}

// GetTimestamp retorna o timestamp da transação
func (t *Transaction) GetTimestamp() valueobjects.Timestamp {
	return t.timestamp
}

// GetSignature retorna a assinatura da transação
func (t *Transaction) GetSignature() valueobjects.Signature {
	return t.signature
}

// GetHash retorna o hash da transação
func (t *Transaction) GetHash() valueobjects.Hash {
	return t.hash
}

// SetID define o ID da transação
func (t *Transaction) SetID(id valueobjects.Hash) {
	t.id = id
}

// SetSignature define a assinatura da transação
func (t *Transaction) SetSignature(signature valueobjects.Signature) {
	t.signature = signature
}

// SetHash define o hash da transação
func (t *Transaction) SetHash(hash valueobjects.Hash) {
	t.hash = hash
}

// IsValid verifica se a transação é válida
func (t *Transaction) IsValid() bool {
	// Validações básicas
	if t.txType == "" {
		return false
	}

	if t.from.IsEmpty() {
		return false
	}

	if t.timestamp.IsZero() {
		return false
	}

	if len(t.data) == 0 {
		return false
	}

	return true
}

// ToBytes serializa a transação para bytes
func (t *Transaction) ToBytes() []byte {
	// Esta implementação será feita na camada de infraestrutura
	// Por enquanto, retorna os dados básicos
	return t.data
}
