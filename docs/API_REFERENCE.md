# Referência da API

## Visão Geral

O Peer-Vote oferece duas interfaces principais: uma API REST para integração web e uma CLI para operação local. Ambas fornecem acesso completo às funcionalidades do sistema.

## REST API

### Configuração
- **Porta Padrão**: 8080
- **Formato**: JSON
- **Autenticação**: Chaves ECDSA
- **CORS**: Configurável

### Endpoints

#### Eleições

##### POST /api/elections
Criar nova eleição.

**Request:**
```json
{
  "title": "Eleição Municipal 2025",
  "description": "Eleição para prefeito da cidade",
  "candidates": [
    {
      "id": "candidate_001",
      "name": "João Silva",
      "description": "Candidato do Partido A"
    },
    {
      "id": "candidate_002", 
      "name": "Maria Santos",
      "description": "Candidata do Partido B"
    }
  ],
  "start_time": "2025-01-15T08:00:00Z",
  "end_time": "2025-01-15T18:00:00Z",
  "allow_anonymous": true,
  "max_votes_per_voter": 1,
  "creator_id": "node_id_here",
  "private_key": "private_key_pem_or_hex"
}
```

**Response:**
```json
{
  "success": true,
  "election": {
    "id": "election_hash_here",
    "title": "Eleição Municipal 2025",
    "status": "PENDING",
    "created_at": "2025-01-14T10:00:00Z"
  },
  "message": "Election created successfully"
}
```

##### GET /api/elections/{id}
Obter detalhes de uma eleição.

**Response:**
```json
{
  "id": "election_hash_here",
  "title": "Eleição Municipal 2025",
  "description": "Eleição para prefeito da cidade",
  "status": "ACTIVE",
  "candidates": [
    {
      "id": "candidate_001",
      "name": "João Silva",
      "description": "Candidato do Partido A",
      "vote_count": 150
    }
  ],
  "start_time": "2025-01-15T08:00:00Z",
  "end_time": "2025-01-15T18:00:00Z",
  "total_votes": 300,
  "created_at": "2025-01-14T10:00:00Z"
}
```

##### GET /api/elections
Listar todas as eleições.

**Query Parameters:**
- `status`: Filtrar por status (PENDING, ACTIVE, CLOSED)
- `limit`: Número máximo de resultados (padrão: 50)
- `offset`: Offset para paginação (padrão: 0)

**Response:**
```json
{
  "elections": [
    {
      "id": "election_hash_1",
      "title": "Eleição Municipal 2025",
      "status": "ACTIVE",
      "start_time": "2025-01-15T08:00:00Z",
      "end_time": "2025-01-15T18:00:00Z",
      "total_votes": 300
    }
  ],
  "total": 1,
  "limit": 50,
  "offset": 0
}
```

##### PUT /api/elections/{id}/status
Alterar status da eleição.

**Request:**
```json
{
  "status": "ACTIVE",
  "updated_by": "node_id_here"
}
```

#### Votos

##### POST /api/votes
Submeter um voto.

**Request:**
```json
{
  "election_id": "election_hash_here",
  "voter_id": "voter_node_id",
  "candidate_id": "candidate_001",
  "is_anonymous": false,
  "private_key": "voter_private_key_pem_or_hex"
}
```

**Response:**
```json
{
  "success": true,
  "vote": {
    "id": "vote_hash_here",
    "election_id": "election_hash_here",
    "candidate_id": "candidate_001",
    "timestamp": "2025-01-15T10:30:00Z",
    "is_anonymous": false
  },
  "message": "Vote submitted successfully"
}
```

##### GET /api/elections/{id}/votes
Listar votos de uma eleição (apenas para auditoria).

**Response:**
```json
{
  "election_id": "election_hash_here",
  "votes": [
    {
      "id": "vote_hash_1",
      "candidate_id": "candidate_001",
      "timestamp": "2025-01-15T10:30:00Z",
      "is_anonymous": false,
      "is_valid": true
    }
  ],
  "total_votes": 300,
  "valid_votes": 298,
  "invalid_votes": 2
}
```

##### POST /api/elections/{id}/audit
Executar auditoria de uma eleição.

**Response:**
```json
{
  "election_id": "election_hash_here",
  "election_title": "Eleição Municipal 2025",
  "audit_passed": true,
  "summary": {
    "total_votes": 300,
    "valid_votes": 298,
    "invalid_votes": 2,
    "anonymous_votes": 50,
    "candidate_results": {
      "candidate_001": 150,
      "candidate_002": 148
    }
  },
  "audit_results": [
    {
      "vote_id": "vote_hash_1",
      "is_valid": true,
      "candidate_id": "candidate_001",
      "timestamp": 1642248600,
      "is_anonymous": false,
      "errors": []
    }
  ],
  "message": "Audit completed successfully"
}
```

#### Blockchain

##### GET /api/blockchain/status
Obter status da blockchain.

**Response:**
```json
{
  "height": 1250,
  "latest_block_hash": "block_hash_here",
  "latest_block_time": "2025-01-15T10:45:00Z",
  "total_transactions": 5000,
  "is_synced": true,
  "is_valid": true
}
```

##### GET /api/blockchain/blocks/{hash}
Obter detalhes de um bloco.

**Response:**
```json
{
  "hash": "block_hash_here",
  "index": 1250,
  "previous_hash": "previous_block_hash",
  "timestamp": "2025-01-15T10:45:00Z",
  "merkle_root": "merkle_root_hash",
  "validator": "validator_node_id",
  "signature": "block_signature",
  "transactions": [
    {
      "id": "tx_hash_1",
      "type": "VOTE",
      "from": "voter_node_id",
      "timestamp": "2025-01-15T10:44:30Z",
      "hash": "tx_hash_here"
    }
  ],
  "transaction_count": 25
}
```

#### Nó

##### GET /api/node/status
Obter status do nó.

**Response:**
```json
{
  "node_id": "node_id_here",
  "version": "1.0.0",
  "uptime": "2h30m15s",
  "is_validator": true,
  "is_synced": true,
  "blockchain_height": 1250,
  "peer_count": 8,
  "last_block_time": "2025-01-15T10:45:00Z"
}
```

##### GET /api/node/peers
Obter informações dos peers conectados.

**Response:**
```json
{
  "peer_count": 8,
  "connected_peers": [
    {
      "id": "peer_node_id_1",
      "addresses": ["/ip4/192.168.1.100/tcp/9000"],
      "connected_at": "2025-01-15T08:30:00Z",
      "last_seen": "2025-01-15T10:45:00Z"
    }
  ],
  "network_status": {
    "listen_addresses": ["/ip4/0.0.0.0/tcp/9000"],
    "discovered_peers": 15,
    "bandwidth_in": "1.2 MB/s",
    "bandwidth_out": "0.8 MB/s"
  }
}
```

### Códigos de Status HTTP

- **200 OK**: Requisição bem-sucedida
- **201 Created**: Recurso criado com sucesso
- **400 Bad Request**: Dados inválidos na requisição
- **401 Unauthorized**: Autenticação necessária
- **403 Forbidden**: Acesso negado
- **404 Not Found**: Recurso não encontrado
- **409 Conflict**: Conflito (ex: voto duplicado)
- **500 Internal Server Error**: Erro interno do servidor

### Tratamento de Erros

**Formato de Erro:**
```json
{
  "success": false,
  "error": {
    "code": "INVALID_ELECTION",
    "message": "Election not found or not active",
    "details": {
      "election_id": "election_hash_here",
      "current_status": "CLOSED"
    }
  }
}
```

## CLI (Command Line Interface)

### Instalação
```bash
# Compilar
make build

# Instalar globalmente
sudo cp build/peer-vote /usr/local/bin/
```

### Comandos Principais

#### peer-vote start
Iniciar nó do Peer-Vote.

```bash
peer-vote start [flags]

Flags:
  --config string     Arquivo de configuração (default "configs/config.yaml")
  --port int          Porta da API REST (default 8080)
  --p2p-port int      Porta P2P (default 9000)
  --verbose           Log detalhado
  --validator         Executar como validador
```

**Exemplo:**
```bash
peer-vote start --port 8080 --p2p-port 9000 --validator --verbose
```

#### peer-vote vote
Submeter um voto.

```bash
peer-vote vote [flags]

Flags:
  --election-id string    ID da eleição
  --candidate-id string   ID do candidato
  --private-key string    Chave privada do eleitor
  --anonymous             Voto anônimo
  --api-url string        URL da API (default "http://localhost:8080")
```

**Exemplo:**
```bash
peer-vote vote \
  --election-id "election_hash_here" \
  --candidate-id "candidate_001" \
  --private-key "path/to/private.key" \
  --api-url "http://localhost:8080"
```

#### peer-vote status
Verificar status do nó.

```bash
peer-vote status [flags]

Flags:
  --api-url string   URL da API (default "http://localhost:8080")
  --json             Saída em formato JSON
```

**Exemplo:**
```bash
peer-vote status --api-url "http://localhost:8080" --json
```

#### peer-vote sync
Sincronizar blockchain.

```bash
peer-vote sync [flags]

Flags:
  --peer string      Peer para sincronizar
  --full             Sincronização completa
  --api-url string   URL da API (default "http://localhost:8080")
```

### Configuração

#### Arquivo de Configuração (config.yaml)
```yaml
# Configuração do nó
node:
  id: "auto-generate"
  private_key_path: "./keys/node.key"
  is_validator: true

# API REST
api:
  host: "0.0.0.0"
  port: 8080
  cors_enabled: true
  cors_origins: ["*"]

# Rede P2P
p2p:
  listen_addresses:
    - "/ip4/0.0.0.0/tcp/9000"
  bootstrap_peers: []
  max_connections: 50
  enable_mdns: true
  enable_dht: true
  namespace: "peer-vote"

# Blockchain
blockchain:
  max_transactions_per_block: 1000
  block_interval: "10s"
  validation_timeout: "30s"

# Consenso
consensus:
  algorithm: "poa"
  validator_timeout: "30s"
  penalty_threshold: 3

# Logs
logging:
  level: "info"
  format: "json"
  output: "stdout"
```

### Variáveis de Ambiente

```bash
# Configuração básica
export PEER_VOTE_CONFIG_PATH="./configs/config.yaml"
export PEER_VOTE_API_PORT="8080"
export PEER_VOTE_P2P_PORT="9000"

# Chaves
export PEER_VOTE_PRIVATE_KEY_PATH="./keys/node.key"
export PEER_VOTE_IS_VALIDATOR="true"

# Rede
export PEER_VOTE_BOOTSTRAP_PEERS="peer1,peer2,peer3"
export PEER_VOTE_NAMESPACE="peer-vote"

# Logs
export PEER_VOTE_LOG_LEVEL="info"
export PEER_VOTE_LOG_FORMAT="json"
```

## Exemplos de Integração

### JavaScript/Node.js
```javascript
const axios = require('axios');

// Criar eleição
async function createElection() {
  const response = await axios.post('http://localhost:8080/api/elections', {
    title: 'Eleição Teste',
    description: 'Eleição de teste',
    candidates: [
      { id: '1', name: 'Candidato A', description: 'Proposta A' },
      { id: '2', name: 'Candidato B', description: 'Proposta B' }
    ],
    start_time: new Date(Date.now() + 3600000).toISOString(),
    end_time: new Date(Date.now() + 86400000).toISOString(),
    allow_anonymous: true,
    max_votes_per_voter: 1,
    creator_id: 'node_id',
    private_key: 'private_key_here'
  });
  
  return response.data.election;
}

// Submeter voto
async function submitVote(electionId, candidateId, privateKey) {
  const response = await axios.post('http://localhost:8080/api/votes', {
    election_id: electionId,
    voter_id: 'voter_id',
    candidate_id: candidateId,
    is_anonymous: false,
    private_key: privateKey
  });
  
  return response.data.vote;
}
```

### Python
```python
import requests
import json

class PeerVoteClient:
    def __init__(self, api_url="http://localhost:8080"):
        self.api_url = api_url
    
    def create_election(self, election_data):
        response = requests.post(
            f"{self.api_url}/api/elections",
            json=election_data
        )
        return response.json()
    
    def submit_vote(self, vote_data):
        response = requests.post(
            f"{self.api_url}/api/votes",
            json=vote_data
        )
        return response.json()
    
    def get_election_results(self, election_id):
        response = requests.get(
            f"{self.api_url}/api/elections/{election_id}"
        )
        return response.json()

# Uso
client = PeerVoteClient()
election = client.create_election({
    "title": "Eleição Python",
    "candidates": [
        {"id": "1", "name": "Candidato A", "description": "Proposta A"}
    ],
    # ... outros campos
})
```
