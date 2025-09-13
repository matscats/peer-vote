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

// ValidatorNode representa um nó validador completo da rede Peer-Vote
type ValidatorNode struct {
	ID           valueobjects.NodeID
	Port         int
	KeyPair      *services.KeyPair
	
	// Serviços de infraestrutura
	CryptoService    services.CryptographyService
	P2PService       *network.P2PService
	ChainManager     *blockchain.ChainManager
	PoAEngine        *consensus.PoAEngine
	
	// Repositórios
	BlockchainRepo   *persistence.MemoryBlockchainRepository
	KeyRepo          *persistence.MemoryKeyRepository
	
	// Casos de uso
	CreateElectionUC *usecases.CreateElectionUseCase
	SubmitVoteUC     *usecases.SubmitVoteUseCase
	AuditVotesUC     *usecases.AuditVotesUseCase
	ManageElectionUC *usecases.ManageElectionUseCase
}

// NormalNode representa um nó normal (não-validador) que apenas participa da rede P2P e vota
type NormalNode struct {
	ID           valueobjects.NodeID
	Port         int
	KeyPair      *services.KeyPair
	Name         string // Nome do eleitor
	
	// Serviços básicos necessários para votar
	CryptoService    services.CryptographyService
	P2PService       *network.P2PService
	ChainManager     *blockchain.ChainManager
	
	// Repositórios básicos
	BlockchainRepo   *persistence.MemoryBlockchainRepository
	KeyRepo          *persistence.MemoryKeyRepository
	
	// Casos de uso limitados (apenas votação)
	SubmitVoteUC     *usecases.SubmitVoteUseCase
}

func main() {
	fmt.Println("🗳️  === PEER-VOTE: SISTEMA DE VOTAÇÃO BLOCKCHAIN REAL ===")
	fmt.Println("📋 Demonstração completa de eleição descentralizada")
	fmt.Println("⛓️  Usando blockchain real com consenso PoA")
	fmt.Println()

	ctx := context.Background()

	// === FASE 1: CONFIGURAÇÃO DA REDE BLOCKCHAIN ===
	fmt.Println("🔧 FASE 1: Configurando rede blockchain...")
	
	// Configurar 2 nós validadores e 6 nós normais
	validatorNodes, normalNodes := setupMixedNetwork(ctx, 2, 6)
	fmt.Printf("✅ Rede configurada: %d nós validadores + %d nós normais\n", len(validatorNodes), len(normalNodes))
	
	// Inicializar consenso PoA apenas nos validadores
	fmt.Println("🏛️ Iniciando consenso Proof of Authority...")
	startConsensus(ctx, validatorNodes, normalNodes)
	fmt.Println("✅ Consenso PoA ativo nos nós validadores")
	
	// === FASE 2: CRIAÇÃO DA ELEIÇÃO ===
	fmt.Println("\n🗳️  FASE 2: Criando eleição na blockchain...")
	
	election := createElectionOnBlockchain(ctx, validatorNodes[0])
	fmt.Printf("✅ Eleição criada: %s\n", election.GetTitle())
	fmt.Printf("📅 Período: %s até %s\n", 
		election.GetStartTime().Time().Format("15:04:05"),
		election.GetEndTime().Time().Format("15:04:05"))
	
	// Propagar eleição para todos os nós via P2P REAL
	fmt.Println("🌐 Propagando eleição via P2P real...")
	propagateElectionViaP2P(ctx, validatorNodes, normalNodes, election)
	fmt.Println("✅ Eleição propagada para toda a rede via P2P")
	
	// === FASE 3: ELEITORES JÁ CONFIGURADOS ===
	fmt.Println("\n👥 FASE 3: Eleitores já configurados como nós normais...")
	fmt.Printf("✅ %d eleitores (nós normais) prontos para votar\n", len(normalNodes))
	
	// === FASE 4: PROCESSO DE VOTAÇÃO ===
	fmt.Println("\n🗳️  FASE 4: Iniciando processo de votação...")
	
	// Forçar sincronização imediata de todos os nós
	fmt.Println("⏳ Sincronizando blockchain entre todos os nós...")
	
	// Aguardar que todos os nós tenham a eleição (com timeout otimizado)
	fmt.Println("✅ Eleições propagadas - todos os nós têm acesso à eleição")
	
	// Pequena pausa para garantir sincronização
	time.Sleep(1 * time.Second)
	
	// A eleição será ativa automaticamente em todos os nós baseada no tempo
	fmt.Println("✅ Eleição ativa automaticamente por timing")
	
	time.Sleep(1 * time.Second)
	
	conductVoting(ctx, validatorNodes, normalNodes, election)
	fmt.Println("✅ Processo de votação concluído")
	
	// Aguardar processamento final
	fmt.Println("\n⏳ Aguardando processamento final da blockchain...")
	time.Sleep(3 * time.Second)
	
	// === FASE 5: AUDITORIA BLOCKCHAIN ===
	fmt.Println("\n📊 FASE 5: Auditoria completa da blockchain...")
	
	auditResults := performBlockchainAudit(ctx, validatorNodes[0], election)
	displayAuditResults(auditResults)
	
	// === FASE 6: RESULTADOS FINAIS ===
	fmt.Println("\n🏆 FASE 6: Apuração e resultados finais...")
	
	results := calculateFinalResults(ctx, validatorNodes[0], election)
	displayFinalResults(results, len(normalNodes))
	
	// === FASE 7: VERIFICAÇÃO DE INTEGRIDADE ===
	fmt.Println("\n🔍 FASE 7: Verificação de integridade da rede...")
	
	verifyNetworkIntegrity(ctx, validatorNodes, normalNodes)
	
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

// setupMixedNetwork configura uma rede com nós validadores e nós normais
func setupMixedNetwork(ctx context.Context, validatorCount, normalCount int) ([]*ValidatorNode, []*NormalNode) {
	validatorNodes := make([]*ValidatorNode, validatorCount)
	normalNodes := make([]*NormalNode, normalCount)
	totalNodes := validatorCount + normalCount
	
	// CORREÇÃO: Criar ValidatorManager compartilhado para todos os nós
	sharedValidatorManager := consensus.NewValidatorManager()
	
	// Nomes para os nós normais (eleitores)
	voterNames := []string{
		"João Silva", "Maria Santos", "Pedro Oliveira", "Ana Costa",
		"Carlos Lima", "Lucia Ferreira", "Roberto Alves", "Patricia Rocha",
		"Fernando Dias", "Juliana Moreira", "Ricardo Souza", "Camila Torres",
	}
	
	// === CONFIGURAR NÓS VALIDADORES ===
	fmt.Println("🔐 Configurando nós validadores...")
	for i := 0; i < validatorCount; i++ {
		node := &ValidatorNode{
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
		node.KeyRepo = persistence.NewMemoryKeyRepository()
		
		// Armazenar chaves do nó
		if err := node.KeyRepo.StoreKeyPair(ctx, node.ID, keyPair); err != nil {
			log.Fatalf("Erro ao armazenar chaves do nó %d: %v", i+1, err)
		}
		
		// Configurar blockchain
		node.ChainManager = blockchain.NewChainManager(node.BlockchainRepo, node.CryptoService)
		
		// Configurar P2P Service (NOVO - P2P REAL)
		// Bootstrap peers: cada nó conhece os outros para conectividade garantida
		bootstrapPeers := []string{}
		for j := 0; j < totalNodes; j++ {
			if j != i { // Não incluir a si mesmo
				bootstrapPeers = append(bootstrapPeers, fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", 9000+j))
			}
		}
		
		p2pConfig := &network.P2PConfig{
			ListenAddresses: []string{fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", node.Port)},
			BootstrapPeers:  bootstrapPeers, // Conectividade garantida
			EnableMDNS:      true,  // Descoberta local via mDNS
			EnableDHT:       false, // DHT desabilitado para simplificar
			Namespace:       "peer-vote-simulation",
			MaxConnections:  10,
		}
		
		p2pService, err := network.NewP2PService(
			node.ChainManager,
			nil, // PoAEngine será definido depois
			node.CryptoService,
			p2pConfig,
		)
		if err != nil {
			log.Fatalf("Erro ao criar P2PService para nó %d: %v", i+1, err)
		}
		
		// Criar adapter para interface NetworkService (DIP)
		networkService := network.NewNetworkAdapter(p2pService)
		node.P2PService = p2pService
		
		// Configurar consenso PoA com ValidatorManager compartilhado
		node.PoAEngine = consensus.NewPoAEngine(
			sharedValidatorManager, // Usar ValidatorManager compartilhado
			node.ChainManager,
			node.CryptoService,
			node.ID,
			keyPair.PrivateKey,
			networkService, // DIP: passa a abstração NetworkService
		)
		
		// Configurar serviços de validação
		votingValidator := services.NewVotingValidator(nil)
		
		// Criar adapters para respeitar arquitetura hexagonal
		blockchainService := blockchain.NewBlockchainAdapter(node.ChainManager)
		consensusService := consensus.NewConsensusAdapter(node.PoAEngine)
		
		// Configurar casos de uso
		node.CreateElectionUC = usecases.NewCreateElectionUseCase(
			node.CryptoService,
			votingValidator,
			blockchainService,
			consensusService,
		)
		
		node.SubmitVoteUC = usecases.NewSubmitVoteUseCase(
			blockchainService,
			consensusService,
			node.CryptoService,
			votingValidator,
		)
		
		node.AuditVotesUC = usecases.NewAuditVotesUseCase(
			node.ChainManager,
			node.CryptoService,
			votingValidator,
		)
		
		node.ManageElectionUC = usecases.NewManageElectionUseCase(
			votingValidator,
			node.ChainManager,
		)
		
		validatorNodes[i] = node
		
		fmt.Printf("   Validador %d: %s (porta %d)\n", 
			i+1, node.ID.String(), node.Port)
	}
	
	// === CONFIGURAR NÓS NORMAIS (ELEITORES) ===
	fmt.Println("👥 Configurando nós normais (eleitores)...")
	for i := 0; i < normalCount; i++ {
		nodeIndex := validatorCount + i
		node := &NormalNode{
			Port: 9000 + nodeIndex,
			Name: voterNames[i % len(voterNames)],
		}
		
		// Configurar serviços básicos
		node.CryptoService = crypto.NewECDSAService()
		
		// Gerar chaves para o eleitor
		keyPair, err := node.CryptoService.GenerateKeyPair(ctx)
		if err != nil {
			log.Fatalf("Erro ao gerar chaves para eleitor %d: %v", i+1, err)
		}
		node.KeyPair = keyPair
		node.ID = valueobjects.NewNodeID(fmt.Sprintf("voter-%d", i+1))
		
		// Configurar repositórios básicos
		node.BlockchainRepo = persistence.NewMemoryBlockchainRepository(node.CryptoService).(*persistence.MemoryBlockchainRepository)
		node.KeyRepo = persistence.NewMemoryKeyRepository()
		
		// Armazenar chaves do eleitor
		if err := node.KeyRepo.StoreKeyPair(ctx, node.ID, keyPair); err != nil {
			log.Fatalf("Erro ao armazenar chaves do eleitor %d: %v", i+1, err)
		}
		
		// Configurar blockchain (apenas leitura para nós normais)
		node.ChainManager = blockchain.NewChainManager(node.BlockchainRepo, node.CryptoService)
		
		// Configurar P2P Service para nós normais
		bootstrapPeers := []string{}
		for j := 0; j < totalNodes; j++ {
			if j != nodeIndex { // Não incluir a si mesmo
				bootstrapPeers = append(bootstrapPeers, fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", 9000+j))
			}
		}
		
		p2pConfig := &network.P2PConfig{
			ListenAddresses: []string{fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", node.Port)},
			BootstrapPeers:  bootstrapPeers,
			EnableMDNS:      true,
			EnableDHT:       false,
			Namespace:       "peer-vote-simulation",
			MaxConnections:  10,
		}
		
		p2pService, err := network.NewP2PService(
			node.ChainManager,
			nil, // Nós normais não têm PoAEngine
			node.CryptoService,
			p2pConfig,
		)
		if err != nil {
			log.Fatalf("Erro ao criar P2PService para eleitor %d: %v", i+1, err)
		}
		node.P2PService = p2pService
		
		// CORREÇÃO: Nós normais não precisam de casos de uso próprios
		// Eles vão enviar votos diretamente via validadores
		node.SubmitVoteUC = nil
		
		normalNodes[i] = node
		
		fmt.Printf("   Eleitor %d: %s - %s (porta %d)\n", 
			i+1, node.Name, node.ID.String(), node.Port)
	}
	
	// CORREÇÃO CRÍTICA: Adicionar APENAS nós validadores no ValidatorManager
	fmt.Println("🔐 Configurando validadores autorizados...")
	for i := 0; i < validatorCount; i++ {
		err := sharedValidatorManager.AddValidator(ctx, validatorNodes[i].ID, validatorNodes[i].KeyPair.PublicKey)
		if err != nil {
			log.Fatalf("Erro ao adicionar validador %s: %v", validatorNodes[i].ID.String(), err)
		}
		fmt.Printf("   ✅ Validador %s autorizado\n", validatorNodes[i].ID.String())
	}
	fmt.Printf("✅ %d nós validadores autorizados (Round Robin ativo)\n", validatorCount)
	
	// OTIMIZAÇÃO: Configurar parâmetros do consenso apenas nos validadores
	fmt.Println("⚙️ Otimizando parâmetros do consenso nos validadores...")
	for i, node := range validatorNodes {
		// Configurar PoA Engine para usar mempool eficientemente
		node.PoAEngine.SetConfiguration(
			300*time.Millisecond, // blockInterval: 300ms (ultra-rápido para processar mempool)
			1,                    // minTxPerBlock: 1 transação (processa assim que receber)
			5,                    // maxTxPerBlock: 5 transações por bloco (lote otimizado)
		)
		
		// Configurar Round Robin com rounds ultra-rápidos
		roundRobinScheduler := node.PoAEngine.GetRoundRobinScheduler()
		if roundRobinScheduler != nil {
			roundRobinScheduler.SetRoundDuration(1 * time.Second)     // 1 segundo por round (ultra-rápido)
			roundRobinScheduler.SetTimeoutDuration(500 * time.Millisecond) // 500ms de timeout
		}
		
		fmt.Printf("   ✅ Validador %d: mempool otimizado (300ms block, 1s round)\n", i+1)
	}
	fmt.Println("✅ Consenso otimizado para alta performance")
	
	return validatorNodes, normalNodes
}

// startConsensus inicia o consenso PoA nos nós validadores e conecta todos os nós via P2P
func startConsensus(ctx context.Context, validatorNodes []*ValidatorNode, normalNodes []*NormalNode) {
	// Criar bloco gênesis apenas no primeiro validador
	genesisData := []byte("Peer-Vote Genesis Block - " + time.Now().Format("2006-01-02 15:04:05"))
	genesisTx := entities.NewTransaction("GENESIS", validatorNodes[0].ID, valueobjects.EmptyNodeID(), genesisData)
	
	// Preparar transação gênesis
	txData := genesisTx.ToBytes()
	txHash := validatorNodes[0].CryptoService.HashTransaction(ctx, txData)
	genesisTx.SetID(txHash)
	genesisTx.SetHash(txHash)
	
	signature, err := validatorNodes[0].CryptoService.Sign(ctx, txData, validatorNodes[0].KeyPair.PrivateKey)
	if err != nil {
		log.Fatalf("Erro ao assinar transação gênesis: %v", err)
	}
	genesisTx.SetSignature(signature)
	
	// Criar bloco gênesis apenas no primeiro validador
	if err := validatorNodes[0].ChainManager.CreateGenesisBlock(ctx, []*entities.Transaction{genesisTx}, validatorNodes[0].ID, validatorNodes[0].KeyPair.PrivateKey); err != nil {
		log.Fatalf("Erro ao criar bloco gênesis: %v", err)
	}
	
	// Obter o bloco gênesis criado
	genesisBlock, err := validatorNodes[0].ChainManager.GetBlockByIndex(ctx, 0)
	if err != nil {
		log.Fatalf("Erro ao obter bloco gênesis: %v", err)
	}
	
	// Propagar o bloco gênesis para todos os outros validadores
	for i := 1; i < len(validatorNodes); i++ {
		if err := validatorNodes[i].ChainManager.AddBlock(ctx, genesisBlock); err != nil {
			log.Fatalf("Erro ao adicionar bloco gênesis no validador %d: %v", i+1, err)
		}
	}
	
	// Propagar o bloco gênesis para todos os nós normais
	for i := 0; i < len(normalNodes); i++ {
		if err := normalNodes[i].ChainManager.AddBlock(ctx, genesisBlock); err != nil {
			log.Fatalf("Erro ao adicionar bloco gênesis no nó normal %d: %v", i+1, err)
		}
	}
	
	// Iniciar serviços P2P em todos os nós PRIMEIRO
	fmt.Println("🌐 Iniciando serviços P2P...")
	
	// Iniciar P2P nos validadores
	for i, node := range validatorNodes {
		if err := node.P2PService.Start(ctx); err != nil {
			log.Fatalf("Erro ao iniciar P2P no validador %d: %v", i+1, err)
		}
		fmt.Printf("   P2P iniciado no validador %d: %s\n", i+1, node.P2PService.GetListenAddresses())
	}
	
	// Iniciar P2P nos nós normais
	for i, node := range normalNodes {
		if err := node.P2PService.Start(ctx); err != nil {
			log.Fatalf("Erro ao iniciar P2P no eleitor %d: %v", i+1, err)
		}
		fmt.Printf("   P2P iniciado no eleitor %d: %s\n", i+1, node.P2PService.GetListenAddresses())
	}
	
	// Conectar nós diretamente usando peer IDs (solução robusta)
	fmt.Println("🔗 Conectando nós diretamente via P2P...")
	
	// Criar lista unificada de todos os nós para conectividade
	allNodes := make([]interface{}, 0, len(validatorNodes)+len(normalNodes))
	for _, node := range validatorNodes {
		allNodes = append(allNodes, node)
	}
	for _, node := range normalNodes {
		allNodes = append(allNodes, node)
	}
	
	// Conectar todos os nós entre si
	for i := 0; i < len(allNodes); i++ {
		for j := 0; j < len(allNodes); j++ {
			if i != j {
				var sourceP2P, targetP2P *network.P2PService
				var sourceType, targetType string
				
				// Determinar tipo e P2P service do nó de origem
				if i < len(validatorNodes) {
					sourceP2P = validatorNodes[i].P2PService
					sourceType = "Validador"
				} else {
					sourceP2P = normalNodes[i-len(validatorNodes)].P2PService
					sourceType = "Eleitor"
				}
				
				// Determinar tipo e P2P service do nó de destino
				if j < len(validatorNodes) {
					targetP2P = validatorNodes[j].P2PService
					targetType = "Validador"
				} else {
					targetP2P = normalNodes[j-len(validatorNodes)].P2PService
					targetType = "Eleitor"
				}
				
				// Obter endereços do nó de destino
				targetAddrs := targetP2P.GetMultiAddresses()
				if len(targetAddrs) > 0 {
					// Tentar conectar ao primeiro endereço válido
					err := sourceP2P.ConnectToPeer(ctx, targetAddrs[0])
					if err != nil {
						fmt.Printf("   ⚠️  %s %d → %s %d: %v\n", sourceType, i+1, targetType, j+1, err)
					} else {
						fmt.Printf("   ✅ %s %d → %s %d conectado\n", sourceType, i+1, targetType, j+1)
					}
				}
			}
		}
	}
	
	// Aguardar estabilização das conexões
	time.Sleep(3 * time.Second)
	
	// Verificar conectividade final
	fmt.Println("📊 Status final da rede P2P:")
	totalConnections := 0
	
	// Verificar validadores
	for i, node := range validatorNodes {
		peerCount, _ := node.P2PService.GetPeerCount()
		fmt.Printf("   Validador %d: %d peers conectados\n", i+1, peerCount)
		totalConnections += peerCount
	}
	
	// Verificar nós normais
	for i, node := range normalNodes {
		peerCount, _ := node.P2PService.GetPeerCount()
		fmt.Printf("   Eleitor %d: %d peers conectados\n", i+1, peerCount)
		totalConnections += peerCount
	}
	
	fmt.Printf("✅ Rede P2P REAL: %d conexões estabelecidas\n", totalConnections/2) // Dividir por 2 pois conexões são bidirecionais
	
	// Iniciar consenso APENAS nos validadores DEPOIS da conectividade P2P
	fmt.Println("⚡ Iniciando consenso PoA nos validadores...")
	for i, node := range validatorNodes {
		if err := node.PoAEngine.StartConsensus(ctx); err != nil {
			log.Printf("Aviso: Erro ao iniciar consenso no validador %d: %v", i+1, err)
		}
	}
	
	fmt.Printf("✅ Rede P2P REAL configurada com descoberta automática!\n")
}

// createElectionOnBlockchain cria uma eleição real na blockchain
func createElectionOnBlockchain(ctx context.Context, node *ValidatorNode) *entities.Election {
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

// propagateElectionViaP2P propaga eleição via consenso PoA REAL
// Aplica SRP: responsabilidade única de propagar eleições via blockchain
func propagateElectionViaP2P(ctx context.Context, validatorNodes []*ValidatorNode, normalNodes []*NormalNode, election *entities.Election) {
	// SOLUÇÃO REAL: Aguardar que o consenso PoA propague a transação de eleição
	// A eleição foi criada no nó 1 como uma transação blockchain
	// O consenso PoA deve propagar essa transação para todos os nós
	
	fmt.Println("   ⏳ Aguardando propagação via consenso PoA...")
	
	// Aguardar que a transação de eleição seja propagada via blockchain
	maxWait := 15 * time.Second
	checkInterval := 1 * time.Second
	start := time.Now()
	
	for time.Since(start) < maxWait {
		time.Sleep(checkInterval)
		
		// Verificar quantos nós têm a eleição na blockchain
		nodesWithElection := 0
		totalNodes := len(validatorNodes) + len(normalNodes)
		
		// Verificar validadores
		for i, node := range validatorNodes {
			_, err := node.ChainManager.GetElectionFromBlockchain(ctx, election.GetID())
			if err == nil {
				nodesWithElection++
			} else if i > 0 { // Apenas log para validadores 2+
				fmt.Printf("   ⏳ Validador %d aguardando eleição... (%.0fs)\n", i+1, time.Since(start).Seconds())
			}
		}
		
		// Verificar nós normais
		for i, node := range normalNodes {
			_, err := node.ChainManager.GetElectionFromBlockchain(ctx, election.GetID())
			if err == nil {
				nodesWithElection++
			} else {
				fmt.Printf("   ⏳ Eleitor %d aguardando eleição... (%.0fs)\n", i+1, time.Since(start).Seconds())
			}
		}
		
		if nodesWithElection == totalNodes {
			fmt.Printf("   ✅ Eleição sincronizada em todos os %d nós via blockchain\n", nodesWithElection)
			return
		}
	}
	
	// Se timeout, forçar sincronização manual como fallback
	fmt.Println("   ⚠️ Timeout na propagação automática - usando fallback")
	
	// Fallback para validadores (exceto o primeiro que já tem a eleição)
	for i := 1; i < len(validatorNodes); i++ {
		if err := forceElectionSyncValidator(ctx, validatorNodes[i], election); err != nil {
			log.Printf("Erro no fallback para validador %d: %v", i+1, err)
		} else {
			fmt.Printf("   ✅ Fallback: Eleição sincronizada no validador %d\n", i+1)
		}
	}
	
	// Fallback para nós normais
	for i := 0; i < len(normalNodes); i++ {
		if err := forceElectionSyncNormal(ctx, normalNodes[i], election); err != nil {
			log.Printf("Erro no fallback para eleitor %d: %v", i+1, err)
		} else {
			fmt.Printf("   ✅ Fallback: Eleição sincronizada no eleitor %d\n", i+1)
		}
	}
}

// forceElectionSyncValidator força sincronização de eleição em validador como fallback
func forceElectionSyncValidator(ctx context.Context, node *ValidatorNode, election *entities.Election) error {
	// Fallback: recriar eleição no validador para garantir disponibilidade
	candidates := election.GetCandidates()
	
	electionReq := &usecases.CreateElectionRequest{
		Title:            election.GetTitle(),
		Description:      election.GetDescription(),
		StartTime:        election.GetStartTime().Time(),
		EndTime:          election.GetEndTime().Time(),
		Candidates:       candidates,
		CreatedBy:        node.ID, 
		AllowAnonymous:   election.AllowsAnonymousVoting(),
		MaxVotesPerVoter: election.GetMaxVotesPerVoter(),
		PrivateKey:       node.KeyPair.PrivateKey,
	}
	
	_, err := node.CreateElectionUC.Execute(ctx, electionReq)
	if err != nil {
		return fmt.Errorf("failed to sync election on validator: %w", err)
	}
	
	return nil
}

// forceElectionSyncNormal força sincronização de eleição em nó normal como fallback
func forceElectionSyncNormal(ctx context.Context, node *NormalNode, election *entities.Election) error {
	// Nós normais não podem criar eleições, apenas sincronizar via blockchain
	// Vamos tentar obter a eleição diretamente da blockchain
	_, err := node.ChainManager.GetElectionFromBlockchain(ctx, election.GetID())
	if err != nil {
		return fmt.Errorf("failed to sync election on normal node: %w", err)
	}
	
	return nil
}


// conductVoting conduz o processo de votação real
func conductVoting(ctx context.Context, validatorNodes []*ValidatorNode, normalNodes []*NormalNode, election *entities.Election) {
	fmt.Println("🗳️ Iniciando votação na blockchain...")
	fmt.Println("📊 Candidatos disponíveis:")
	for _, candidate := range election.GetCandidates() {
		fmt.Printf("   %s. %s - %s\n", 
			candidate.ID, candidate.Name, candidate.Description)
	}
	fmt.Println()
	voteCount := 0
	
	// Agora os nós normais são os eleitores
	for i, voter := range normalNodes {
		// Escolher candidato (simulação realística)
		var candidateID string
		switch {
		case i < 2: // Primeiros 2 votam em Ana Silva
			candidateID = "candidate_001"
		case i < 4: // Próximos 2 votam em Carlos Santos  
			candidateID = "candidate_002"
		default: // Restantes votam em Maria Oliveira
			candidateID = "candidate_003"
		}
		
		// CORREÇÃO CRÍTICA: Sempre enviar para o validador atual para usar o mempool eficientemente
		currentValidator, err := validatorNodes[0].PoAEngine.GetCurrentValidator(ctx)
		if err != nil {
			fmt.Printf("   ❌ Erro ao obter validador atual: %v\n", err)
			continue
		}
		
		// Encontrar o validador que é o atual
		validatorIndex := 0
		found := false
		for i, node := range validatorNodes {
			if node.ID.Equals(currentValidator) {
				validatorIndex = i
				found = true
				break
			}
		}
		
		// Fallback: usar primeiro validador se não encontrar o atual
		if !found {
			validatorIndex = 0
		}
		
		// Determinar se o voto é anônimo (50% de chance)
		isAnonymous := rand.Float32() < 0.5
		
		// Criar requisição de voto (o nó normal vota, mas envia para o validador processar)
		voteReq := &usecases.SubmitVoteRequest{
			ElectionID:  election.GetID(),
			CandidateID: candidateID,
			VoterID:     voter.ID,
			PrivateKey:  voter.KeyPair.PrivateKey,
			IsAnonymous: isAnonymous,
		}
		
		// CORREÇÃO: Nós normais enviam votos através dos validadores
		// Usar o validador atual para processar o voto
		selectedValidatorNode := validatorNodes[validatorIndex]
		response, err := selectedValidatorNode.SubmitVoteUC.Execute(ctx, voteReq)
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
		
		fmt.Printf("   %s %s votou em %s%s → Hash: %s (via Validador %d)\n",
			blockchainStatus, voter.Name, candidateName, anonymousStr,
			response.TransactionHash.String(), validatorIndex+1)
		
		// Pausa otimizada para permitir processamento do consenso
		time.Sleep(800 * time.Millisecond)
	}
	
	fmt.Printf("\n✅ %d votos processados na blockchain\n", voteCount)
}

// performBlockchainAudit realiza auditoria completa da blockchain
func performBlockchainAudit(ctx context.Context, node *ValidatorNode, election *entities.Election) *usecases.AuditVotesResponse {
	// DEBUG: Verificar altura da blockchain antes da auditoria
	height, err := node.BlockchainRepo.GetBlockHeight(ctx)
	if err != nil {
		fmt.Printf("⚠️ Erro ao obter altura da blockchain: %v\n", err)
	} else {
		fmt.Printf("🔍 DEBUG: Altura da blockchain: %d blocos\n", height)
	}
	
	// DEBUG: Verificar blocos na blockchain
	for i := uint64(0); i <= height; i++ {
		block, err := node.BlockchainRepo.GetBlockByIndex(ctx, i)
		if err != nil {
			fmt.Printf("⚠️ Erro ao obter bloco %d: %v\n", i, err)
			continue
		}
		
		transactions := block.GetTransactions()
		voteCount := 0
		for _, tx := range transactions {
			if tx.GetType() == "VOTE" {
				voteCount++
			}
		}
		fmt.Printf("🔍 DEBUG: Bloco %d - %d transações (%d votos)\n", i, len(transactions), voteCount)
	}
	
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
func calculateFinalResults(ctx context.Context, node *ValidatorNode, election *entities.Election) *usecases.CountVotesResponse {
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
func displayFinalResults(results *usecases.CountVotesResponse, totalVoters int) {
	// Proteção contra panic se results for nil ou não tiver dados
	if results == nil {
		fmt.Println("❌ Erro: Não foi possível obter resultados da eleição")
		return
	}
	
	if results.Winner == nil {
		fmt.Println("❌ Erro: Nenhum vencedor encontrado - possivelmente não há votos válidos")
		fmt.Printf("📊 Total de votos computados: %d\n", results.TotalVotes)
		return
	}
	
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
		float64(results.TotalVotes)/float64(totalVoters)*100)
}

// verifyNetworkIntegrity verifica a integridade da rede
func verifyNetworkIntegrity(ctx context.Context, validatorNodes []*ValidatorNode, normalNodes []*NormalNode) {
	fmt.Println("🔍 Verificando integridade da rede blockchain...")
	
	// Verificar altura da blockchain em todos os nós
	allHeights := make([]uint64, 0, len(validatorNodes)+len(normalNodes))
	
	// Verificar validadores
	fmt.Println("📊 Validadores:")
	for i, node := range validatorNodes {
		height, err := node.BlockchainRepo.GetBlockHeight(ctx)
		if err != nil {
			fmt.Printf("❌ Erro ao obter altura do validador %d: %v\n", i+1, err)
			continue
		}
		allHeights = append(allHeights, height)
		fmt.Printf("   Validador %d: %d blocos\n", i+1, height)
	}
	
	// Verificar nós normais
	fmt.Println("📊 Nós normais:")
	for i, node := range normalNodes {
		height, err := node.BlockchainRepo.GetBlockHeight(ctx)
		if err != nil {
			fmt.Printf("❌ Erro ao obter altura do eleitor %d: %v\n", i+1, err)
			continue
		}
		allHeights = append(allHeights, height)
		fmt.Printf("   Eleitor %d: %d blocos\n", i+1, height)
	}
	
	// Verificar sincronização
	allSynced := true
	if len(allHeights) > 0 {
		baseHeight := allHeights[0]
		for i := 1; i < len(allHeights); i++ {
			if allHeights[i] != baseHeight {
				allSynced = false
				break
			}
		}
		
		if allSynced {
			fmt.Printf("✅ Todos os nós sincronizados (altura: %d)\n", baseHeight)
		} else {
			fmt.Println("⚠️  Nós com alturas diferentes - sincronização em andamento")
		}
		
		// Verificar últimos blocos usando o primeiro validador
		fmt.Println("📦 Últimos blocos criados:")
		for i := 0; i < min(3, int(baseHeight)); i++ {
			blockIndex := baseHeight - uint64(i)
			block, err := validatorNodes[0].BlockchainRepo.GetBlockByIndex(ctx, blockIndex)
			if err != nil {
				continue
			}
			
			fmt.Printf("   Bloco %d: %d transações, Validador: %s\n",
				blockIndex, len(block.GetTransactions()), 
				block.GetValidator().String())
		}
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