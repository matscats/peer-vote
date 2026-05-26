#!/usr/bin/env bash

# TCC Test 4: Escalabilidade - Votos por Bloco
# Objetivo: Medir quantos votos conseguem ser processados sob diferentes cargas

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

# Cargas progressivas para testar
LOAD_LEVELS=(10 25 50 75 100 150 200)
# Tempo fixo de espera para processamento
WAIT_TIME=10

echo "=========================================="
echo "TCC - Teste 4: Escalabilidade"
echo "=========================================="
echo ""
echo "Objetivo: Medir capacidade de processamento"
echo "Níveis de carga: ${LOAD_LEVELS[@]}"
echo "Tempo de medição: ${WAIT_TIME}s"
echo "Intervalo de blocos: 5s"
echo ""

# Clean up
echo "Limpando dados antigos..."
pkill -9 -f 'bin/bootstrap' 2>/dev/null || true
pkill -9 -f 'bin/node' 2>/dev/null || true
sleep 2
rm -rf data/* logs/*
mkdir -p data logs

# Check voters
MAX_VOTERS=200
VOTER_COUNT=$(ls -1 keys/voters/*.key 2>/dev/null | wc -l | tr -d ' ')
if [ "$VOTER_COUNT" -lt "$MAX_VOTERS" ]; then
    echo "Gerando $MAX_VOTERS votantes..."
    (cd scripts && ./generate_voters.sh $MAX_VOTERS)
    (cd scripts && ./update_eligibility.sh)
fi

echo ""
echo "Iniciando rede..."

# Start bootstrap
./bin/bootstrap -port 4000 > logs/bootstrap.log 2>&1 &
sleep 3

BOOTSTRAP_ADDR=$(grep "/ip4/127.0.0.1/tcp/4000/p2p/" logs/bootstrap.log | head -1 | sed 's/.*\(\/ip4\/127\.0\.0\.1\/tcp\/4000\/p2p\/[^ ]*\).*/\1/')

# Update configs
for config in configs/config{1,2,3}.json; do
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
    ./bin/node -config configs/config${i}.json > logs/node${i}.log 2>&1 &
    sleep 2
done

echo "✓ Aguardando estabilização (10s)..."
sleep 10

# Get peer IDs
NODE1_ID=$(grep "P2P Host created with ID:" logs/node1.log | awk '{print $NF}')
NODE2_ID=$(grep "P2P Host created with ID:" logs/node2.log | awk '{print $NF}')
NODE3_ID=$(grep "P2P Host created with ID:" logs/node3.log | awk '{print $NF}')

TARGET_PEERS=(
    "/ip4/127.0.0.1/tcp/4001/p2p/${NODE1_ID}"
    "/ip4/127.0.0.1/tcp/4002/p2p/${NODE2_ID}"
    "/ip4/127.0.0.1/tcp/4003/p2p/${NODE3_ID}"
)

echo ""
echo "=========================================="
echo "Executando Testes de Escalabilidade"
echo "=========================================="
echo ""

CANDIDATES=("candidate-a" "candidate-b" "candidate-c")

# Results file
RESULTS_FILE="logs/escalabilidade_results.txt"
echo "Resultados do Teste de Escalabilidade" > $RESULTS_FILE
echo "=====================================" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE
echo "Objetivo: Medir capacidade de processamento sob diferentes cargas" >> $RESULTS_FILE
echo "Intervalo de blocos: 5s | Tempo de medição: ${WAIT_TIME}s" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE

# Run tests at each load level
for NUM_VOTERS in "${LOAD_LEVELS[@]}"; do
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Teste: $NUM_VOTERS votos simultâneos"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    # Clean state between tests
    echo "  Reiniciando rede..."
    pkill -9 -f 'bin/bootstrap' 2>/dev/null || true
    pkill -9 -f 'bin/node' 2>/dev/null || true
    sleep 2
    
    rm -rf data/* logs/*
    mkdir -p data logs
    
    # Restart bootstrap
    ./bin/bootstrap -port 4000 > logs/bootstrap.log 2>&1 &
    sleep 3
    
    BOOTSTRAP_ADDR=$(grep "/ip4/127.0.0.1/tcp/4000/p2p/" logs/bootstrap.log | head -1 | sed 's/.*\(\/ip4\/127\.0\.0\.1\/tcp\/4000\/p2p\/[^ ]*\).*/\1/')
    
    # Update configs
    for config in configs/config{1,2,3}.json; do
        if command -v jq &> /dev/null; then
            jq ".bootstrap_peers = [\"$BOOTSTRAP_ADDR\"]" $config > ${config}.tmp && mv ${config}.tmp $config
        else
            sed -i.bak "s|\"bootstrap_peers\": \[.*\]|\"bootstrap_peers\": [\"$BOOTSTRAP_ADDR\"]|" $config
            rm -f ${config}.bak
        fi
    done
    
    # Restart nodes
    for i in 1 2 3; do
        ./bin/node -config configs/config${i}.json > logs/node${i}.log 2>&1 &
        sleep 2
    done
    
    sleep 5
    
    # Get peer IDs
    NODE1_ID=$(grep "P2P Host created with ID:" logs/node1.log | awk '{print $NF}')
    NODE2_ID=$(grep "P2P Host created with ID:" logs/node2.log | awk '{print $NF}')
    NODE3_ID=$(grep "P2P Host created with ID:" logs/node3.log | awk '{print $NF}')
    
    TARGET_PEERS=(
        "/ip4/127.0.0.1/tcp/4001/p2p/${NODE1_ID}"
        "/ip4/127.0.0.1/tcp/4002/p2p/${NODE2_ID}"
        "/ip4/127.0.0.1/tcp/4003/p2p/${NODE3_ID}"
    )
    
    # Submit votes
    echo "  Submetendo $NUM_VOTERS votos..."
    VOTE_COUNT=0
    for key_file in keys/voters/*.key; do
        if [ $VOTE_COUNT -ge $NUM_VOTERS ]; then
            break
        fi
        
        VOTER_ID=$(basename "$key_file" .key)
        CANDIDATE=${CANDIDATES[$((RANDOM % 3))]}
        TARGET_INDEX=$((VOTE_COUNT % 3))
        TARGET_PEER=${TARGET_PEERS[$TARGET_INDEX]}
        
        ./bin/submit-vote \
            -voter "$VOTER_ID" \
            -choice "$CANDIDATE" \
            -key "keys/voters/${VOTER_ID}.key" \
            -peer "$TARGET_PEER" > logs/vote_${VOTER_ID}.log 2>&1 &
        
        VOTE_COUNT=$((VOTE_COUNT + 1))
    done
    
    echo "  Aguardando ${WAIT_TIME}s para processamento..."
    sleep $WAIT_TIME
    
    # Count results
    VOTES_SUBMITTED=$(grep -l "Vote submitted successfully" logs/vote_*.log 2>/dev/null | wc -l | tr -d ' ')
    
    # Get total votes finalized
    VOTES_FINALIZED=$(grep "finalized successfully" logs/node1.log 2>/dev/null | tail -1 | grep -o "voted count: [0-9]*" | awk '{print $3}')
    if [ -z "$VOTES_FINALIZED" ]; then
        VOTES_FINALIZED=0
    fi
    
    # Count blocks created
    BLOCKS_CREATED=$(grep "finalized successfully" logs/node1.log 2>/dev/null | wc -l | tr -d ' ')
    
    if [ $VOTES_SUBMITTED -gt 0 ]; then
        SUCCESS_RATE=$(echo "scale=2; ($VOTES_FINALIZED / $VOTES_SUBMITTED) * 100" | bc)
    else
        SUCCESS_RATE="0.00"
    fi
    
    if [ $BLOCKS_CREATED -gt 0 ]; then
        AVG_VOTES_PER_BLOCK=$(echo "scale=2; $VOTES_FINALIZED / $BLOCKS_CREATED" | bc)
    else
        AVG_VOTES_PER_BLOCK="0.00"
    fi
    
    # Display results
    echo ""
    echo "  Resultados:"
    echo "    Votos submetidos:     $VOTES_SUBMITTED"
    echo "    Votos finalizados:    $VOTES_FINALIZED"
    echo "    Taxa de sucesso:      ${SUCCESS_RATE}%"
    echo "    Blocos criados:       $BLOCKS_CREATED"
    echo "    Votos/bloco (média):  $AVG_VOTES_PER_BLOCK"
    echo ""
    
    # Save to file
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" >> $RESULTS_FILE
    echo "Carga: $NUM_VOTERS votos simultâneos" >> $RESULTS_FILE
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" >> $RESULTS_FILE
    echo "  Votos submetidos:     $VOTES_SUBMITTED" >> $RESULTS_FILE
    echo "  Votos finalizados:    $VOTES_FINALIZED" >> $RESULTS_FILE
    echo "  Taxa de sucesso:      ${SUCCESS_RATE}%" >> $RESULTS_FILE
    echo "  Blocos criados:       $BLOCKS_CREATED" >> $RESULTS_FILE
    echo "  Votos/bloco (média):  $AVG_VOTES_PER_BLOCK" >> $RESULTS_FILE
    echo "" >> $RESULTS_FILE
    
    # Clean up vote logs
    rm -f logs/vote_*.log
    
    sleep 2
done

# Stop nodes
pkill -9 -f 'bin/bootstrap' 2>/dev/null || true
pkill -9 -f 'bin/node' 2>/dev/null || true

echo ""
echo "=========================================="
echo "Resumo Final"
echo "=========================================="
echo ""

cat $RESULTS_FILE

echo ""
echo "Resultados salvos em: simulation/logs/escalabilidade_results.txt"
echo ""

echo "✓ Teste 4 concluído!"
