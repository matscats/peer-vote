# Módulo Consenso (Proof of Authority)

## Visão Geral

O módulo de Consenso implementa o algoritmo Proof of Authority (PoA) para validação de blocos e transações na blockchain. Diferente do Proof of Work, o PoA utiliza validadores pré-autorizados que se alternam na criação de blocos.

## Componentes

### PoAEngine
Motor principal do consenso que coordena todo o processo de validação.

**Funcionalidades:**
- Gerenciamento do pool de transações
- Criação de blocos por validadores autorizados
- Validação de blocos recebidos
- Coordenação com Round Robin scheduler
- Penalização de validadores maliciosos

**Estados:**
- **Stopped**: Motor parado
- **Running**: Processando transações e criando blocos
- **Syncing**: Sincronizando com outros nós

### ValidatorManager
Gerenciador de validadores autorizados no sistema.

**Funcionalidades:**
- Adicionar/remover validadores
- Verificar autorização de validadores
- Gerenciar status dos validadores
- Calcular estatísticas de performance
- Aplicar penalidades

**Status de Validador:**
- **Active**: Validador ativo e autorizado
- **Inactive**: Validador temporariamente inativo
- **Penalized**: Validador penalizado
- **Removed**: Validador removido do sistema

### RoundRobinScheduler
Scheduler que implementa a seleção de validadores em turnos.

**Funcionalidades:**
- Seleção justa de validadores
- Rotação automática de turnos
- Recuperação de falhas de validadores
- Sincronização de turnos entre nós
- Prevenção de monopolização

**Algoritmo:**
1. Lista validadores ativos
2. Ordena por ID para consistência
3. Seleciona próximo validador no turno
4. Rotaciona após cada bloco
5. Pula validadores inativos

### PenaltySystem
Sistema de penalidades para validadores maliciosos ou ineficientes.

**Tipos de Penalidade:**
- **Miss Block**: Perder turno de validação
- **Invalid Block**: Propor bloco inválido
- **Double Signing**: Assinar múltiplos blocos
- **Network Issues**: Problemas de conectividade

## Fluxo de Consenso

### 1. Inicialização
```
┌─────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Start     │───▶│  Load Validators │───▶│  Start Scheduler│
│  Consensus  │    │                  │    │                 │
└─────────────┘    └──────────────────┘    └─────────────────┘
```

### 2. Processamento de Transações
```
┌─────────────┐    ┌──────────────────┐    ┌─────────────────┐
│ Receive TX  │───▶│   Validate TX    │───▶│  Add to Pool    │
│             │    │                  │    │                 │
└─────────────┘    └──────────────────┘    └─────────────────┘
```

### 3. Criação de Bloco
```
┌─────────────┐    ┌──────────────────┐    ┌─────────────────┐
│ My Turn?    │───▶│  Collect TXs     │───▶│  Create Block   │
│             │    │                  │    │                 │
└─────────────┘    └──────────────────┘    └─────────────────┘
         │                                           │
         ▼                                           ▼
┌─────────────┐    ┌──────────────────┐    ┌─────────────────┐
│    Wait     │    │   Sign Block     │───▶│ Broadcast Block │
│             │    │                  │    │                 │
└─────────────┘    └──────────────────┘    └─────────────────┘
```

### 4. Validação de Bloco
```
┌─────────────┐    ┌──────────────────┐    ┌─────────────────┐
│Receive Block│───▶│  Validate Block  │───▶│  Add to Chain   │
│             │    │                  │    │                 │
└─────────────┘    └──────────────────┘    └─────────────────┘
         │                    │                       │
         │                    ▼                       ▼
         │           ┌──────────────────┐    ┌─────────────────┐
         └──────────▶│  Reject Block    │    │  Update State   │
                     │                  │    │                 │
                     └──────────────────┘    └─────────────────┘
```

## Estruturas de Dados

### Validator
```go
type Validator struct {
    NodeID       valueobjects.NodeID
    PublicKey    *services.PublicKey
    Status       ValidatorStatus
    AddedAt      valueobjects.Timestamp
    LastActiveAt valueobjects.Timestamp
    MissedRounds int
    TotalRounds  int
    PenaltyCount int
}
```

### ConsensusState
```go
type ConsensusState struct {
    CurrentRound     uint64
    CurrentValidator valueobjects.NodeID
    LastBlockTime    valueobjects.Timestamp
    PendingTxs       []*entities.Transaction
    IsRunning        bool
}
```

## Configuração

### Parâmetros do PoA
```go
type PoAConfig struct {
    BlockInterval    time.Duration // Intervalo entre blocos
    MinTxPerBlock    int          // Mínimo de transações por bloco
    MaxTxPerBlock    int          // Máximo de transações por bloco
    MaxPendingTxs    int          // Máximo de transações pendentes
    ValidatorTimeout time.Duration // Timeout para validador
    PenaltyThreshold int          // Limite para penalização
}
```

### Exemplo de Configuração
```yaml
consensus:
  algorithm: "poa"
  block_interval: "5s"
  min_transactions_per_block: 1
  max_transactions_per_block: 1000
  max_pending_transactions: 10000
  validator_timeout: "30s"
  penalty_threshold: 3
```

## Validação

### Validação de Bloco PoA
1. **Validador Autorizado**: Verificar se o criador é validador ativo
2. **Turno Correto**: Verificar se é o turno do validador
3. **Timestamp**: Validar ordem temporal dos blocos
4. **Assinatura**: Verificar assinatura do validador
5. **Transações**: Validar todas as transações do bloco
6. **Estrutura**: Verificar estrutura do bloco

### Validação de Transação
1. **Formato**: Verificar estrutura da transação
2. **Assinatura**: Validar assinatura digital
3. **Duplicação**: Verificar se não existe na blockchain
4. **Regras de Negócio**: Aplicar regras específicas do tipo
5. **Recursos**: Verificar disponibilidade de recursos

## Segurança

### Proteções Implementadas
- **Autorização**: Apenas validadores autorizados podem criar blocos
- **Rotação**: Impede monopolização por um único validador
- **Penalidades**: Pune comportamento malicioso
- **Timeout**: Recupera de validadores inativos
- **Verificação**: Valida todos os blocos recebidos

### Ataques Mitigados
- **51% Attack**: Não aplicável (validadores autorizados)
- **Double Spending**: Validação de transações duplicadas
- **Block Withholding**: Timeout e rotação de validadores
- **Selfish Mining**: Não aplicável (sem mineração)
- **Eclipse Attack**: Descoberta distribuída de peers

## Performance

### Métricas
- **Block Time**: Tempo médio entre blocos
- **Transaction Throughput**: Transações por segundo
- **Validator Performance**: Estatísticas por validador
- **Network Latency**: Tempo de propagação de blocos
- **Consensus Efficiency**: Taxa de blocos válidos

### Otimizações
- Pool de transações otimizado
- Validação paralela
- Cache de validadores ativos
- Pré-validação de transações
- Batch processing

## Uso

### Exemplo: Iniciar Consenso
```go
// Configurar PoA Engine
poaEngine := consensus.NewPoAEngine(
    validatorManager,
    chainManager,
    cryptoService,
    myNodeID,
    privateKey,
)

// Iniciar consenso
err := poaEngine.StartConsensus(ctx)
if err != nil {
    log.Fatalf("Erro ao iniciar consenso: %v", err)
}
```

### Exemplo: Adicionar Validador
```go
// Adicionar novo validador
err := validatorManager.AddValidator(ctx, nodeID, publicKey)
if err != nil {
    log.Printf("Erro ao adicionar validador: %v", err)
}
```

### Exemplo: Processar Transação
```go
// Adicionar transação ao pool
err := poaEngine.AddTransaction(ctx, transaction)
if err != nil {
    log.Printf("Erro ao adicionar transação: %v", err)
}
```

## Monitoramento

### Logs Importantes
- Início/parada do consenso
- Criação de blocos
- Validação de blocos
- Penalizações aplicadas
- Mudanças de validadores

### Alertas
- Validador perdeu turno
- Bloco inválido recebido
- Timeout de consenso
- Falha na sincronização
- Penalidade aplicada
