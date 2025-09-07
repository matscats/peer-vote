package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Flags globais
	configFile string
	verbose    bool
	nodeID     string
)

// rootCmd representa o comando base quando chamado sem subcomandos
var rootCmd = &cobra.Command{
	Use:   "peer-vote",
	Short: "Sistema de votação descentralizado baseado em blockchain",
	Long: `Peer-Vote é um sistema de votação descentralizado que utiliza blockchain 
com algoritmo de consenso Proof of Authority (PoA) e comunicação P2P via libp2p.

Cada nó da rede pode atuar como eleitor, garantindo transparência e 
descentralização no processo eleitoral.`,
	Version: "1.0.0",
}

// Execute adiciona todos os comandos filhos ao comando raiz e define flags apropriadamente.
// É chamado pelo main.main(). Só precisa acontecer uma vez para o rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Flags globais
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "arquivo de configuração (padrão: $HOME/.peer-vote.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "saída verbosa")
	rootCmd.PersistentFlags().StringVar(&nodeID, "node-id", "", "ID do nó (gerado automaticamente se não especificado)")

	// Flags locais
	rootCmd.Flags().BoolP("toggle", "t", false, "Mensagem de ajuda para toggle")
}

// initConfig lê o arquivo de configuração e variáveis de ambiente se definidas.
func initConfig() {
	if configFile != "" {
		// Usar arquivo de configuração especificado pela flag
		fmt.Printf("📄 Usando arquivo de configuração: %s\n", configFile)
	} else {
		// Procurar por arquivo de configuração no diretório home
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		configPath := home + "/.peer-vote.yaml"
		if _, err := os.Stat(configPath); err == nil {
			configFile = configPath
			fmt.Printf("📄 Arquivo de configuração encontrado: %s\n", configFile)
		}
	}

	if verbose {
		fmt.Println("🔍 Modo verboso ativado")
	}
}
