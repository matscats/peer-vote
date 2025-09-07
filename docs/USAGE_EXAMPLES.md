# Exemplos de Uso - Peer-Vote

Este documento demonstra como usar o sistema Peer-Vote em diferentes cenários.

## 🚀 Cenários de Uso

### 1. Configuração Inicial

#### Inicializar um Nó Validador

```bash
# 1. Gerar chaves do validador
mkdir -p keys
openssl ecparam -genkey -name prime256v1 -noout -out keys/validator.key

# 2. Configurar como validador
cp configs/config.yaml configs/validator.yaml
# Editar validator.yaml:
# consensus.validator.is_validator: true
# consensus.validator.private_key_path: "./keys/validator.key"

# 3. Iniciar nó validador
make start-validator
```

#### Inicializar um Nó Peer

```bash
# 1. Configurar como peer
cp configs/config.yaml configs/peer.yaml
# Editar peer.yaml:
# consensus.validator.is_validator: false

# 2. Iniciar nó peer
make start-peer
```

### 2. Criando uma Eleição

#### Via API REST

```bash
# Criar eleição para presidente estudantil
curl -X POST http://localhost:8080/api/v1/elections \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Eleição para Presidente Estudantil 2024",
    "description": "Eleição para escolher o presidente do grêmio estudantil",
    "candidates": [
      {
        "id": "candidate_1",
        "name": "Ana Silva",
        "description": "Proposta focada em melhorias na infraestrutura"
      },
      {
        "id": "candidate_2", 
        "name": "João Santos",
        "description": "Proposta focada em atividades culturais"
      },
      {
        "id": "candidate_3",
        "name": "Maria Oliveira", 
        "description": "Proposta focada em apoio acadêmico"
      }
    ],
    "start_time": "2024-03-01T08:00:00Z",
    "end_time": "2024-03-01T18:00:00Z",
    "allow_anonymous": false,
    "max_votes_per_voter": 1
  }'
```

#### Resposta Esperada

```json
{
  "id": "0x1a2b3c4d5e6f...",
  "title": "Eleição para Presidente Estudantil 2024",
  "status": "PENDING",
  "created_at": "2024-02-15T10:30:00Z",
  "transaction_hash": "0xabcdef123456..."
}
```

### 3. Votação

#### Submeter um Voto

```bash
# Votar na Ana Silva
curl -X POST http://localhost:8080/api/v1/votes \
  -H "Content-Type: application/json" \
  -d '{
    "election_id": "0x1a2b3c4d5e6f...",
    "candidate_id": "candidate_1",
    "voter_signature": "0x987654321..."
  }'
```

#### Verificar se o Voto foi Registrado

```bash
# Consultar transação do voto
curl http://localhost:8080/api/v1/transactions/0xvote_transaction_hash
```

### 4. Consultando Resultados

#### Resultados Parciais (Eleição Ativa)

```bash
# Consultar resultados em tempo real
curl http://localhost:8080/api/v1/elections/0x1a2b3c4d5e6f.../results
```

#### Resultados Finais (Eleição Encerrada)

```bash
# Consultar resultados finais
curl http://localhost:8080/api/v1/elections/0x1a2b3c4d5e6f.../results?final=true
```

#### Resposta dos Resultados

```json
{
  "election_id": "0x1a2b3c4d5e6f...",
  "status": "CLOSED",
  "total_votes": 150,
  "results": [
    {
      "candidate_id": "candidate_1",
      "candidate_name": "Ana Silva",
      "vote_count": 75,
      "percentage": 50.0
    },
    {
      "candidate_id": "candidate_2",
      "candidate_name": "João Santos", 
      "vote_count": 45,
      "percentage": 30.0
    },
    {
      "candidate_id": "candidate_3",
      "candidate_name": "Maria Oliveira",
      "vote_count": 30,
      "percentage": 20.0
    }
  ],
  "winner": {
    "candidate_id": "candidate_1",
    "candidate_name": "Ana Silva"
  }
}
```

### 5. Auditoria e Verificação

#### Verificar Integridade da Blockchain

```bash
# Verificar integridade de toda a cadeia
curl http://localhost:8080/api/v1/blockchain/validate
```

#### Consultar Bloco Específico

```bash
# Consultar bloco por hash
curl http://localhost:8080/api/v1/blocks/0xblock_hash

# Consultar bloco por índice
curl http://localhost:8080/api/v1/blocks/123
```

#### Verificar Prova de Inclusão (Merkle Proof)

```bash
# Verificar se um voto está incluído em um bloco
curl http://localhost:8080/api/v1/blocks/0xblock_hash/proof/0xvote_hash
```

### 6. Monitoramento da Rede

#### Status da Rede P2P

```bash
# Status geral da rede
curl http://localhost:8080/api/v1/network/status
```

#### Peers Conectados

```bash
# Lista de peers conectados
curl http://localhost:8080/api/v1/network/peers
```

#### Status do Consenso

```bash
# Status do algoritmo de consenso
curl http://localhost:8080/api/v1/consensus/status
```

### 7. Cenários Avançados

#### Eleição com Votação Anônima

```bash
curl -X POST http://localhost:8080/api/v1/elections \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Pesquisa de Satisfação Anônima",
    "description": "Avaliação anônima dos serviços",
    "candidates": [
      {"id": "muito_satisfeito", "name": "Muito Satisfeito"},
      {"id": "satisfeito", "name": "Satisfeito"},
      {"id": "neutro", "name": "Neutro"},
      {"id": "insatisfeito", "name": "Insatisfeito"}
    ],
    "start_time": "2024-03-01T00:00:00Z",
    "end_time": "2024-03-07T23:59:59Z",
    "allow_anonymous": true,
    "max_votes_per_voter": 1
  }'
```

#### Eleição com Múltiplos Votos

```bash
curl -X POST http://localhost:8080/api/v1/elections \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Escolha de Atividades Extracurriculares",
    "description": "Escolha até 3 atividades de seu interesse",
    "candidates": [
      {"id": "futebol", "name": "Futebol"},
      {"id": "basquete", "name": "Basquete"},
      {"id": "teatro", "name": "Teatro"},
      {"id": "musica", "name": "Música"},
      {"id": "debate", "name": "Debate"},
      {"id": "robotica", "name": "Robótica"}
    ],
    "start_time": "2024-03-01T00:00:00Z",
    "end_time": "2024-03-15T23:59:59Z",
    "allow_anonymous": false,
    "max_votes_per_voter": 3
  }'
```

### 8. Troubleshooting

#### Verificar Health do Nó

```bash
# Health check básico
curl http://localhost:8080/health

# Health check detalhado
curl http://localhost:8080/health?detailed=true
```

#### Logs de Debug

```bash
# Executar com logs detalhados
./build/peer-vote -log-level debug

# Ou via make
make run-dev
```

#### Sincronização Manual

```bash
# Forçar sincronização com a rede
curl -X POST http://localhost:8080/api/v1/network/sync
```

### 9. Scripts de Automação

#### Script para Teste Completo

```bash
#!/bin/bash
# test-election.sh

echo "🗳️  Testando sistema de votação completo"

# 1. Criar eleição
echo "📝 Criando eleição..."
ELECTION_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/elections \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Teste Automatizado",
    "candidates": [
      {"id": "A", "name": "Opção A"},
      {"id": "B", "name": "Opção B"}
    ],
    "start_time": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'",
    "end_time": "'$(date -u -d '+1 hour' +%Y-%m-%dT%H:%M:%SZ)'"
  }')

ELECTION_ID=$(echo $ELECTION_RESPONSE | jq -r '.id')
echo "✅ Eleição criada: $ELECTION_ID"

# 2. Submeter votos
echo "🗳️  Submetendo votos..."
for i in {1..10}; do
  CANDIDATE=$([ $((i % 2)) -eq 0 ] && echo "A" || echo "B")
  curl -s -X POST http://localhost:8080/api/v1/votes \
    -H "Content-Type: application/json" \
    -d "{\"election_id\":\"$ELECTION_ID\",\"candidate_id\":\"$CANDIDATE\"}" > /dev/null
  echo "  Voto $i submetido para candidato $CANDIDATE"
done

# 3. Verificar resultados
echo "📊 Consultando resultados..."
sleep 5  # Aguardar processamento
RESULTS=$(curl -s http://localhost:8080/api/v1/elections/$ELECTION_ID/results)
echo $RESULTS | jq '.'

echo "✅ Teste completo finalizado!"
```

### 10. Integração com Aplicações

#### Exemplo em JavaScript/Node.js

```javascript
// peer-vote-client.js
const axios = require('axios');

class PeerVoteClient {
  constructor(baseURL = 'http://localhost:8080') {
    this.api = axios.create({ baseURL });
  }

  async createElection(electionData) {
    const response = await this.api.post('/api/v1/elections', electionData);
    return response.data;
  }

  async submitVote(voteData) {
    const response = await this.api.post('/api/v1/votes', voteData);
    return response.data;
  }

  async getResults(electionId) {
    const response = await this.api.get(`/api/v1/elections/${electionId}/results`);
    return response.data;
  }

  async getNetworkStatus() {
    const response = await this.api.get('/api/v1/network/status');
    return response.data;
  }
}

// Uso
async function example() {
  const client = new PeerVoteClient();
  
  // Criar eleição
  const election = await client.createElection({
    title: "Eleição Teste",
    candidates: [
      { id: "1", name: "Candidato 1" },
      { id: "2", name: "Candidato 2" }
    ],
    start_time: new Date().toISOString(),
    end_time: new Date(Date.now() + 3600000).toISOString() // +1 hora
  });
  
  console.log('Eleição criada:', election.id);
  
  // Submeter voto
  await client.submitVote({
    election_id: election.id,
    candidate_id: "1"
  });
  
  // Consultar resultados
  const results = await client.getResults(election.id);
  console.log('Resultados:', results);
}
```

## 📚 Próximos Passos

1. **Explore a API**: Use os endpoints para entender o funcionamento
2. **Monitore a Rede**: Acompanhe o status dos peers e consenso
3. **Teste Cenários**: Experimente diferentes tipos de eleição
4. **Integre**: Desenvolva aplicações que usem o Peer-Vote
5. **Contribua**: Ajude no desenvolvimento do projeto

Para mais detalhes, consulte:
- [Guia de Implementação](IMPLEMENTATION_GUIDE.md)
- [Documentação da API](docs/api.md)
- [Arquitetura do Sistema](peer-vote/README.md)
