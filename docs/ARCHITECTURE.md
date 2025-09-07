# Arquitetura do Sistema Peer-Vote

## Visão Geral

O Peer-Vote é um sistema de votação descentralizado que utiliza blockchain e consenso Proof of Authority (PoA) para garantir transparência, segurança e integridade nas eleições.

## Princípios Arquiteturais

### Arquitetura Hexagonal (Clean Architecture)

O sistema segue os princípios da Arquitetura Hexagonal, organizando o código em camadas bem definidas:

```
┌─────────────────────────────────────────────────────────┐
│                    Infrastructure                        │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐ │
│  │     CLI     │ │  REST API   │ │      P2P Network    │ │
│  └─────────────┘ └─────────────┘ └─────────────────────┘ │
└─────────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────────┐
│                     Application                         │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐ │
│  │  Use Cases  │ │  Consensus  │ │     Blockchain      │ │
│  └─────────────┘ └─────────────┘ └─────────────────────┘ │
└─────────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────────┐
│                       Domain                            │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐ │
│  │  Entities   │ │   Services  │ │   Value Objects     │ │
│  └─────────────┘ └─────────────┘ └─────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

### Camadas

#### 1. Domain (Domínio)
- **Entities**: Eleições, Votos, Blocos, Transações
- **Value Objects**: Hash, NodeID, Timestamp, Signature
- **Services**: Interfaces para serviços de domínio
- **Repositories**: Interfaces para persistência

#### 2. Application (Aplicação)
- **Use Cases**: Casos de uso de negócio
- **Services**: Implementações de serviços de domínio

#### 3. Infrastructure (Infraestrutura)
- **Blockchain**: Gerenciamento da cadeia de blocos
- **Consensus**: Algoritmo Proof of Authority
- **Network**: Comunicação P2P com libp2p
- **Persistence**: Repositórios em memória
- **Crypto**: Criptografia ECDSA
- **CLI**: Interface de linha de comando
- **REST**: API REST

## Componentes Principais

### Blockchain
- **ChainManager**: Gerencia a cadeia de blocos
- **BlockBuilder**: Constrói e valida blocos
- **Merkle Tree**: Garante integridade das transações

### Consenso PoA
- **PoAEngine**: Motor de consenso Proof of Authority
- **ValidatorManager**: Gerencia validadores autorizados
- **RoundRobin**: Seleção de validadores em turnos

### Rede P2P
- **P2PService**: Serviço principal de rede
- **Discovery**: Descoberta de peers via mDNS/DHT
- **Protocols**: Protocolos de comunicação
- **Sync**: Sincronização de dados

### Criptografia
- **ECDSAService**: Assinaturas digitais ECDSA
- **Hash Functions**: SHA-256 para integridade
- **Key Management**: Gerenciamento de chaves

## Fluxo de Dados

### Criação de Eleição
1. Usuário cria eleição via CLI/API
2. Eleição é validada e persistida
3. Transação de eleição é criada
4. Transação é propagada via P2P
5. Nós processam e ativam eleição

### Processo de Votação
1. Eleitor submete voto
2. Voto é validado e assinado
3. Transação de voto é criada
4. Transação é adicionada ao pool PoA
5. Validador cria bloco com transações
6. Bloco é propagado e validado
7. Blockchain é atualizada

### Auditoria
1. Solicitação de auditoria
2. Recuperação de votos da blockchain
3. Validação de integridade
4. Contagem e resultados

## Segurança

### Criptografia
- **ECDSA P-256**: Assinaturas digitais
- **SHA-256**: Hashing criptográfico
- **Merkle Tree**: Integridade de transações

### Consenso
- **Proof of Authority**: Validadores autorizados
- **Round Robin**: Seleção justa de validadores
- **Penalidades**: Sistema de punições

### Rede
- **Noise Protocol**: Comunicação segura
- **Peer Authentication**: Autenticação de peers
- **Gossip Protocol**: Propagação confiável

## Escalabilidade

### Horizontal
- Adição de novos nós validadores
- Descoberta automática de peers
- Sincronização distribuída

### Performance
- Pool de transações otimizado
- Validação paralela
- Cache de dados frequentes

## Monitoramento

### Métricas
- Status da rede P2P
- Performance do consenso
- Integridade da blockchain
- Estatísticas de votação

### Logs
- Eventos de consenso
- Transações processadas
- Erros e alertas
- Auditoria de ações
