# Peer-Vote 🗳️

Um sistema de votação descentralizado baseado em blockchain com algoritmo de consenso Proof of Authority (PoA) e comunicação P2P via libp2p.

## 🎯 Visão Geral

O **Peer-Vote** é um sistema de votação transparente e descentralizado que garante a integridade dos votos através de tecnologia blockchain. Cada nó da rede pode atuar como eleitor, e o sistema utiliza Proof of Authority com seleção Round Robin para validação de blocos.

### Características Principais

- 🔐 **Segurança**: Criptografia ECDSA e Merkle Trees
- 🌐 **Descentralização**: Rede P2P com libp2p
- ⚡ **Consenso Eficiente**: Proof of Authority com Round Robin
- 🔍 **Transparência**: Blockchain auditável publicamente
- 🏗️ **Arquitetura Limpa**: Código modular seguindo princípios SOLID
- 📊 **API REST**: Interface completa para integração

## 🏛️ Arquitetura

O projeto segue os princípios da **Clean Architecture**, organizando o código em camadas bem definidas:

```
┌─────────────────────────────────────────────────────────────┐
│                    Interfaces Layer                        │
│                   (API REST, CLI)                          │
├─────────────────────────────────────────────────────────────┤
│                   Application Layer                        │
│              (Use Cases, App Services)                     │
├─────────────────────────────────────────────────────────────┤
│                     Domain Layer                           │
│           (Entities, Value Objects, Interfaces)            │
├─────────────────────────────────────────────────────────────┤
│                 Infrastructure Layer                       │
│        (Blockchain, Consensus, Network, Persistence)       │
└─────────────────────────────────────────────────────────────┘
```

### Componentes Técnicos

- **Blockchain**: Implementação com Merkle Tree para eficiência
- **Consenso**: Proof of Authority com seleção Round Robin
- **Rede P2P**: Comunicação via libp2p
- **Criptografia**: ECDSA para assinaturas digitais
- **Persistência**: Armazenamento flexível (arquivo/memória)

## 🚀 Início Rápido

### Pré-requisitos

- Go 1.21 ou superior
- Git

### Instalação

```bash
# Clone o repositório
git clone https://github.com/matscats/peer-vote.git
cd peer-vote

# Baixe as dependências
make deps

# Configure o ambiente de desenvolvimento
make setup-dev

# Compile o projeto
make build
```

### Executando um Nó

```bash
# Executar com configuração padrão
make run

# Executar em modo desenvolvimento
make run-dev

# Executar como validador
make start-validator

# Executar como peer
make start-peer
```

## 📖 Documentação

### Estrutura do Projeto

```
peer-vote/
├── peer-vote/              # Código fonte principal
│   ├── domain/            # Camada de domínio
│   ├── application/       # Camada de aplicação
│   ├── infrastructure/    # Camada de infraestrutura
│   └── interfaces/        # Camada de interfaces
├── cmd/                   # Entry points da aplicação
├── configs/               # Arquivos de configuração
├── test/                  # Testes (unit, integration, e2e)
├── docs/                  # Documentação
└── scripts/               # Scripts utilitários
```

### Guias

- [📋 Guia de Implementação](IMPLEMENTATION_GUIDE.md) - Roadmap detalhado
- [🏗️ Arquitetura](peer-vote/README.md) - Detalhes da arquitetura
- [⚙️ Configuração](configs/config.yaml) - Opções de configuração
- [🧪 Testes](test/README.md) - Como executar testes

## 🛠️ Desenvolvimento

### Comandos Úteis

```bash
# Executar testes
make test

# Executar linter
make lint

# Formatar código
make fmt

# Executar todas as verificações
make check

# Gerar relatório de cobertura
make coverage

# Build para múltiplas plataformas
make build-all
```

### Fluxo de Desenvolvimento

1. **Fork** o repositório
2. **Crie** uma branch para sua feature (`git checkout -b feature/nova-feature`)
3. **Implemente** seguindo os princípios de Clean Code
4. **Teste** sua implementação (`make test`)
5. **Verifique** a qualidade do código (`make check`)
6. **Commit** suas mudanças (`git commit -am 'Add nova feature'`)
7. **Push** para a branch (`git push origin feature/nova-feature`)
8. **Abra** um Pull Request

## 🔧 API REST

### Endpoints Principais

```bash
# Status do nó
GET /health

# Listar eleições
GET /api/v1/elections

# Criar eleição
POST /api/v1/elections

# Submeter voto
POST /api/v1/votes

# Consultar blockchain
GET /api/v1/blocks

# Status da rede
GET /api/v1/network/status
```

### Exemplos de Uso

```bash
# Criar uma eleição
curl -X POST http://localhost:8080/api/v1/elections \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Eleição Teste",
    "description": "Uma eleição de teste",
    "candidates": [
      {"id": "1", "name": "Candidato A"},
      {"id": "2", "name": "Candidato B"}
    ],
    "start_time": "2024-01-01T00:00:00Z",
    "end_time": "2024-12-31T23:59:59Z"
  }'

# Submeter um voto
curl -X POST http://localhost:8080/api/v1/votes \
  -H "Content-Type: application/json" \
  -d '{
    "election_id": "hash_da_eleicao",
    "candidate_id": "1"
  }'
```

## 🧪 Testes

O projeto inclui uma suíte completa de testes:

```bash
# Testes unitários
make test-unit

# Testes de integração
make test-integration

# Testes end-to-end
make test-e2e

# Todos os testes
make test

# Benchmarks
make benchmark
```

## 📊 Monitoramento

### Métricas

- Endpoint de métricas: `http://localhost:9090/metrics`
- Health check: `http://localhost:8080/health`
- Status da rede: `http://localhost:8080/api/v1/network/status`

### Logs

Os logs são estruturados e configuráveis via arquivo de configuração:

```yaml
logging:
  level: "info"
  format: "json"
  output: "stdout"
```

## 🔒 Segurança

### Criptografia

- **Assinaturas**: ECDSA com curva P-256
- **Hashes**: SHA-256
- **Merkle Trees**: Para eficiência na verificação

### Validação

- Verificação de assinaturas digitais
- Validação de integridade da blockchain
- Prevenção de double-spending
- Timeout de validadores

## 🐳 Docker

```bash
# Build da imagem
make docker-build

# Executar container
make docker-run

# Ou usando docker-compose
docker-compose up -d
```

## 📈 Roadmap

- [x] **Fase 1**: Estruturas base e interfaces
- [ ] **Fase 2**: Implementação da blockchain
- [ ] **Fase 3**: Algoritmo de consenso PoA
- [ ] **Fase 4**: Rede P2P com libp2p
- [ ] **Fase 5**: Sistema de votação
- [ ] **Fase 6**: Interface e monitoramento

Veja o [Guia de Implementação](IMPLEMENTATION_GUIDE.md) para detalhes completos.

## 🤝 Contribuindo

Contribuições são bem-vindas! Por favor:

1. Leia o [Guia de Contribuição](CONTRIBUTING.md)
2. Siga os princípios de Clean Code e SOLID
3. Mantenha cobertura de testes > 80%
4. Documente suas mudanças

## 📄 Licença

Este projeto está licenciado sob a [MIT License](LICENSE).

## 👥 Autores

- **Mateus** - *Desenvolvimento inicial* - [@matscats](https://github.com/matscats)

## 🙏 Agradecimentos

- Comunidade Go pela excelente linguagem
- Projeto libp2p pela infraestrutura P2P
- Princípios de Clean Architecture por Robert C. Martin

---

**Peer-Vote** - Democratizando a votação através da tecnologia blockchain 🚀
