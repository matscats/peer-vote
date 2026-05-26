#!/bin/bash

# TCC Test 1: Validação Funcional Básica
# Objetivo: Validar ciclo completo de consenso com carga mínima

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "=========================================="
echo "TCC - Teste 1: Validação Funcional"
echo "=========================================="
echo ""

# Clean up
echo "Limpando dados antigos..."
rm -rf ../data/* ../logs/*
pkill -9 -f 'bin/bootstrap' 2>/dev/null || true
pkill -9 -f 'bin/node' 2>/dev/null || true
sleep 2

# Build if needed
if [ ! -f "../bin/node" ]; then
    echo "Compilando binários..."
    ./build.sh
fi

# Check voters
if [ ! -d "../keys/voters" ] || [ $(ls -1 ../keys/voters/*.key 2>/dev/null | wc -l) -lt 5 ]; then
    echo "Gerando votantes..."
    ./generate_voters.sh 10
    ./update_eligibility.sh
fi

echo ""
echo "Iniciando rede..."

# Start bootstrap
../bin/bootstrap -port 4000 > ../logs/bootstrap.log 2>&1 &
BOOTSTRAP_PID=$!
sleep 3

# Get bootstrap address
BOOTSTRAP_ADDR=$(grep "/ip4/127.0.0.1/tcp/4000/p2p/" ../logs/bootstrap.log | head -1 | sed 's/.*\(\/ip4\/127\.0\.0\.1\/tcp\/4000\/p2p\/[^ ]*\).*/\1/')

if [ -z "$BOOTSTRAP_ADDR" ]; then
    echo "❌ Erro ao obter endereço do bootstrap"
    kill $BOOTSTRAP_PID 2>/dev/null
    exit 1
fi

echo "✓ Bootstrap: $BOOTSTRAP_ADDR"

# Update configs
for config in ../configs/config{1,2,3}.json; do
    if command -v jq &> /dev/null; then
        jq ".bootstrap_peers = [\"$BOOTSTRAP_ADDR\"]" $config > ${config}.tmp && mv ${config}.tmp $config
    else
        sed -i.bak "s|\"bootstrap_peers\": \[.*\]|\"bootstrap_peers\": [\"$BOOTSTRAP_ADDR\"]|" $config
        rm -f ${config}.bak
    fi
done

# Start nodes
echo "✓ Iniciando 3 validadores..."
for i in 1 2 3; do
    (cd .. && ./bin/node -config configs/config${i}.json) > ../logs/node${i}.log 2>&1 &
    sleep 2
done

echo "✓ Aguardando estabilização (15s)..."
sleep 15

# Get peer IDs
NODE1_ID=$(grep "P2P Host created with ID:" ../logs/node1.log | awk '{print $NF}')
NODE2_ID=$(grep "P2P Host created with ID:" ../logs/node2.log | awk '{print $NF}')
NODE3_ID=$(grep "P2P Host created with ID:" ../logs/node3.log | awk '{print $NF}')

TARGET_PEERS=(
    "/ip4/127.0.0.1/tcp/4001/p2p/${NODE1_ID}"
    "/ip4/127.0.0.1/tcp/4002/p2p/${NODE2_ID}"
    "/ip4/127.0.0.1/tcp/4003/p2p/${NODE3_ID}"
)

echo ""
echo "Submetendo 5 votos..."

# Submit 5 votes
CANDIDATES=("candidate-a" "candidate-b" "candidate-c")
for i in 1 2 3 4 5; do
    VOTER_ID=$(printf "voter%03d" $i)
    CANDIDATE=${CANDIDATES[$((RANDOM % 3))]}
    TARGET_PEER=${TARGET_PEERS[$(((i-1) % 3))]}
    
    (cd ../.. && ./simulation/bin/submit-vote \
        -voter "$VOTER_ID" \
        -choice "$CANDIDATE" \
        -key "simulation/keys/voters/${VOTER_ID}.key" \
        -peer "$TARGET_PEER") > ../logs/vote_${VOTER_ID}.log 2>&1 &
    
    sleep 1
done

echo "✓ Aguardando processamento (30s)..."
sleep 30

echo ""
echo "=========================================="
echo "Resultados"
echo "=========================================="

# Collect metrics
VOTES_SUBMITTED=$(grep -l "Vote submitted successfully" ../logs/vote_*.log 2>/dev/null | wc -l | tr -d ' ')
VOTES_IN_MEMPOOL=$(grep "added to mempool" ../logs/node1.log 2>/dev/null | wc -l | tr -d ' ')
BLOCKS_FINALIZED=$(grep "finalized successfully" ../logs/node1.log 2>/dev/null | wc -l | tr -d ' ')
VOTES_FINALIZED=$(grep "finalized successfully" ../logs/node1.log 2>/dev/null | tail -1 | grep -o "voted count: [0-9]*" | awk '{print $3}')

if [ -z "$VOTES_FINALIZED" ]; then
    VOTES_FINALIZED=0
fi

echo "Votos submetidos: $VOTES_SUBMITTED"
echo "Votos no mempool: $VOTES_IN_MEMPOOL"
echo "Blocos finalizados: $BLOCKS_FINALIZED"
echo "Votos finalizados: $VOTES_FINALIZED"
echo ""

# Check round-robin
echo "Verificação de Round-Robin:"
grep "Proposed new block at height" ../logs/node*.log | head -5 | while read line; do
    HEIGHT=$(echo "$line" | grep -o "height [0-9]*" | awk '{print $2}')
    NODE=$(echo "$line" | grep -o "node[0-9]")
    echo "  Altura $HEIGHT: $NODE"
done

echo ""

# Success check
if [ $VOTES_FINALIZED -ge 5 ]; then
    echo "✅ TESTE PASSOU"
    echo "   - Todos os votos foram finalizados"
    echo "   - Consenso funcionando corretamente"
else
    echo "⚠️  TESTE INCOMPLETO"
    echo "   - Esperado: 5 votos finalizados"
    echo "   - Obtido: $VOTES_FINALIZED votos"
fi

echo ""
echo "Logs salvos em: simulation/logs/"
echo ""

# Cleanup
echo "Parando nós..."
pkill -9 -f 'bin/bootstrap' 2>/dev/null || true
pkill -9 -f 'bin/node' 2>/dev/null || true

echo "✓ Teste 1 concluído!"
