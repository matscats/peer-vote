#!/bin/bash

# TCC Test 3: Performance e Throughput
# Objetivo: Medir capacidade de processamento do sistema

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

NUM_VOTERS=50
DURATION=30

echo "=========================================="
echo "TCC - Teste 3: Performance e Throughput"
echo "=========================================="
echo ""
echo "Configuração:"
echo "  Votos: $NUM_VOTERS"
echo "  Duração: ${DURATION}s"
echo ""

# Clean up
echo "Limpando dados antigos..."
rm -rf ../data/* ../logs/*
pkill -9 -f 'bin/bootstrap' 2>/dev/null || true
pkill -9 -f 'bin/node' 2>/dev/null || true
sleep 2

# Check voters
if [ $(ls -1 ../keys/voters/*.key 2>/dev/null | wc -l) -lt $NUM_VOTERS ]; then
    echo "Gerando $NUM_VOTERS votantes..."
    ./generate_voters.sh $NUM_VOTERS
    ./update_eligibility.sh
fi

echo ""
echo "Iniciando rede..."

# Start bootstrap
../bin/bootstrap -port 4000 > ../logs/bootstrap.log 2>&1 &
sleep 3

BOOTSTRAP_ADDR=$(grep "/ip4/127.0.0.1/tcp/4000/p2p/" ../logs/bootstrap.log | head -1 | sed 's/.*\(\/ip4\/127\.0\.0\.1\/tcp\/4000\/p2p\/[^ ]*\).*/\1/')

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

echo "✓ Aguardando estabilização (10s)..."
sleep 10

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
echo "Submetendo $NUM_VOTERS votos..."

START_TIME=$(date +%s)

# Submit votes
VOTE_COUNT=0
CANDIDATES=("candidate-a" "candidate-b" "candidate-c")

for key_file in ../keys/voters/*.key; do
    if [ $VOTE_COUNT -ge $NUM_VOTERS ]; then
        break
    fi
    
    VOTER_ID=$(basename "$key_file" .key)
    CANDIDATE=${CANDIDATES[$((RANDOM % 3))]}
    TARGET_INDEX=$((VOTE_COUNT % 3))
    TARGET_PEER=${TARGET_PEERS[$TARGET_INDEX]}
    
    (cd ../.. && ./simulation/bin/submit-vote \
        -voter "$VOTER_ID" \
        -choice "$CANDIDATE" \
        -key "simulation/keys/voters/${VOTER_ID}.key" \
        -peer "$TARGET_PEER") > ../logs/vote_${VOTER_ID}.log 2>&1 &
    
    VOTE_COUNT=$((VOTE_COUNT + 1))
    
    if [ $((VOTE_COUNT % 10)) -eq 0 ]; then
        echo "  Submetidos: $VOTE_COUNT/$NUM_VOTERS"
    fi
done

SUBMIT_END=$(date +%s)
SUBMIT_DURATION=$((SUBMIT_END - START_TIME))

echo "✓ Todos os votos submetidos em ${SUBMIT_DURATION}s"
echo ""
echo "Aguardando processamento (${DURATION}s)..."
sleep $DURATION

END_TIME=$(date +%s)
TOTAL_DURATION=$((END_TIME - START_TIME))

echo ""
echo "=========================================="
echo "Resultados"
echo "=========================================="

# Collect metrics
VOTES_SUBMITTED=$(grep -l "Vote submitted successfully" ../logs/vote_*.log 2>/dev/null | wc -l | tr -d ' ')
VOTES_FAILED=$((NUM_VOTERS - VOTES_SUBMITTED))
BLOCKS_FINALIZED=$(grep "finalized successfully" ../logs/node1.log 2>/dev/null | wc -l | tr -d ' ')
VOTES_FINALIZED=$(grep "finalized successfully" ../logs/node1.log 2>/dev/null | tail -1 | grep -o "voted count: [0-9]*" | awk '{print $3}')

if [ -z "$VOTES_FINALIZED" ]; then
    VOTES_FINALIZED=0
fi

VOTE_LOSS=$((VOTES_SUBMITTED - VOTES_FINALIZED))
THROUGHPUT=$(echo "scale=2; $VOTES_FINALIZED / $TOTAL_DURATION" | bc)

if [ $BLOCKS_FINALIZED -gt 0 ]; then
    AVG_VOTES_PER_BLOCK=$(echo "scale=2; $VOTES_FINALIZED / $BLOCKS_FINALIZED" | bc)
else
    AVG_VOTES_PER_BLOCK=0
fi

echo "Submissão:"
echo "  Votos submetidos: $VOTES_SUBMITTED"
echo "  Votos falhados: $VOTES_FAILED"
echo "  Taxa de submissão: $(echo "scale=2; $VOTES_SUBMITTED / $SUBMIT_DURATION" | bc) votos/s"
echo ""
echo "Blockchain:"
echo "  Blocos finalizados: $BLOCKS_FINALIZED"
echo "  Votos finalizados: $VOTES_FINALIZED"
echo "  Média votos/bloco: $AVG_VOTES_PER_BLOCK"
echo ""
echo "Performance:"
echo "  Throughput: $THROUGHPUT votos/s"
echo "  Latência end-to-end: ${TOTAL_DURATION}s"
echo "  Perda de votos: $VOTE_LOSS ($((VOTE_LOSS * 100 / VOTES_SUBMITTED))%)"
echo ""

# Vote distribution
echo "Distribuição por candidato:"
for candidate in "${CANDIDATES[@]}"; do
    COUNT=$(grep -l "choice=$candidate" ../logs/vote_*.log 2>/dev/null | wc -l | tr -d ' ')
    echo "  $candidate: $COUNT votos"
done

echo ""

# Assessment
if [ $VOTE_LOSS -eq 0 ]; then
    echo "✅ EXCELENTE: Nenhum voto perdido!"
elif [ $VOTE_LOSS -lt 3 ]; then
    echo "✅ BOM: Perda mínima de votos"
elif [ $VOTE_LOSS -lt 5 ]; then
    echo "⚠️  ACEITÁVEL: Alguma perda de votos"
else
    echo "❌ RUIM: Perda significativa de votos"
fi

echo ""
echo "Logs salvos em: simulation/logs/"
echo "Blockchain salva em: simulation/data/"
echo ""

# Cleanup
echo "Parando nós..."
pkill -9 -f 'bin/bootstrap' 2>/dev/null || true
pkill -9 -f 'bin/node' 2>/dev/null || true

echo "✓ Teste 3 concluído!"
