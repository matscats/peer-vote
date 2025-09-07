package blockchain

import (
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
)

// MerkleNode representa um nó na Merkle Tree
type MerkleNode struct {
	Hash  valueobjects.Hash
	Left  *MerkleNode
	Right *MerkleNode
	Data  []byte // Dados originais (apenas para folhas)
}

// MerkleTree representa uma Merkle Tree completa
type MerkleTree struct {
	Root   *MerkleNode
	Leaves []*MerkleNode
}

// MerkleProof representa uma prova de inclusão na Merkle Tree
type MerkleProof struct {
	LeafHash   valueobjects.Hash
	LeafIndex  int
	Siblings   []valueobjects.Hash
	Directions []bool // true = direita, false = esquerda
}

// NewMerkleTree cria uma nova Merkle Tree a partir de dados
func NewMerkleTree(data [][]byte) (*MerkleTree, error) {
	if len(data) == 0 {
		return nil, errors.New("cannot create merkle tree with empty data")
	}

	// Criar nós folha
	leaves := make([]*MerkleNode, len(data))
	for i, d := range data {
		hash := hashData(d)
		leaves[i] = &MerkleNode{
			Hash: hash,
			Data: d,
		}
	}

	// Construir a árvore
	root, err := buildTree(leaves)
	if err != nil {
		return nil, fmt.Errorf("failed to build merkle tree: %w", err)
	}

	return &MerkleTree{
		Root:   root,
		Leaves: leaves,
	}, nil
}

// buildTree constrói a árvore recursivamente
func buildTree(nodes []*MerkleNode) (*MerkleNode, error) {
	if len(nodes) == 0 {
		return nil, errors.New("cannot build tree with empty nodes")
	}

	if len(nodes) == 1 {
		return nodes[0], nil
	}

	var nextLevel []*MerkleNode

	// Processar pares de nós
	for i := 0; i < len(nodes); i += 2 {
		left := nodes[i]
		var right *MerkleNode

		if i+1 < len(nodes) {
			right = nodes[i+1]
		} else {
			// Se número ímpar de nós, duplicar o último
			right = nodes[i]
		}

		// Criar nó pai
		parentHash := hashPair(left.Hash, right.Hash)
		parent := &MerkleNode{
			Hash:  parentHash,
			Left:  left,
			Right: right,
		}

		nextLevel = append(nextLevel, parent)
	}

	// Recursão para o próximo nível
	return buildTree(nextLevel)
}

// GetRoot retorna o hash da raiz da árvore
func (mt *MerkleTree) GetRoot() valueobjects.Hash {
	if mt.Root == nil {
		return valueobjects.EmptyHash()
	}
	return mt.Root.Hash
}

// GenerateProof gera uma prova de inclusão para um dado específico
func (mt *MerkleTree) GenerateProof(data []byte) (*MerkleProof, error) {
	targetHash := hashData(data)
	
	// Encontrar o índice da folha
	leafIndex := -1
	for i, leaf := range mt.Leaves {
		if leaf.Hash.Equals(targetHash) {
			leafIndex = i
			break
		}
	}

	if leafIndex == -1 {
		return nil, errors.New("data not found in merkle tree")
	}

	// Gerar a prova
	proof := &MerkleProof{
		LeafHash:   targetHash,
		LeafIndex:  leafIndex,
		Siblings:   []valueobjects.Hash{},
		Directions: []bool{},
	}

	// Percorrer a árvore coletando siblings
	err := mt.collectSiblings(mt.Root, targetHash, leafIndex, len(mt.Leaves), proof)
	if err != nil {
		return nil, fmt.Errorf("failed to generate proof: %w", err)
	}

	return proof, nil
}

// collectSiblings coleta os nós irmãos necessários para a prova
func (mt *MerkleTree) collectSiblings(node *MerkleNode, targetHash valueobjects.Hash, leafIndex, totalLeaves int, proof *MerkleProof) error {
	if node == nil {
		return errors.New("node is nil")
	}

	// Se é uma folha, não há siblings
	if node.Left == nil && node.Right == nil {
		return nil
	}

	// Determinar qual lado contém o target
	leftContains := mt.containsTarget(node.Left, targetHash, leafIndex, totalLeaves/2)
	
	if leftContains {
		// Target está à esquerda, sibling é à direita
		if node.Right != nil {
			proof.Siblings = append(proof.Siblings, node.Right.Hash)
			proof.Directions = append(proof.Directions, true) // sibling à direita
		}
		return mt.collectSiblings(node.Left, targetHash, leafIndex, totalLeaves/2, proof)
	} else {
		// Target está à direita, sibling é à esquerda
		if node.Left != nil {
			proof.Siblings = append(proof.Siblings, node.Left.Hash)
			proof.Directions = append(proof.Directions, false) // sibling à esquerda
		}
		return mt.collectSiblings(node.Right, targetHash, leafIndex-totalLeaves/2, totalLeaves/2, proof)
	}
}

// containsTarget verifica se um nó contém o hash alvo
func (mt *MerkleTree) containsTarget(node *MerkleNode, targetHash valueobjects.Hash, leafIndex, rangeSize int) bool {
	if node == nil {
		return false
	}

	// Se é uma folha, verificar diretamente
	if node.Left == nil && node.Right == nil {
		return node.Hash.Equals(targetHash)
	}

	// Para nós internos, verificar se o índice está no range
	return leafIndex < rangeSize
}

// VerifyProof verifica se uma prova de inclusão é válida
func VerifyProof(proof *MerkleProof, rootHash valueobjects.Hash) bool {
	if proof == nil {
		return false
	}

	currentHash := proof.LeafHash

	// Reconstruir o caminho até a raiz
	for i, sibling := range proof.Siblings {
		if i >= len(proof.Directions) {
			return false
		}

		if proof.Directions[i] {
			// Sibling à direita
			currentHash = hashPair(currentHash, sibling)
		} else {
			// Sibling à esquerda
			currentHash = hashPair(sibling, currentHash)
		}
	}

	return currentHash.Equals(rootHash)
}

// UpdateLeaf atualiza uma folha da árvore e reconstrói
func (mt *MerkleTree) UpdateLeaf(index int, newData []byte) error {
	if index < 0 || index >= len(mt.Leaves) {
		return errors.New("leaf index out of range")
	}

	// Atualizar a folha
	newHash := hashData(newData)
	mt.Leaves[index].Hash = newHash
	mt.Leaves[index].Data = newData

	// Reconstruir a árvore
	root, err := buildTree(mt.Leaves)
	if err != nil {
		return fmt.Errorf("failed to rebuild tree: %w", err)
	}

	mt.Root = root
	return nil
}

// GetLeafCount retorna o número de folhas na árvore
func (mt *MerkleTree) GetLeafCount() int {
	return len(mt.Leaves)
}

// GetLeafData retorna os dados de uma folha específica
func (mt *MerkleTree) GetLeafData(index int) ([]byte, error) {
	if index < 0 || index >= len(mt.Leaves) {
		return nil, errors.New("leaf index out of range")
	}

	return mt.Leaves[index].Data, nil
}

// GetLeafHash retorna o hash de uma folha específica
func (mt *MerkleTree) GetLeafHash(index int) (valueobjects.Hash, error) {
	if index < 0 || index >= len(mt.Leaves) {
		return valueobjects.EmptyHash(), errors.New("leaf index out of range")
	}

	return mt.Leaves[index].Hash, nil
}

// IsValid verifica se a árvore é válida
func (mt *MerkleTree) IsValid() bool {
	if mt.Root == nil || len(mt.Leaves) == 0 {
		return false
	}

	// Verificar se todas as folhas têm dados
	for _, leaf := range mt.Leaves {
		if len(leaf.Data) == 0 {
			return false
		}
		
		// Verificar se o hash da folha está correto
		expectedHash := hashData(leaf.Data)
		if !leaf.Hash.Equals(expectedHash) {
			return false
		}
	}

	// Verificar a integridade da árvore reconstruindo
	expectedRoot, err := buildTree(mt.Leaves)
	if err != nil {
		return false
	}

	return mt.Root.Hash.Equals(expectedRoot.Hash)
}

// hashData calcula o hash SHA-256 de dados
func hashData(data []byte) valueobjects.Hash {
	hash := sha256.Sum256(data)
	return valueobjects.NewHash(hash[:])
}

// hashPair calcula o hash de dois hashes concatenados
func hashPair(left, right valueobjects.Hash) valueobjects.Hash {
	combined := append(left.Bytes(), right.Bytes()...)
	hash := sha256.Sum256(combined)
	return valueobjects.NewHash(hash[:])
}
