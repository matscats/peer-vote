#!/bin/bash

# Q3 - Tolerância a falhas: operação após crash-stop de um validador.
# Este script preserva o critério usado no experimento descrito no TCC:
# validar que há progresso durante a falha, com pelo menos um novo bloco e
# pelo menos um novo voto finalizado pelos validadores remanescentes.

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

finalized_blocks_from_log() {
    grep "finalized successfully" "$1" 2>/dev/null | wc -l | tr -d ' '
}

voted_count_from_log() {
    local value
    value=$(grep "finalized successfully" "$1" 2>/dev/null \
        | tail -1 \
        | grep -o "voted count: [0-9]*" \
        | awk '{print $3}' || true)
    echo "${value:-0}"
}

height_from_log() {
    local value
    value=$(grep -E "chain height:|Current blockchain height:|Blockchain loaded successfully, current height:" "$1" 2>/dev/null \
        | grep -Eo "height:? [0-9]+|chain height: [0-9]+" \
        | grep -Eo "[0-9]+" \
        | tail -1 || true)
    echo "${value:-0}"
}

submit_vote_async() {
    local voter_id="$1"
    local choice="$2"
    local peer="$3"
    local log_file="$4"

    (cd "$PROJECT_ROOT" && ./simulation/bin/submit-vote \
        -voter "$voter_id" \
        -choice "$choice" \
        -key "simulation/keys/voters/${voter_id}.key" \
        -peer "$peer") > "$log_file" 2>&1 &
    disown $! 2>/dev/null || true
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
echo "Q3 - Tolerância a falhas"
echo "=========================================="
echo "Cenário: crash-stop do Node 2"
echo ""

cleanup
rm -rf "$SIM_DIR/data"/* "$SIM_DIR/logs"/*
mkdir -p "$SIM_DIR/bin" "$SIM_DIR/data" "$SIM_DIR/logs"
CONFIG_BACKUP_DIR="$(mktemp -d /tmp/peer-vote-config-backup.XXXXXX)"
cp "$SIM_DIR"/configs/config{1,2,3}.json "$CONFIG_BACKUP_DIR/"

echo "Compilando binários..."
(cd "$PROJECT_ROOT" && go build -o simulation/bin/bootstrap cmd/bootstrap/main.go)
(cd "$PROJECT_ROOT" && go build -o simulation/bin/node cmd/node/main.go)
(cd "$PROJECT_ROOT" && go build -o simulation/bin/submit-vote cmd/submit-vote/main.go)

for voter in voter001 voter002 voter003 voter004 voter005 voter006 voter007 voter008; do
    if [ ! -f "$SIM_DIR/keys/voters/${voter}.key" ]; then
        echo "ERRO: chave simulation/keys/voters/${voter}.key não encontrada."
        exit 1
    fi
done

echo "Iniciando bootstrap e validadores..."
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
NODE2_PID=$(start_node 2 "$SIM_DIR/logs/node2.log")
sleep 2
start_node 3 "$SIM_DIR/logs/node3.log" >/dev/null
sleep 10

NODE1_ID=$(grep "P2P Host created with ID:" "$SIM_DIR/logs/node1.log" | awk '{print $NF}' | tail -1)
NODE2_ID=$(grep "P2P Host created with ID:" "$SIM_DIR/logs/node2.log" | awk '{print $NF}' | tail -1)
NODE3_ID=$(grep "P2P Host created with ID:" "$SIM_DIR/logs/node3.log" | awk '{print $NF}' | tail -1)

TARGET_PEERS=(
    "/ip4/127.0.0.1/tcp/4001/p2p/${NODE1_ID}"
    "/ip4/127.0.0.1/tcp/4002/p2p/${NODE2_ID}"
    "/ip4/127.0.0.1/tcp/4003/p2p/${NODE3_ID}"
)

echo ""
echo "Fase 1: operação normal"
echo "Submetendo 3 votos..."
for i in 1 2 3; do
    VOTER_ID=$(printf "voter%03d" "$i")
    TARGET_PEER=${TARGET_PEERS[$(((i - 1) % 3))]}
    submit_vote_async "$VOTER_ID" "candidate-a" "$TARGET_PEER" "$SIM_DIR/logs/vote_${VOTER_ID}_phase1.log"
done

sleep 10

INITIAL_BLOCKS=$(finalized_blocks_from_log "$SIM_DIR/logs/node1.log")
INITIAL_VOTES=$(voted_count_from_log "$SIM_DIR/logs/node1.log")

echo "Estado inicial:"
echo "  Validadores ativos: 3/3"
echo "  Blocos finalizados: $INITIAL_BLOCKS"
echo "  Votos finalizados: $INITIAL_VOTES"

echo ""
echo "Fase 2: injeção de falha"
echo "Aplicando SIGKILL no Node 2..."
kill -9 "$NODE2_PID" 2>/dev/null || true
sleep 5

echo ""
echo "Fase 3: operação sob falha"
echo "Submetendo 5 votos aos validadores remanescentes..."
for i in 4 5 6 7 8; do
    VOTER_ID=$(printf "voter%03d" "$i")
    if [ $(((i - 4) % 2)) -eq 0 ]; then
        TARGET_PEER=${TARGET_PEERS[0]}
    else
        TARGET_PEER=${TARGET_PEERS[2]}
    fi

    submit_vote_async "$VOTER_ID" "candidate-b" "$TARGET_PEER" "$SIM_DIR/logs/vote_${VOTER_ID}_phase3.log"
    sleep 1
done

sleep 15

DURING_BLOCKS_TOTAL=$(finalized_blocks_from_log "$SIM_DIR/logs/node1.log")
DURING_VOTES_TOTAL=$(voted_count_from_log "$SIM_DIR/logs/node1.log")
BLOCKS_DURING=$((DURING_BLOCKS_TOTAL - INITIAL_BLOCKS))
VOTES_DURING=$((DURING_VOTES_TOTAL - INITIAL_VOTES))
HEIGHT1=$(height_from_log "$SIM_DIR/logs/node1.log")
HEIGHT3=$(height_from_log "$SIM_DIR/logs/node3.log")

echo ""
echo "Resultado:"
echo "  Fase 1: $INITIAL_VOTES votos, $INITIAL_BLOCKS blocos"
echo "  Fase 3: $VOTES_DURING votos novos, $BLOCKS_DURING blocos novos"
echo "  Total no Node 1: $DURING_VOTES_TOTAL votos, $DURING_BLOCKS_TOTAL blocos"
echo "  Alturas dos remanescentes: node1=${HEIGHT1}, node3=${HEIGHT3}"

echo ""
echo "Validação:"
if [ "$BLOCKS_DURING" -gt 0 ]; then
    echo "✅ Sistema continuou finalizando blocos durante a falha"
else
    echo "❌ Sistema não finalizou novos blocos durante a falha"
fi

if [ "$VOTES_DURING" -gt 0 ]; then
    echo "✅ Votos foram processados durante a falha"
else
    echo "⚠️  Nenhum voto novo foi finalizado durante a falha"
fi

if [ "$BLOCKS_DURING" -gt 0 ] && [ "$VOTES_DURING" -gt 0 ]; then
    echo "✅ TESTE PASSOU"
else
    echo "⚠️  TESTE INCOMPLETO"
fi
