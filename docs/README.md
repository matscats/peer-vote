# Documentação do Peer-Vote

Bem-vindo à documentação completa do sistema Peer-Vote. Esta pasta contém toda a documentação técnica organizada por módulos e funcionalidades.

## 📚 Índice da Documentação

### 🏗️ Arquitetura e Design
- **[ARCHITECTURE.md](ARCHITECTURE.md)** - Visão geral da arquitetura do sistema
  - Princípios arquiteturais (Arquitetura Hexagonal)
  - Componentes principais
  - Fluxo de dados
  - Segurança e escalabilidade

- **[IMPLEMENTATION_GUIDE.md](IMPLEMENTATION_GUIDE.md)** - Guia detalhado de implementação
  - Estrutura do projeto
  - Padrões de código
  - Diretrizes de desenvolvimento

### 🔧 Módulos do Sistema

#### Blockchain
- **[BLOCKCHAIN.md](BLOCKCHAIN.md)** - Módulo de blockchain e gerenciamento de blocos
  - ChainManager e BlockBuilder
  - Estruturas de dados (Block, Transaction)
  - Merkle Tree e validação
  - Persistência e performance

#### Consenso
- **[CONSENSUS.md](CONSENSUS.md)** - Sistema de consenso Proof of Authority
  - PoAEngine e ValidatorManager
  - Round Robin Scheduler
  - Sistema de penalidades
  - Fluxo de consenso

#### Rede P2P
- **[P2P_NETWORK.md](P2P_NETWORK.md)** - Rede peer-to-peer e comunicação
  - P2PService e LibP2PHost
  - Descoberta de peers (mDNS, DHT)
  - Protocolos de comunicação
  - Sincronização de dados

#### Sistema de Votação
- **[VOTING_SYSTEM.md](VOTING_SYSTEM.md)** - Sistema de votação e validação
  - Entidades (Election, Vote, Candidate)
  - Casos de uso (Create, Submit, Audit)
  - Validação e segurança
  - Anonimato e auditoria

### 🔌 APIs e Integração

- **[API_REFERENCE.md](API_REFERENCE.md)** - Referência completa da REST API e CLI
  - Endpoints REST detalhados
  - Comandos CLI com exemplos
  - Códigos de status e tratamento de erros
  - Exemplos de integração (JavaScript, Python)

- **[USAGE_EXAMPLES.md](USAGE_EXAMPLES.md)** - Exemplos práticos de uso
  - Cenários de uso comum
  - Scripts de exemplo
  - Configurações típicas
  - Troubleshooting

## 🎯 Como Usar Esta Documentação

### Para Desenvolvedores
1. Comece com **[ARCHITECTURE.md](ARCHITECTURE.md)** para entender a estrutura geral
2. Leia **[IMPLEMENTATION_GUIDE.md](IMPLEMENTATION_GUIDE.md)** para diretrizes de código
3. Consulte os módulos específicos conforme necessário
4. Use **[API_REFERENCE.md](API_REFERENCE.md)** para integração

### Para Operadores
1. Consulte **[USAGE_EXAMPLES.md](USAGE_EXAMPLES.md)** para configuração
2. Use **[API_REFERENCE.md](API_REFERENCE.md)** para comandos CLI
3. Consulte módulos específicos para troubleshooting

### Para Integradores
1. Comece com **[API_REFERENCE.md](API_REFERENCE.md)**
2. Veja exemplos em **[USAGE_EXAMPLES.md](USAGE_EXAMPLES.md)**
3. Consulte **[VOTING_SYSTEM.md](VOTING_SYSTEM.md)** para regras de negócio

## 🔄 Fluxo de Leitura Recomendado

### Iniciantes
```
README.md → ARCHITECTURE.md → USAGE_EXAMPLES.md → API_REFERENCE.md
```

### Desenvolvedores
```
ARCHITECTURE.md → IMPLEMENTATION_GUIDE.md → [Módulos específicos] → API_REFERENCE.md
```

### Administradores de Sistema
```
USAGE_EXAMPLES.md → API_REFERENCE.md → P2P_NETWORK.md → CONSENSUS.md
```

## 📖 Convenções da Documentação

### Símbolos Utilizados
- 🎯 **Visão Geral**: Introdução e contexto
- 🔧 **Componentes**: Partes técnicas do sistema
- 📊 **Estruturas**: Formatos de dados e APIs
- 🔐 **Segurança**: Aspectos de segurança
- ⚡ **Performance**: Otimizações e métricas
- 💡 **Exemplos**: Código e casos de uso
- ⚠️ **Alertas**: Pontos importantes de atenção

### Formato de Código
- **Go**: Exemplos de implementação
- **JSON**: Formatos de API
- **YAML**: Arquivos de configuração
- **Bash**: Comandos de terminal

## 🔗 Links Úteis

- **[README Principal](../README.md)** - Visão geral do projeto
- **[Exemplos](../examples/)** - Código de exemplo executável
- **[Configurações](../configs/)** - Arquivos de configuração
- **[Makefile](../Makefile)** - Comandos de build e desenvolvimento

## 📝 Contribuindo com a Documentação

Para contribuir com a documentação:

1. Mantenha a estrutura modular
2. Use exemplos práticos
3. Inclua diagramas quando necessário
4. Mantenha links atualizados
5. Siga as convenções de formatação

### Template para Novos Documentos
```markdown
# Título do Módulo

## Visão Geral
Breve descrição do módulo...

## Componentes
Lista dos componentes principais...

## Uso
Exemplos práticos...

## Configuração
Parâmetros e configurações...

## Segurança
Aspectos de segurança...

## Performance
Métricas e otimizações...
```

---

**Documentação mantida pela equipe do Peer-Vote** 📚
