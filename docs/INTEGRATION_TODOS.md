# Mapeamento de TODOs e Integração Final

## 📋 **Visão Geral**

Este documento mapeia todos os TODOs pendentes no código e identifica as integrações necessárias para conectar todas as pontas do sistema Peer-Vote.

## 🔍 **TODOs Identificados no Código**

### 🌐 **REST API Handlers**

#### **`blockchain_handler.go`**
- **TODO 1**: `Hash: "calculated-hash"` (5 ocorrências)
  - **Localização**: Linhas 98, 141, 178, 204, 231
  - **Problema**: Handlers retornam hash hardcoded em vez de calcular o hash real do bloco
  - **Solução**: Integrar com `ChainManager.calculateBlockHash()`

- **TODO 2**: `IsValid: true` (1 ocorrência)
  - **Localização**: Linha 238
  - **Problema**: Validação da cadeia sempre retorna true
  - **Solução**: Integrar com `BlockchainRepository.ValidateChain()`

#### **`node_handler.go`**
- **TODO 3**: `DiscoveredPeers: 0`
  - **Localização**: Linha 69
  - **Problema**: Contagem de peers descobertos não implementada
  - **Solução**: Integrar com `DiscoveryService.GetDiscoveredPeers()`

- **TODO 4**: `MultiAddresses: status.ListenAddrs`
  - **Localização**: Linha 71
  - **Problema**: MultiAddresses usando ListenAddrs
  - **Solução**: Implementar conversão para formato MultiAddr do libp2p

#### **`vote_handler.go`**
- **TODO 5**: Conversão de PrivateKey
  - **Localização**: Linhas 63, 76, 82
  - **Problema**: API REST não consegue converter string para PrivateKey
  - **Solução**: Implementar parser de chave privada em formato PEM/hex

### 🖥️ **CLI Interface**

#### **`start.go`**
- **TODO 6**: `BlockchainRepository: nil`
  - **Localização**: Linha 98
  - **Problema**: Servidor REST não tem acesso ao repositório blockchain
  - **Solução**: Integrar `MemoryBlockchainRepository`

- **TODO 7**: `NetworkService: nil`
  - **Localização**: Linha 99
  - **Problema**: Servidor REST não tem acesso ao serviço de rede
  - **Solução**: Integrar `P2PService`

#### **`vote.go`**
- **TODO 8**: Carregamento de chave privada de string
  - **Localização**: Linha 95
  - **Problema**: CLI não consegue carregar chave privada de string
  - **Solução**: Implementar parser usando `ECDSAService.LoadKeyPair()`

## 🧩 **Componentes Já Implementados e Prontos**

### ✅ **Blockchain (Fase 2)**
- **`ChainManager`**: Gerenciamento completo da blockchain
- **`BlockBuilder`**: Construção e validação de blocos
- **`MerkleTree`**: Árvore Merkle completa com provas
- **`MemoryBlockchainRepository`**: Persistência em memória

### ✅ **Consenso (Fase 3)**
- **`PoAEngine`**: Algoritmo Proof of Authority
- **`RoundRobin`**: Seleção de validadores
- **`ValidatorManager`**: Gerenciamento de validadores
- **`PenaltySystem`**: Sistema de penalidades

### ✅ **Rede P2P (Fase 4)**
- **`P2PService`**: Serviço P2P completo
- **`LibP2PHost`**: Host libp2p configurado
- **`DiscoveryService`**: Descoberta mDNS + DHT
- **`SyncService`**: Sincronização de blockchain
- **`ProtocolManager`**: Protocolos de comunicação

### ✅ **Votação (Fase 5)**
- **`CreateElectionUseCase`**: Criação de eleições
- **`SubmitVoteUseCase`**: Submissão de votos
- **`AuditVotesUseCase`**: Auditoria de votos
- **`VotingValidator`**: Validação de votos
- **Repositórios**: Election e Vote em memória

### ✅ **Criptografia**
- **`ECDSAService`**: Serviço completo de criptografia
- **Geração de chaves**: KeyPair generation
- **Assinatura/Verificação**: Sign/Verify
- **Persistência**: LoadKeyPair/SaveKeyPair

## 🔗 **Integrações Necessárias**

### 🎯 **Prioridade Alta - Funcionalidade Crítica**

#### **1. Integrar Blockchain nos Handlers REST**
```go
// Em blockchain_handler.go
func (h *BlockchainHandler) GetBlock(w http.ResponseWriter, r *http.Request) {
    // ANTES: Hash: "calculated-hash"
    // DEPOIS: Hash: h.chainManager.CalculateBlockHash(ctx, block).String()
}
```

#### **2. Integrar P2P Service na CLI**
```go
// Em start.go
func runStartCommand(cmd *cobra.Command, args []string) {
    // Criar P2P Service
    p2pService := network.NewP2PService(config)
    
    // Integrar no REST server
    deps := &rest.Dependencies{
        NetworkService: p2pService,
        // ...
    }
}
```

#### **3. Integrar Blockchain Repository na CLI**
```go
// Em start.go
func runStartCommand(cmd *cobra.Command, args []string) {
    // Criar repositório blockchain
    blockchainRepo := persistence.NewMemoryBlockchainRepository()
    
    // Integrar no REST server
    deps := &rest.Dependencies{
        BlockchainRepository: blockchainRepo,
        // ...
    }
}
```

### 🎯 **Prioridade Média - Melhorias**

#### **4. Implementar Parser de Chave Privada**
```go
// Nova função em crypto/ecdsa.go
func (e *ECDSAService) ParsePrivateKeyFromString(keyStr string) (*PrivateKey, error) {
    // Implementar parsing de PEM ou hex
}
```

#### **5. Integrar Descoberta de Peers**
```go
// Em node_handler.go
func (h *NodeHandler) GetNodeStatus(w http.ResponseWriter, r *http.Request) {
    discoveredPeers := h.networkService.GetDiscoveredPeersCount()
    // ...
}
```

#### **6. Implementar MultiAddresses**
```go
// Em node_handler.go
func convertToMultiAddresses(listenAddrs []string) []string {
    // Converter para formato libp2p MultiAddr
}
```

### 🎯 **Prioridade Baixa - Otimizações**

#### **7. Cache de Hash de Blocos**
- Implementar cache para evitar recálculo de hashes
- Usar no blockchain_handler.go

#### **8. Validação Real de Cadeia**
- Integrar `BlockchainRepository.ValidateChain()` real
- Remover hardcoded `IsValid: true`

## 📋 **Plano de Implementação**

### **Etapa 1: Integrações Críticas (1-2 horas)**
1. ✅ Integrar `MemoryBlockchainRepository` na CLI
2. ✅ Integrar `P2PService` na CLI  
3. ✅ Conectar `ChainManager` nos handlers REST
4. ✅ Implementar cálculo real de hash nos handlers

### **Etapa 2: Parser de Chaves (30 min)**
1. ✅ Implementar `ParsePrivateKeyFromString()` 
2. ✅ Integrar no `vote_handler.go`
3. ✅ Integrar no CLI `vote.go`

### **Etapa 3: Melhorias P2P (30 min)**
1. ✅ Implementar contagem de peers descobertos
2. ✅ Converter para MultiAddresses
3. ✅ Integrar estatísticas de rede

### **Etapa 4: Validações (15 min)**
1. ✅ Integrar validação real de cadeia
2. ✅ Remover hardcoded values

## 🧪 **Testes de Integração Necessários**

### **Cenário 1: Fluxo Completo de Votação**
1. Iniciar nó com `peer-vote start`
2. Criar eleição via API REST
3. Submeter voto via CLI
4. Auditar resultados via API
5. Verificar sincronização P2P

### **Cenário 2: Rede Multi-Nó**
1. Iniciar múltiplos nós
2. Verificar descoberta automática
3. Sincronizar blockchain entre nós
4. Testar consenso distribuído

### **Cenário 3: Persistência e Recuperação**
1. Criar dados (eleições, votos, blocos)
2. Parar e reiniciar nó
3. Verificar integridade dos dados
4. Testar ressincronização

## 🎯 **Resultado Esperado**

Após implementar todas as integrações:

- ✅ **API REST funcional** com dados reais
- ✅ **CLI completa** com todas as operações
- ✅ **P2P funcionando** com descoberta e sincronização
- ✅ **Blockchain integrada** com consenso PoA
- ✅ **Sistema de votação** end-to-end funcional

## 📊 **Métricas de Sucesso**

- **0 TODOs** restantes no código
- **100% dos handlers** usando dados reais
- **Rede P2P** com descoberta automática
- **Blockchain** sincronizada entre nós
- **Votação** funcionando end-to-end
- **Testes** passando em todos os cenários

---

**Status Atual**: ✅ **CONCLUÍDO**  
**Tempo Gasto**: ⏱️ **~3 horas**  
**Complexidade**: 🟢 **Resolvida** (todas as integrações implementadas)
