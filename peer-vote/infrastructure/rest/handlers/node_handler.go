package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/matscats/peer-vote/peer-vote/domain/services"
)

// NodeHandler gerencia endpoints relacionados aos nós da rede
type NodeHandler struct {
	networkService services.NetworkService
}

// NewNodeHandler cria um novo handler de nós
func NewNodeHandler(networkService services.NetworkService) *NodeHandler {
	return &NodeHandler{
		networkService: networkService,
	}
}

// NodeStatusResponse representa o status de um nó
type NodeStatusResponse struct {
	NodeID          string `json:"node_id"`
	IsRunning       bool   `json:"is_running"`
	ConnectedPeers  int    `json:"connected_peers"`
	DiscoveredPeers int    `json:"discovered_peers"`
	ListenAddresses []string `json:"listen_addresses"`
	MultiAddresses  []string `json:"multi_addresses"`
}

// NetworkStatusResponse representa o status da rede
type NetworkStatusResponse struct {
	TotalNodes      int    `json:"total_nodes"`
	ConnectedNodes  int    `json:"connected_nodes"`
	NetworkHealth   string `json:"network_health"`
	LastSyncTime    int64  `json:"last_sync_time"`
}

// RegisterRoutes registra as rotas do handler
func (h *NodeHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/nodes/status", h.GetNodeStatus).Methods("GET")
	router.HandleFunc("/nodes/network", h.GetNetworkStatus).Methods("GET")
	router.HandleFunc("/nodes/peers", h.GetPeers).Methods("GET")
	router.HandleFunc("/nodes/health", h.HealthCheck).Methods("GET")
}

// GetNodeStatus obtém o status do nó atual
func (h *NodeHandler) GetNodeStatus(w http.ResponseWriter, r *http.Request) {
	// Verificar se o serviço de rede está disponível
	if h.networkService == nil {
		http.Error(w, "Network service not available", http.StatusServiceUnavailable)
		return
	}

	// Obter status da rede
	status, err := h.networkService.GetNetworkStatus(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Converter para resposta
	nodeStatus := NodeStatusResponse{
		NodeID:          status.NodeID.String(),
		IsRunning:       status.IsRunning,
		ConnectedPeers:  status.PeerCount,
		DiscoveredPeers: 0, // TODO: Implementar contagem de peers descobertos
		ListenAddresses: status.ListenAddrs,
		MultiAddresses:  status.ListenAddrs, // TODO: Implementar MultiAddresses separadamente
	}

	// Retornar resposta
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nodeStatus)
}

// GetNetworkStatus obtém o status geral da rede
func (h *NodeHandler) GetNetworkStatus(w http.ResponseWriter, r *http.Request) {
	// Verificar se o serviço de rede está disponível
	if h.networkService == nil {
		http.Error(w, "Network service not available", http.StatusServiceUnavailable)
		return
	}

	// Obter status da rede
	status, err := h.networkService.GetNetworkStatus(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Determinar saúde da rede
	networkHealth := "healthy"
	if !status.IsRunning {
		networkHealth = "offline"
	} else if status.PeerCount == 0 {
		networkHealth = "isolated"
	} else if status.PeerCount < 3 {
		networkHealth = "degraded"
	}

	// Converter para resposta
	networkStatus := NetworkStatusResponse{
		TotalNodes:     status.PeerCount + 1, // +1 para o nó atual
		ConnectedNodes: status.PeerCount,
		NetworkHealth:  networkHealth,
		LastSyncTime:   status.LastSync.Unix(),
	}

	// Retornar resposta
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(networkStatus)
}

// GetPeers obtém a lista de peers conectados
func (h *NodeHandler) GetPeers(w http.ResponseWriter, r *http.Request) {
	// Verificar se o serviço de rede está disponível
	if h.networkService == nil {
		http.Error(w, "Network service not available", http.StatusServiceUnavailable)
		return
	}

	// Obter lista de peers
	peers, err := h.networkService.GetConnectedPeers(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Converter para formato de resposta
	peerList := make([]map[string]interface{}, len(peers))
	for i, peer := range peers {
		peerList[i] = map[string]interface{}{
			"peer_id":   peer.ID.String(),
			"connected": true,
		}
	}

	// Retornar resposta
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"peers": peerList,
		"count": len(peers),
	})
}

// HealthCheck verifica a saúde do nó
func (h *NodeHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	// Verificações básicas de saúde
	checks := map[string]interface{}{
		"network_service": h.networkService != nil,
		"timestamp":       r.Context().Value("timestamp"),
	}

	// Verificar status da rede se disponível
	if h.networkService != nil {
		status, err := h.networkService.GetNetworkStatus(r.Context())
		checks["network_running"] = err == nil && status.IsRunning
		checks["has_peers"] = err == nil && status.PeerCount > 0
	}

	// Determinar status geral
	allHealthy := true
	for _, check := range checks {
		if b, ok := check.(bool); ok && !b {
			allHealthy = false
			break
		}
	}

	// Definir código de status HTTP
	statusCode := http.StatusOK
	if !allHealthy {
		statusCode = http.StatusServiceUnavailable
	}

	// Retornar resposta
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": map[string]string{
			"overall": func() string {
				if allHealthy {
					return "healthy"
				}
				return "unhealthy"
			}(),
		},
		"checks": checks,
	})
}
