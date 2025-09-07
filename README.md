# Peer-Vote

Sistema de votação descentralizado baseado em blockchain com consenso Proof of Authority (PoA).

## 🎯 Visão Geral

O Peer-Vote é um sistema de votação eletrônica descentralizado que utiliza tecnologia blockchain para garantir transparência, segurança e auditabilidade em eleições. O sistema implementa consenso Proof of Authority (PoA) e comunicação peer-to-peer (P2P) para criar uma rede distribuída de nós validadores.

### Características Principais

- **🔗 Blockchain**: Armazenamento imutável de votos e eleições
- **🏛️ Consenso PoA**: Validadores autorizados com rotação Round Robin
- **🌐 Rede P2P**: Comunicação descentralizada com libp2p
- **🔐 Criptografia**: Assinaturas digitais ECDSA e hashing SHA-256
- **🗳️ Votação Anônima**: Suporte a votos anônimos com auditoria
- **📊 Auditoria Completa**: Verificação de integridade e contagem
- **🔄 Sincronização**: Sincronização automática entre nós
- **📱 APIs**: REST API e CLI para integração

## 🏗️ Arquitetura

O sistema segue os princípios da **Arquitetura Hexagonal (Clean Architecture)**, organizando o código em camadas bem definidas:

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

## 🚀 Início Rápido

### Pré-requisitos

- Go 1.21 ou superior
- Make (para comandos de build)

### Instalação

```bash
# Clonar repositório
git clone https://github.com/matscats/peer-vote.git
cd peer-vote

# Instalar dependências
go mod download

# Compilar
make build
```

### Executar Nó

```bash
# Iniciar nó validador
./build/peer-vote start --validator --verbose

# Ou usar make
make cli-start
```

### Exemplo de Votação

```bash
# Executar simulação completa
make example-complete

# Ou executar exemplo específico
go run ./examples/complete_voting_simulation.go
```

## 📖 Documentação

A documentação completa está organizada na pasta `docs/`:

### Arquitetura e Design
- **[ARCHITECTURE.md](docs/ARCHITECTURE.md)** - Visão geral da arquitetura do sistema
- **[IMPLEMENTATION_GUIDE.md](docs/IMPLEMENTATION_GUIDE.md)** - Guia de implementação detalhado

### Módulos do Sistema
- **[BLOCKCHAIN.md](docs/BLOCKCHAIN.md)** - Módulo de blockchain e gerenciamento de blocos
- **[CONSENSUS.md](docs/CONSENSUS.md)** - Sistema de consenso Proof of Authority
- **[P2P_NETWORK.md](docs/P2P_NETWORK.md)** - Rede peer-to-peer e comunicação
- **[VOTING_SYSTEM.md](docs/VOTING_SYSTEM.md)** - Sistema de votação e validação

### APIs e Integração
- **[API_REFERENCE.md](docs/API_REFERENCE.md)** - Referência completa da REST API e CLI
- **[USAGE_EXAMPLES.md](docs/USAGE_EXAMPLES.md)** - Exemplos práticos de uso

## 🛠️ Desenvolvimento

### Comandos Make

```bash
# Build
make build                 # Compilar aplicação
make build-all             # Compilar para todas as plataformas

# Testes
make test                  # Executar testes
make test-coverage         # Testes com cobertura

# Exemplos
make example-blockchain    # Exemplo de blockchain
make example-consensus     # Exemplo de consenso
make example-p2p          # Exemplo de P2P
make example-voting       # Exemplo de votação
make example-complete     # Simulação completa

# CLI
make cli-start            # Iniciar nó
make cli-status           # Status do nó
make cli-help             # Ajuda da CLI

# Desenvolvimento
make clean                # Limpar builds
make deps                 # Instalar dependências
```

### Estrutura do Projeto

```
peer-vote/
├── cmd/                  # Pontos de entrada da aplicação
├── configs/              # Arquivos de configuração
├── docs/                 # Documentação
├── examples/             # Exemplos de uso
├── peer-vote/            # Código fonte principal
│   ├── application/      # Casos de uso
│   ├── domain/          # Entidades e regras de negócio
│   └── infrastructure/   # Implementações de infraestrutura
├── build/               # Binários compilados
├── go.mod              # Dependências Go
├── Makefile            # Comandos de build
└── README.md           # Este arquivo
```

## 🔧 Configuração

### Arquivo de Configuração

Crie um arquivo `configs/config.yaml`:

```yaml
# Configuração do nó
node:
  is_validator: true
  private_key_path: "./keys/node.key"

# API REST
api:
  host: "0.0.0.0"
  port: 8080

# Rede P2P
p2p:
  listen_addresses:
    - "/ip4/0.0.0.0/tcp/9000"
  max_connections: 50
  enable_mdns: true
  enable_dht: true
  namespace: "peer-vote"

# Blockchain
blockchain:
  max_transactions_per_block: 1000
  block_interval: "10s"

# Consenso
consensus:
  algorithm: "poa"
  validator_timeout: "30s"
```

### Variáveis de Ambiente

```bash
export PEER_VOTE_CONFIG_PATH="./configs/config.yaml"
export PEER_VOTE_API_PORT="8080"
export PEER_VOTE_P2P_PORT="9000"
export PEER_VOTE_IS_VALIDATOR="true"
```

## 🔐 Segurança

### Criptografia
- **ECDSA P-256**: Assinaturas digitais
- **SHA-256**: Hashing criptográfico
- **Noise Protocol**: Comunicação P2P segura

### Consenso
- **Proof of Authority**: Validadores autorizados
- **Round Robin**: Seleção justa de validadores
- **Penalidades**: Sistema de punições para validadores maliciosos

### Votação
- **Assinaturas Digitais**: Autenticação de votos
- **Prevenção de Voto Duplo**: Verificação de histórico
- **Votos Anônimos**: Privacidade com auditabilidade
- **Auditoria Completa**: Verificação de integridade

## 📊 Monitoramento

### Métricas Disponíveis
- Status da rede P2P
- Performance do consenso
- Altura da blockchain
- Estatísticas de votação
- Conectividade de peers

### APIs de Status
```bash
# Status do nó
curl http://localhost:8080/api/node/status

# Status da blockchain
curl http://localhost:8080/api/blockchain/status

# Peers conectados
curl http://localhost:8080/api/node/peers
```

## 🤝 Contribuição

1. Fork o projeto
2. Crie uma branch para sua feature (`git checkout -b feature/AmazingFeature`)
3. Commit suas mudanças (`git commit -m 'Add some AmazingFeature'`)
4. Push para a branch (`git push origin feature/AmazingFeature`)
5. Abra um Pull Request

### Diretrizes de Código

- Siga os princípios de Clean Code
- Mantenha a Arquitetura Hexagonal
- Implemente testes para novas funcionalidades
- Documente APIs e interfaces públicas

## 📄 Licença

Este projeto está licenciado sob a Licença MIT - veja o arquivo [LICENSE](LICENSE) para detalhes.

## 🙏 Agradecimentos

- [libp2p](https://libp2p.io/) - Biblioteca de rede P2P
- [Go](https://golang.org/) - Linguagem de programação
- [Cobra](https://github.com/spf13/cobra) - Framework CLI
- [Gorilla Mux](https://github.com/gorilla/mux) - Router HTTP

## 📞 Suporte

- **Documentação**: [docs/](docs/)
- **Exemplos**: [examples/](examples/)
- **Issues**: [GitHub Issues](https://github.com/matscats/peer-vote/issues)

---

**Peer-Vote** - Sistema de Votação Descentralizado 🗳️