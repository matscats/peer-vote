package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/matscats/peer-vote/peer-vote/application/usecases"
	"github.com/matscats/peer-vote/peer-vote/domain/repositories"
	"github.com/matscats/peer-vote/peer-vote/domain/services"
	"github.com/matscats/peer-vote/peer-vote/infrastructure/blockchain"
	"github.com/matscats/peer-vote/peer-vote/infrastructure/rest/handlers"
)

// ServerConfig representa a configuração do servidor REST
type ServerConfig struct {
	Host         string        `json:"host"`
	Port         int           `json:"port"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout"`
}

// DefaultServerConfig retorna configuração padrão do servidor
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Host:         "localhost",
		Port:         8080,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// Server representa o servidor REST da API
type Server struct {
	config     *ServerConfig
	httpServer *http.Server
	router     *mux.Router

	// Handlers
	electionHandler   *handlers.ElectionHandler
	voteHandler       *handlers.VoteHandler
	blockchainHandler *handlers.BlockchainHandler
	nodeHandler       *handlers.NodeHandler
}

// Dependencies representa as dependências necessárias para o servidor
type Dependencies struct {
	// Use Cases
	CreateElectionUseCase *usecases.CreateElectionUseCase
	ManageElectionUseCase *usecases.ManageElectionUseCase
	SubmitVoteUseCase     *usecases.SubmitVoteUseCase
	AuditVotesUseCase     *usecases.AuditVotesUseCase

	// Repositories
	BlockchainRepository repositories.BlockchainRepository

	// Services
	NetworkService  services.NetworkService
	ChainManager    *blockchain.ChainManager
	CryptoService   services.CryptographyService
}

// NewServer cria um novo servidor REST
func NewServer(config *ServerConfig, deps *Dependencies) *Server {
	if config == nil {
		config = DefaultServerConfig()
	}

	// Criar router principal
	router := mux.NewRouter()

	// Criar handlers
	electionHandler := handlers.NewElectionHandler(
		deps.CreateElectionUseCase,
		deps.ManageElectionUseCase,
	)

	voteHandler := handlers.NewVoteHandler(
		deps.SubmitVoteUseCase,
		deps.AuditVotesUseCase,
		deps.CryptoService,
	)

	blockchainHandler := handlers.NewBlockchainHandler(
		deps.BlockchainRepository,
		deps.ChainManager,
	)

	nodeHandler := handlers.NewNodeHandler(
		deps.NetworkService,
	)

	server := &Server{
		config:            config,
		router:            router,
		electionHandler:   electionHandler,
		voteHandler:       voteHandler,
		blockchainHandler: blockchainHandler,
		nodeHandler:       nodeHandler,
	}

	// Configurar rotas
	server.setupRoutes()

	// Configurar servidor HTTP
	server.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
		Handler:      server.router,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		IdleTimeout:  config.IdleTimeout,
	}

	return server
}

// setupRoutes configura todas as rotas da API
func (s *Server) setupRoutes() {
	// Middleware global
	s.router.Use(s.loggingMiddleware)
	s.router.Use(s.corsMiddleware)
	s.router.Use(s.contentTypeMiddleware)

	// API versioning
	api := s.router.PathPrefix("/api/v1").Subrouter()

	// Registrar rotas dos handlers
	s.electionHandler.RegisterRoutes(api)
	s.voteHandler.RegisterRoutes(api)
	s.blockchainHandler.RegisterRoutes(api)
	s.nodeHandler.RegisterRoutes(api)

	// Rota de informações da API
	api.HandleFunc("/info", s.getAPIInfo).Methods("GET")

	// Rota de documentação
	s.router.HandleFunc("/", s.getDocumentation).Methods("GET")
}

// Start inicia o servidor REST
func (s *Server) Start(ctx context.Context) error {
	log.Printf("🚀 Starting REST API server on %s:%d", s.config.Host, s.config.Port)

	// Canal para capturar erros do servidor
	errChan := make(chan error, 1)

	// Iniciar servidor em goroutine
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("failed to start server: %w", err)
		}
	}()

	// Aguardar contexto cancelado ou erro
	select {
	case <-ctx.Done():
		log.Println("🛑 Shutting down REST API server...")
		return s.Stop()
	case err := <-errChan:
		return err
	}
}

// Stop para o servidor REST
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	log.Println("✅ REST API server stopped successfully")
	return nil
}

// GetAddress retorna o endereço do servidor
func (s *Server) GetAddress() string {
	return fmt.Sprintf("http://%s:%d", s.config.Host, s.config.Port)
}

// Middlewares

// loggingMiddleware adiciona logging das requisições
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("📡 %s %s - %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// corsMiddleware adiciona headers CORS
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// contentTypeMiddleware define content type padrão
func (s *Server) contentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Adicionar timestamp ao contexto
		ctx := context.WithValue(r.Context(), "timestamp", time.Now())
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Handlers de informação

// getAPIInfo retorna informações sobre a API
func (s *Server) getAPIInfo(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"name":        "Peer-Vote REST API",
		"version":     "1.0.0",
		"description": "API REST para sistema de votação descentralizado",
		"endpoints": map[string]interface{}{
			"elections": "/api/v1/elections",
			"votes":     "/api/v1/votes",
			"blocks":    "/api/v1/blocks",
			"nodes":     "/api/v1/nodes",
		},
		"documentation": "/",
		"timestamp":     time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// getDocumentation retorna documentação básica da API
func (s *Server) getDocumentation(w http.ResponseWriter, r *http.Request) {
	docs := `
<!DOCTYPE html>
<html>
<head>
    <title>Peer-Vote REST API</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        h1 { color: #333; }
        h2 { color: #666; }
        code { background: #f4f4f4; padding: 2px 4px; border-radius: 3px; }
        .endpoint { margin: 20px 0; padding: 15px; border-left: 4px solid #007cba; background: #f9f9f9; }
    </style>
</head>
<body>
    <h1>🗳️ Peer-Vote REST API</h1>
    <p>API REST para sistema de votação descentralizado baseado em blockchain.</p>
    
    <h2>📋 Endpoints Principais</h2>
    
    <div class="endpoint">
        <h3>Eleições</h3>
        <p><code>GET /api/v1/elections</code> - Listar eleições</p>
        <p><code>POST /api/v1/elections</code> - Criar eleição</p>
        <p><code>GET /api/v1/elections/{id}</code> - Obter eleição</p>
        <p><code>PUT /api/v1/elections/{id}/status</code> - Atualizar status</p>
        <p><code>GET /api/v1/elections/{id}/results</code> - Obter resultados</p>
    </div>
    
    <div class="endpoint">
        <h3>Votos</h3>
        <p><code>POST /api/v1/votes</code> - Submeter voto</p>
        <p><code>GET /api/v1/votes/audit/{election_id}</code> - Auditar votos</p>
        <p><code>GET /api/v1/votes/count/{election_id}</code> - Contar votos</p>
    </div>
    
    <div class="endpoint">
        <h3>Blockchain</h3>
        <p><code>GET /api/v1/blocks</code> - Listar blocos</p>
        <p><code>GET /api/v1/blocks/{index}</code> - Obter bloco por índice</p>
        <p><code>GET /api/v1/blocks/latest</code> - Último bloco</p>
        <p><code>GET /api/v1/chain/status</code> - Status da blockchain</p>
    </div>
    
    <div class="endpoint">
        <h3>Nós</h3>
        <p><code>GET /api/v1/nodes/status</code> - Status do nó</p>
        <p><code>GET /api/v1/nodes/network</code> - Status da rede</p>
        <p><code>GET /api/v1/nodes/health</code> - Health check</p>
    </div>
    
    <h2>ℹ️ Informações</h2>
    <p><code>GET /api/v1/info</code> - Informações da API</p>
</body>
</html>
    `

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(docs))
}
