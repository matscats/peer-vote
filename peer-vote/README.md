# Peer-Vote - Estrutura do Projeto

## Arquitetura Clean Architecture

Este projeto segue os princípios da Clean Architecture, organizando o código em camadas bem definidas com dependências apontando sempre para o interior.

```
peer-vote/
├── domain/                 # Camada de Domínio (Núcleo do negócio)
│   ├── entities/          # Entidades de negócio
│   ├── repositories/      # Interfaces de repositórios
│   ├── services/          # Interfaces de serviços de domínio
│   └── valueobjects/      # Objetos de valor imutáveis
├── application/           # Camada de Aplicação (Casos de uso)
│   ├── usecases/         # Casos de uso da aplicação
│   └── services/         # Serviços de aplicação
├── infrastructure/       # Camada de Infraestrutura (Detalhes técnicos)
│   ├── blockchain/       # Implementação da blockchain
│   ├── consensus/        # Algoritmo de consenso PoA
│   ├── network/          # Comunicação P2P (libp2p)
│   ├── persistence/      # Persistência de dados
│   └── crypto/           # Operações criptográficas
└── interfaces/           # Camada de Interface (Adaptadores)
    ├── api/              # API REST
    └── cli/              # Interface de linha de comando
```

## Princípios Aplicados

### 1. Inversão de Dependência (DIP)
- Módulos de alto nível não dependem de módulos de baixo nível
- Ambos dependem de abstrações (interfaces)
- Infraestrutura implementa interfaces definidas no domínio

### 2. Responsabilidade Única (SRP)
- Cada módulo tem uma única razão para mudar
- Separação clara entre lógica de negócio e detalhes técnicos

### 3. Aberto/Fechado (OCP)
- Extensível para novos comportamentos
- Fechado para modificações no código existente

### 4. Substituição de Liskov (LSP)
- Implementações podem ser substituídas sem quebrar o sistema
- Contratos bem definidos via interfaces

### 5. Segregação de Interfaces (ISP)
- Interfaces específicas para cada cliente
- Evita dependências desnecessárias

## Fluxo de Dependências

```
Interfaces → Application → Domain
     ↑            ↑
Infrastructure ────┘
```

- **Domain**: Não depende de ninguém (núcleo puro)
- **Application**: Depende apenas do Domain
- **Infrastructure**: Implementa interfaces do Domain
- **Interfaces**: Orquestra Application e Infrastructure

## Estrutura Detalhada

### Domain Layer
```
domain/
├── entities/
│   ├── block.go           # Entidade Block
│   ├── transaction.go     # Entidade Transaction
│   ├── vote.go            # Entidade Vote
│   ├── voter.go           # Entidade Voter
│   └── election.go        # Entidade Election
├── repositories/
│   ├── blockchain.go      # Interface BlockchainRepository
│   ├── vote.go            # Interface VoteRepository
│   └── node.go            # Interface NodeRepository
├── services/
│   ├── consensus.go       # Interface ConsensusService
│   ├── network.go         # Interface NetworkService
│   ├── crypto.go          # Interface CryptographyService
│   └── validation.go      # Interface ValidationService
└── valueobjects/
    ├── hash.go            # Value Object Hash
    ├── signature.go       # Value Object Signature
    ├── node_id.go         # Value Object NodeID
    └── timestamp.go       # Value Object Timestamp
```

### Application Layer
```
application/
├── usecases/
│   ├── submit_vote.go     # Caso de uso: Submeter voto
│   ├── create_election.go # Caso de uso: Criar eleição
│   ├── sync_blockchain.go # Caso de uso: Sincronizar blockchain
│   └── validate_block.go  # Caso de uso: Validar bloco
└── services/
    ├── election.go        # Serviço de eleição
    ├── voting.go          # Serviço de votação
    └── synchronization.go # Serviço de sincronização
```

### Infrastructure Layer
```
infrastructure/
├── blockchain/
│   ├── merkle_tree.go     # Implementação Merkle Tree
│   ├── block_builder.go   # Construtor de blocos
│   └── chain_manager.go   # Gerenciador da cadeia
├── consensus/
│   ├── poa.go             # Proof of Authority
│   ├── round_robin.go     # Seleção Round Robin
│   └── validator.go       # Validador de consenso
├── network/
│   ├── libp2p_host.go     # Host libp2p
│   ├── protocols.go       # Protocolos de rede
│   └── discovery.go       # Descoberta de pares
├── persistence/
│   ├── file_store.go      # Armazenamento em arquivo
│   └── memory_store.go    # Armazenamento em memória
└── crypto/
    ├── ecdsa.go           # Implementação ECDSA
    └── hashing.go         # Funções de hash
```

### Interfaces Layer
```
interfaces/
├── api/
│   ├── handlers/          # Handlers HTTP
│   ├── middleware/        # Middlewares
│   └── routes.go          # Definição de rotas
└── cli/
    ├── commands/          # Comandos CLI
    └── main.go            # Entry point CLI
```

## Comunicação Entre Camadas

1. **Interface → Application**: Controllers chamam casos de uso
2. **Application → Domain**: Casos de uso usam entidades e serviços
3. **Infrastructure → Domain**: Implementa interfaces do domínio
4. **Application → Infrastructure**: Via injeção de dependência

## Benefícios da Arquitetura

- **Testabilidade**: Fácil criação de mocks e testes unitários
- **Manutenibilidade**: Mudanças isoladas em cada camada
- **Flexibilidade**: Troca de implementações sem afetar o núcleo
- **Escalabilidade**: Adição de novos recursos de forma organizada
- **Independência**: Domínio independente de frameworks e bibliotecas
