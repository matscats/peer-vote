# Módulo Blockchain

## Visão Geral

O módulo Blockchain é responsável por gerenciar a cadeia de blocos que armazena todas as transações de votação de forma imutável e transparente.

## Componentes

### ChainManager
Gerenciador principal da blockchain que coordena todas as operações da cadeia.

**Funcionalidades:**
- Adição de novos blocos
- Validação da cadeia
- Recuperação de blocos e transações
- Cálculo de hashes de blocos
- Verificação de integridade

**Métodos Principais:**
```go
func (cm *ChainManager) AddBlock(ctx context.Context, block *entities.Block) error
func (cm *ChainManager) GetBlock(ctx context.Context, hash valueobjects.Hash) (*entities.Block, error)
func (cm *ChainManager) ValidateChain(ctx context.Context) error
func (cm *ChainManager) GetLatestBlock(ctx context.Context) (*entities.Block, error)
```

### BlockBuilder
Construtor e validador de blocos que garante a estrutura correta dos blocos.

**Funcionalidades:**
- Construção de blocos com transações
- Cálculo de Merkle Tree
- Validação de estrutura de blocos
- Serialização/deserialização
- Verificação de assinaturas

**Processo de Construção:**
1. Coleta transações válidas
2. Calcula Merkle Root
3. Cria header do bloco
4. Assina bloco com validador
5. Valida estrutura final

### Merkle Tree
Implementação de árvore de Merkle para garantir integridade das transações.

**Características:**
- Hash SHA-256
- Estrutura binária balanceada
- Prova de integridade eficiente
- Detecção de alterações

## Estruturas de Dados

### Block (Bloco)
```go
type Block struct {
    header       *BlockHeader
    transactions []*Transaction
    merkleRoot   valueobjects.Hash
}
```

### BlockHeader (Cabeçalho do Bloco)
```go
type BlockHeader struct {
    index        uint64
    previousHash valueobjects.Hash
    timestamp    valueobjects.Timestamp
    merkleRoot   valueobjects.Hash
    nonce        uint64
    validator    valueobjects.NodeID
    signature    valueobjects.Signature
}
```

### Transaction (Transação)
```go
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
```

## Tipos de Transação

### VOTE (Voto)
Transação que representa um voto em uma eleição.
- Contém dados do voto serializado
- Assinada pelo eleitor (se não anônimo)
- Validada pelo sistema de votação

### CREATE_ELECTION (Criação de Eleição)
Transação especial para criar uma nova eleição.
- Contém dados da eleição
- Assinada pelo criador
- Propagada para todos os nós

### VALIDATOR (Validador)
Transação para gerenciar validadores do sistema.
- Adicionar/remover validadores
- Atualizar permissões
- Configurações de consenso

## Validação

### Validação de Bloco
1. **Estrutura**: Verificar campos obrigatórios
2. **Hash**: Validar hash do bloco anterior
3. **Merkle Root**: Verificar integridade das transações
4. **Timestamp**: Validar ordem temporal
5. **Assinatura**: Verificar assinatura do validador
6. **Transações**: Validar cada transação individualmente

### Validação de Transação
1. **Formato**: Verificar estrutura da transação
2. **Hash**: Validar hash da transação
3. **Assinatura**: Verificar assinatura digital
4. **Dados**: Validar conteúdo específico do tipo
5. **Duplicação**: Verificar se não existe duplicata

## Persistência

### MemoryBlockchainRepository
Implementação em memória para desenvolvimento e testes.

**Funcionalidades:**
- Armazenamento de blocos em memória
- Índices por hash e altura
- Validação de cadeia
- Recuperação eficiente

### Estrutura de Armazenamento
```go
type MemoryBlockchainRepository struct {
    blocks       map[string]*entities.Block  // Hash -> Block
    blocksByIndex map[uint64]*entities.Block // Index -> Block
    latestBlock  *entities.Block
    height       uint64
}
```

## Segurança

### Integridade
- **Merkle Tree**: Detecta alterações em transações
- **Hash Encadeado**: Liga blocos de forma imutável
- **Assinaturas**: Autentica origem dos blocos

### Imutabilidade
- Blocos não podem ser alterados após adição
- Cadeia de hashes previne modificações
- Consenso distribuído valida mudanças

## Performance

### Otimizações
- Cache de blocos recentes
- Índices por hash e altura
- Validação paralela de transações
- Compressão de dados históricos

### Métricas
- Altura da cadeia
- Número de transações
- Tempo de validação
- Taxa de blocos por segundo

## Uso

### Exemplo: Adicionar Bloco
```go
// Criar transações
transactions := []*entities.Transaction{vote1, vote2, vote3}

// Construir bloco
block := entities.NewBlock(height, previousHash, transactions, validatorID)

// Adicionar à cadeia
err := chainManager.AddBlock(ctx, block)
```

### Exemplo: Validar Cadeia
```go
// Validar integridade completa
err := chainManager.ValidateChain(ctx)
if err != nil {
    log.Printf("Blockchain inválida: %v", err)
}
```

## Configuração

### Parâmetros
- **Block Size**: Número máximo de transações por bloco
- **Block Time**: Intervalo entre blocos
- **Validation Rules**: Regras de validação customizadas
- **Storage Path**: Caminho para persistência (futuro)

### Exemplo de Configuração
```yaml
blockchain:
  max_transactions_per_block: 1000
  block_interval: "10s"
  validation_timeout: "30s"
  storage_path: "./blockchain_data"
```
