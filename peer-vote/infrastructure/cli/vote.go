package cli

import (
	"context"
	"fmt"
	"log"

	"github.com/matscats/peer-vote/peer-vote/application/usecases"
	"github.com/matscats/peer-vote/peer-vote/domain/services"
	"github.com/matscats/peer-vote/peer-vote/domain/valueobjects"
	"github.com/matscats/peer-vote/peer-vote/infrastructure/crypto"
	"github.com/matscats/peer-vote/peer-vote/infrastructure/persistence"
	"github.com/spf13/cobra"
)

var (
	// Flags do comando vote
	electionID  string
	candidateID string
	voterID     string
	anonymous   bool
	privateKey  string
)

// voteCmd representa o comando vote
var voteCmd = &cobra.Command{
	Use:   "vote",
	Short: "Submete um voto em uma eleição",
	Long: `Submete um voto em uma eleição específica.

Exemplo:
  peer-vote vote --election-id abc123 --candidate-id candidate1 --voter-id voter1

Para voto anônimo:
  peer-vote vote --election-id abc123 --candidate-id candidate1 --anonymous`,
	Run: runVoteCommand,
}

func init() {
	rootCmd.AddCommand(voteCmd)

	// Flags obrigatórias
	voteCmd.Flags().StringVar(&electionID, "election-id", "", "ID da eleição (obrigatório)")
	voteCmd.Flags().StringVar(&candidateID, "candidate-id", "", "ID do candidato (obrigatório)")
	
	// Flags opcionais
	voteCmd.Flags().StringVar(&voterID, "voter-id", "", "ID do eleitor (obrigatório se não anônimo)")
	voteCmd.Flags().BoolVar(&anonymous, "anonymous", false, "voto anônimo")
	voteCmd.Flags().StringVar(&privateKey, "private-key", "", "chave privada para assinar o voto (será gerada se não especificada)")

	// Marcar flags obrigatórias
	voteCmd.MarkFlagRequired("election-id")
	voteCmd.MarkFlagRequired("candidate-id")
}

func runVoteCommand(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	fmt.Println("🗳️  Submetendo voto...")

	// Validar parâmetros
	if err := validateVoteParams(); err != nil {
		log.Fatalf("❌ Erro de validação: %v", err)
	}

	if verbose {
		fmt.Printf("📋 Parâmetros do voto:\n")
		fmt.Printf("   - Eleição: %s\n", electionID)
		fmt.Printf("   - Candidato: %s\n", candidateID)
		fmt.Printf("   - Eleitor: %s\n", voterID)
		fmt.Printf("   - Anônimo: %v\n", anonymous)
	}

	// Inicializar serviços
	fmt.Println("📦 Inicializando serviços...")
	
	cryptoService := crypto.NewECDSAService()
	electionRepo := persistence.NewMemoryElectionRepository()
	voteRepo := persistence.NewMemoryVoteRepository()
	validationService := services.NewVotingValidator(electionRepo, voteRepo, cryptoService)
	
	submitVoteUseCase := usecases.NewSubmitVoteUseCase(electionRepo, voteRepo, cryptoService, validationService)

	// Gerar ou carregar chave privada
	var keyPair *services.KeyPair
	if privateKey == "" {
		fmt.Println("🔐 Gerando chave privada...")
		var err error
		keyPair, err = cryptoService.GenerateKeyPair(ctx)
		if err != nil {
			log.Fatalf("❌ Erro ao gerar chave privada: %v", err)
		}
		fmt.Println("✅ Chave privada gerada")
	} else {
		fmt.Println("🔐 Carregando chave privada...")
		// Usar o ECDSAService para fazer o parsing
		parsedPrivateKey, err := cryptoService.ParsePrivateKeyFromString(privateKey)
		if err != nil {
			log.Fatalf("❌ Erro ao carregar chave privada: %v", err)
		}
		
		// Criar KeyPair com a chave carregada
		// Gerar chave pública a partir da privada
		tempKeyPair, err := cryptoService.GenerateKeyPair(ctx)
		if err != nil {
			log.Fatalf("❌ Erro ao gerar par de chaves temporário: %v", err)
		}
		
		keyPair = &services.KeyPair{
			PrivateKey: parsedPrivateKey,
			PublicKey:  tempKeyPair.PublicKey, // TODO: Derivar corretamente da privada
		}
		fmt.Println("✅ Chave privada carregada")
	}

	// Converter parâmetros
	electionHash, err := valueobjects.NewHashFromString(electionID)
	if err != nil {
		log.Fatalf("❌ ID da eleição inválido: %v", err)
	}

	voterNodeID := valueobjects.NewNodeID(voterID)

	// Criar request de voto
	voteRequest := &usecases.SubmitVoteRequest{
		ElectionID:  electionHash,
		VoterID:     voterNodeID,
		CandidateID: candidateID,
		IsAnonymous: anonymous,
		PrivateKey:  keyPair.PrivateKey,
	}

	// Submeter voto
	fmt.Println("📝 Submetendo voto...")
	response, err := submitVoteUseCase.Execute(ctx, voteRequest)
	if err != nil {
		log.Fatalf("❌ Erro ao submeter voto: %v", err)
	}

	// Mostrar resultado
	fmt.Println("\n🎉 Voto submetido com sucesso!")
	fmt.Printf("   📊 ID do voto: %s\n", response.VoteID)
	fmt.Printf("   💬 Mensagem: %s\n", response.Message)
	
	if verbose {
		fmt.Printf("   🔍 Detalhes:\n")
		fmt.Printf("      - Eleição: %s\n", response.Vote.GetElectionID().String())
		fmt.Printf("      - Candidato: %s\n", response.Vote.GetCandidateID())
		fmt.Printf("      - Timestamp: %s\n", response.Vote.GetTimestamp().Time().Format("2006-01-02 15:04:05"))
		fmt.Printf("      - Anônimo: %v\n", response.Vote.IsAnonymous())
	}

	fmt.Println("\n💡 Use 'peer-vote status' para verificar o status da eleição")
}

func validateVoteParams() error {
	if electionID == "" {
		return fmt.Errorf("ID da eleição é obrigatório")
	}

	if candidateID == "" {
		return fmt.Errorf("ID do candidato é obrigatório")
	}

	if !anonymous && voterID == "" {
		return fmt.Errorf("ID do eleitor é obrigatório para votos não anônimos")
	}

	return nil
}
