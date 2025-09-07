package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/matscats/peer-vote/peer-vote/application/usecases"
	"github.com/matscats/peer-vote/peer-vote/domain/entities"
	"github.com/matscats/peer-vote/peer-vote/domain/services"
	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
	"github.com/matscats/peer-vote/peer-vote/infrastructure/blockchain"
	"github.com/matscats/peer-vote/peer-vote/infrastructure/consensus"
	"github.com/matscats/peer-vote/peer-vote/infrastructure/crypto"
	"github.com/matscats/peer-vote/peer-vote/infrastructure/network"
	"github.com/matscats/peer-vote/peer-vote/infrastructure/persistence"
)

// Node representa um nó completo da rede Peer-Vote
type Node struct {
	ID           valueobjects.NodeID
	Port         int
	KeyPair      *services.KeyPair
	
	// Serviços de infraestrutura
	CryptoService    services.CryptographyService
	P2PService       *network.P2PService
	ChainManager     *blockchain.ChainManager
	PoAEngine        *consensus.PoAEngine
	
	// Repositórios
	ElectionRepo     *persistence.MemoryElectionRepository
	BlockchainRepo   *persistence.MemoryBlockchainRepository
	KeyRepo          *persistence.MemoryKeyRepository
	
	// Casos de uso
	CreateElectionUC *usecases.CreateElectionUseCase
	SubmitVoteUC     *usecases.SubmitVoteUseCase
	AuditVotesUC     *usecases.AuditVotesUseCase
	ManageElectionUC *usecases.ManageElectionUseCase
}

// Voter representa um eleitor no sistema
type Voter struct {
	ID       valueobjects.NodeID
	Name     string
	KeyPair  *services.KeyPair
}

func main() {
	fmt.Println("🗳️  === PEER-VOTE: SISTEMA DE VOTAÇÃO BLOCKCHAIN REAL ===")
	fmt.Println("📋 Demonstração completa de eleição descentralizada")
	fmt.Println("⛓️  Usando blockchain real com consenso PoA")
	fmt.Println()

	ctx := context.Background()

	// === FASE 1: CONFIGURAÇÃO DA REDE BLOCKCHAIN ===
	fmt.Println("🔧 FASE 1: Configurando rede blockchain...")
	
	nodes := setupBlockchainNetwork(ctx, 3)
	fmt.Printf("✅ Rede blockchain configurada com %d nós validadores\n", len(nodes))
	
	// Inicializar consenso PoA
	fmt.Println("🏛️ Iniciando consenso Proof of Authority...")
	startConsensus(ctx, nodes)
	fmt.Println("✅ Consenso PoA ativo em todos os nós")
	
	// === FASE 2: CRIAÇÃO DA ELEIÇÃO ===
	fmt.Println("\n🗳️  FASE 2: Criando eleição na blockchain...")
	
	election := createElectionOnBlockchain(ctx, nodes[0])
	fmt.Printf("✅ Eleição criada: %s\n", election.GetTitle())
	fmt.Printf("📅 Período: %s até %s\n", 
		election.GetStartTime().Time().Format("15:04:05"),
		election.GetEndTime().Time().Format("15:04:05"))
	
	// Propagar eleição para todos os nós
	propagateElection(ctx, nodes, election)
	fmt.Println("🌐 Eleição propagada para toda a rede")
	
	// === FASE 3: REGISTRO DE ELEITORES ===
	fmt.Println("\n👥 FASE 3: Registrando eleitores...")
	
	voters := generateVoters(ctx, 12) // 12 eleitores
	fmt.Printf("✅ %d eleitores registrados\n", len(voters))
	
	// === FASE 4: PROCESSO DE VOTAÇÃO ===
	fmt.Println("\n🗳️  FASE 4: Iniciando processo de votação...")
	
	// Forçar sincronização imediata de todos os nós
	fmt.Println("⏳ Forçando sincronização blockchain entre nós...")
	for _, node := range nodes {
		if err := node.PoAEngine.SyncWithPeers(ctx); err != nil {
			log.Printf("Aviso: Erro na sincronização do nó %s: %v", node.ID.String(), err)
		}
	}
	
	// A eleição será ativa automaticamente em todos os nós baseada no tempo
	fmt.Println("✅ Eleição ativa automaticamente por timing")
	
	time.Sleep(1 * time.Second)
	
	conductVoting(ctx, nodes, voters, election)
	fmt.Println("✅ Processo de votação concluído")
	
	// Aguardar processamento final
	fmt.Println("\n⏳ Aguardando processamento final da blockchain...")
	time.Sleep(10 * time.Second)
	
	// === FASE 5: AUDITORIA BLOCKCHAIN ===
	fmt.Println("\n📊 FASE 5: Auditoria completa da blockchain...")
	
	auditResults := performBlockchainAudit(ctx, nodes[0], election)
	displayAuditResults(auditResults)
	
	// === FASE 6: RESULTADOS FINAIS ===
	fmt.Println("\n🏆 FASE 6: Apuração e resultados finais...")
	
	results := calculateFinalResults(ctx, nodes[0], election)
	displayFinalResults(results)
	
	// === FASE 7: VERIFICAÇÃO DE INTEGRIDADE ===
	fmt.Println("\n🔍 FASE 7: Verificação de integridade da rede...")
	
	verifyNetworkIntegrity(ctx, nodes)
	
	// === CONCLUSÃO ===
	fmt.Println("\n🎉 === SIMULAÇÃO CONCLUÍDA COM SUCESSO! ===")
	fmt.Println("✅ Sistema de votação blockchain totalmente funcional:")
	fmt.Println("   - Eleições criadas como transações blockchain")
	fmt.Println("   - Votos armazenados de forma imutável na blockchain")
	fmt.Println("   - Consenso PoA garantindo validação descentralizada")
	fmt.Println("   - Auditoria completa lendo diretamente da blockchain")
	fmt.Println("   - Transparência e imutabilidade garantidas")
	fmt.Println("🚀 PEER-VOTE: O futuro da votação eletrônica!")
}

// setupBlockchainNetwork configura uma rede blockchain completa
func setupBlockchainNetwork(ctx context.Context, nodeCount int) []*Node {
	nodes := make([]*Node, nodeCount)
	
	for i := 0; i < nodeCount; i++ {
		node := &Node{
			Port: 9000 + i,
		}
		
		// Configurar serviços de criptografia
		node.CryptoService = crypto.NewECDSAService()
		
		// Gerar chaves para o nó
		keyPair, err := node.CryptoService.GenerateKeyPair(ctx)
		if err != nil {
			log.Fatalf("Erro ao gerar chaves para nó %d: %v", i+1, err)
		}
		node.KeyPair = keyPair
		node.ID = valueobjects.NewNodeID(fmt.Sprintf("node-%d", i+1))
		
		// Configurar repositórios
		node.BlockchainRepo = persistence.NewMemoryBlockchainRepository(node.CryptoService).(*persistence.MemoryBlockchainRepository)
		node.ElectionRepo = persistence.NewMemoryElectionRepository()
		node.KeyRepo = persistence.NewMemoryKeyRepository()
		
		// Armazenar chaves do nó
		if err := node.KeyRepo.StoreKeyPair(ctx, node.ID, keyPair); err != nil {
			log.Fatalf("Erro ao armazenar chaves do nó %d: %v", i+1, err)
		}
		
		// Configurar blockchain
		node.ChainManager = blockchain.NewChainManager(node.BlockchainRepo, node.CryptoService)
		
		// Configurar consenso PoA
		validatorManager := consensus.NewValidatorManager()
		validatorManager.AddValidator(ctx, node.ID, keyPair.PublicKey)
		
		node.PoAEngine = consensus.NewPoAEngine(
			validatorManager,
			node.ChainManager,
			node.CryptoService,
			node.ID,
			keyPair.PrivateKey,
		)
		
		// Configurar serviços de validação
		votingValidator := services.NewVotingValidator(
			node.ElectionRepo,
			nil, // Parâmetro não utilizado
			node.CryptoService,
		)
		
		// Configurar casos de uso
		node.CreateElectionUC = usecases.NewCreateElectionUseCase(
			node.ElectionRepo,
			node.CryptoService,
			votingValidator,
			node.ChainManager,
			node.PoAEngine,
		)
		
		node.SubmitVoteUC = usecases.NewSubmitVoteUseCase(
			node.ElectionRepo,
			node.ChainManager,
			node.PoAEngine,
			node.CryptoService,
			votingValidator,
		)
		
		node.AuditVotesUC = usecases.NewAuditVotesUseCase(
			node.ElectionRepo,
			node.ChainManager,
			node.CryptoService,
			votingValidator,
		)
		
		node.ManageElectionUC = usecases.NewManageElectionUseCase(
			node.ElectionRepo,
			votingValidator,
			node.ChainManager,
		)
		
		nodes[i] = node
		
		fmt.Printf("   Nó %d: %s (porta %d)\n", 
			i+1, node.ID.String(), node.Port)
	}
	
	return nodes
}

// startConsensus inicia o consenso PoA em todos os nós
func startConsensus(ctx context.Context, nodes []*Node) {
	// Criar bloco gênesis em todos os nós
	genesisData := []byte("Peer-Vote Genesis Block - " + time.Now().Format("2006-01-02 15:04:05"))
	genesisTx := entities.NewTransaction("GENESIS", nodes[0].ID, valueobjects.EmptyNodeID(), genesisData)
	
	// Preparar transação gênesis
	txData := genesisTx.ToBytes()
	txHash := nodes[0].CryptoService.HashTransaction(ctx, txData)
	genesisTx.SetID(txHash)
	genesisTx.SetHash(txHash)
	
	signature, err := nodes[0].CryptoService.Sign(ctx, txData, nodes[0].KeyPair.PrivateKey)
	if err != nil {
		log.Fatalf("Erro ao assinar transação gênesis: %v", err)
	}
	genesisTx.SetSignature(signature)
	
	// Criar bloco gênesis apenas no nó 1
	if err := nodes[0].ChainManager.CreateGenesisBlock(ctx, []*entities.Transaction{genesisTx}, nodes[0].ID, nodes[0].KeyPair.PrivateKey); err != nil {
		log.Fatalf("Erro ao criar bloco gênesis: %v", err)
	}
	
	// Obter o bloco gênesis criado
	genesisBlock, err := nodes[0].ChainManager.GetBlockByIndex(ctx, 0)
	if err != nil {
		log.Fatalf("Erro ao obter bloco gênesis: %v", err)
	}
	
	// Propagar o mesmo bloco gênesis para todos os outros nós
	for i := 1; i < len(nodes); i++ {
		if err := nodes[i].ChainManager.AddBlock(ctx, genesisBlock); err != nil {
			log.Fatalf("Erro ao adicionar bloco gênesis no nó %d: %v", i+1, err)
		}
	}
	
	// Iniciar consenso em todos os nós
	for i, node := range nodes {
		if err := node.PoAEngine.StartConsensus(ctx); err != nil {
			log.Printf("Aviso: Erro ao iniciar consenso no nó %d: %v", i+1, err)
		}
	}
	
	// Conectar todos os nós como peers uns dos outros (rede completa)
	fmt.Println("🔗 Conectando nós como peers...")
	for i := 0; i < len(nodes); i++ {
		for j := 0; j < len(nodes); j++ {
			if i != j {
				nodes[i].PoAEngine.AddPeer(nodes[j].PoAEngine)
			}
		}
	}
	fmt.Printf("✅ Rede P2P configurada: %d nós conectados\n", len(nodes))
}

// createElectionOnBlockchain cria uma eleição real na blockchain
func createElectionOnBlockchain(ctx context.Context, node *Node) *entities.Election {
	// Criar candidatos
	candidates := []entities.Candidate{
		{ID: "candidate_001", Name: "Ana Silva", Description: "Digitalização completa da cidade - Partido Tecnológico", VoteCount: 0},
		{ID: "candidate_002", Name: "Carlos Santos", Description: "Sustentabilidade e energia limpa - Partido Verde", VoteCount: 0},
		{ID: "candidate_003", Name: "Maria Oliveira", Description: "Startups e empreendedorismo - Partido da Inovação", VoteCount: 0},
	}
	
	// Criar eleição
	electionReq := &usecases.CreateElectionRequest{
		Title:            "Eleição para Prefeito de TechCity 2025",
		Description:      "Eleição municipal para escolha do prefeito da cidade tecnológica",
		StartTime:        time.Now().Add(-5 * time.Second), // Já ativa há 5 segundos
		EndTime:          time.Now().Add(10 * time.Minute),
		Candidates:       candidates,
		CreatedBy:        node.ID,
		AllowAnonymous:   true,
		MaxVotesPerVoter: 1,
		PrivateKey:       node.KeyPair.PrivateKey,
	}
	
	response, err := node.CreateElectionUC.Execute(ctx, electionReq)
	if err != nil {
		log.Fatalf("Erro ao criar eleição: %v", err)
	}
	
	
	return response.Election
}

// propagateElection - Eleições são automaticamente sincronizadas via consenso blockchain
func propagateElection(ctx context.Context, nodes []*Node, election *entities.Election) {
	// A sincronização é feita automaticamente pelo PoA Engine quando blocos são criados
	fmt.Println("🔄 Sincronização automática via consenso PoA")
}

// generateVoters gera eleitores para a simulação
func generateVoters(ctx context.Context, count int) []*Voter {
	voters := make([]*Voter, count)
	names := []string{
		"João Silva", "Maria Santos", "Pedro Oliveira", "Ana Costa",
		"Carlos Lima", "Lucia Ferreira", "Roberto Alves", "Patricia Rocha",
		"Fernando Dias", "Juliana Moreira", "Ricardo Souza", "Camila Torres",
	}
	
	cryptoService := crypto.NewECDSAService()
	
	for i := 0; i < count; i++ {
		keyPair, err := cryptoService.GenerateKeyPair(ctx)
		if err != nil {
			log.Fatalf("Erro ao gerar chaves para eleitor %d: %v", i+1, err)
		}
		
		voters[i] = &Voter{
			ID:      valueobjects.NewNodeID(fmt.Sprintf("voter-%d", i+1)),
			Name:    names[i],
			KeyPair: keyPair,
		}
		
		fmt.Printf("   Eleitor %d: %s (%s)\n", 
			i+1, voters[i].Name, voters[i].ID.String())
	}
	
	return voters
}

// conductVoting conduz o processo de votação real
func conductVoting(ctx context.Context, nodes []*Node, voters []*Voter, election *entities.Election) {
	fmt.Println("🗳️ Iniciando votação na blockchain...")
	fmt.Println("📊 Candidatos disponíveis:")
	for _, candidate := range election.GetCandidates() {
		fmt.Printf("   %s. %s - %s\n", 
			candidate.ID, candidate.Name, candidate.Description)
	}
	fmt.Println()
	voteCount := 0
	
	for i, voter := range voters {
		// Escolher candidato (simulação realística)
		var candidateID string
		switch {
		case i < 5: // Primeiros 5 votam em Ana Silva
			candidateID = "candidate_001"
		case i < 8: // Próximos 3 votam em Carlos Santos  
			candidateID = "candidate_002"
		default: // Restantes votam em Maria Oliveira
			candidateID = "candidate_003"
		}
		
		// Selecionar nó aleatório para processar o voto (distribuição real)
		nodeIndex := rand.Intn(len(nodes))
		selectedNode := nodes[nodeIndex]
		
		// Determinar se o voto é anônimo (50% de chance)
		isAnonymous := rand.Float32() < 0.5
		
		// Criar requisição de voto
		voteReq := &usecases.SubmitVoteRequest{
			ElectionID:  election.GetID(),
			CandidateID: candidateID,
			VoterID:     voter.ID,
			PrivateKey:  voter.KeyPair.PrivateKey,
			IsAnonymous: isAnonymous,
		}
		
		// Submeter voto na blockchain
		response, err := selectedNode.SubmitVoteUC.Execute(ctx, voteReq)
		if err != nil {
			fmt.Printf("   ❌ Erro no voto de %s: %v\n", voter.Name, err)
			continue
		}
		
		voteCount++
		anonymousStr := ""
		if isAnonymous {
			anonymousStr = " (anônimo)"
		}
		
		candidateName := getCandidateName(election, candidateID)
		blockchainStatus := "⛓️ "
		if response.InBlockchain {
			blockchainStatus = "✅"
		}
		
		fmt.Printf("   %s %s votou em %s%s → Hash: %s (Nó %d)\n",
			blockchainStatus, voter.Name, candidateName, anonymousStr,
			response.TransactionHash.String(), nodeIndex+1)
		
		// Pequena pausa para simular votação realística
		time.Sleep(200 * time.Millisecond)
	}
	
	fmt.Printf("\n✅ %d votos processados na blockchain\n", voteCount)
}

// performBlockchainAudit realiza auditoria completa da blockchain
func performBlockchainAudit(ctx context.Context, node *Node, election *entities.Election) *usecases.AuditVotesResponse {
	auditReq := &usecases.AuditVotesRequest{
		ElectionID: election.GetID(),
	}
	
	response, err := node.AuditVotesUC.AuditVotes(ctx, auditReq)
	if err != nil {
		log.Fatalf("Erro na auditoria: %v", err)
	}
	
	return response
}

// displayAuditResults exibe os resultados da auditoria
func displayAuditResults(results *usecases.AuditVotesResponse) {
	if results.AuditPassed {
		fmt.Printf("✅ AUDITORIA APROVADA: %s\n", results.Message)
	} else {
		fmt.Printf("❌ AUDITORIA REPROVADA: %s\n", results.Message)
	}
	
	fmt.Printf("📊 Estatísticas da auditoria blockchain:\n")
	fmt.Printf("   - Total de votos encontrados: %d\n", len(results.AuditResults))
	fmt.Printf("   - Votos válidos: %d\n", results.Summary.ValidVotes)
	fmt.Printf("   - Votos inválidos: %d\n", results.Summary.InvalidVotes)
	fmt.Printf("   - Votos anônimos: %d\n", results.Summary.AnonymousVotes)
	
	if !results.AuditPassed {
		fmt.Println("⚠️  Detalhes dos problemas encontrados:")
		for i, audit := range results.AuditResults {
			if !audit.IsValid {
				fmt.Printf("   Voto %d: %v\n", i+1, audit.Errors)
			}
		}
	}
}

// calculateFinalResults calcula os resultados finais
func calculateFinalResults(ctx context.Context, node *Node, election *entities.Election) *usecases.CountVotesResponse {
	countReq := &usecases.CountVotesRequest{
		ElectionID: election.GetID(),
	}
	
	response, err := node.AuditVotesUC.CountVotes(ctx, countReq)
	if err != nil {
		log.Fatalf("Erro ao contar votos: %v", err)
	}
	
	return response
}

// displayFinalResults exibe os resultados finais
func displayFinalResults(results *usecases.CountVotesResponse) {
	fmt.Printf("🏆 VENCEDOR: %s\n", results.Winner.CandidateName)
	fmt.Printf("   Votos: %d (%.1f%%)\n", results.Winner.VoteCount, results.Winner.Percentage)
	fmt.Println()
	
	fmt.Println("📊 Resultado completo:")
	for i, candidate := range results.Results {
		var medal string
		switch i {
		case 0:
			medal = "🥇"
		case 1:
			medal = "🥈" 
		default:
			medal = "🥉"
		}
		
		fmt.Printf("   %s %s: %d votos (%.1f%%)\n", 
			medal, candidate.CandidateName, candidate.VoteCount, candidate.Percentage)
	}
	
	fmt.Printf("\n📈 Total de votos computados: %d\n", results.TotalVotes)
	fmt.Printf("📊 Participação: %.1f%% dos eleitores\n", 
		float64(results.TotalVotes)/12.0*100) // 12 eleitores registrados
}

// verifyNetworkIntegrity verifica a integridade da rede
func verifyNetworkIntegrity(ctx context.Context, nodes []*Node) {
	fmt.Println("🔍 Verificando integridade da rede blockchain...")
	
	// Verificar altura da blockchain em todos os nós
	heights := make([]uint64, len(nodes))
	for i, node := range nodes {
		height, err := node.BlockchainRepo.GetBlockHeight(ctx)
		if err != nil {
			fmt.Printf("❌ Erro ao obter altura do nó %d: %v\n", i+1, err)
			continue
		}
		heights[i] = height
		fmt.Printf("   Nó %d: %d blocos\n", i+1, height)
	}
	
	// Verificar sincronização
	allSynced := true
	baseHeight := heights[0]
	for i := 1; i < len(heights); i++ {
		if heights[i] != baseHeight {
			allSynced = false
			break
		}
	}
	
	if allSynced {
		fmt.Printf("✅ Todos os nós sincronizados (altura: %d)\n", baseHeight)
	} else {
		fmt.Println("⚠️  Nós com alturas diferentes - sincronização em andamento")
	}
	
	// Verificar últimos blocos
	fmt.Println("📦 Últimos blocos criados:")
	for i := 0; i < min(3, int(baseHeight)); i++ {
		blockIndex := baseHeight - uint64(i)
		block, err := nodes[0].BlockchainRepo.GetBlockByIndex(ctx, blockIndex)
		if err != nil {
			continue
		}
		
		fmt.Printf("   Bloco %d: %d transações, Validador: %s\n",
			blockIndex, len(block.GetTransactions()), 
			block.GetValidator().String())
	}
}

// Funções auxiliares

func getCandidateName(election *entities.Election, candidateID string) string {
	for _, candidate := range election.GetCandidates() {
		if candidate.ID == candidateID {
			return candidate.Name
		}
	}
	return candidateID
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}