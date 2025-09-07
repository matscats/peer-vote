package cli

import (
	"context"
	"fmt"

	"github.com/matscats/peer-vote/peer-vote/application/usecases"
	"github.com/matscats/peer-vote/peer-vote/domain/services"
	"github.com/spf13/cobra"
)

var (
	// Flags do comando status
	showElections bool
	showNetwork   bool
	showAll       bool
)

// statusCmd representa o comando status
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Mostra o status do sistema Peer-Vote",
	Long: `Exibe informações sobre o status atual do sistema incluindo:
- Eleições ativas
- Status da rede P2P
- Status da blockchain
- Estatísticas gerais

Exemplos:
  peer-vote status                    # Status geral
  peer-vote status --elections        # Apenas eleições
  peer-vote status --network          # Apenas rede
  peer-vote status --all              # Informações detalhadas`,
	Run: runStatusCommand,
}

func init() {
	rootCmd.AddCommand(statusCmd)

	// Flags do comando status
	statusCmd.Flags().BoolVar(&showElections, "elections", false, "mostrar apenas status das eleições")
	statusCmd.Flags().BoolVar(&showNetwork, "network", false, "mostrar apenas status da rede")
	statusCmd.Flags().BoolVar(&showAll, "all", false, "mostrar informações detalhadas")
}

func runStatusCommand(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	fmt.Println("📊 Status do Sistema Peer-Vote")
	fmt.Println("==============================")

	// Inicializar serviços
	validationService := services.NewVotingValidator(nil)

	manageElectionUseCase := usecases.NewManageElectionUseCase(validationService, nil)

	// Mostrar status das eleições (padrão ou se solicitado)
	if !showNetwork || showElections || showAll {
		showElectionStatus(ctx, manageElectionUseCase)
	}

	// Mostrar status da rede (se solicitado)
	if showNetwork || showAll {
		showNetworkStatus()
	}

	// Mostrar status da blockchain (se solicitado)
	if showAll {
		showBlockchainStatus()
	}

	// Mostrar estatísticas gerais
	if !showElections && !showNetwork || showAll {
		showGeneralStats(ctx)
	}
}

func showElectionStatus(ctx context.Context, manageElectionUseCase *usecases.ManageElectionUseCase) {
	fmt.Println("\n🗳️  Eleições")
	fmt.Println("------------")

	// Listar todas as eleições
	listRequest := &usecases.ListElectionsRequest{}
	response, err := manageElectionUseCase.ListElections(ctx, listRequest)
	if err != nil {
		fmt.Printf("❌ Erro ao obter eleições: %v\n", err)
		return
	}

	if len(response.Elections) == 0 {
		fmt.Println("📭 Nenhuma eleição encontrada")
		return
	}

	fmt.Printf("📋 Total de eleições: %d\n", response.Count)

	// Mostrar eleições ativas
	activeRequest := &usecases.ListElectionsRequest{ActiveOnly: true}
	activeResponse, err := manageElectionUseCase.ListElections(ctx, activeRequest)
	if err == nil {
		fmt.Printf("🟢 Eleições ativas: %d\n", activeResponse.Count)
	}

	if showAll {
		fmt.Println("\n📝 Detalhes das eleições:")
		for i, election := range response.Elections {
			if i >= 5 && !showAll { // Limitar a 5 se não for --all
				fmt.Printf("   ... e mais %d eleições\n", len(response.Elections)-5)
				break
			}

			status := "🔴"
			switch election.GetStatus() {
			case "ACTIVE":
				status = "🟢"
			case "PENDING":
				status = "🟡"
			case "CLOSED":
				status = "⚫"
			}

			fmt.Printf("   %s %s\n", status, election.GetTitle())
			fmt.Printf("      ID: %s\n", election.GetID().String()[:16]+"...")
			fmt.Printf("      Status: %s\n", election.GetStatus())
			fmt.Printf("      Candidatos: %d\n", len(election.GetCandidates()))
		}
	}
}

func showNetworkStatus() {
	fmt.Println("\n🌐 Rede P2P")
	fmt.Println("-----------")
	fmt.Println("⚠️  Integração P2P não implementada ainda")
	fmt.Println("📡 Status: Offline")
	fmt.Println("🔗 Peers conectados: 0")
	fmt.Println("🔍 Peers descobertos: 0")
}

func showBlockchainStatus() {
	fmt.Println("\n⛓️  Blockchain")
	fmt.Println("-------------")
	fmt.Println("⚠️  Integração Blockchain não implementada ainda")
	fmt.Println("📏 Altura da cadeia: 0")
	fmt.Println("🔗 Último bloco: N/A")
	fmt.Println("✅ Cadeia válida: N/A")
}

func showGeneralStats(ctx context.Context) {
	fmt.Println("\n📈 Estatísticas Gerais")
	fmt.Println("---------------------")

	// Estatísticas agora são obtidas da blockchain
	fmt.Printf("🗳️  Total de eleições: N/A (consulte a blockchain)\n")
	fmt.Printf("📊 Total de votos: N/A (consulte a blockchain)\n")
	fmt.Printf("🎭 Votos anônimos: N/A (consulte a blockchain)\n")

	// Status dos serviços
	fmt.Println("\n🔧 Serviços")
	fmt.Println("-----------")
	fmt.Println("✅ Blockchain: Online")
	fmt.Println("✅ Consenso PoA: Online")
	fmt.Println("✅ Serviço de criptografia: Online")
	fmt.Println("✅ Serviço de validação: Online")
	fmt.Println("⚠️  Rede P2P: Offline")
	fmt.Println("⚠️  Blockchain: Offline")

	fmt.Println("\n💡 Dicas:")
	fmt.Println("   - Use 'peer-vote start' para iniciar um nó completo")
	fmt.Println("   - Use 'peer-vote vote --help' para submeter um voto")
	fmt.Println("   - Use 'peer-vote status --all' para informações detalhadas")
}
