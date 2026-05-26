#!/bin/bash

# TCC Test 2: Validação de Segurança (Double Voting)
# Objetivo: Verificar que votos duplicados são rejeitados

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "=========================================="
echo "TCC - Teste 2: Segurança (Double Voting)"
echo "=========================================="
echo ""

# Clean up
echo "Limpando dados antigos..."
rm -rf ../data/* ../logs/*
pkill -9 -f 'bin/bootstrap' 2>/dev/null || true
pkill -9 -f 'bin/node' 2>/dev/null || true
sleep 2

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

echo "✓ Aguardando estabilização (15s)..."
sleep 15

# Get peer ID
NODE1_ID=$(grep "P2P Host created with ID:" ../logs/node1.log | awk '{print $NF}')
TARGET_PEER="/ip4/127.0.0.1/tcp/4001/p2p/${NODE1_ID}"

echo ""
echo "=========================================="
echo "Teste de Double Voting"
echo "=========================================="
echo ""

# First vote
echo "1. Submetendo primeiro voto de voter001..."
(cd ../.. && ./simulation/bin/submit-vote \
    -voter "voter001" \
    -choice "candidate-a" \
    -key "simulation/keys/voters/voter001.key" \
    -peer "$TARGET_PEER") > ../logs/vote_first.log 2>&1

sleep 2

if grep -q "Vote submitted successfully" ../logs/vote_first.log; then
    echo "   ✅ Primeiro voto aceito"
else
    echo "   ❌ Primeiro voto falhou"
    cat ../logs/vote_first.log
fi

# Wait for finalization
echo ""
echo "2. Aguardando finalização do primeiro voto (15s)..."
sleep 15

# Check if first vote was finalized
VOTES_FINALIZED=$(grep "finalized successfully" ../logs/node1.log 2>/dev/null | tail -1 | grep -o "voted count: [0-9]*" | awk '{print $3}')
if [ -z "$VOTES_FINALIZED" ]; then
    VOTES_FINALIZED=0
fi

echo "   Votos finalizados: $VOTES_FINALIZED"

# Second vote (duplicate)
echo ""
echo "3. Tentando submeter segundo voto de voter001 (duplicado)..."
(cd ../.. && ./simulation/bin/submit-vote \
    -voter "voter001" \
    -choice "candidate-b" \
    -key "simulation/keys/voters/voter001.key" \
    -peer "$TARGET_PEER") > ../logs/vote_second.log 2>&1

sleep 2

# Check if second vote was rejected
if grep -q "Vote submitted successfully" ../logs/vote_second.log; then
    echo "   ❌ Segundo voto foi aceito (ERRO!)"
    SECOND_ACCEPTED=true
else
    echo "   ✅ Segundo voto foi rejeitado"
    SECOND_ACCEPTED=false
fi

# Wait a bit more
sleep 10

echo ""
echo "=========================================="
echo "Resultados"
echo "=========================================="

# Final count
FINAL_VOTES=$(grep "finalized successfully" ../logs/node1.log 2>/dev/null | tail -1 | grep -o "voted count: [0-9]*" | awk '{print $3}')
if [ -z "$FINAL_VOTES" ]; then
    FINAL_VOTES=0
fi

echo "Votos finalizados na blockchain: $FINAL_VOTES"
echo ""

# Check logs for rejection
echo "Verificando logs de rejeição:"
if grep -q "already voted\|already in mempool\|duplicate" ../logs/node*.log 2>/dev/null; then
    echo "  ✅ Sistema detectou duplicação"
    grep "already voted\|already in mempool\|duplicate" ../logs/node*.log 2>/dev/null | head -3 | sed 's/^/  /'
else
    echo "  ⚠️  Nenhuma mensagem de rejeição encontrada nos logs"
fi

echo ""

# Final verdict
if [ "$FINAL_VOTES" -eq 1 ] && [ "$SECOND_ACCEPTED" = false ]; then
    echo "✅ TESTE PASSOU"
    echo "   - Primeiro voto aceito e finalizado"
    echo "   - Segundo voto rejeitado"
    echo "   - Blockchain contém apenas 1 voto"
else
    echo "⚠️  TESTE FALHOU"
    echo "   - Votos na blockchain: $FINAL_VOTES (esperado: 1)"
    echo "   - Segundo voto aceito: $SECOND_ACCEPTED (esperado: false)"
fi

echo ""
echo "Logs salvos em: simulation/logs/"
echo ""

# Cleanup
echo "Parando nós..."
pkill -9 -f 'bin/bootstrap' 2>/dev/null || true
pkill -9 -f 'bin/node' 2>/dev/null || true

echo "✓ Teste 2 concluído!"
