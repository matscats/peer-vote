package domain

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"

	"peer-vote/crypto"
	"peer-vote/voting/domain"
)

// Block represents a data structure containing a collection of votes, metadata, and cryptographic proof
// This is a domain entity with behavior that enforces blockchain invariants
type Block struct {
	height       uint64
	previousHash crypto.Hash
	timestamp    time.Time
	votes        []*domain.Vote
	proposer     crypto.PublicKey
	signature    crypto.Signature
	hash         crypto.Hash // Computed once, immutable
}

// NewBlock creates a new Block entity
// Constructor enforces height and hash chain invariants
func NewBlock(height uint64, prevHash crypto.Hash, votes []*domain.Vote, proposer crypto.PublicKey) (*Block, error) {
	if height > 0 && prevHash == (crypto.Hash{}) {
		return nil, fmt.Errorf("non-genesis block must have valid previous hash")
	}

	if len(proposer) == 0 {
		return nil, fmt.Errorf("proposer public key cannot be empty")
	}

	// Create block without signature and hash (will be set later)
	block := &Block{
		height:       height,
		previousHash: prevHash,
		timestamp:    time.Now().UTC(),
		votes:        votes,
		proposer:     proposer,
	}

	return block, nil
}

// Sign signs the block with the given signer and computes the block hash
// This method should be called by the block proposer after creating the block
func (b *Block) Sign(signer crypto.Signer) error {
	if b.signature != nil {
		return fmt.Errorf("block already signed")
	}

	// Verify the signer's public key matches the proposer
	if !bytes.Equal(signer.PublicKey(), b.proposer) {
		return fmt.Errorf("signer public key does not match proposer")
	}

	// Build message to sign (all block data except signature and hash)
	message := b.buildSignatureMessage()

	// Sign the message
	sig, err := signer.Sign(message)
	if err != nil {
		return fmt.Errorf("failed to sign block: %w", err)
	}

	b.signature = sig

	// Compute and set the block hash (includes signature)
	b.hash = b.ComputeHash()

	return nil
}

// Verify verifies the block's signature and structural integrity
func (b *Block) Verify() error {
	if b.signature == nil {
		return fmt.Errorf("block has no signature")
	}

	if b.hash == (crypto.Hash{}) {
		return fmt.Errorf("block has no hash")
	}

	// Verify signature
	message := b.buildSignatureMessage()
	if err := b.signature.Verify(b.proposer, message); err != nil {
		return fmt.Errorf("invalid block signature: %w", err)
	}

	// Verify hash is correct
	computedHash := b.ComputeHash()
	if b.hash != computedHash {
		return fmt.Errorf("block hash mismatch: expected %s, got %s", b.hash.String(), computedHash.String())
	}

	// Verify height invariant for non-genesis blocks
	if b.height > 0 && b.previousHash == (crypto.Hash{}) {
		return fmt.Errorf("non-genesis block must have valid previous hash")
	}

	return nil
}

// ComputeHash computes the SHA-256 hash of the block
// The hash includes all block data including the signature
func (b *Block) ComputeHash() crypto.Hash {
	var buf bytes.Buffer

	// Write height
	binary.Write(&buf, binary.BigEndian, b.height)

	// Write previous hash
	buf.Write(b.previousHash.Bytes())

	// Write timestamp
	binary.Write(&buf, binary.BigEndian, b.timestamp.Unix())

	// Write votes
	for _, vote := range b.votes {
		buf.WriteString(vote.VoterID())
		buf.WriteString(vote.Choice())
		binary.Write(&buf, binary.BigEndian, vote.Timestamp().Unix())
		buf.Write(vote.Signature())
	}

	// Write proposer
	buf.Write(b.proposer)

	// Write signature (if present)
	if b.signature != nil {
		buf.Write(b.signature)
	}

	return crypto.NewHash(buf.Bytes())
}

// buildSignatureMessage builds the message that should be signed
// This includes all block data except the signature and hash
func (b *Block) buildSignatureMessage() []byte {
	var buf bytes.Buffer

	// Write height
	binary.Write(&buf, binary.BigEndian, b.height)

	// Write previous hash
	buf.Write(b.previousHash.Bytes())

	// Write timestamp
	binary.Write(&buf, binary.BigEndian, b.timestamp.Unix())

	// Write votes
	for _, vote := range b.votes {
		buf.WriteString(vote.VoterID())
		buf.WriteString(vote.Choice())
		binary.Write(&buf, binary.BigEndian, vote.Timestamp().Unix())
		buf.Write(vote.Signature())
	}

	// Write proposer
	buf.Write(b.proposer)

	return buf.Bytes()
}

// Height returns the block height
func (b *Block) Height() uint64 {
	return b.height
}

// PreviousHash returns the hash of the previous block
func (b *Block) PreviousHash() crypto.Hash {
	return b.previousHash
}

// Timestamp returns the block creation timestamp
func (b *Block) Timestamp() time.Time {
	return b.timestamp
}

// Votes returns the votes included in the block
func (b *Block) Votes() []*domain.Vote {
	return b.votes
}

// Proposer returns the public key of the block proposer
func (b *Block) Proposer() crypto.PublicKey {
	return b.proposer
}

// Signature returns the block signature
func (b *Block) Signature() crypto.Signature {
	return b.signature
}

// Hash returns the block hash
func (b *Block) Hash() crypto.Hash {
	return b.hash
}

// Serialize serializes the block to bytes using deterministic encoding
// Format:
//
//	8 bytes: height (big endian)
//	32 bytes: previous hash
//	8 bytes: timestamp (Unix seconds, big endian)
//	4 bytes: number of votes (big endian)
//	For each vote:
//	  4 bytes: voterID length (big endian)
//	  N bytes: voterID
//	  4 bytes: choice length (big endian)
//	  N bytes: choice
//	  8 bytes: timestamp (Unix seconds, big endian)
//	  4 bytes: signature length (big endian)
//	  N bytes: signature
//	4 bytes: proposer length (big endian)
//	N bytes: proposer public key
//	4 bytes: signature length (big endian)
//	N bytes: block signature
//	32 bytes: block hash
func (b *Block) Serialize() ([]byte, error) {
	var buf bytes.Buffer

	// Write height
	if err := binary.Write(&buf, binary.BigEndian, b.height); err != nil {
		return nil, fmt.Errorf("failed to write height: %w", err)
	}

	// Write previous hash
	if _, err := buf.Write(b.previousHash.Bytes()); err != nil {
		return nil, fmt.Errorf("failed to write previous hash: %w", err)
	}

	// Write timestamp
	if err := binary.Write(&buf, binary.BigEndian, b.timestamp.Unix()); err != nil {
		return nil, fmt.Errorf("failed to write timestamp: %w", err)
	}

	// Write number of votes
	if err := binary.Write(&buf, binary.BigEndian, uint32(len(b.votes))); err != nil {
		return nil, fmt.Errorf("failed to write vote count: %w", err)
	}

	// Write each vote
	for i, vote := range b.votes {
		// VoterID
		voterID := vote.VoterID()
		if err := binary.Write(&buf, binary.BigEndian, uint32(len(voterID))); err != nil {
			return nil, fmt.Errorf("failed to write voterID length for vote %d: %w", i, err)
		}
		if _, err := buf.WriteString(voterID); err != nil {
			return nil, fmt.Errorf("failed to write voterID for vote %d: %w", i, err)
		}

		// PublicKey
		pubKey := vote.PublicKey()
		if err := binary.Write(&buf, binary.BigEndian, uint32(len(pubKey))); err != nil {
			return nil, fmt.Errorf("failed to write publicKey length for vote %d: %w", i, err)
		}
		if _, err := buf.Write(pubKey); err != nil {
			return nil, fmt.Errorf("failed to write publicKey for vote %d: %w", i, err)
		}

		// Choice
		choice := vote.Choice()
		if err := binary.Write(&buf, binary.BigEndian, uint32(len(choice))); err != nil {
			return nil, fmt.Errorf("failed to write choice length for vote %d: %w", i, err)
		}
		if _, err := buf.WriteString(choice); err != nil {
			return nil, fmt.Errorf("failed to write choice for vote %d: %w", i, err)
		}

		// Timestamp
		if err := binary.Write(&buf, binary.BigEndian, vote.Timestamp().Unix()); err != nil {
			return nil, fmt.Errorf("failed to write timestamp for vote %d: %w", i, err)
		}

		// Signature
		sig := vote.Signature()
		if err := binary.Write(&buf, binary.BigEndian, uint32(len(sig))); err != nil {
			return nil, fmt.Errorf("failed to write signature length for vote %d: %w", i, err)
		}
		if _, err := buf.Write(sig); err != nil {
			return nil, fmt.Errorf("failed to write signature for vote %d: %w", i, err)
		}
	}

	// Write proposer
	if err := binary.Write(&buf, binary.BigEndian, uint32(len(b.proposer))); err != nil {
		return nil, fmt.Errorf("failed to write proposer length: %w", err)
	}
	if _, err := buf.Write(b.proposer); err != nil {
		return nil, fmt.Errorf("failed to write proposer: %w", err)
	}

	// Write block signature
	if err := binary.Write(&buf, binary.BigEndian, uint32(len(b.signature))); err != nil {
		return nil, fmt.Errorf("failed to write signature length: %w", err)
	}
	if _, err := buf.Write(b.signature); err != nil {
		return nil, fmt.Errorf("failed to write signature: %w", err)
	}

	// Write block hash
	if _, err := buf.Write(b.hash.Bytes()); err != nil {
		return nil, fmt.Errorf("failed to write hash: %w", err)
	}

	return buf.Bytes(), nil
}

// DeserializeBlock deserializes a block from bytes with validation
// This function reconstructs a Block entity from serialized data and validates its integrity
func DeserializeBlock(data []byte) (*Block, error) {
	buf := bytes.NewReader(data)

	// Read height
	var height uint64
	if err := binary.Read(buf, binary.BigEndian, &height); err != nil {
		return nil, fmt.Errorf("failed to read height: %w", err)
	}

	// Read previous hash
	prevHashBytes := make([]byte, 32)
	if _, err := buf.Read(prevHashBytes); err != nil {
		return nil, fmt.Errorf("failed to read previous hash: %w", err)
	}
	var previousHash crypto.Hash
	copy(previousHash[:], prevHashBytes)

	// Read timestamp
	var timestampUnix int64
	if err := binary.Read(buf, binary.BigEndian, &timestampUnix); err != nil {
		return nil, fmt.Errorf("failed to read timestamp: %w", err)
	}
	timestamp := time.Unix(timestampUnix, 0).UTC()

	// Read number of votes
	var voteCount uint32
	if err := binary.Read(buf, binary.BigEndian, &voteCount); err != nil {
		return nil, fmt.Errorf("failed to read vote count: %w", err)
	}

	// Read each vote
	votes := make([]*domain.Vote, voteCount)
	for i := uint32(0); i < voteCount; i++ {
		// Read voterID
		var voterIDLen uint32
		if err := binary.Read(buf, binary.BigEndian, &voterIDLen); err != nil {
			return nil, fmt.Errorf("failed to read voterID length for vote %d: %w", i, err)
		}
		voterIDBytes := make([]byte, voterIDLen)
		if _, err := buf.Read(voterIDBytes); err != nil {
			return nil, fmt.Errorf("failed to read voterID for vote %d: %w", i, err)
		}
		voterID := string(voterIDBytes)

		// Read publicKey
		var pubKeyLen uint32
		if err := binary.Read(buf, binary.BigEndian, &pubKeyLen); err != nil {
			return nil, fmt.Errorf("failed to read publicKey length for vote %d: %w", i, err)
		}
		publicKey := make([]byte, pubKeyLen)
		if _, err := buf.Read(publicKey); err != nil {
			return nil, fmt.Errorf("failed to read publicKey for vote %d: %w", i, err)
		}

		// Read choice
		var choiceLen uint32
		if err := binary.Read(buf, binary.BigEndian, &choiceLen); err != nil {
			return nil, fmt.Errorf("failed to read choice length for vote %d: %w", i, err)
		}
		choiceBytes := make([]byte, choiceLen)
		if _, err := buf.Read(choiceBytes); err != nil {
			return nil, fmt.Errorf("failed to read choice for vote %d: %w", i, err)
		}
		choice := string(choiceBytes)

		// Read timestamp
		var voteTimestampUnix int64
		if err := binary.Read(buf, binary.BigEndian, &voteTimestampUnix); err != nil {
			return nil, fmt.Errorf("failed to read timestamp for vote %d: %w", i, err)
		}
		voteTimestamp := time.Unix(voteTimestampUnix, 0).UTC()

		// Read signature
		var sigLen uint32
		if err := binary.Read(buf, binary.BigEndian, &sigLen); err != nil {
			return nil, fmt.Errorf("failed to read signature length for vote %d: %w", i, err)
		}
		signature := make([]byte, sigLen)
		if _, err := buf.Read(signature); err != nil {
			return nil, fmt.Errorf("failed to read signature for vote %d: %w", i, err)
		}

		// Create vote without validation (we'll validate the block as a whole)
		// We use NewVoteUnsafe here because block validation will verify all votes
		votes[i] = domain.NewVoteUnsafe(voterID, choice, voteTimestamp, signature, publicKey)
	}

	// Read proposer
	var proposerLen uint32
	if err := binary.Read(buf, binary.BigEndian, &proposerLen); err != nil {
		return nil, fmt.Errorf("failed to read proposer length: %w", err)
	}
	proposer := make([]byte, proposerLen)
	if _, err := buf.Read(proposer); err != nil {
		return nil, fmt.Errorf("failed to read proposer: %w", err)
	}

	// Read block signature
	var sigLen uint32
	if err := binary.Read(buf, binary.BigEndian, &sigLen); err != nil {
		return nil, fmt.Errorf("failed to read signature length: %w", err)
	}
	signature := make([]byte, sigLen)
	if _, err := buf.Read(signature); err != nil {
		return nil, fmt.Errorf("failed to read signature: %w", err)
	}

	// Read block hash
	hashBytes := make([]byte, 32)
	if _, err := buf.Read(hashBytes); err != nil {
		return nil, fmt.Errorf("failed to read hash: %w", err)
	}
	var hash crypto.Hash
	copy(hash[:], hashBytes)

	// Create block with all fields
	block := &Block{
		height:       height,
		previousHash: previousHash,
		timestamp:    timestamp,
		votes:        votes,
		proposer:     proposer,
		signature:    signature,
		hash:         hash,
	}

	if block.Height() == 0 {
		if err := VerifyGenesisBlock(block); err != nil {
			return nil, fmt.Errorf("deserialized genesis block failed validation: %w", err)
		}
	} else if err := block.Verify(); err != nil {
		return nil, fmt.Errorf("deserialized block failed validation: %w", err)
	}

	return block, nil
}
