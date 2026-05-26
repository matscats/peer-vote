package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: analyze_blocks <data_dir>")
	}

	dataDir := os.Args[1]

	// Read block file
	blockFile := filepath.Join(dataDir, "blk00000.dat")
	f, err := os.Open(blockFile)
	if err != nil {
		log.Fatalf("Failed to open block file: %v", err)
	}
	defer f.Close()

	fmt.Printf("Analyzing blocks from: %s\n\n", blockFile)

	blockNum := 0
	totalVotes := 0
	blocksWithVotes := 0
	votersSeen := make(map[string]string) // voterID -> choice

	for {
		// Read magic number (4 bytes)
		var magic uint32
		if err := binary.Read(f, binary.BigEndian, &magic); err != nil {
			if err == io.EOF {
				break
			}
			log.Fatalf("Failed to read magic number: %v", err)
		}

		// Read block size
		var blockSize uint32
		if err := binary.Read(f, binary.BigEndian, &blockSize); err != nil {
			log.Fatalf("Failed to read block size: %v", err)
		}

		// Read block data
		blockData := make([]byte, blockSize)
		if _, err := io.ReadFull(f, blockData); err != nil {
			log.Fatalf("Failed to read block %d data (size %d): %v", blockNum, blockSize, err)
		}

		// Parse block to count votes
		voteCount, voters := parseBlockVotes(blockData)

		if voteCount > 0 {
			blocksWithVotes++
			totalVotes += voteCount
			fmt.Printf("Block %d: %d votes\n", blockNum, voteCount)
			for voterID, choice := range voters {
				fmt.Printf("  - %s voted for %s\n", voterID, choice)
				if prevChoice, exists := votersSeen[voterID]; exists {
					fmt.Printf("    ⚠️  WARNING: %s already voted for %s!\n", voterID, prevChoice)
				}
				votersSeen[voterID] = choice
			}
		}

		blockNum++
	}

	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("Total blocks: %d\n", blockNum)
	fmt.Printf("Blocks with votes: %d\n", blocksWithVotes)
	fmt.Printf("Blocks without votes: %d\n", blockNum-blocksWithVotes)
	fmt.Printf("Total votes across all blocks: %d\n", totalVotes)
	fmt.Printf("Unique voters: %d\n", len(votersSeen))

	if totalVotes > len(votersSeen) {
		fmt.Printf("\n⚠️  WARNING: Vote duplication detected!\n")
		fmt.Printf("   Total votes: %d, Unique voters: %d\n", totalVotes, len(votersSeen))
	} else {
		fmt.Printf("\n✅ No vote duplication detected\n")
	}
}

func parseBlockVotes(data []byte) (int, map[string]string) {
	// Simple parser to extract vote count from block
	// Block format: height(8) + prevHash(32) + timestamp(8) + voteCount(4) + votes...

	if len(data) < 52 {
		return 0, nil
	}

	// Skip height (8 bytes) + prevHash (32 bytes) + timestamp (8 bytes)
	offset := 48

	// Read vote count
	voteCount := binary.BigEndian.Uint32(data[offset : offset+4])
	offset += 4

	voters := make(map[string]string)

	// Parse each vote
	for i := uint32(0); i < voteCount && offset < len(data); i++ {
		// Read voterID length
		if offset+4 > len(data) {
			break
		}
		voterIDLen := binary.BigEndian.Uint32(data[offset : offset+4])
		offset += 4

		// Read voterID
		if offset+int(voterIDLen) > len(data) {
			break
		}
		voterID := string(data[offset : offset+int(voterIDLen)])
		offset += int(voterIDLen)

		// Read publicKey length
		if offset+4 > len(data) {
			break
		}
		pubKeyLen := binary.BigEndian.Uint32(data[offset : offset+4])
		offset += 4

		// Skip publicKey
		if offset+int(pubKeyLen) > len(data) {
			break
		}
		offset += int(pubKeyLen)

		// Read choice length
		if offset+4 > len(data) {
			break
		}
		choiceLen := binary.BigEndian.Uint32(data[offset : offset+4])
		offset += 4

		// Read choice
		if offset+int(choiceLen) > len(data) {
			break
		}
		choice := string(data[offset : offset+int(choiceLen)])
		offset += int(choiceLen)

		voters[voterID] = choice

		// Skip timestamp (8 bytes) + signature length (4 bytes) + signature
		if offset+12 > len(data) {
			break
		}
		offset += 8 // timestamp
		sigLen := binary.BigEndian.Uint32(data[offset : offset+4])
		offset += 4
		if offset+int(sigLen) > len(data) {
			break
		}
		offset += int(sigLen)
	}

	return int(voteCount), voters
}
