# Módulo Rede P2P

## Visão Geral

O módulo de Rede P2P implementa a comunicação descentralizada entre nós usando libp2p. Permite descoberta automática de peers, propagação de transações e blocos, e sincronização de dados.

## Componentes

### P2PService
Serviço principal que coordena toda a comunicação P2P.

**Funcionalidades:**
- Gerenciamento de conexões com peers
- Descoberta automática de nós
- Propagação de transações e blocos
- Sincronização de blockchain
- Protocolos de comunicação
- Estatísticas de rede

**Estados:**
- **Stopped**: Serviço parado
- **Starting**: Inicializando componentes
- **Running**: Operacional e conectado
- **Stopping**: Finalizando conexões

### LibP2PHost
Wrapper para o host libp2p que gerencia conexões de baixo nível.

**Funcionalidades:**
- Criação e configuração do host libp2p
- Gerenciamento de identidade do nó
- Configuração de transporte (TCP, QUIC)
- Segurança com Noise Protocol
- Multiplexing de streams

### DiscoveryService
Serviço de descoberta de peers na rede.

**Métodos de Descoberta:**
- **mDNS**: Descoberta local via multicast DNS
- **DHT**: Distributed Hash Table para descoberta global
- **Bootstrap**: Conexão com nós conhecidos
- **Peer Exchange**: Troca de informações de peers

### ProtocolManager
Gerenciador de protocolos de comunicação específicos.

**Protocolos Implementados:**
- **Transaction Gossip**: Propagação de transações
- **Block Gossip**: Propagação de blocos
- **Sync Protocol**: Sincronização de dados
- **Peer Info**: Troca de informações de peers

### SyncService
Serviço de sincronização de blockchain entre nós.

**Funcionalidades:**
- Sincronização inicial de blockchain
- Recuperação de blocos perdidos
- Validação de dados recebidos
- Resolução de conflitos
- Monitoramento de progresso

## Arquitetura de Rede

### Topologia
```
     ┌─────────┐         ┌─────────┐         ┌─────────┐
     │  Node A │◄────────┤  Node B │────────►│  Node C │
     │         │         │         │         │         │
     └────┬────┘         └────┬────┘         └────┬────┘
          │                   │                   │
          │              ┌────▼────┐              │
          └─────────────►│  Node D │◄─────────────┘
                         │         │
                         └─────────┘
```

### Camadas de Protocolo
```
┌─────────────────────────────────────────────────────┐
│                Application Layer                    │
│  (Transactions, Blocks, Sync, Peer Info)          │
└─────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────┐
│                Protocol Layer                       │
│  (Gossip, Request/Response, Stream Multiplexing)   │
└─────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────┐
│                Security Layer                       │
│  (Noise Protocol, TLS, Authentication)             │
└─────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────┐
│                Transport Layer                      │
│  (TCP, QUIC, WebSocket, WebRTC)                    │
└─────────────────────────────────────────────────────┘
```

## Descoberta de Peers

### mDNS (Local Discovery)
Descoberta automática de nós na rede local.

**Configuração:**
```go
type MDNSConfig struct {
    ServiceName string        // Nome do serviço
    Interval    time.Duration // Intervalo de anúncio
    TTL         time.Duration // Time to live
}
```

**Processo:**
1. Anunciar presença na rede local
2. Escutar anúncios de outros nós
3. Estabelecer conexões automáticas
4. Manter lista de peers locais

### DHT (Global Discovery)
Descoberta distribuída usando Kademlia DHT.

**Funcionalidades:**
- Armazenamento distribuído de informações de peers
- Roteamento eficiente de consultas
- Tolerância a falhas
- Escalabilidade global

**Bootstrap:**
```go
bootstrapPeers := []string{
    "/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
    "/ip4/104.236.179.241/tcp/4001/p2p/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM",
}
```

## Protocolos de Comunicação

### Transaction Gossip
Propagação eficiente de transações na rede.

**Algoritmo:**
1. Nó recebe nova transação
2. Valida transação localmente
3. Propaga para subset de peers
4. Peers replicam para seus vizinhos
5. Evita duplicação com cache

**Formato da Mensagem:**
```go
type TransactionGossipMessage struct {
    TransactionID string
    Transaction   []byte
    Timestamp     int64
    Signature     []byte
}
```

### Block Gossip
Propagação de blocos validados.

**Processo:**
1. Validador cria novo bloco
2. Assina bloco com chave privada
3. Propaga para todos os peers
4. Peers validam e retransmitem
5. Bloco é adicionado à blockchain

### Sync Protocol
Sincronização de blockchain entre nós.

**Tipos de Sincronização:**
- **Full Sync**: Sincronização completa da blockchain
- **Fast Sync**: Sincronização otimizada (apenas headers)
- **Light Sync**: Sincronização leve (apenas necessário)
- **Incremental**: Sincronização de blocos perdidos

## Configuração

### P2P Configuration
```go
type P2PConfig struct {
    ListenAddresses []string      // Endereços de escuta
    BootstrapPeers  []string      // Peers de bootstrap
    MaxConnections  int           // Máximo de conexões
    EnableMDNS      bool          // Habilitar mDNS
    EnableDHT       bool          // Habilitar DHT
    Namespace       string        // Namespace da rede
}
```

### Exemplo de Configuração
```yaml
p2p:
  listen_addresses:
    - "/ip4/0.0.0.0/tcp/9000"
    - "/ip4/0.0.0.0/udp/9000/quic"
  bootstrap_peers:
    - "/ip4/127.0.0.1/tcp/9001/p2p/12D3Ko..."
  max_connections: 50
  enable_mdns: true
  enable_dht: true
  namespace: "peer-vote"
```

## Segurança

### Noise Protocol
Protocolo de segurança para comunicação criptografada.

**Características:**
- Autenticação mútua
- Forward secrecy
- Resistência a ataques
- Performance otimizada

### Peer Authentication
Autenticação de peers usando chaves públicas.

**Processo:**
1. Peer apresenta identidade (Peer ID)
2. Prova posse da chave privada
3. Verificação da assinatura
4. Estabelecimento de canal seguro

### Rate Limiting
Proteção contra spam e ataques DDoS.

**Limites:**
- Mensagens por segundo por peer
- Tamanho máximo de mensagens
- Conexões simultâneas
- Bandwidth por peer

## Performance

### Otimizações
- **Connection Pooling**: Reutilização de conexões
- **Message Batching**: Agrupamento de mensagens
- **Compression**: Compressão de dados
- **Caching**: Cache de peers e rotas
- **Load Balancing**: Distribuição de carga

### Métricas
- Número de peers conectados
- Latência de rede
- Throughput de mensagens
- Taxa de descoberta de peers
- Eficiência de propagação

## Uso

### Exemplo: Inicializar P2P
```go
// Configurar P2P
config := &network.P2PConfig{
    ListenAddresses: []string{"/ip4/0.0.0.0/tcp/9000"},
    MaxConnections:  50,
    EnableMDNS:      true,
    EnableDHT:       true,
    Namespace:       "peer-vote",
}

// Criar serviço P2P
p2pService, err := network.NewP2PService(
    chainManager,
    poaEngine,
    cryptoService,
    config,
)

// Iniciar serviço
err = p2pService.Start(ctx)
```

### Exemplo: Propagar Transação
```go
// Criar transação
transaction := entities.NewTransaction(
    "VOTE",
    voterID,
    valueobjects.EmptyNodeID(),
    voteData,
)

// Propagar via P2P
err := p2pService.BroadcastTransaction(ctx, transaction)
if err != nil {
    log.Printf("Erro ao propagar transação: %v", err)
}
```

### Exemplo: Conectar a Peer
```go
// Endereço do peer
peerAddr := "/ip4/127.0.0.1/tcp/9001/p2p/12D3Ko..."

// Conectar
err := p2pService.ConnectToPeer(ctx, peerAddr)
if err != nil {
    log.Printf("Erro ao conectar: %v", err)
}
```

## Monitoramento

### Status da Rede
```go
type NetworkStatus struct {
    PeerCount      int
    ConnectedPeers []PeerInfo
    DiscoveredPeers int
    ListenAddrs    []string
    Bandwidth      BandwidthStats
    LastSync       time.Time
}
```

### Logs Importantes
- Conexões estabelecidas/perdidas
- Descoberta de novos peers
- Propagação de mensagens
- Erros de rede
- Estatísticas de performance

### Alertas
- Perda de conectividade
- Falha na descoberta de peers
- Latência alta
- Bandwidth insuficiente
- Ataques detectados
