# Plano de Testes para Validação Experimental do TCC

## 📋 Visão Geral

Este documento descreve a bateria de testes que será executada para validar o sistema de votação blockchain desenvolvido. Os testes cobrem aspectos funcionais, de segurança, performance, escalabilidade e tolerância a falhas.

**Objetivo**: Coletar dados experimentais para o Capítulo 5 (Resultados) do TCC.

**Tempo total estimado**: 8-10 minutos de execução + análise

---

## 🎯 Estrutura dos Testes

Os testes estão organizados em 5 categorias principais, cada uma mapeada para uma seção específica do Capítulo 5:

| Teste | Objetivo | Seção TCC | Tempo |
|-------|----------|-----------|-------|
| 1. Validação Funcional | Provar funcionamento correto | 5.2 | 1 min |
| 2. Segurança | Validar rejeição de votos duplicados | 5.2 | 1 min |
| 3. Performance | Medir throughput e latência | 5.3 | 1 min |
| 4. Escalabilidade | Identificar limites do sistema | 5.4 | 4 min |
| 5. Tolerância a Falhas | Provar tolerância bizantina | 5.5 | 1 min |

---

## 📝 Detalhamento dos Testes

### Teste 1: Validação Funcional Básica

**Arquivo**: `scripts/quick_test.sh`

**Objetivo**: Validar o ciclo completo de consenso com carga mínima

**Configuração**:
- 3 validadores (Node 1, Node 2, Node 3)
- 5 votos distribuídos entre os validadores
- Intervalo de bloco: 5 segundos
- Timeout de bloco vazio: 1 segundo

**Cenário de Execução**:
1. Inicializar rede (bootstrap + 3 validadores)
2. Aguardar estabilização (15 segundos)
3. Submeter 5 votos (1 para cada validador, distribuído)
4. Aguardar finalização (30 segundos)
5. Coletar métricas

**Métricas Coletadas**:
- Votos aceitos no mempool
- Blocos finalizados
- Altura da blockchain
- Peers conectados por nó
- Tempo de propagação de votos
- Distribuição de líderes (round-robin)

**Validações**:
- ✅ Todos os 5 votos foram finalizados
- ✅ Seleção de líder segue fórmula L(h) = v_{h mod 3}
- ✅ Blocos contêm assinaturas válidas
- ✅ Estado é consistente entre os 3 nós
- ✅ Nenhum erro crítico nos logs

**Resultados Esperados**:
- Taxa de sucesso: 100% (5/5 votos finalizados)
- Blocos finalizados: 2-4 blocos
- Latência: < 30 segundos

**Para o TCC (Seção 5.2.1)**:
- Tabela: Altura do bloco vs Líder esperado vs Líder real
- Gráfico: Timeline de eventos (voto → mempool → proposta → aprovação → finalização)
- Análise: Validação da fórmula de seleção de líder



---

### Teste 2: Validação de Segurança (Double Voting)

**Arquivo**: `scripts/test_validation_failure.sh`

**Objetivo**: Verificar que o sistema rejeita corretamente votos duplicados

**Configuração**:
- 3 validadores
- 1 votante tenta votar 2 vezes
- Mesmo voterID, mesma ou diferente escolha

**Cenário de Execução**:
1. Inicializar rede
2. Submeter primeiro voto de voter001 → deve ser aceito
3. Aguardar finalização do primeiro voto
4. Submeter segundo voto de voter001 → deve ser rejeitado
5. Verificar logs de rejeição

**Métricas Coletadas**:
- Primeiro voto: aceito no mempool (sim/não)
- Primeiro voto: finalizado na blockchain (sim/não)
- Segundo voto: rejeitado (sim/não)
- Mensagem de erro apropriada (sim/não)
- Estado final: apenas 1 voto de voter001 (sim/não)

**Validações**:
- ✅ Primeiro voto aceito e finalizado
- ✅ Segundo voto rejeitado com erro "already voted"
- ✅ Blockchain contém apenas 1 voto de voter001
- ✅ Estado marca voter001 como já votado
- ✅ Mempool não aceita segundo voto

**Resultados Esperados**:
- Primeiro voto: 100% sucesso
- Segundo voto: 100% rejeição
- Mensagem de erro: "vote already in mempool" ou "voter has already voted"

**Para o TCC (Seção 5.2.2)**:
- Tabela: Tentativa de voto vs Resultado (aceito/rejeitado)
- Logs: Mensagens de erro do sistema
- Análise: Validação das 4 camadas de proteção (mempool, state, block validation, finalization)

---

### Teste 3: Análise de Performance e Throughput

**Arquivo**: `scripts/stress_test_advanced.sh 50 0 30`

**Objetivo**: Medir capacidade de processamento do sistema sob carga moderada

**Configuração**:
- 3 validadores
- 50 votos submetidos o mais rápido possível
- Duração: 30 segundos de observação
- Sem rate limiting (máxima velocidade)

**Cenário de Execução**:
1. Inicializar rede
2. Submeter 50 votos simultaneamente (sem delay)
3. Registrar tempo de submissão
4. Aguardar 30 segundos para processamento
5. Coletar métricas finais

**Métricas Coletadas**:
- **Submissão**:
  - Votos submetidos com sucesso
  - Tempo de submissão total
  - Taxa de submissão (votos/segundo)
  
- **Processamento**:
  - Votos finalizados na blockchain
  - Blocos criados
  - Votos por bloco (média, min, max)
  
- **Performance**:
  - Throughput (votos finalizados / tempo total)
  - Latência end-to-end (primeiro voto → última finalização)
  - Perda de votos (votos submetidos - votos finalizados)
  
- **Rede**:
  - Peers conectados
  - Distribuição de carga entre validadores

**Validações**:
- ✅ Throughput > 1.5 votos/segundo
- ✅ Perda de votos < 5%
- ✅ Batching eficiente (média > 2 votos/bloco)
- ✅ Sistema não crasha sob carga

**Resultados Esperados**:
- Throughput: 1.5 - 2.5 votos/segundo
- Latência: 25-35 segundos
- Perda de votos: 0-2 votos (0-4%)
- Votos por bloco: 2-5 (média)

**Para o TCC (Seção 5.3)**:
- Gráfico: Throughput ao longo do tempo
- Gráfico: Distribuição de votos por bloco (histograma)
- Tabela: Métricas de performance (throughput, latência, perda)
- Análise: Comparação com throughput teórico (limitado por intervalo de bloco)

---

### Teste 4: Teste de Escalabilidade (Carga Progressiva)

**Arquivo**: `scripts/progressive_load_test.sh`

**Objetivo**: Identificar o ponto de saturação do sistema através de carga crescente

**Configuração**:
- 3 validadores (mesma rede mantida entre testes)
- Níveis de carga: 10, 25, 50, 100 votos
- Duração por nível: 20 segundos
- Mesma rede para todos os níveis (sem restart)

**Cenário de Execução**:
1. Inicializar rede uma única vez
2. Para cada nível de carga:
   - Registrar estado inicial (blocos, votos)
   - Submeter N votos
   - Aguardar 20 segundos
   - Registrar estado final
   - Calcular métricas delta
3. Gerar relatório comparativo

**Métricas Coletadas (por nível)**:
- Votos submetidos
- Votos finalizados
- Perda de votos (absoluta e %)
- Blocos criados
- Throughput (votos/segundo)
- Votos por bloco (média)

**Validações**:
- ✅ Até 50 votos: perda mínima (< 2%)
- ✅ 100 votos: perda aceitável (< 10%)
- ✅ Throughput satura em algum ponto
- ✅ Sistema não crasha em nenhum nível

**Resultados Esperados**:

| Nível | Throughput | Perda | Status |
|-------|------------|-------|--------|
| 10    | 0.4-0.5 v/s | 0% | Excelente |
| 25    | 0.8-1.0 v/s | 0% | Excelente |
| 50    | 1.5-2.0 v/s | 0-2% | Bom |
| 100   | 2.0-2.5 v/s | 2-10% | Aceitável |

**Para o TCC (Seção 5.4)**:
- Gráfico: Throughput vs Carga (linha)
- Gráfico: Perda de votos vs Carga (linha)
- Tabela: Métricas por nível de carga
- Análise: Identificação do ponto de saturação (~50-75 votos)
- Discussão: Causas da saturação (mempool, intervalo de bloco)

---

### Teste 5: Tolerância a Falhas (Crash Failure)

**Arquivo**: `scripts/byzantine_failure_test.sh crash`

**Objetivo**: Validar que o sistema continua operando com falha de 1 validador

**Configuração**:
- 3 validadores inicialmente
- 1 validador (Node 2) será crashado abruptamente
- Votos submetidos antes, durante e após a falha

**Cenário de Execução**:

**Fase 1: Operação Normal**
1. Inicializar 3 validadores
2. Submeter 3 votos
3. Verificar que sistema funciona normalmente
4. Registrar estado inicial (blocos, votos)

**Fase 2: Injeção de Falha**
5. Matar Node 2 abruptamente (SIGKILL)
6. Registrar timestamp da falha

**Fase 3: Operação Sob Falha**
7. Submeter 5 votos adicionais
8. Observar comportamento do sistema
9. Aguardar 15 segundos

**Fase 4: Análise**
10. Comparar estado antes/durante/após falha
11. Verificar logs de detecção de falha
12. Validar que sistema continuou operando

**Métricas Coletadas**:

| Métrica | Antes | Durante | Após |
|---------|-------|---------|------|
| Validadores ativos | 3/3 | 2/3 | 2/3 |
| Blocos finalizados | X | Y | Z |
| Votos processados | 3 | 5 | 8 |
| Disponibilidade | 100% | ? | ? |

**Validações**:
- ✅ Sistema detecta falha (timeout)
- ✅ Sistema continua com 2/3 validadores (maioria)
- ✅ Blocos continuam sendo finalizados
- ✅ Votos submetidos durante falha são processados
- ✅ Nenhum voto é perdido
- ✅ Disponibilidade = 100%

**Resultados Esperados**:
- Blocos finalizados durante falha: > 0
- Votos processados durante falha: 5/5 (100%)
- Tempo de detecção de falha: ~10 segundos (timeout)
- Disponibilidade: 100%

**Para o TCC (Seção 5.5)**:
- Tabela: Comparação antes/durante/após falha
- Timeline: Diagrama temporal da falha
- Logs: Detecção de timeout e avanço de round
- Análise: Validação de tolerância f = (n-1)/2 = 1
- Discussão: Propriedades de safety e liveness mantidas

---

## 📊 Análise Complementar: Integridade da Blockchain

**Arquivo**: `scripts/analyze_blocks.go`

**Objetivo**: Verificar integridade e consistência da blockchain persistida

**Execução**: Após qualquer teste que gere blocos

**Análises Realizadas**:
1. **Contagem**:
   - Total de blocos
   - Total de votos
   - Blocos vazios vs com votos

2. **Integridade**:
   - Detecção de votos duplicados
   - Verificação de continuidade (hashes)
   - Validação de assinaturas

3. **Distribuição**:
   - Votos por bloco (histograma)
   - Votos por candidato
   - Blocos por proposer

**Validações**:
- ✅ Zero votos duplicados
- ✅ Continuidade da chain (hash[i] = prevHash[i+1])
- ✅ Todas as assinaturas válidas
- ✅ Altura sequencial (0, 1, 2, 3...)

**Para o TCC (Seção 5.6)**:
- Histograma: Distribuição de votos por bloco
- Tabela: Estatísticas da blockchain
- Análise: Presença de blocos vazios (esperado devido a empty block wait)
- Verificação: Integridade criptográfica

---

## 🚀 Execução dos Testes

### Pré-requisitos

```bash
cd simulation/scripts

# 1. Build dos binários
./build.sh

# 2. Gerar votantes (se necessário)
./generate_voters.sh 100
./update_eligibility.sh
```

### Execução Individual

```bash
# Teste 1: Validação Funcional
./quick_test.sh

# Teste 2: Segurança
./test_validation_failure.sh

# Teste 3: Performance
./stress_test_advanced.sh 50 0 30

# Teste 4: Escalabilidade
./progressive_load_test.sh

# Teste 5: Tolerância a Falhas
./byzantine_failure_test.sh crash

# Análise de Blocos (após qualquer teste)
go run analyze_blocks.go
```

### Execução Completa (Script Master)

```bash
# Executa todos os testes e gera relatório consolidado
./run_all_tcc_tests.sh
```

---

## 📈 Coleta de Dados para o TCC

### Dados Brutos

Após cada teste, os seguintes arquivos estarão disponíveis:

```
simulation/
├── logs/
│   ├── bootstrap.log          # Logs do bootstrap node
│   ├── node1.log              # Logs do validador 1
│   ├── node2.log              # Logs do validador 2
│   ├── node3.log              # Logs do validador 3
│   ├── vote_*.log             # Logs de submissão de votos
│   └── progressive_load_results.txt  # Resultados do teste 4
└── data/
    ├── data1/                 # Blockchain do node 1
    ├── data2/                 # Blockchain do node 2
    └── data3/                 # Blockchain do node 3
```

### Extração de Métricas

Para cada teste, extrair dos logs:

```bash
# Votos no mempool
grep "added to mempool" logs/node1.log | wc -l

# Blocos finalizados
grep "finalized successfully" logs/node1.log | wc -l

# Votos finalizados
grep "finalized successfully" logs/node1.log | tail -1 | grep -o "voted count: [0-9]*"

# Peers conectados
grep "Connected peers:" logs/node1.log | tail -1

# Erros
grep "ERROR" logs/node*.log | wc -l
```

### Geração de Gráficos

Os seguintes gráficos devem ser gerados para o TCC:

1. **Teste 1**: Timeline de eventos
2. **Teste 3**: 
   - Throughput ao longo do tempo
   - Distribuição de votos por bloco (histograma)
3. **Teste 4**:
   - Throughput vs Carga (linha)
   - Perda de votos vs Carga (linha)
4. **Teste 5**: Timeline da falha
5. **Análise**: Histograma de votos por bloco

---

## ✅ Checklist de Validação

Antes de considerar os testes completos, verificar:

### Teste 1: Validação Funcional
- [ ] Todos os votos foram finalizados
- [ ] Round-robin funcionou corretamente
- [ ] Nenhum erro crítico nos logs
- [ ] Estado consistente entre nós

### Teste 2: Segurança
- [ ] Primeiro voto aceito
- [ ] Segundo voto rejeitado
- [ ] Mensagem de erro apropriada
- [ ] Blockchain contém apenas 1 voto

### Teste 3: Performance
- [ ] Throughput > 1.5 v/s
- [ ] Perda < 5%
- [ ] Batching eficiente
- [ ] Sistema estável

### Teste 4: Escalabilidade
- [ ] Dados coletados para todos os níveis
- [ ] Ponto de saturação identificado
- [ ] Degradação graceful (sem crash)
- [ ] Relatório gerado

### Teste 5: Tolerância a Falhas
- [ ] Sistema continuou operando
- [ ] Blocos finalizados durante falha
- [ ] Votos processados
- [ ] Disponibilidade 100%

### Análise de Blocos
- [ ] Zero duplicação
- [ ] Continuidade verificada
- [ ] Estatísticas geradas
- [ ] Histograma criado

---

## 📝 Notas Importantes

### Limitações do Ambiente de Teste

1. **Localhost**: Todos os nós executam na mesma máquina
   - Latência mínima (não realista)
   - Sem perda de pacotes
   - Recursos compartilhados

2. **Rede Confiável**: Sem simulação de falhas de rede
   - Sem jitter
   - Sem partições (exceto teste 5)
   - Sem congestionamento

3. **Número de Validadores**: Fixo em 3
   - Não testa escalabilidade de validadores
   - Apenas testa escalabilidade de votos

### Reprodutibilidade

Para garantir reprodutibilidade:
- Sempre limpar dados antes de cada teste (`rm -rf ../data/* ../logs/*`)
- Usar mesmas configurações (configs/*.json)
- Registrar versão do Go e sistema operacional
- Documentar hardware (CPU, RAM)

### Interpretação dos Resultados

- **Throughput**: Limitado principalmente pelo intervalo de bloco (5s)
- **Perda de votos**: Esperada em cargas muito altas (>100 votos)
- **Blocos vazios**: Esperados devido ao mecanismo de empty block wait
- **Latência**: Dominada pelo intervalo de bloco, não pela rede

---

## 🎯 Mapeamento para Capítulo 5 do TCC

### Seção 5.1: Ambiente Experimental
- Configuração dos validadores
- Parâmetros do protocolo
- Infraestrutura de teste

### Seção 5.2: Validação Funcional
- Teste 1: Consenso básico
- Teste 2: Segurança (double voting)

### Seção 5.3: Análise de Performance
- Teste 3: Throughput e latência

### Seção 5.4: Testes de Escalabilidade
- Teste 4: Carga progressiva

### Seção 5.5: Tolerância a Falhas
- Teste 5: Crash failure

### Seção 5.6: Integridade da Blockchain
- Análise de blocos

### Seção 5.7: Discussão dos Resultados
- Síntese de todos os testes
- Comparação com trabalhos relacionados
- Limitações identificadas

---

## 📚 Referências para Metodologia

Os testes foram inspirados em metodologias de:
- **Hyperledger Caliper**: Framework de benchmark para blockchain
- **Tendermint**: Testes de consenso BFT
- **Ethereum 2.0**: Testes de rede P2P
- **Bitcoin Core**: Testes de stress e regressão

---

**Última atualização**: Março 2026
**Autor**: Mateus Cavalcanti Alves Teixeira Silva
**Orientador**: [Nome do Orientador]
