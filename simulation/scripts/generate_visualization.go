package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"time"
)

// NetworkNode represents a node in the network
type NetworkNode struct {
	ID       string
	Type     string // "bootstrap", "validator", "voter"
	PeerID   string
	Address  string
	Position Position
}

// Position for visualization
type Position struct {
	X float64
	Y float64
}

// Connection between nodes
type Connection struct {
	From string
	To   string
}

// Event represents a blockchain event
type Event struct {
	Timestamp time.Time
	Node      string
	Type      string // "vote_received", "block_proposed", "block_approved", "block_finalized"
	Details   string
}

// Block represents a finalized block
type Block struct {
	Height    int
	Hash      string
	Proposer  string
	VoteCount int
	Timestamp time.Time
}

// VoteDistribution tracks votes per candidate
type VoteDistribution map[string]int

// NetworkStats contains network statistics
type NetworkStats struct {
	TotalNodes       int
	ValidatorNodes   int
	ConnectedPeers   map[string]int
	TotalBlocks      int
	TotalVotes       int
	VoteDistribution VoteDistribution
	AverageBlockTime float64
	Events           []Event
	Blocks           []Block
	Nodes            []NetworkNode
	Connections      []Connection
}

func main() {
	fmt.Println("==========================================")
	fmt.Println("Generating Network Visualization")
	fmt.Println("==========================================")

	// Parse logs
	stats := parseAllLogs()

	// Generate HTML visualization
	if err := generateHTML(stats); err != nil {
		fmt.Printf("Error generating HTML: %v\n", err)
		os.Exit(1)
	}

	// Generate JSON data for external tools
	if err := generateJSON(stats); err != nil {
		fmt.Printf("Error generating JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n✅ Visualization generated successfully!")
	fmt.Println("\nOutput files:")
	fmt.Println("  - simulation/visualization.html (open in browser)")
	fmt.Println("  - simulation/network_data.json (raw data)")
}

func parseAllLogs() *NetworkStats {
	stats := &NetworkStats{
		ConnectedPeers:   make(map[string]int),
		VoteDistribution: make(VoteDistribution),
		Nodes:            []NetworkNode{},
		Connections:      []Connection{},
		Events:           []Event{},
		Blocks:           []Block{},
	}

	// Parse bootstrap log
	parseBootstrapLog(stats)

	// Parse validator logs
	for i := 1; i <= 3; i++ {
		parseValidatorLog(stats, i)
	}

	// Parse vote logs
	parseVoteLogs(stats)

	// Calculate statistics
	calculateStats(stats)

	return stats
}

func parseBootstrapLog(stats *NetworkStats) {
	logPath := filepath.Join("..", "logs", "bootstrap.log")
	file, err := os.Open(logPath)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	peerIDRegex := regexp.MustCompile(`/p2p/([A-Za-z0-9]+)`)
	addressRegex := regexp.MustCompile(`(/ip4/127\.0\.0\.1/tcp/\d+/p2p/[A-Za-z0-9]+)`)

	for scanner.Scan() {
		line := scanner.Text()

		// Extract bootstrap peer ID and address
		// Format: "  /ip4/127.0.0.1/tcp/4000/p2p/12D3Koo..."
		if strings.Contains(line, "/ip4/127.0.0.1/tcp/") && strings.Contains(line, "/p2p/") {
			if matches := addressRegex.FindStringSubmatch(line); len(matches) > 1 {
				address := matches[1]
				if peerMatches := peerIDRegex.FindStringSubmatch(address); len(peerMatches) > 1 {
					stats.Nodes = append(stats.Nodes, NetworkNode{
						ID:       "bootstrap",
						Type:     "bootstrap",
						PeerID:   peerMatches[1],
						Address:  address,
						Position: Position{X: 400, Y: 100},
					})
					stats.TotalNodes++
					return // Only add once
				}
			}
		}
	}
}

func parseValidatorLog(stats *NetworkStats, nodeNum int) {
	logPath := filepath.Join("..", "logs", fmt.Sprintf("node%d.log", nodeNum))
	file, err := os.Open(logPath)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	timestampRegex := regexp.MustCompile(`^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2})`)

	nodeID := fmt.Sprintf("node%d", nodeNum)
	var peerID string
	var address string

	// Position nodes around bootstrap
	x := 400 + 200*float64(nodeNum)*0.8*float64(nodeNum%2*2-1)
	y := 300 + 150*float64((nodeNum-1)%2*2-1)

	for scanner.Scan() {
		line := scanner.Text()

		// Extract timestamp
		var timestamp time.Time
		if matches := timestampRegex.FindStringSubmatch(line); len(matches) > 1 {
			timestamp, _ = time.Parse("2006/01/02 15:04:05", matches[1])
		}

		// Extract peer ID from "P2P Host created with ID: 12D3Koo..."
		if strings.Contains(line, "P2P Host created with ID:") {
			parts := strings.Split(line, "P2P Host created with ID:")
			if len(parts) > 1 {
				peerID = strings.TrimSpace(parts[1])
			}
		}

		// Extract address from "Listening on: [/ip4/127.0.0.1/tcp/4001 ...]"
		if strings.Contains(line, "Listening on:") && peerID != "" {
			re := regexp.MustCompile(`/ip4/127\.0\.0\.1/tcp/(\d+)`)
			if matches := re.FindStringSubmatch(line); len(matches) > 1 {
				port := matches[1]
				address = fmt.Sprintf("/ip4/127.0.0.1/tcp/%s/p2p/%s", port, peerID)

				// Add node once we have both peer ID and address
				stats.Nodes = append(stats.Nodes, NetworkNode{
					ID:       nodeID,
					Type:     "validator",
					PeerID:   peerID,
					Address:  address,
					Position: Position{X: x, Y: y},
				})
				stats.TotalNodes++
				stats.ValidatorNodes++

				// Add connection to bootstrap
				stats.Connections = append(stats.Connections, Connection{
					From: nodeID,
					To:   "bootstrap",
				})
			}
		}

		// Track connections
		if strings.Contains(line, "Connected peers:") {
			re := regexp.MustCompile(`Connected peers: (\d+)`)
			if matches := re.FindStringSubmatch(line); len(matches) > 1 {
				var count int
				fmt.Sscanf(matches[1], "%d", &count)
				stats.ConnectedPeers[nodeID] = count
			}
		}

		// Track events
		if strings.Contains(line, "Vote from") && strings.Contains(line, "added to mempool") {
			re := regexp.MustCompile(`Vote from (\S+) added to mempool`)
			if matches := re.FindStringSubmatch(line); len(matches) > 1 {
				stats.Events = append(stats.Events, Event{
					Timestamp: timestamp,
					Node:      nodeID,
					Type:      "vote_received",
					Details:   fmt.Sprintf("Vote from %s", matches[1]),
				})
			}
		}

		if strings.Contains(line, "Proposed new block") {
			re := regexp.MustCompile(`Proposed new block at height (\d+) with (\d+) votes`)
			if matches := re.FindStringSubmatch(line); len(matches) > 2 {
				stats.Events = append(stats.Events, Event{
					Timestamp: timestamp,
					Node:      nodeID,
					Type:      "block_proposed",
					Details:   fmt.Sprintf("Height %s with %s votes", matches[1], matches[2]),
				})
			}
		}

		if strings.Contains(line, "Block at height") && strings.Contains(line, "finalized successfully") {
			re := regexp.MustCompile(`Block at height (\d+) finalized successfully.*voted count: (\d+)`)
			if matches := re.FindStringSubmatch(line); len(matches) > 2 {
				var height, voteCount int
				fmt.Sscanf(matches[1], "%d", &height)
				fmt.Sscanf(matches[2], "%d", &voteCount)

				stats.Events = append(stats.Events, Event{
					Timestamp: timestamp,
					Node:      nodeID,
					Type:      "block_finalized",
					Details:   fmt.Sprintf("Height %d with %d total votes", height, voteCount),
				})

				stats.Blocks = append(stats.Blocks, Block{
					Height:    height,
					Proposer:  nodeID,
					VoteCount: voteCount,
					Timestamp: timestamp,
				})
			}
		}
	}
}

func parseVoteLogs(stats *NetworkStats) {
	logsDir := filepath.Join("..", "logs")
	files, err := os.ReadDir(logsDir)
	if err != nil {
		return
	}

	voterCount := 0
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "vote_") && strings.HasSuffix(file.Name(), ".log") {
			parseVoteLog(stats, filepath.Join(logsDir, file.Name()), voterCount)
			voterCount++
		}
	}
}

func parseVoteLog(stats *NetworkStats, logPath string, voterIndex int) {
	file, err := os.Open(logPath)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	var voterID string
	var candidate string
	var peerID string
	var targetNode string

	for scanner.Scan() {
		line := scanner.Text()

		// Extract voter ID from filename or log
		if voterID == "" && strings.Contains(line, "voter=") {
			re := regexp.MustCompile(`voter=([a-zA-Z0-9_-]+)`)
			if matches := re.FindStringSubmatch(line); len(matches) > 1 {
				voterID = matches[1]
			}
		}

		// Extract candidate from vote
		if strings.Contains(line, "choice=") {
			re := regexp.MustCompile(`choice=([a-zA-Z0-9_-]+)`)
			if matches := re.FindStringSubmatch(line); len(matches) > 1 {
				candidate = matches[1]
				stats.VoteDistribution[candidate]++
				stats.TotalVotes++
			}
		}

		// Extract peer ID
		if strings.Contains(line, "Created temporary peer with ID:") {
			parts := strings.Split(line, "Created temporary peer with ID:")
			if len(parts) > 1 {
				peerID = strings.TrimSpace(parts[1])
			}
		}

		// Extract target node
		if strings.Contains(line, "Connecting to target peer:") && strings.Contains(line, "/tcp/") {
			re := regexp.MustCompile(`/tcp/(\d+)/`)
			if matches := re.FindStringSubmatch(line); len(matches) > 1 {
				port := matches[1]
				// Map port to node
				switch port {
				case "4001":
					targetNode = "node1"
				case "4002":
					targetNode = "node2"
				case "4003":
					targetNode = "node3"
				}
			}
		}
	}

	// Add voter node if we have the info
	if voterID != "" {
		// Position voters in a circle around the network
		// Distribute them evenly around the center
		radius := 250.0
		x := 400 + radius*float64(1+voterIndex%3)*0.3*float64(1-2*(voterIndex%2))
		y := 400 + radius*float64(1-2*((voterIndex/2)%2))*0.8

		// Clamp to viewbox
		if x < 50 {
			x = 50
		}
		if x > 750 {
			x = 750
		}
		if y < 50 {
			y = 50
		}
		if y > 550 {
			y = 550
		}

		stats.Nodes = append(stats.Nodes, NetworkNode{
			ID:       voterID,
			Type:     "voter",
			PeerID:   peerID,
			Address:  "",
			Position: Position{X: x, Y: y},
		})
		stats.TotalNodes++

		// Add connection to target node if known
		if targetNode != "" {
			stats.Connections = append(stats.Connections, Connection{
				From: voterID,
				To:   targetNode,
			})
		}
	}
}

func calculateStats(stats *NetworkStats) {
	// Sort blocks by height
	sort.Slice(stats.Blocks, func(i, j int) bool {
		return stats.Blocks[i].Height < stats.Blocks[j].Height
	})

	// Calculate average block time
	if len(stats.Blocks) > 1 {
		totalDuration := stats.Blocks[len(stats.Blocks)-1].Timestamp.Sub(stats.Blocks[0].Timestamp)
		stats.AverageBlockTime = totalDuration.Seconds() / float64(len(stats.Blocks)-1)
	}

	stats.TotalBlocks = len(stats.Blocks)

	// Sort events by timestamp
	sort.Slice(stats.Events, func(i, j int) bool {
		return stats.Events[i].Timestamp.Before(stats.Events[j].Timestamp)
	})
}

func generateHTML(stats *NetworkStats) error {
	tmpl := `<!DOCTYPE html>
<html lang="pt-BR">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Visualização da Rede Blockchain - Simulação</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: #333;
            padding: 20px;
        }
        .container {
            max-width: 1400px;
            margin: 0 auto;
            background: white;
            border-radius: 15px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            overflow: hidden;
        }
        header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 30px;
            text-align: center;
        }
        h1 { font-size: 2.5em; margin-bottom: 10px; }
        .subtitle { font-size: 1.1em; opacity: 0.9; }
        .content { padding: 30px; }
        .section {
            margin-bottom: 40px;
            background: #f8f9fa;
            padding: 25px;
            border-radius: 10px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        h2 {
            color: #667eea;
            margin-bottom: 20px;
            font-size: 1.8em;
            border-bottom: 3px solid #667eea;
            padding-bottom: 10px;
        }
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin-bottom: 20px;
        }
        .stat-card {
            background: white;
            padding: 20px;
            border-radius: 8px;
            text-align: center;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
            transition: transform 0.2s;
        }
        .stat-card:hover { transform: translateY(-5px); }
        .stat-value {
            font-size: 2.5em;
            font-weight: bold;
            color: #667eea;
            margin: 10px 0;
        }
        .stat-label {
            color: #666;
            font-size: 0.9em;
            text-transform: uppercase;
            letter-spacing: 1px;
        }
        #network-map {
            width: 100%;
            height: 500px;
            border: 2px solid #ddd;
            border-radius: 8px;
            background: white;
        }
        .node {
            cursor: pointer;
            transition: all 0.3s;
        }
        .node:hover { transform: scale(1.1); }
        .node-bootstrap { fill: #ff6b6b; }
        .node-validator { fill: #4ecdc4; }
        .node-voter { fill: #95e1d3; }
        .connection { stroke: #ddd; stroke-width: 2; opacity: 0.6; }
        .connection-voter { stroke: #95e1d3; stroke-width: 1.5; opacity: 0.4; stroke-dasharray: 5,5; }
        .node-label {
            font-size: 12px;
            fill: #333;
            text-anchor: middle;
            font-weight: bold;
        }
        .voter-label {
            font-size: 10px;
            fill: #666;
            text-anchor: middle;
        }
        .timeline {
            max-height: 400px;
            overflow-y: auto;
            background: white;
            border-radius: 8px;
            padding: 15px;
        }
        .event {
            padding: 12px;
            margin-bottom: 10px;
            border-left: 4px solid #667eea;
            background: #f8f9fa;
            border-radius: 4px;
            transition: all 0.2s;
        }
        .event:hover {
            background: #e9ecef;
            transform: translateX(5px);
        }
        .event-vote_received { border-left-color: #51cf66; }
        .event-block_proposed { border-left-color: #ffd43b; }
        .event-block_finalized { border-left-color: #ff6b6b; }
        .event-time {
            font-size: 0.85em;
            color: #666;
            margin-bottom: 5px;
        }
        .event-node {
            font-weight: bold;
            color: #667eea;
        }
        .blocks-table {
            width: 100%;
            border-collapse: collapse;
            background: white;
            border-radius: 8px;
            overflow: hidden;
        }
        .blocks-table th {
            background: #667eea;
            color: white;
            padding: 15px;
            text-align: left;
        }
        .blocks-table td {
            padding: 12px 15px;
            border-bottom: 1px solid #ddd;
        }
        .blocks-table tr:hover { background: #f8f9fa; }
        .chart-container {
            background: white;
            padding: 20px;
            border-radius: 8px;
            margin-top: 20px;
        }
        .bar {
            display: flex;
            align-items: center;
            margin-bottom: 15px;
        }
        .bar-label {
            width: 120px;
            font-weight: bold;
            color: #333;
        }
        .bar-fill {
            height: 30px;
            background: linear-gradient(90deg, #667eea 0%, #764ba2 100%);
            border-radius: 5px;
            display: flex;
            align-items: center;
            padding: 0 10px;
            color: white;
            font-weight: bold;
            transition: width 0.5s ease;
        }
        .legend {
            display: flex;
            gap: 20px;
            margin-top: 20px;
            flex-wrap: wrap;
        }
        .legend-item {
            display: flex;
            align-items: center;
            gap: 8px;
        }
        .legend-color {
            width: 20px;
            height: 20px;
            border-radius: 4px;
        }
        footer {
            text-align: center;
            padding: 20px;
            background: #f8f9fa;
            color: #666;
            font-size: 0.9em;
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>🔗 Visualização da Rede Blockchain</h1>
            <p class="subtitle">Sistema de Votação Descentralizado - Simulação para TCC</p>
        </header>

        <div class="content">
            <!-- Statistics Overview -->
            <div class="section">
                <h2>📊 Estatísticas Gerais</h2>
                <div class="stats-grid">
                    <div class="stat-card">
                        <div class="stat-label">Total de Nós</div>
                        <div class="stat-value">{{.TotalNodes}}</div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-label">Validadores</div>
                        <div class="stat-value">{{.ValidatorNodes}}</div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-label">Blocos Finalizados</div>
                        <div class="stat-value">{{.TotalBlocks}}</div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-label">Total de Votos</div>
                        <div class="stat-value">{{.TotalVotes}}</div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-label">Tempo Médio/Bloco</div>
                        <div class="stat-value">{{printf "%.1f" .AverageBlockTime}}s</div>
                    </div>
                </div>
            </div>

            <!-- Network Map -->
            <div class="section">
                <h2>🗺️ Mapa da Rede P2P</h2>
                <svg id="network-map" viewBox="0 0 800 600">
                    <!-- Connections -->
                    <g id="connections">
                        {{range .Connections}}
                        {{$fromNode := index $.NodeMap .From}}
                        {{$toNode := index $.NodeMap .To}}
                        <line class="connection{{if eq $fromNode.Type "voter"}}-voter{{end}}" 
                              x1="{{$fromNode.Position.X}}" 
                              y1="{{$fromNode.Position.Y}}"
                              x2="{{$toNode.Position.X}}" 
                              y2="{{$toNode.Position.Y}}" />
                        {{end}}
                    </g>
                    
                    <!-- Nodes -->
                    <g id="nodes">
                        {{range .Nodes}}
                        <g class="node">
                            <circle class="node-{{.Type}}" 
                                    cx="{{.Position.X}}" 
                                    cy="{{.Position.Y}}" 
                                    r="{{if eq .Type "bootstrap"}}30{{else if eq .Type "validator"}}25{{else}}15{{end}}" />
                            <text class="{{if eq .Type "voter"}}voter-label{{else}}node-label{{end}}" 
                                  x="{{.Position.X}}" 
                                  y="{{.Position.Y}}" 
                                  dy="{{if eq .Type "voter"}}4{{else}}5{{end}}">{{.ID}}</text>
                            {{if ne .Type "voter"}}
                            <text class="node-label" 
                                  x="{{.Position.X}}" 
                                  y="{{.Position.Y}}" 
                                  dy="50" 
                                  style="font-size: 10px; fill: #666;">{{slice .PeerID 0 8}}...</text>
                            {{end}}
                        </g>
                        {{end}}
                    </g>
                </svg>
                
                <div class="legend">
                    <div class="legend-item">
                        <div class="legend-color node-bootstrap"></div>
                        <span>Bootstrap Node</span>
                    </div>
                    <div class="legend-item">
                        <div class="legend-color node-validator"></div>
                        <span>Validator Node</span>
                    </div>
                    <div class="legend-item">
                        <div class="legend-color node-voter"></div>
                        <span>Voter Node</span>
                    </div>
                </div>
            </div>

            <!-- Vote Distribution -->
            <div class="section">
                <h2>🗳️ Distribuição de Votos</h2>
                <div class="chart-container">
                    {{range $candidate, $count := .VoteDistribution}}
                    <div class="bar">
                        <div class="bar-label">{{$candidate}}</div>
                        <div class="bar-fill" style="width: {{mul $count 50}}px;">{{$count}} votos</div>
                    </div>
                    {{end}}
                </div>
            </div>

            <!-- Timeline -->
            <div class="section">
                <h2>⏱️ Timeline de Eventos (Últimos 50)</h2>
                <div class="timeline">
                    {{range .RecentEvents}}
                    <div class="event event-{{.Type}}">
                        <div class="event-time">{{.Timestamp.Format "15:04:05"}}</div>
                        <div><span class="event-node">{{.Node}}</span>: {{.Details}}</div>
                    </div>
                    {{end}}
                </div>
            </div>

            <!-- Blocks Table -->
            <div class="section">
                <h2>📦 Blocos Finalizados (Últimos 20)</h2>
                <table class="blocks-table">
                    <thead>
                        <tr>
                            <th>Altura</th>
                            <th>Proposer</th>
                            <th>Votos no Bloco</th>
                            <th>Timestamp</th>
                        </tr>
                    </thead>
                    <tbody>
                        {{range .RecentBlocks}}
                        <tr>
                            <td>#{{.Height}}</td>
                            <td>{{.Proposer}}</td>
                            <td>{{.VoteCount}}</td>
                            <td>{{.Timestamp.Format "15:04:05"}}</td>
                        </tr>
                        {{end}}
                    </tbody>
                </table>
            </div>
        </div>

        <footer>
            <p>Gerado automaticamente pela simulação do sistema de votação blockchain</p>
            <p>Dados coletados dos logs em: {{.GeneratedAt}}</p>
        </footer>
    </div>
</body>
</html>`

	// Prepare template data
	data := prepareTemplateData(stats)

	// Parse and execute template
	t, err := template.New("visualization").Funcs(template.FuncMap{
		"slice": func(s string, start, end int) string {
			if end > len(s) {
				end = len(s)
			}
			return s[start:end]
		},
		"mul": func(a, b int) int {
			return a * b
		},
	}).Parse(tmpl)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Create output file
	outputPath := filepath.Join("..", "visualization.html")
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Execute template
	if err := t.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	fmt.Printf("✅ HTML visualization saved to: %s\n", outputPath)
	return nil
}

func prepareTemplateData(stats *NetworkStats) map[string]interface{} {
	// Get recent events (last 50)
	recentEvents := stats.Events
	if len(recentEvents) > 50 {
		recentEvents = recentEvents[len(recentEvents)-50:]
	}

	// Get recent blocks (last 20)
	recentBlocks := stats.Blocks
	if len(recentBlocks) > 20 {
		recentBlocks = recentBlocks[len(recentBlocks)-20:]
	}

	// Create node map for template
	nodeMap := make(map[string]NetworkNode)
	for _, node := range stats.Nodes {
		nodeMap[node.ID] = node
	}

	return map[string]interface{}{
		"TotalNodes":       stats.TotalNodes,
		"ValidatorNodes":   stats.ValidatorNodes,
		"TotalBlocks":      stats.TotalBlocks,
		"TotalVotes":       stats.TotalVotes,
		"AverageBlockTime": stats.AverageBlockTime,
		"VoteDistribution": stats.VoteDistribution,
		"Nodes":            stats.Nodes,
		"Connections":      stats.Connections,
		"NodeMap":          nodeMap,
		"RecentEvents":     recentEvents,
		"RecentBlocks":     recentBlocks,
		"GeneratedAt":      time.Now().Format("2006-01-02 15:04:05"),
	}
}

func generateJSON(stats *NetworkStats) error {
	outputPath := filepath.Join("..", "network_data.json")
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create JSON file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(stats); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	fmt.Printf("✅ JSON data saved to: %s\n", outputPath)
	return nil
}
