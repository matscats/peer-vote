#!/bin/bash

# Q3 - Tolerância a falhas: operação após crash-stop e sincronização no retorno.

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
    grep -E "chain height:|Current blockchain height:|Blockchain loaded successfully, current height:|Sync completed successfully at height" "$1" 2>/dev/null \
        | grep -Eo "height:? [0-9]+|chain height: [0-9]+" \
        | grep -Eo "[0-9]+" \
        | tail -1
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
echo "Q3 - Falha por parada e sincronização"
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

for voter in voter001 voter002 voter003 voter004 voter005 voter006; do
    if [ ! -f "$SIM_DIR/keys/voters/${voter}.key" ]; then
        echo "ERRO: chave simulation/keys/voters/${voter}.key não encontrada."
        exit 1
    fi
done

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
NODE3_PID=$(start_node 3 "$SIM_DIR/logs/node3.log")
sleep 12

NODE1_ID=$(grep "P2P Host created with ID:" "$SIM_DIR/logs/node1.log" | awk '{print $NF}' | tail -1)
NODE2_ID=$(grep "P2P Host created with ID:" "$SIM_DIR/logs/node2.log" | awk '{print $NF}' | tail -1)
NODE3_ID=$(grep "P2P Host created with ID:" "$SIM_DIR/logs/node3.log" | awk '{print $NF}' | tail -1)

PEER1="/ip4/127.0.0.1/tcp/4001/p2p/${NODE1_ID}"
PEER2="/ip4/127.0.0.1/tcp/4002/p2p/${NODE2_ID}"
PEER3="/ip4/127.0.0.1/tcp/4003/p2p/${NODE3_ID}"

echo "Submetendo votos antes da falha..."
submit_vote "voter001" "candidate-a" "$PEER1" "$SIM_DIR/logs/vote_voter001_before.log"
submit_vote "voter002" "candidate-a" "$PEER2" "$SIM_DIR/logs/vote_voter002_before.log"
submit_vote "voter003" "candidate-a" "$PEER3" "$SIM_DIR/logs/vote_voter003_before.log"
sleep 15

H3_BEFORE=$(height_from_log "$SIM_DIR/logs/node3.log")

echo "Aplicando crash-stop no Node 3..."
kill -9 "$NODE3_PID" 2>/dev/null || true
sleep 2

submit_vote "voter004" "candidate-b" "$PEER1" "$SIM_DIR/logs/vote_voter004_offline.log"
sleep 1
submit_vote "voter005" "candidate-b" "$PEER2" "$SIM_DIR/logs/vote_voter005_offline.log"
sleep 14

H1_OFFLINE=$(height_from_log "$SIM_DIR/logs/node1.log")
H2_OFFLINE=$(height_from_log "$SIM_DIR/logs/node2.log")

echo "Reiniciando Node 3..."
start_node 3 "$SIM_DIR/logs/node3_restart.log" >/dev/null
sleep 18

H3_SYNC=$(height_from_log "$SIM_DIR/logs/node3_restart.log")
NODE3_RESTART_ID=$(grep "P2P Host created with ID:" "$SIM_DIR/logs/node3_restart.log" | awk '{print $NF}' | tail -1)
PEER3_RESTART="/ip4/127.0.0.1/tcp/4003/p2p/${NODE3_RESTART_ID}"

submit_vote "voter006" "candidate-c" "$PEER3_RESTART" "$SIM_DIR/logs/vote_voter006_after_restart.log"
sleep 16

H1_FINAL=$(height_from_log "$SIM_DIR/logs/node1.log")
H2_FINAL=$(height_from_log "$SIM_DIR/logs/node2.log")
H3_FINAL=$(height_from_log "$SIM_DIR/logs/node3_restart.log")

SUM1=$(shasum "$SIM_DIR/data/data1/blk00000.dat" | awk '{print $1}')
SUM2=$(shasum "$SIM_DIR/data/data2/blk00000.dat" | awk '{print $1}')
SUM3=$(shasum "$SIM_DIR/data/data3/blk00000.dat" | awk '{print $1}')
SYNC_EVIDENCE=$(grep -c "Sync completed successfully" "$SIM_DIR/logs/node3_restart.log" 2>/dev/null || true)

echo ""
echo "Resultado:"
echo "  Altura do Node 3 antes da falha: ${H3_BEFORE:-0}"
echo "  Alturas com Node 3 offline: node1=${H1_OFFLINE:-0}, node2=${H2_OFFLINE:-0}"
echo "  Altura do Node 3 após sync: ${H3_SYNC:-0}"
echo "  Alturas finais: node1=${H1_FINAL:-0}, node2=${H2_FINAL:-0}, node3=${H3_FINAL:-0}"
echo "  Checksum node1: $SUM1"
echo "  Checksum node2: $SUM2"
echo "  Checksum node3: $SUM3"
echo "  Evidências de sync: $SYNC_EVIDENCE"

if [ "${H1_OFFLINE:-0}" -gt "${H3_BEFORE:-0}" ] \
    && [ "${H3_FINAL:-0}" -eq "${H1_FINAL:-0}" ] \
    && [ "${H1_FINAL:-0}" -eq "${H2_FINAL:-0}" ] \
    && [ "$SUM1" = "$SUM2" ] \
    && [ "$SUM1" = "$SUM3" ] \
    && [ "$SYNC_EVIDENCE" -gt 0 ]; then
    echo "✅ TESTE PASSOU"
else
    echo "❌ TESTE FALHOU"
    exit 1
fi
