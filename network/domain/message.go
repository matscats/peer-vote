package domain

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"

	"peer-vote/blockchain/domain"
	"peer-vote/crypto"
	votingdomain "peer-vote/voting/domain"
)

// MessageType represents the type of network message
type MessageType uint8

const (
	MsgTypeVote MessageType = iota
	MsgTypeBlockProposal
	MsgTypeBlockApproval
	MsgTypeSyncRequest
	MsgTypeSyncResponse
)

// Message represents a network message with type and payload
type Message struct {
	Type    MessageType
	Payload []byte
}

// VoteMessage represents a vote being broadcast over the network
type VoteMessage struct {
	Vote *votingdomain.Vote
}

// BlockProposalMessage represents a block proposal being broadcast
type BlockProposalMessage struct {
	Block *domain.Block
}

// BlockApprovalMessage represents a validator's approval of a block
type BlockApprovalMessage struct {
	BlockHash crypto.Hash
	Validator crypto.PublicKey
}

// SyncRequestMessage represents a request for blocks during synchronization
type SyncRequestMessage struct {
	FromHeight uint64
	ToHeight   uint64
}

// SyncResponseMessage represents a response containing requested blocks
type SyncResponseMessage struct {
	Blocks []*domain.Block
}

// SerializeVote serializes a vote for network transmission
// Format:
//
//	4 bytes: voterID length (big endian)
//	N bytes: voterID
//	4 bytes: publicKey length (big endian)
//	N bytes: publicKey
//	4 bytes: choice length (big endian)
//	N bytes: choice
//	8 bytes: timestamp (Unix seconds, big endian)
//	4 bytes: signature length (big endian)
//	N bytes: signature
func SerializeVote(vote *votingdomain.Vote) ([]byte, error) {
	var buf bytes.Buffer

	// Write voterID
	voterID := vote.VoterID()
	if err := binary.Write(&buf, binary.BigEndian, uint32(len(voterID))); err != nil {
		return nil, fmt.Errorf("failed to write voterID length: %w", err)
	}
	if _, err := buf.WriteString(voterID); err != nil {
		return nil, fmt.Errorf("failed to write voterID: %w", err)
	}

	// Write publicKey
	pubKey := vote.PublicKey()
	if err := binary.Write(&buf, binary.BigEndian, uint32(len(pubKey))); err != nil {
		return nil, fmt.Errorf("failed to write publicKey length: %w", err)
	}
	if _, err := buf.Write(pubKey); err != nil {
		return nil, fmt.Errorf("failed to write publicKey: %w", err)
	}

	// Write choice
	choice := vote.Choice()
	if err := binary.Write(&buf, binary.BigEndian, uint32(len(choice))); err != nil {
		return nil, fmt.Errorf("failed to write choice length: %w", err)
	}
	if _, err := buf.WriteString(choice); err != nil {
		return nil, fmt.Errorf("failed to write choice: %w", err)
	}

	// Write timestamp
	if err := binary.Write(&buf, binary.BigEndian, vote.Timestamp().Unix()); err != nil {
		return nil, fmt.Errorf("failed to write timestamp: %w", err)
	}

	// Write signature
	sig := vote.Signature()
	if err := binary.Write(&buf, binary.BigEndian, uint32(len(sig))); err != nil {
		return nil, fmt.Errorf("failed to write signature length: %w", err)
	}
	if _, err := buf.Write(sig); err != nil {
		return nil, fmt.Errorf("failed to write signature: %w", err)
	}

	return buf.Bytes(), nil
}

// DeserializeVote deserializes a vote from network data
func DeserializeVote(data []byte) (*votingdomain.Vote, error) {
	buf := bytes.NewReader(data)

	// Read voterID
	var voterIDLen uint32
	if err := binary.Read(buf, binary.BigEndian, &voterIDLen); err != nil {
		return nil, fmt.Errorf("failed to read voterID length: %w", err)
	}
	voterIDBytes := make([]byte, voterIDLen)
	if _, err := buf.Read(voterIDBytes); err != nil {
		return nil, fmt.Errorf("failed to read voterID: %w", err)
	}
	voterID := string(voterIDBytes)

	// Read publicKey
	var pubKeyLen uint32
	if err := binary.Read(buf, binary.BigEndian, &pubKeyLen); err != nil {
		return nil, fmt.Errorf("failed to read publicKey length: %w", err)
	}
	publicKey := make([]byte, pubKeyLen)
	if _, err := buf.Read(publicKey); err != nil {
		return nil, fmt.Errorf("failed to read publicKey: %w", err)
	}

	// Read choice
	var choiceLen uint32
	if err := binary.Read(buf, binary.BigEndian, &choiceLen); err != nil {
		return nil, fmt.Errorf("failed to read choice length: %w", err)
	}
	choiceBytes := make([]byte, choiceLen)
	if _, err := buf.Read(choiceBytes); err != nil {
		return nil, fmt.Errorf("failed to read choice: %w", err)
	}
	choice := string(choiceBytes)

	// Read timestamp
	var timestampUnix int64
	if err := binary.Read(buf, binary.BigEndian, &timestampUnix); err != nil {
		return nil, fmt.Errorf("failed to read timestamp: %w", err)
	}
	timestamp := timestampFromUnix(timestampUnix)

	// Read signature
	var sigLen uint32
	if err := binary.Read(buf, binary.BigEndian, &sigLen); err != nil {
		return nil, fmt.Errorf("failed to read signature length: %w", err)
	}
	signature := make([]byte, sigLen)
	if _, err := buf.Read(signature); err != nil {
		return nil, fmt.Errorf("failed to read signature: %w", err)
	}

	// Create vote WITH validation - this will verify the signature
	vote, err := votingdomain.NewVote(voterID, choice, timestamp, signature, publicKey)
	if err != nil {
		return nil, fmt.Errorf("vote validation failed: %w", err)
	}

	return vote, nil
}

// SerializeBlock serializes a block for network transmission
// Uses the Block's built-in Serialize method
func SerializeBlock(block *domain.Block) ([]byte, error) {
	return block.Serialize()
}

// DeserializeBlock deserializes a block from network data
// Uses the Block's built-in DeserializeBlock function
func DeserializeBlock(data []byte) (*domain.Block, error) {
	return domain.DeserializeBlock(data)
}

// SerializeApproval serializes a block approval for network transmission
// Format:
//
//	32 bytes: block hash
//	4 bytes: validator public key length (big endian)
//	N bytes: validator public key
func SerializeApproval(blockHash crypto.Hash, validator crypto.PublicKey) ([]byte, error) {
	var buf bytes.Buffer

	// Write block hash
	if _, err := buf.Write(blockHash.Bytes()); err != nil {
		return nil, fmt.Errorf("failed to write block hash: %w", err)
	}

	// Write validator public key
	if err := binary.Write(&buf, binary.BigEndian, uint32(len(validator))); err != nil {
		return nil, fmt.Errorf("failed to write validator length: %w", err)
	}
	if _, err := buf.Write(validator); err != nil {
		return nil, fmt.Errorf("failed to write validator: %w", err)
	}

	return buf.Bytes(), nil
}

// DeserializeApproval deserializes a block approval from network data
func DeserializeApproval(data []byte) (crypto.Hash, crypto.PublicKey, error) {
	buf := bytes.NewReader(data)

	// Read block hash
	hashBytes := make([]byte, 32)
	if _, err := buf.Read(hashBytes); err != nil {
		return crypto.Hash{}, nil, fmt.Errorf("failed to read block hash: %w", err)
	}
	var blockHash crypto.Hash
	copy(blockHash[:], hashBytes)

	// Read validator public key
	var validatorLen uint32
	if err := binary.Read(buf, binary.BigEndian, &validatorLen); err != nil {
		return crypto.Hash{}, nil, fmt.Errorf("failed to read validator length: %w", err)
	}
	validator := make([]byte, validatorLen)
	if _, err := buf.Read(validator); err != nil {
		return crypto.Hash{}, nil, fmt.Errorf("failed to read validator: %w", err)
	}

	return blockHash, validator, nil
}

// SerializeSyncRequest serializes a sync request for network transmission
// Format:
//
//	8 bytes: from height (big endian)
//	8 bytes: to height (big endian)
func SerializeSyncRequest(fromHeight, toHeight uint64) ([]byte, error) {
	var buf bytes.Buffer

	if err := binary.Write(&buf, binary.BigEndian, fromHeight); err != nil {
		return nil, fmt.Errorf("failed to write from height: %w", err)
	}

	if err := binary.Write(&buf, binary.BigEndian, toHeight); err != nil {
		return nil, fmt.Errorf("failed to write to height: %w", err)
	}

	return buf.Bytes(), nil
}

// DeserializeSyncRequest deserializes a sync request from network data
func DeserializeSyncRequest(data []byte) (uint64, uint64, error) {
	buf := bytes.NewReader(data)

	var fromHeight uint64
	if err := binary.Read(buf, binary.BigEndian, &fromHeight); err != nil {
		return 0, 0, fmt.Errorf("failed to read from height: %w", err)
	}

	var toHeight uint64
	if err := binary.Read(buf, binary.BigEndian, &toHeight); err != nil {
		return 0, 0, fmt.Errorf("failed to read to height: %w", err)
	}

	return fromHeight, toHeight, nil
}

// SerializeSyncResponse serializes a sync response for network transmission
// Format:
//
//	4 bytes: number of blocks (big endian)
//	For each block:
//	  4 bytes: block data length (big endian)
//	  N bytes: serialized block data
func SerializeSyncResponse(blocks []*domain.Block) ([]byte, error) {
	var buf bytes.Buffer

	// Write number of blocks
	if err := binary.Write(&buf, binary.BigEndian, uint32(len(blocks))); err != nil {
		return nil, fmt.Errorf("failed to write block count: %w", err)
	}

	// Write each block
	for i, block := range blocks {
		blockData, err := block.Serialize()
		if err != nil {
			return nil, fmt.Errorf("failed to serialize block %d: %w", i, err)
		}

		// Write block length
		if err := binary.Write(&buf, binary.BigEndian, uint32(len(blockData))); err != nil {
			return nil, fmt.Errorf("failed to write block %d length: %w", i, err)
		}

		// Write block data
		if _, err := buf.Write(blockData); err != nil {
			return nil, fmt.Errorf("failed to write block %d data: %w", i, err)
		}
	}

	return buf.Bytes(), nil
}

// DeserializeSyncResponse deserializes a sync response from network data
func DeserializeSyncResponse(data []byte) ([]*domain.Block, error) {
	buf := bytes.NewReader(data)

	// Read number of blocks
	var blockCount uint32
	if err := binary.Read(buf, binary.BigEndian, &blockCount); err != nil {
		return nil, fmt.Errorf("failed to read block count: %w", err)
	}

	// Read each block
	blocks := make([]*domain.Block, blockCount)
	for i := uint32(0); i < blockCount; i++ {
		// Read block length
		var blockLen uint32
		if err := binary.Read(buf, binary.BigEndian, &blockLen); err != nil {
			return nil, fmt.Errorf("failed to read block %d length: %w", i, err)
		}

		// Read block data
		blockData := make([]byte, blockLen)
		if _, err := buf.Read(blockData); err != nil {
			return nil, fmt.Errorf("failed to read block %d data: %w", i, err)
		}

		// Deserialize block
		block, err := domain.DeserializeBlock(blockData)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize block %d: %w", i, err)
		}

		blocks[i] = block
	}

	return blocks, nil
}

// timestampFromUnix is a helper to create a time.Time from Unix timestamp
func timestampFromUnix(unix int64) time.Time {
	return time.Unix(unix, 0).UTC()
}
