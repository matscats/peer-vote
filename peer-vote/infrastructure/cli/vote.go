package cli

import (
	"fmt"

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
	fmt.Println("❌ Este comando CLI está desatualizado após migração para blockchain")
	fmt.Println("💡 Use a API REST ou a simulação completa para submeter votos")
}