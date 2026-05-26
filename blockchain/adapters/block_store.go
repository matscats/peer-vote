package adapters

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"peer-vote/blockchain/domain"
	"peer-vote/crypto"
)

const (
	// MagicNumber is a 4-byte prefix for block data (similar to Bitcoin's 0xD9B4BEF9)
	MagicNumber uint32 = 0xD9B4BEF9

	// MaxFileSize is the maximum size of a .dat file before rotation (100 MB)
	MaxFileSize int64 = 100 * 1024 * 1024

	// BlockFilePrefix is the prefix for block data files
	BlockFilePrefix = "blk"

	// IndexFileName is the name of the index file
	IndexFileName = "index.dat"
)

// BlockIndex represents an entry in the block index
type BlockIndex struct {
	Height     uint64 // Block height
	FileNumber uint32 // File number (e.g., 0 for blk00000.dat)
	Offset     int64  // Offset within the file
	Size       uint32 // Size of the block data
	Hash       crypto.Hash
}

// BlockStore implements the BlockRepository port using file-based storage
// Blocks are stored in .dat files similar to Bitcoin's blk*.dat format
type BlockStore struct {
	dataDir           string
	currentFile       uint32
	currentOffset     int64
	index             map[uint64]*BlockIndex // height -> index entry
	hashIndex         map[crypto.Hash]uint64 // hash -> height
	mu                sync.RWMutex
	currentFileHandle *os.File
}

// NewBlockStore creates a new BlockStore
func NewBlockStore(dataDir string) (*BlockStore, error) {
	// Create data directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	store := &BlockStore{
		dataDir:   dataDir,
		index:     make(map[uint64]*BlockIndex),
		hashIndex: make(map[crypto.Hash]uint64),
	}

	// Load existing index
	if err := store.loadIndex(); err != nil {
		return nil, fmt.Errorf("failed to load index: %w", err)
	}

	// Determine current file and offset
	if len(store.index) > 0 {
		// Find the highest file number and offset
		maxHeight := uint64(0)
		for height := range store.index {
			if height > maxHeight {
				maxHeight = height
			}
		}
		lastEntry := store.index[maxHeight]
		store.currentFile = lastEntry.FileNumber
		store.currentOffset = lastEntry.Offset + int64(lastEntry.Size) + 8 // +8 for magic and size prefix
	}

	return store, nil
}

// Store persists a block to storage
func (s *BlockStore) Store(block *domain.Block) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if block already exists
	if _, exists := s.index[block.Height()]; exists {
		return fmt.Errorf("block at height %d already exists", block.Height())
	}

	// Serialize the block
	blockData, err := block.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize block: %w", err)
	}

	// Check if we need to rotate to a new file
	if s.currentOffset+int64(len(blockData))+8 > MaxFileSize {
		if err := s.rotateFile(); err != nil {
			return fmt.Errorf("failed to rotate file: %w", err)
		}
	}

	// Open current file for appending
	filePath := s.getFilePath(s.currentFile)
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	// Get current offset
	offset, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("failed to seek to end of file: %w", err)
	}

	// Write magic number
	if err := binary.Write(file, binary.BigEndian, MagicNumber); err != nil {
		return fmt.Errorf("failed to write magic number: %w", err)
	}

	// Write block size
	blockSize := uint32(len(blockData))
	if err := binary.Write(file, binary.BigEndian, blockSize); err != nil {
		return fmt.Errorf("failed to write block size: %w", err)
	}

	// Write block data
	if _, err := file.Write(blockData); err != nil {
		return fmt.Errorf("failed to write block data: %w", err)
	}

	// Sync to disk
	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	// Update index
	indexEntry := &BlockIndex{
		Height:     block.Height(),
		FileNumber: s.currentFile,
		Offset:     offset,
		Size:       blockSize,
		Hash:       block.Hash(),
	}
	s.index[block.Height()] = indexEntry
	s.hashIndex[block.Hash()] = block.Height()

	// Update current offset
	s.currentOffset = offset + int64(blockSize) + 8

	// Persist index
	if err := s.saveIndex(); err != nil {
		return fmt.Errorf("failed to save index: %w", err)
	}

	return nil
}

// GetByHeight retrieves a block by its height
func (s *BlockStore) GetByHeight(height uint64) (*domain.Block, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	indexEntry, exists := s.index[height]
	if !exists {
		return nil, fmt.Errorf("block at height %d not found", height)
	}

	return s.readBlock(indexEntry)
}

// GetByHash retrieves a block by its hash
func (s *BlockStore) GetByHash(hash crypto.Hash) (*domain.Block, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	height, exists := s.hashIndex[hash]
	if !exists {
		return nil, fmt.Errorf("block with hash %s not found", hash.String())
	}

	indexEntry := s.index[height]
	return s.readBlock(indexEntry)
}

// GetTip retrieves the current chain tip (highest block)
func (s *BlockStore) GetTip() (*domain.Block, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.index) == 0 {
		return nil, fmt.Errorf("no blocks in storage")
	}

	// Find the highest height
	maxHeight := uint64(0)
	for height := range s.index {
		if height > maxHeight {
			maxHeight = height
		}
	}

	indexEntry := s.index[maxHeight]
	return s.readBlock(indexEntry)
}

// GetAll retrieves all blocks in the chain
func (s *BlockStore) GetAll() ([]*domain.Block, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.index) == 0 {
		return []*domain.Block{}, nil
	}

	// Find the highest height
	maxHeight := uint64(0)
	for height := range s.index {
		if height > maxHeight {
			maxHeight = height
		}
	}

	// Read blocks in order
	blocks := make([]*domain.Block, maxHeight+1)
	for height := uint64(0); height <= maxHeight; height++ {
		indexEntry, exists := s.index[height]
		if !exists {
			return nil, fmt.Errorf("missing block at height %d", height)
		}

		block, err := s.readBlock(indexEntry)
		if err != nil {
			return nil, fmt.Errorf("failed to read block at height %d: %w", height, err)
		}

		blocks[height] = block
	}

	return blocks, nil
}

// readBlock reads a block from disk using the index entry
func (s *BlockStore) readBlock(indexEntry *BlockIndex) (*domain.Block, error) {
	filePath := s.getFilePath(indexEntry.FileNumber)

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	// Seek to the block offset
	if _, err := file.Seek(indexEntry.Offset, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to offset %d: %w", indexEntry.Offset, err)
	}

	// Read and verify magic number
	var magic uint32
	if err := binary.Read(file, binary.BigEndian, &magic); err != nil {
		return nil, fmt.Errorf("failed to read magic number: %w", err)
	}
	if magic != MagicNumber {
		return nil, fmt.Errorf("invalid magic number: expected 0x%X, got 0x%X", MagicNumber, magic)
	}

	// Read block size
	var size uint32
	if err := binary.Read(file, binary.BigEndian, &size); err != nil {
		return nil, fmt.Errorf("failed to read block size: %w", err)
	}

	// Verify size matches index
	if size != indexEntry.Size {
		return nil, fmt.Errorf("block size mismatch: index says %d, file says %d", indexEntry.Size, size)
	}

	// Read block data
	blockData := make([]byte, size)
	if _, err := io.ReadFull(file, blockData); err != nil {
		return nil, fmt.Errorf("failed to read block data: %w", err)
	}

	// Deserialize block
	block, err := domain.DeserializeBlock(blockData)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize block: %w", err)
	}

	// Verify block hash matches index
	if block.Hash() != indexEntry.Hash {
		return nil, fmt.Errorf("block hash mismatch: index says %s, block says %s",
			indexEntry.Hash.String(), block.Hash().String())
	}

	return block, nil
}

// rotateFile closes the current file and opens a new one
func (s *BlockStore) rotateFile() error {
	if s.currentFileHandle != nil {
		if err := s.currentFileHandle.Close(); err != nil {
			return fmt.Errorf("failed to close current file: %w", err)
		}
		s.currentFileHandle = nil
	}

	s.currentFile++
	s.currentOffset = 0

	return nil
}

// getFilePath returns the file path for a given file number
func (s *BlockStore) getFilePath(fileNumber uint32) string {
	filename := fmt.Sprintf("%s%05d.dat", BlockFilePrefix, fileNumber)
	return filepath.Join(s.dataDir, filename)
}

// loadIndex loads the index from disk
func (s *BlockStore) loadIndex() error {
	indexPath := filepath.Join(s.dataDir, IndexFileName)

	// Check if index file exists
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		// No index file, start fresh
		return nil
	}

	file, err := os.Open(indexPath)
	if err != nil {
		return fmt.Errorf("failed to open index file: %w", err)
	}
	defer file.Close()

	// Read index entries
	for {
		var entry BlockIndex

		// Read height
		if err := binary.Read(file, binary.BigEndian, &entry.Height); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to read height: %w", err)
		}

		// Read file number
		if err := binary.Read(file, binary.BigEndian, &entry.FileNumber); err != nil {
			return fmt.Errorf("failed to read file number: %w", err)
		}

		// Read offset
		if err := binary.Read(file, binary.BigEndian, &entry.Offset); err != nil {
			return fmt.Errorf("failed to read offset: %w", err)
		}

		// Read size
		if err := binary.Read(file, binary.BigEndian, &entry.Size); err != nil {
			return fmt.Errorf("failed to read size: %w", err)
		}

		// Read hash
		hashBytes := make([]byte, 32)
		if _, err := io.ReadFull(file, hashBytes); err != nil {
			return fmt.Errorf("failed to read hash: %w", err)
		}
		copy(entry.Hash[:], hashBytes)

		// Add to index
		s.index[entry.Height] = &entry
		s.hashIndex[entry.Hash] = entry.Height
	}

	return nil
}

// saveIndex saves the index to disk
func (s *BlockStore) saveIndex() error {
	indexPath := filepath.Join(s.dataDir, IndexFileName)

	// Create temporary file
	tempPath := indexPath + ".tmp"
	file, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create temp index file: %w", err)
	}

	// Write all index entries in height order
	maxHeight := uint64(0)
	for height := range s.index {
		if height > maxHeight {
			maxHeight = height
		}
	}

	for height := uint64(0); height <= maxHeight; height++ {
		entry, exists := s.index[height]
		if !exists {
			file.Close()
			os.Remove(tempPath)
			return fmt.Errorf("missing index entry for height %d", height)
		}

		// Write height
		if err := binary.Write(file, binary.BigEndian, entry.Height); err != nil {
			file.Close()
			os.Remove(tempPath)
			return fmt.Errorf("failed to write height: %w", err)
		}

		// Write file number
		if err := binary.Write(file, binary.BigEndian, entry.FileNumber); err != nil {
			file.Close()
			os.Remove(tempPath)
			return fmt.Errorf("failed to write file number: %w", err)
		}

		// Write offset
		if err := binary.Write(file, binary.BigEndian, entry.Offset); err != nil {
			file.Close()
			os.Remove(tempPath)
			return fmt.Errorf("failed to write offset: %w", err)
		}

		// Write size
		if err := binary.Write(file, binary.BigEndian, entry.Size); err != nil {
			file.Close()
			os.Remove(tempPath)
			return fmt.Errorf("failed to write size: %w", err)
		}

		// Write hash
		if _, err := file.Write(entry.Hash.Bytes()); err != nil {
			file.Close()
			os.Remove(tempPath)
			return fmt.Errorf("failed to write hash: %w", err)
		}
	}

	// Sync to disk
	if err := file.Sync(); err != nil {
		file.Close()
		os.Remove(tempPath)
		return fmt.Errorf("failed to sync index file: %w", err)
	}

	file.Close()

	// Atomically replace old index with new one
	if err := os.Rename(tempPath, indexPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename temp index file: %w", err)
	}

	return nil
}
