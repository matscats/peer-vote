#!/bin/bash

# Script para executar todos os testes do TCC em sequência
# Salva resultados em simulation/results_tcc/

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
RESULTS_DIR="../results_tcc"
mkdir -p "$RESULTS_DIR"

echo "=========================================="
echo "Executando Todos os Testes do TCC"
echo "=========================================="
echo ""
echo "Timestamp: $(date)"
echo "Resultados em: $RESULTS_DIR"
echo ""

# Build first
if [ ! -f "../bin/node" ]; then
    echo "Compilando binários..."
    ./build.sh
    echo ""
fi

# Generate voters
if [ $(ls -1 ../keys/voters/*.key 2>/dev/null | wc -l) -lt 100 ]; then
    echo "Gerando 100 votantes..."
    ./generate_voters.sh 100
    ./update_eligibility.sh
    echo ""
fi

START_TIME=$(date +%s)

# Test 1
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Teste 1/5: Validação Funcional"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
./tcc_test1_funcional.sh 2>&1 | tee "$RESULTS_DIR/test1_${TIMESTAMP}.log"
cp -r ../data "$RESULTS_DIR/test1_data_${TIMESTAMP}" 2>/dev/null || true
echo ""
sleep 3

# Test 2
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Teste 2/5: Segurança"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
./tcc_test2_seguranca.sh 2>&1 | tee "$RESULTS_DIR/test2_${TIMESTAMP}.log"
cp -r ../data "$RESULTS_DIR/test2_data_${TIMESTAMP}" 2>/dev/null || true
echo ""
sleep 3

# Test 3
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Teste 3/5: Performance"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
./tcc_test3_performance.sh 2>&1 | tee "$RESULTS_DIR/test3_${TIMESTAMP}.log"
cp -r ../data "$RESULTS_DIR/test3_data_${TIMESTAMP}" 2>/dev/null || true
echo ""
sleep 3

# Test 4
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Teste 4/5: Escalabilidade"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
./tcc_test4_escalabilidade.sh 2>&1 | tee "$RESULTS_DIR/test4_${TIMESTAMP}.log"
cp -r ../data "$RESULTS_DIR/test4_data_${TIMESTAMP}" 2>/dev/null || true
cp ../logs/escalabilidade_results.txt "$RESULTS_DIR/test4_escalabilidade_${TIMESTAMP}.txt" 2>/dev/null || true
echo ""
sleep 3

# Test 5
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Teste 5/5: Tolerância a Falhas"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
./tcc_test5_tolerancia.sh 2>&1 | tee "$RESULTS_DIR/test5_${TIMESTAMP}.log"
cp -r ../data "$RESULTS_DIR/test5_data_${TIMESTAMP}" 2>/dev/null || true
echo ""

END_TIME=$(date +%s)
TOTAL_DURATION=$((END_TIME - START_TIME))

echo "=========================================="
echo "Todos os Testes Concluídos!"
echo "=========================================="
echo ""
echo "Tempo total: ${TOTAL_DURATION}s ($(($TOTAL_DURATION / 60))m $(($TOTAL_DURATION % 60))s)"
echo ""
echo "Resultados salvos em: $RESULTS_DIR"
echo "  - test1_${TIMESTAMP}.log (Validação Funcional)"
echo "  - test2_${TIMESTAMP}.log (Segurança)"
echo "  - test3_${TIMESTAMP}.log (Performance)"
echo "  - test4_${TIMESTAMP}.log (Escalabilidade)"
echo "  - test5_${TIMESTAMP}.log (Tolerância a Falhas)"
echo ""
echo "Blockchain de cada teste salva em:"
echo "  - test[1-5]_data_${TIMESTAMP}/"
echo ""
echo "Próximo passo: Analisar resultados e escrever Capítulo 5 do TCC"
echo ""
