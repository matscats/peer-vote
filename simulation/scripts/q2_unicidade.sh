#!/bin/bash

# Q2 - Unicidade: o sistema previne voto duplicado?

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
SIM_DIR="$PROJECT_ROOT/simulation"
CONFIG_BACKUP_DIR=""

export GOCACHE="${GOCACHE:-/tmp/peer-vote-go-cache}"

cleanup() {
    pkill -9 -f 'simulation/bin/bootstrap|./bin/bootstrap|bin/bootstrap' 2>/dev/null || true
    pkill -9 -f 'simulation/bin/node|./bin/node|bin/node' 2>/dev/null || true
    pkill -9 -f 'simulation/bin/submit-vote|submit-vote' 2>/dev/null || true
    if [ -n "$CONFIG_BACKUP_DIR" ] && [ -d "$CONFIG_BACKUP_DIR" ]; then
        cp "$CONFIG_BACKUP_DIR"/config{1,2,3}.json "$SIM_DIR/configs/"
        rm -rf "$CONFIG_BACKUP_DIR"
    fi
}

trap cleanup EXIT

voted_count_from_log() {
    grep "finalized successfully" "$1" 2>/dev/null \
        | tail -1 \
        | grep -o "voted count: [0-9]*" \
        | awk '{print $3}'
}

start_node() {
    local node_id="$1"
    local log_file="$2"
    (cd "$SIM_DIR" && exec ./bin/node -config "configs/config${node_id}.json") > "$log_file" 2>&1 &
    local pid=$!
    disown "$pid" 2>/dev/null || true
    echo "$pid"
}

echo "=========================================="
echo "Q2 - Unicidade de voto"
echo "=========================================="

cleanup
rm -rf "$SIM_DIR/data"/* "$SIM_DIR/logs"/*
mkdir -p "$SIM_DIR/bin" "$SIM_DIR/data" "$SIM_DIR/logs"
CONFIG_BACKUP_DIR="$(mktemp -d /tmp/peer-vote-config-backup.XXXXXX)"
cp "$SIM_DIR"/configs/config{1,2,3}.json "$CONFIG_BACKUP_DIR/"

echo "Compilando binários..."
(cd "$PROJECT_ROOT" && go build -o simulation/bin/bootstrap cmd/bootstrap/main.go)
(cd "$PROJECT_ROOT" && go build -o simulation/bin/node cmd/node/main.go)
(cd "$PROJECT_ROOT" && go build -o simulation/bin/submit-vote cmd/submit-vote/main.go)

if [ ! -f "$SIM_DIR/keys/voters/voter001.key" ]; then
    echo "ERRO: chave simulation/keys/voters/voter001.key não encontrada."
    exit 1
fi

"$SIM_DIR/bin/bootstrap" -port 4000 > "$SIM_DIR/logs/bootstrap.log" 2>&1 &
disown $! 2>/dev/null || true
sleep 3

BOOTSTRAP_ADDR=$(grep "/ip4/127.0.0.1/tcp/4000/p2p/" "$SIM_DIR/logs/bootstrap.log" | head -1 | sed 's/.*\(\/ip4\/127\.0\.0\.1\/tcp\/4000\/p2p\/[^ ]*\).*/\1/')
if [ -z "$BOOTSTRAP_ADDR" ]; then
    echo "ERRO: não foi possível obter endereço do bootstrap."
    exit 1
fi

for config in "$SIM_DIR"/configs/config{1,2,3}.json; do
    if command -v jq >/dev/null 2>&1; then
        jq ".bootstrap_peers = [\"$BOOTSTRAP_ADDR\"]" "$config" > "${config}.tmp" && mv "${config}.tmp" "$config"
    else
        sed -i.bak "s|\"bootstrap_peers\": \[.*\]|\"bootstrap_peers\": [\"$BOOTSTRAP_ADDR\"]|" "$config"
        rm -f "${config}.bak"
    fi
done

start_node 1 "$SIM_DIR/logs/node1.log" >/dev/null
sleep 2
start_node 2 "$SIM_DIR/logs/node2.log" >/dev/null
sleep 2
start_node 3 "$SIM_DIR/logs/node3.log" >/dev/null
sleep 12

NODE1_ID=$(grep "P2P Host created with ID:" "$SIM_DIR/logs/node1.log" | awk '{print $NF}' | tail -1)
PEER1="/ip4/127.0.0.1/tcp/4001/p2p/${NODE1_ID}"

echo "Submetendo primeiro voto de voter001..."
(cd "$PROJECT_ROOT" && ./simulation/bin/submit-vote \
    -voter "voter001" \
    -choice "candidate-a" \
    -key "simulation/keys/voters/voter001.key" \
    -peer "$PEER1") > "$SIM_DIR/logs/vote_first.log" 2>&1

sleep 15

echo "Tentando submeter voto duplicado de voter001..."
set +e
(cd "$PROJECT_ROOT" && ./simulation/bin/submit-vote \
    -voter "voter001" \
    -choice "candidate-b" \
    -key "simulation/keys/voters/voter001.key" \
    -peer "$PEER1") > "$SIM_DIR/logs/vote_duplicate.log" 2>&1
DUPLICATE_EXIT=$?
set -e

sleep 10

FINALIZED=$(voted_count_from_log "$SIM_DIR/logs/node1.log")
FINALIZED=${FINALIZED:-0}
REJECTION_LINES=$(grep -h "already voted\|already in mempool\|duplicate" "$SIM_DIR"/logs/node*.log 2>/dev/null | wc -l | tr -d ' ')

echo ""
echo "Resultado:"
echo "  Votos finalizados na blockchain: $FINALIZED"
echo "  Saída do segundo envio: $DUPLICATE_EXIT"
echo "  Evidências de rejeição nos logs: $REJECTION_LINES"

if [ "$FINALIZED" -eq 1 ] && [ "$REJECTION_LINES" -gt 0 ]; then
    echo "✅ TESTE PASSOU"
else
    echo "❌ TESTE FALHOU"
    exit 1
fi
