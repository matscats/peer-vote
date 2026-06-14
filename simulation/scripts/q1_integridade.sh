#!/bin/bash

# Q1 - Integridade: votos válidos são finalizados corretamente?

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
SIM_DIR="$PROJECT_ROOT/simulation"
CONFIG_BACKUP_DIR=""

export GOCACHE="${GOCACHE:-/tmp/peer-vote-go-cache}"

cleanup() {
    pkill -9 -f 'simulation/bin/bootstrap' 2>/dev/null || true
    pkill -9 -f 'simulation/bin/node' 2>/dev/null || true
    if [ -n "$CONFIG_BACKUP_DIR" ] && [ -d "$CONFIG_BACKUP_DIR" ]; then
        cp "$CONFIG_BACKUP_DIR"/config{1,2,3}.json "$SIM_DIR/configs/"
        rm -rf "$CONFIG_BACKUP_DIR"
    fi
}

trap cleanup EXIT

height_from_log() {
    grep -E "chain height:|Current blockchain height:|Blockchain loaded successfully, current height:" "$1" 2>/dev/null \
        | grep -Eo "height:? [0-9]+|chain height: [0-9]+" \
        | grep -Eo "[0-9]+" \
        | tail -1
}

voted_count_from_log() {
    grep "finalized successfully" "$1" 2>/dev/null \
        | tail -1 \
        | grep -o "voted count: [0-9]*" \
        | awk '{print $3}'
}

submit_vote() {
    local voter_id="$1"
    local choice="$2"
    local peer="$3"
    local log_file="$4"

    (cd "$PROJECT_ROOT" && ./simulation/bin/submit-vote \
        -voter "$voter_id" \
        -choice "$choice" \
        -key "simulation/keys/voters/${voter_id}.key" \
        -peer "$peer") > "$log_file" 2>&1 &
}

start_node() {
    local node_id="$1"
    local log_file="$2"
    (cd "$SIM_DIR" && exec ./bin/node -config "configs/config${node_id}.json") > "$log_file" 2>&1 &
    echo $!
}

echo "=========================================="
echo "Q1 - Integridade"
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

if [ "$(find "$SIM_DIR/keys/voters" -name 'voter005.key' 2>/dev/null | wc -l | tr -d ' ')" -eq 0 ]; then
    echo "ERRO: chaves de votantes não encontradas em simulation/keys/voters/."
    exit 1
fi

echo "Iniciando bootstrap e validadores..."
"$SIM_DIR/bin/bootstrap" -port 4000 > "$SIM_DIR/logs/bootstrap.log" 2>&1 &
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
NODE2_ID=$(grep "P2P Host created with ID:" "$SIM_DIR/logs/node2.log" | awk '{print $NF}' | tail -1)
NODE3_ID=$(grep "P2P Host created with ID:" "$SIM_DIR/logs/node3.log" | awk '{print $NF}' | tail -1)

PEERS=(
    "/ip4/127.0.0.1/tcp/4001/p2p/${NODE1_ID}"
    "/ip4/127.0.0.1/tcp/4002/p2p/${NODE2_ID}"
    "/ip4/127.0.0.1/tcp/4003/p2p/${NODE3_ID}"
)

echo "Submetendo 5 votos válidos..."
for i in 1 2 3 4 5; do
    VOTER_ID=$(printf "voter%03d" "$i")
    PEER=${PEERS[$(((i - 1) % 3))]}
    submit_vote "$VOTER_ID" "candidate-a" "$PEER" "$SIM_DIR/logs/vote_${VOTER_ID}.log"
    sleep 1
done

sleep 30

SUBMITTED=$(grep -l "Vote submitted successfully" "$SIM_DIR"/logs/vote_*.log 2>/dev/null | wc -l | tr -d ' ')
FINALIZED=$(voted_count_from_log "$SIM_DIR/logs/node1.log")
FINALIZED=${FINALIZED:-0}
HEIGHT1=$(height_from_log "$SIM_DIR/logs/node1.log")
HEIGHT2=$(height_from_log "$SIM_DIR/logs/node2.log")
HEIGHT3=$(height_from_log "$SIM_DIR/logs/node3.log")

echo ""
echo "Resultado:"
echo "  Votos submetidos: $SUBMITTED"
echo "  Votos finalizados: $FINALIZED"
echo "  Alturas finais: node1=${HEIGHT1:-0}, node2=${HEIGHT2:-0}, node3=${HEIGHT3:-0}"

if [ "$SUBMITTED" -eq 5 ] && [ "$FINALIZED" -ge 5 ]; then
    echo "✅ TESTE PASSOU"
else
    echo "❌ TESTE FALHOU"
    exit 1
fi
