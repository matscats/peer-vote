# Sistema de Votação

## Visão Geral

O Sistema de Votação é o módulo central que gerencia eleições, candidatos, eleitores e votos. Implementa todas as regras de negócio para garantir eleições justas, transparentes e auditáveis.

## Componentes

### Entities (Entidades)

#### Election (Eleição)
Representa uma eleição no sistema.

**Atributos:**
- **ID**: Identificador único da eleição
- **Título**: Nome da eleição
- **Descrição**: Descrição detalhada
- **Candidatos**: Lista de candidatos
- **Período**: Data/hora de início e fim
- **Status**: Estado atual da eleição
- **Configurações**: Regras específicas

**Status Possíveis:**
- **PENDING**: Eleição criada, aguardando ativação
- **ACTIVE**: Eleição em andamento, aceitando votos
- **CLOSED**: Eleição encerrada
- **CANCELLED**: Eleição cancelada

#### Vote (Voto)
Representa um voto individual em uma eleição.

**Atributos:**
- **ID**: Identificador único do voto
- **Eleição**: ID da eleição
- **Eleitor**: ID do eleitor (se não anônimo)
- **Candidato**: ID do candidato escolhido
- **Timestamp**: Momento do voto
- **Assinatura**: Assinatura digital
- **Anônimo**: Flag de anonimato
- **Nonce**: Valor único para evitar duplicação

#### Candidate (Candidato)
Representa um candidato em uma eleição.

**Atributos:**
- **ID**: Identificador único
- **Nome**: Nome do candidato
- **Descrição**: Informações do candidato
- **Contagem**: Número de votos recebidos

### Use Cases (Casos de Uso)

#### CreateElectionUseCase
Caso de uso para criar novas eleições.

**Processo:**
1. Validar dados da eleição
2. Verificar permissões do criador
3. Criar entidade Election
4. Persistir no repositório
5. Propagar via blockchain

**Validações:**
- Título não vazio
- Pelo menos 2 candidatos
- Data de fim posterior ao início
- Criador autorizado

#### SubmitVoteUseCase
Caso de uso para submeter votos.

**Processo:**
1. Validar eleição ativa
2. Verificar elegibilidade do eleitor
3. Prevenir votação dupla
4. Criar entidade Vote
5. Assinar voto
6. Persistir e propagar

**Validações:**
- Eleição está ativa
- Eleitor não votou anteriormente
- Candidato existe na eleição
- Assinatura válida (se não anônimo)

#### AuditVotesUseCase
Caso de uso para auditoria de votos.

**Processo:**
1. Recuperar todos os votos da eleição
2. Validar integridade de cada voto
3. Verificar assinaturas
4. Contar votos por candidato
5. Gerar relatório de auditoria

**Verificações:**
- Integridade dos dados
- Validade das assinaturas
- Ausência de duplicatas
- Conformidade com regras

#### ManageElectionUseCase
Caso de uso para gerenciar eleições.

**Operações:**
- Ativar eleição
- Encerrar eleição
- Cancelar eleição
- Atualizar configurações

## Validação de Votos

### VotingValidator
Serviço de domínio que implementa todas as regras de validação.

#### Validações Implementadas

**1. Validação de Eleição**
```go
func (v *VotingValidator) ValidateElection(ctx context.Context, election *entities.Election) error
```
- Estrutura da eleição
- Candidatos válidos
- Período de votação
- Status da eleição

**2. Validação de Voto**
```go
func (v *VotingValidator) ValidateVote(ctx context.Context, vote *entities.Vote, election *entities.Election) error
```
- Estrutura do voto
- Eleição correspondente
- Timing da eleição
- Candidato válido
- Elegibilidade do eleitor
- Prevenção de voto duplo

**3. Validação de Timing**
```go
func (v *VotingValidator) ValidateElectionTiming(ctx context.Context, election *entities.Election) error
```
- Eleição iniciada
- Eleição não encerrada
- Status ACTIVE

**4. Prevenção de Voto Duplo**
```go
func (v *VotingValidator) PreventDoubleVoting(ctx context.Context, voterID valueobjects.NodeID, electionID valueobjects.Hash) error
```
- Verificar histórico de votos
- Respeitar limite de votos por eleitor
- Detectar tentativas de fraude

**5. Validação de Assinatura**
```go
func (v *VotingValidator) ValidateVoteSignature(ctx context.Context, vote *entities.Vote, publicKey *PublicKey) error
```
- Verificar assinatura digital
- Autenticar eleitor
- Garantir integridade

## Anonimato

### Votos Anônimos
O sistema suporta votos anônimos para proteger a privacidade dos eleitores.

**Características:**
- ID do eleitor omitido
- Assinatura opcional
- Nonce único para evitar duplicação
- Auditoria de integridade mantida

**Implementação:**
```go
// Voto anônimo
vote := entities.NewVote(electionID, voterID, candidateID, true)

// Serialização omite voter ID
voteData, _ := vote.ToBytes()
// {"election_id": "...", "candidate_id": "...", "timestamp": ..., "is_anonymous": true, "nonce": "..."}
```

### Auditoria de Votos Anônimos
Votos anônimos ainda podem ser auditados para integridade:

**Verificações Possíveis:**
- Estrutura do voto
- Timestamp válido
- Candidato existe
- Nonce único
- Dados não alterados

**Limitações:**
- Não é possível verificar elegibilidade
- Não é possível prevenir voto duplo completamente
- Confiança no sistema de nonce

## Persistência

### ElectionRepository
Interface para persistência de eleições.

**Métodos:**
```go
type ElectionRepository interface {
    CreateElection(ctx context.Context, election *entities.Election) error
    GetElection(ctx context.Context, id valueobjects.Hash) (*entities.Election, error)
    UpdateElection(ctx context.Context, election *entities.Election) error
    ListElections(ctx context.Context) ([]*entities.Election, error)
    IncrementCandidateVotes(ctx context.Context, electionID valueobjects.Hash, candidateID string) error
}
```

### VoteRepository
Interface para persistência de votos.

**Métodos:**
```go
type VoteRepository interface {
    CreateVote(ctx context.Context, vote *entities.Vote) error
    GetVote(ctx context.Context, id valueobjects.Hash) (*entities.Vote, error)
    ListVotesByElection(ctx context.Context, electionID valueobjects.Hash) ([]*entities.Vote, error)
    HasVoterVoted(ctx context.Context, electionID valueobjects.Hash, voterID valueobjects.NodeID) (bool, error)
    GetVoterVoteCount(ctx context.Context, electionID valueobjects.Hash, voterID valueobjects.NodeID) (int, error)
}
```

## Auditoria

### Processo de Auditoria
1. **Coleta de Dados**: Recuperar todos os votos da eleição
2. **Validação Individual**: Validar cada voto separadamente
3. **Verificação de Integridade**: Verificar assinaturas e estrutura
4. **Contagem**: Contar votos por candidato
5. **Relatório**: Gerar relatório detalhado

### Relatório de Auditoria
```go
type AuditVotesResponse struct {
    ElectionID    valueobjects.Hash
    ElectionTitle string
    AuditResults  []VoteAuditResult
    Summary       ElectionAuditSummary
    Message       string
    AuditPassed   bool
}

type ElectionAuditSummary struct {
    TotalVotes       uint64
    ValidVotes       uint64
    InvalidVotes     uint64
    AnonymousVotes   uint64
    CandidateResults map[string]uint64
}
```

### Tipos de Problemas Detectados
- Votos com estrutura inválida
- Assinaturas inválidas
- Votos duplicados
- Candidatos inexistentes
- Timestamps inválidos
- Dados corrompidos

## Segurança

### Proteções Implementadas
- **Assinaturas Digitais**: Autenticação de votos
- **Prevenção de Voto Duplo**: Verificação de histórico
- **Validação de Timing**: Respeito ao período eleitoral
- **Verificação de Candidatos**: Apenas candidatos válidos
- **Auditoria Completa**: Verificação de integridade
- **Nonce Único**: Prevenção de duplicação

### Ataques Mitigados
- **Vote Stuffing**: Prevenção de votos múltiplos
- **Ballot Tampering**: Assinaturas digitais
- **Timing Attacks**: Validação de período
- **Candidate Injection**: Validação de candidatos
- **Replay Attacks**: Nonce único e timestamps

## Configuração

### Parâmetros de Eleição
```go
type ElectionConfig struct {
    AllowAnonymous   bool          // Permitir votos anônimos
    MaxVotesPerVoter int           // Máximo de votos por eleitor
    RequireSignature bool          // Exigir assinatura
    AuditEnabled     bool          // Habilitar auditoria
    VotingPeriod     time.Duration // Período de votação
}
```

### Exemplo de Configuração
```yaml
voting:
  allow_anonymous: true
  max_votes_per_voter: 1
  require_signature: true
  audit_enabled: true
  voting_period: "24h"
  
validation:
  strict_timing: true
  signature_required: true
  prevent_double_voting: true
```

## Uso

### Exemplo: Criar Eleição
```go
candidates := []entities.Candidate{
    {ID: "1", Name: "Candidato A", Description: "Proposta A"},
    {ID: "2", Name: "Candidato B", Description: "Proposta B"},
}

request := &usecases.CreateElectionRequest{
    Title:            "Eleição Municipal 2025",
    Description:      "Eleição para prefeito",
    Candidates:       candidates,
    StartTime:        time.Now().Add(1 * time.Hour),
    EndTime:          time.Now().Add(25 * time.Hour),
    CreatedBy:        creatorID,
    AllowAnonymous:   true,
    MaxVotesPerVoter: 1,
}

response, err := createElectionUseCase.Execute(ctx, request)
```

### Exemplo: Submeter Voto
```go
request := &usecases.SubmitVoteRequest{
    ElectionID:  electionID,
    VoterID:     voterID,
    CandidateID: "1",
    IsAnonymous: false,
    PrivateKey:  voterPrivateKey,
}

response, err := submitVoteUseCase.Execute(ctx, request)
```

### Exemplo: Auditar Eleição
```go
request := &usecases.AuditVotesRequest{
    ElectionID: electionID,
}

response, err := auditVotesUseCase.AuditVotes(ctx, request)
if response.AuditPassed {
    fmt.Printf("Auditoria aprovada: %d votos válidos", response.Summary.ValidVotes)
}
```
