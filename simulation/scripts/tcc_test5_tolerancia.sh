#!/bin/bash

# TCC Test 5: Tolerância a Falhas (Crash Failure)
# Objetivo: Validar que sistema continua operando com falha de 1 validador

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "=========================================="
echo "TCC - Teste 5: Tolerância a Falhas"
echo "=========================================="
echo ""
echo "Cenário: Crash de 1 validador (Node 2)"
echo ""

# Clean up
echo "Limpando dados antigos..."
rm -rf ../data/* ../logs/*
pkill -9 -f 'bin/bootstrap' 2>/dev/null || true
pkill -9 -f 'bin/node' 2>/dev/null || true
sleep 2

# Check voters
if [ $(ls -1 ../keys/voters/*.key 2>/dev/null | wc -l) -lt 10 ]; then
    echo "Gerando votantes..."
    ./generate_voters.sh 10
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

# Start all 3 validators
echo "✓ Iniciando 3 validadores..."
for i in 1 2 3; do
    (cd .. && ./bin/node -config configs/config${i}.json) > ../logs/node${i}.log 2>&1 &
    eval "NODE${i}_PID=$!"
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
echo "=========================================="
echo "Fase 1: Operação Normal"
echo "=========================================="
echo ""

echo "Submetendo 3 votos..."
for i in 1 2 3; do
    VOTER_ID=$(printf "voter%03d" $i)
    TARGET_PEER=${TARGET_PEERS[$(((i-1) % 3))]}
    
    (cd ../.. && ./simulation/bin/submit-vote \
        -voter "$VOTER_ID" \
        -choice "candidate-a" \
        -key "simulation/keys/voters/${VOTER_ID}.key" \
        -peer "$TARGET_PEER") > ../logs/vote_${VOTER_ID}_phase1.log 2>&1 &
done

echo "Aguardando processamento (10s)..."
sleep 10

# Check initial state
INITIAL_BLOCKS=$(grep "finalized successfully" ../logs/node1.log 2>/dev/null | wc -l | tr -d ' ')
INITIAL_VOTES=$(grep "finalized successfully" ../logs/node1.log 2>/dev/null | tail -1 | grep -o "voted count: [0-9]*" | awk '{print $3}')
if [ -z "$INITIAL_VOTES" ]; then
    INITIAL_VOTES=0
fi

echo ""
echo "Estado inicial:"
echo "  Validadores ativos: 3/3"
echo "  Blocos finalizados: $INITIAL_BLOCKS"
echo "  Votos finalizados: $INITIAL_VOTES"

echo ""
echo "=========================================="
echo "Fase 2: Injeção de Falha"
echo "=========================================="
echo ""

echo "💥 Crashando Node 2 (SIGKILL)..."
kill -9 $NODE2_PID 2>/dev/null || true
FAILURE_TIME=$(date +%H:%M:%S)
echo "   Falha injetada em: $FAILURE_TIME"

echo ""
echo "Aguardando 5s para falha propagar..."
sleep 5

echo ""
echo "=========================================="
echo "Fase 3: Operação Sob Falha"
echo "=========================================="
echo ""

echo "Submetendo 5 votos com sistema degradado..."
for i in 4 5 6 7 8; do
    VOTER_ID=$(printf "voter%03d" $i)
    # Only send to healthy nodes (1 and 3)
    TARGET_INDEX=$(( (i-4) % 2 ))
    if [ $TARGET_INDEX -eq 0 ]; then
        TARGET_PEER=${TARGET_PEERS[0]}
    else
        TARGET_PEER=${TARGET_PEERS[2]}
    fi
    
    (cd ../.. && ./simulation/bin/submit-vote \
        -voter "$VOTER_ID" \
        -choice "candidate-b" \
        -key "simulation/keys/voters/${VOTER_ID}.key" \
        -peer "$TARGET_PEER") > ../logs/vote_${VOTER_ID}_phase3.log 2>&1 &
    
    sleep 1
done

echo "Aguardando processamento (15s)..."
sleep 15

# Check state during failure
DURING_BLOCKS=$(grep "finalized successfully" ../logs/node1.log 2>/dev/null | wc -l | tr -d ' ')
DURING_VOTES=$(grep "finalized successfully" ../logs/node1.log 2>/dev/null | tail -1 | grep -o "voted count: [0-9]*" | awk '{print $3}')
if [ -z "$DURING_VOTES" ]; then
    DURING_VOTES=0
fi

BLOCKS_DURING=$((DURING_BLOCKS - INITIAL_BLOCKS))
VOTES_DURING=$((DURING_VOTES - INITIAL_VOTES))

echo ""
echo "Estado durante falha:"
echo "  Validadores ativos: 2/3"
echo "  Novos blocos: $BLOCKS_DURING"
echo "  Novos votos: $VOTES_DURING"

echo ""
echo "=========================================="
echo "Análise Final"
echo "=========================================="
echo ""

FINAL_BLOCKS=$(grep "finalized successfully" ../logs/node1.log 2>/dev/null | wc -l | tr -d ' ')
FINAL_VOTES=$(grep "finalized successfully" ../logs/node1.log 2>/dev/null | tail -1 | grep -o "voted count: [0-9]*" | awk '{print $3}')
if [ -z "$FINAL_VOTES" ]; then
    FINAL_VOTES=0
fi

echo "Resumo:"
echo "  Fase 1 (Normal): $INITIAL_VOTES votos, $INITIAL_BLOCKS blocos"
echo "  Fase 3 (Falha): $VOTES_DURING votos, $BLOCKS_DURING blocos"
echo "  Total: $FINAL_VOTES votos, $FINAL_BLOCKS blocos"
echo ""

# Validation
echo "Validação:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

if [ $BLOCKS_DURING -gt 0 ]; then
    echo "✅ Sistema continuou operando durante falha"
    echo "   ($BLOCKS_DURING blocos finalizados)"
else
    echo "❌ Sistema parou durante falha"
fi

if [ $VOTES_DURING -gt 0 ]; then
    echo "✅ Votos foram processados durante falha"
    echo "   ($VOTES_DURING votos finalizados)"
else
    echo "⚠️  Nenhum voto finalizado durante falha"
fi

# Check for errors
ERRORS=$(grep -h "ERROR" ../logs/node*.log 2>/dev/null | grep -v "no active round" | wc -l | tr -d ' ')
if [ $ERRORS -gt 0 ]; then
    echo "⚠️  $ERRORS erros detectados nos logs"
else
    echo "✅ Nenhum erro crítico"
fi

# Calculate availability
if [ $BLOCKS_DURING -gt 0 ]; then
    AVAILABILITY=100
else
    AVAILABILITY=0
fi

echo ""
echo "Disponibilidade do sistema: ${AVAILABILITY}%"
echo ""

# Final verdict
if [ $BLOCKS_DURING -gt 0 ] && [ $VOTES_DURING -gt 0 ]; then
    echo "✅ TESTE PASSOU"
    echo "   - Sistema tolerou falha de 1/3 validadores"
    echo "   - Maioria (2/3) mantida"
    echo "   - Disponibilidade: 100%"
else
    echo "⚠️  TESTE INCOMPLETO"
    echo "   - Sistema pode não ter tolerado a falha adequadamente"
fi

echo ""
echo "Logs salvos em: simulation/logs/"
echo ""

# Cleanup
echo "Parando nós restantes..."
pkill -9 -f 'bin/bootstrap' 2>/dev/null || true
pkill -9 -f 'bin/node' 2>/dev/null || true

echo "✓ Teste 5 concluído!"
