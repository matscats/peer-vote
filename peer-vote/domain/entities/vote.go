package entities

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
)

// Vote representa um voto em uma eleição
type Vote struct {
	id          valueobjects.Hash
	electionID  valueobjects.Hash
	voterID     valueobjects.NodeID
	candidateID string
	timestamp   valueobjects.Timestamp
	signature   valueobjects.Signature
	isAnonymous bool
	nonce       string 
}

// VoteData representa os dados serializáveis de um voto
type VoteData struct {
	ElectionID  string `json:"election_id"`
	VoterID     string `json:"voter_id,omitempty"` // Omitido se anônimo
	CandidateID string `json:"candidate_id"`
	Timestamp   int64  `json:"timestamp"`
	IsAnonymous bool   `json:"is_anonymous"`
	Nonce       string `json:"nonce"`
}

// generateNonce gera um nonce aleatório para garantir unicidade
func generateNonce() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// NewVote cria um novo voto
func NewVote(electionID valueobjects.Hash, voterID valueobjects.NodeID, candidateID string, isAnonymous bool) *Vote {
	return &Vote{
		electionID:  electionID,
		voterID:     voterID,
		candidateID: candidateID,
		timestamp:   valueobjects.NewTimestamp(time.Now()),
		isAnonymous: isAnonymous,
		nonce:       generateNonce(),
	}
}

// GetID retorna o ID do voto
func (v *Vote) GetID() valueobjects.Hash {
	return v.id
}

// GetElectionID retorna o ID da eleição
func (v *Vote) GetElectionID() valueobjects.Hash {
	return v.electionID
}

// GetVoterID retorna o ID do eleitor
func (v *Vote) GetVoterID() valueobjects.NodeID {
	return v.voterID
}

// GetCandidateID retorna o ID do candidato
func (v *Vote) GetCandidateID() string {
	return v.candidateID
}

// GetTimestamp retorna o timestamp do voto
func (v *Vote) GetTimestamp() valueobjects.Timestamp {
	return v.timestamp
}

// GetSignature retorna a assinatura do voto
func (v *Vote) GetSignature() valueobjects.Signature {
	return v.signature
}

// IsAnonymous verifica se o voto é anônimo
func (v *Vote) IsAnonymous() bool {
	return v.isAnonymous
}

// SetID define o ID do voto
func (v *Vote) SetID(id valueobjects.Hash) {
	v.id = id
}

// SetSignature define a assinatura do voto
func (v *Vote) SetSignature(signature valueobjects.Signature) {
	v.signature = signature
}

// IsValid verifica se o voto é válido
func (v *Vote) IsValid() bool {
	// Validações básicas
	if v.electionID.IsEmpty() {
		return false
	}

	if !v.isAnonymous && v.voterID.IsEmpty() {
		return false
	}

	if v.candidateID == "" {
		return false
	}

	if v.timestamp.IsZero() {
		return false
	}

	if v.signature.IsEmpty() {
		return false
	}

	return true
}

// ToBytes serializa o voto para bytes
func (v *Vote) ToBytes() ([]byte, error) {
	data := VoteData{
		ElectionID:  v.electionID.String(),
		CandidateID: v.candidateID,
		Timestamp:   v.timestamp.Unix(),
		IsAnonymous: v.isAnonymous,
		Nonce:       v.nonce,
	}

	// Só inclui o voter ID se não for anônimo
	if !v.isAnonymous {
		data.VoterID = v.voterID.String()
	}

	return json.Marshal(data)
}

// FromBytes deserializa um voto de bytes
func (v *Vote) FromBytes(data []byte) error {
	var voteData VoteData
	if err := json.Unmarshal(data, &voteData); err != nil {
		return err
	}

	electionID, err := valueobjects.NewHashFromString(voteData.ElectionID)
	if err != nil {
		return err
	}

	v.electionID = electionID
	v.candidateID = voteData.CandidateID
	v.timestamp = valueobjects.Unix(voteData.Timestamp, 0)
	v.isAnonymous = voteData.IsAnonymous
	v.nonce = voteData.Nonce

	if !v.isAnonymous && voteData.VoterID != "" {
		v.voterID = valueobjects.NewNodeID(voteData.VoterID)
	}

	return nil
}

// Copy retorna uma cópia do voto
func (v *Vote) Copy() *Vote {
	return &Vote{
		id:          v.id.Copy(),
		electionID:  v.electionID.Copy(),
		voterID:     v.voterID.Copy(),
		candidateID: v.candidateID,
		timestamp:   v.timestamp,
		signature:   v.signature.Copy(),
		isAnonymous: v.isAnonymous,
	}
}
