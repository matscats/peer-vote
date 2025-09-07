package cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/matscats/peer-vote/peer-vote/application/usecases"
	"github.com/matscats/peer-vote/peer-vote/domain/services"
	"github.com/matscats/peer-vote/peer-vote/infrastructure/blockchain"
	"github.com/matscats/peer-vote/peer-vote/infrastructure/consensus"
	"github.com/matscats/peer-vote/peer-vote/infrastructure/crypto"
	"github.com/matscats/peer-vote/peer-vote/infrastructure/network"
	"github.com/matscats/peer-vote/peer-vote/infrastructure/persistence"
	"github.com/matscats/peer-vote/peer-vote/infrastructure/rest"
	"github.com/spf13/cobra"
)

var (
	// Flags do comando start
	restPort    int
	restHost    string
	p2pPort     int
	enableRest  bool
	enableP2P   bool
)

// startCmd representa o comando start
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Inicia um nó do Peer-Vote",
	Long: `Inicia um nó completo do sistema Peer-Vote incluindo:
- Servidor REST API (se habilitado)
- Rede P2P (se habilitado)
- Serviços de blockchain e consenso
- Repositórios de dados`,
	Run: runStartCommand,
}

func init() {
	rootCmd.AddCommand(startCmd)

	// Flags do comando start
	startCmd.Flags().IntVar(&restPort, "rest-port", 8080, "porta do servidor REST API")
	startCmd.Flags().StringVar(&restHost, "rest-host", "localhost", "host do servidor REST API")
	startCmd.Flags().IntVar(&p2pPort, "p2p-port", 9000, "porta da rede P2P")
	startCmd.Flags().BoolVar(&enableRest, "enable-rest", true, "habilitar servidor REST API")
	startCmd.Flags().BoolVar(&enableP2P, "enable-p2p", true, "habilitar rede P2P")
}

func runStartCommand(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fmt.Println("🚀 Iniciando nó Peer-Vote...")
	
	if verbose {
		fmt.Printf("📋 Configurações:\n")
		fmt.Printf("   - REST API: %s:%d (habilitado: %v)\n", restHost, restPort, enableRest)
		fmt.Printf("   - P2P: porta %d (habilitado: %v)\n", p2pPort, enableP2P)
		fmt.Printf("   - Node ID: %s\n", nodeID)
	}

	// 1. Inicializar serviços base
	fmt.Println("📦 Inicializando serviços...")
	
	// Serviços de infraestrutura
	cryptoService := crypto.NewECDSAService()
	electionRepo := persistence.NewMemoryElectionRepository()
	blockchainRepo := persistence.NewMemoryBlockchainRepository(cryptoService)
	
	// Serviços de blockchain
	chainManager := blockchain.NewChainManager(blockchainRepo, cryptoService)
	
	// Gerar chave para este nó
	keyPair, err := cryptoService.GenerateKeyPair(ctx)
	if err != nil {
		log.Fatalf("❌ Erro ao gerar chave do nó: %v", err)
	}
	
	myNodeID := cryptoService.GenerateNodeID(ctx, keyPair.PublicKey)
	
	// Serviços de consenso
	validatorManager := consensus.NewValidatorManager()
	poaEngine := consensus.NewPoAEngine(validatorManager, chainManager, cryptoService, myNodeID, keyPair.PrivateKey)
	
	// Serviços de domínio
	validationService := services.NewVotingValidator(electionRepo, nil, cryptoService)
	
	// Casos de uso
	createElectionUseCase := usecases.NewCreateElectionUseCase(electionRepo, cryptoService, validationService, chainManager, poaEngine)
	manageElectionUseCase := usecases.NewManageElectionUseCase(electionRepo, validationService, chainManager)
	submitVoteUseCase := usecases.NewSubmitVoteUseCase(electionRepo, chainManager, poaEngine, cryptoService, validationService)
	auditVotesUseCase := usecases.NewAuditVotesUseCase(electionRepo, chainManager, cryptoService, validationService)

	// Serviço P2P (se habilitado)
	var p2pService *network.P2PService
	if enableP2P {
		fmt.Printf("🔗 Inicializando rede P2P na porta %d...\n", p2pPort)
		
		p2pConfig := &network.P2PConfig{
			ListenAddresses: []string{fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", p2pPort)},
			BootstrapPeers:  []string{}, // Sem bootstrap peers por padrão
			MaxConnections:  50,
			EnableMDNS:      true,
			EnableDHT:       true,
			Namespace:       "peer-vote",
		}
		
		p2pService, err = network.NewP2PService(chainManager, poaEngine, cryptoService, p2pConfig)
		if err != nil {
			log.Fatalf("❌ Erro ao criar serviço P2P: %v", err)
		}
		
		// Iniciar P2P service
		if err := p2pService.Start(ctx); err != nil {
			log.Fatalf("❌ Erro ao iniciar P2P: %v", err)
		}
		
		fmt.Printf("✅ Rede P2P iniciada: %s\n", myNodeID.String()[:16]+"...")
	}

	fmt.Println("✅ Serviços inicializados com sucesso")

	// 2. Inicializar servidor REST (se habilitado)
	var restServer *rest.Server
	if enableRest {
		fmt.Printf("🌐 Iniciando servidor REST API em %s:%d...\n", restHost, restPort)
		
		restConfig := &rest.ServerConfig{
			Host: restHost,
			Port: restPort,
		}

		var networkService services.NetworkService
		if p2pService != nil {
			networkService = network.NewNetworkAdapter(p2pService)
		}

		deps := &rest.Dependencies{
			CreateElectionUseCase: createElectionUseCase,
			ManageElectionUseCase: manageElectionUseCase,
			SubmitVoteUseCase:     submitVoteUseCase,
			AuditVotesUseCase:     auditVotesUseCase,
			BlockchainRepository:  blockchainRepo,
			NetworkService:        networkService,
			ChainManager:          chainManager,
			CryptoService:         cryptoService,
		}

		restServer = rest.NewServer(restConfig, deps)
		
		// Iniciar servidor REST em goroutine
		go func() {
			if err := restServer.Start(ctx); err != nil {
				log.Printf("❌ Erro no servidor REST: %v", err)
			}
		}()

		fmt.Printf("✅ Servidor REST iniciado: %s\n", restServer.GetAddress())
	}

	// 3. Inicializar rede P2P (se habilitado)
	if enableP2P {
		fmt.Printf("🔗 Rede P2P será iniciada na porta %d\n", p2pPort)
		fmt.Println("⚠️  Integração P2P será implementada na integração final")
	}

	// 4. Mostrar informações do nó
	fmt.Println("\n🎯 Nó Peer-Vote iniciado com sucesso!")
	fmt.Println("=====================================")
	
	if enableRest {
		fmt.Printf("📡 REST API: %s\n", restServer.GetAddress())
		fmt.Printf("📖 Documentação: %s/\n", restServer.GetAddress())
		fmt.Printf("ℹ️  Info da API: %s/api/v1/info\n", restServer.GetAddress())
	}
	
	if enableP2P && p2pService != nil {
		fmt.Printf("🔗 P2P: porta %d (Node ID: %s)\n", p2pPort, myNodeID.String()[:16]+"...")
	}
	
	fmt.Println("\n💡 Comandos úteis:")
	fmt.Println("   peer-vote status    - Verificar status do nó")
	fmt.Println("   peer-vote vote      - Submeter um voto")
	fmt.Println("   Ctrl+C              - Parar o nó")

	// 5. Aguardar sinal de interrupção
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("\n⏳ Nó em execução... Pressione Ctrl+C para parar")
	
	select {
	case <-sigChan:
		fmt.Println("\n🛑 Sinal de interrupção recebido, parando nó...")
	case <-ctx.Done():
		fmt.Println("\n🛑 Contexto cancelado, parando nó...")
	}

	// 6. Shutdown graceful
	fmt.Println("🔄 Parando serviços...")
	
	if restServer != nil {
		if err := restServer.Stop(); err != nil {
			log.Printf("❌ Erro ao parar servidor REST: %v", err)
		}
	}
	
	if p2pService != nil {
		if err := p2pService.Stop(ctx); err != nil {
			log.Printf("❌ Erro ao parar P2P: %v", err)
		}
	}

	fmt.Println("✅ Nó Peer-Vote parado com sucesso!")
}
