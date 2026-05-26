# Fluxo de Execução dos Testes

## 🔄 Visão Geral do Processo

```
┌─────────────────────────────────────────────────────────────┐
│                    INÍCIO DOS TESTES                        │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│              PRÉ-REQUISITOS                                 │
│  • Build dos binários (bootstrap, node, submit-vote)        │
│  • Geração de 100 votantes                                  │
│  • Limpeza de dados antigos                                 │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│  TESTE 1: Validação Funcional (~1 min)                     │
│  ┌───────────────────────────────────────────────────────┐ │
│  │ 1. Iniciar 3 validadores                              │ │
│  │ 2. Submeter 5 votos                                   │ │
│  │ 3. Verificar consenso PoA                             │ │
│  │ 4. Validar round-robin                                │ │
│  │ ✅ Resultado: 5/5 votos finalizados                   │ │
│  └───────────────────────────────────────────────────────┘ │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│  TESTE 2: Segurança - Double Voting (~1 min)               │
│  ┌───────────────────────────────────────────────────────┐ │
│  │ 1. Submeter voto de voter001 → ✅ Aceito             │ │
│  │ 2. Aguardar finalização                               │ │
│  │ 3. Submeter 2º voto de voter001 → ❌ Rejeitado       │ │
│  │ 4. Verificar mensagem de erro                         │ │
│  │ ✅ Resultado: Duplicação detectada                    │ │
│  └───────────────────────────────────────────────────────┘ │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│  TESTE 3: Performance e Throughput (~1 min)                │
│  ┌───────────────────────────────────────────────────────┐ │
│  │ 1. Submeter 50 votos simultaneamente                  │ │
│  │ 2. Medir tempo de submissão                           │ │
│  │ 3. Aguardar 30s para processamento                    │ │
│  │ 4. Calcular métricas:                                 │ │
│  │    • Throughput (votos/s)                             │ │
│  │    • Latência end-to-end                              │ │
│  │    • Perda de votos                                   │ │
│  │    • Votos por bloco                                  │ │
│  │ ✅ Resultado: 1.5-2.5 v/s, perda < 5%                │ │
│  └───────────────────────────────────────────────────────┘ │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│  TESTE 4: Escalabilidade - Carga Progressiva (~4 min)      │
│  ┌───────────────────────────────────────────────────────┐ │
│  │ Nível 1: 10 votos  → Throughput: 0.4-0.5 v/s         │ │
│  │          ↓ (20s)                                       │ │
│  │ Nível 2: 25 votos  → Throughput: 0.8-1.0 v/s         │ │
│  │          ↓ (20s)                                       │ │
│  │ Nível 3: 50 votos  → Throughput: 1.5-2.0 v/s         │ │
│  │          ↓ (20s)                                       │ │
│  │ Nível 4: 100 votos → Throughput: 2.0-2.5 v/s         │ │
│  │          ↓ (20s)                                       │ │
│  │ ✅ Resultado: Saturação em ~50-75 votos              │ │
│  └───────────────────────────────────────────────────────┘ │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│  TESTE 5: Tolerância a Falhas - Crash (~1 min)             │
│  ┌───────────────────────────────────────────────────────┐ │
│  │ Fase 1: Operação Normal                               │ │
│  │   • 3 validadores ativos                              │ │
│  │   • Submeter 3 votos → ✅ Finalizados                │ │
│  │                                                        │ │
│  │ Fase 2: Injeção de Falha                              │ │
│  │   • 💥 Node 2 crasha (SIGKILL)                        │ │
│  │   • 2/3 validadores restantes                         │ │
│  │                                                        │ │
│  │ Fase 3: Operação Sob Falha                            │ │
│  │   • Submeter 5 votos                                  │ │
│  │   • Sistema continua operando                         │ │
│  │   • ✅ 5/5 votos finalizados                          │ │
│  │                                                        │ │
│  │ ✅ Resultado: 100% disponibilidade                    │ │
│  └───────────────────────────────────────────────────────┘ │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│  ANÁLISE COMPLEMENTAR: Integridade da Blockchain           │
│  ┌───────────────────────────────────────────────────────┐ │
│  │ • Verificar continuidade (hashes)                     │ │
│  │ • Detectar votos duplicados                           │ │
│  │ • Analisar distribuição de votos por bloco           │ │
│  │ • Validar assinaturas                                 │ │
│  │ ✅ Resultado: 0 duplicações, chain íntegra           │ │
│  └───────────────────────────────────────────────────────┘ │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│              GERAÇÃO DE RELATÓRIO                           │
│  • Relatório consolidado (report_TIMESTAMP.txt)             │
│  • Logs individuais (test[1-5]_TIMESTAMP.log)               │
│  • Blockchain persistida (test[1-5]_data_TIMESTAMP/)        │
│  • Análise de integridade (blockchain_analysis.txt)         │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                    FIM DOS TESTES                           │
│              Todos os dados coletados! ✅                   │
└─────────────────────────────────────────────────────────────┘
```

---

## 📊 Dados Gerados por Teste

### Teste 1: Validação Funcional
```
Dados:
├── Votos finalizados: 5
├── Blocos criados: 2-4
├── Líderes por altura: [Node1, Node2, Node3, Node1, ...]
└── Logs: Propagação P2P, consenso, finalização

Para TCC:
├── Tabela: Altura vs Líder esperado vs Líder real
└── Timeline: Eventos do consenso
```

### Teste 2: Segurança
```
Dados:
├── Primeiro voto: Aceito ✅
├── Segundo voto: Rejeitado ❌
├── Mensagem de erro: "already voted"
└── Blockchain: 1 voto de voter001

Para TCC:
├── Tabela: Tentativas vs Resultado
└── Logs: Mensagens de rejeição
```

### Teste 3: Performance
```
Dados:
├── Throughput: 1.5-2.5 votos/s
├── Latência: 25-35s
├── Perda: 0-2 votos (0-4%)
├── Votos/bloco: 2-5 (média)
└── Distribuição: Histograma

Para TCC:
├── Gráfico: Throughput ao longo do tempo
├── Gráfico: Histograma de votos/bloco
└── Tabela: Métricas de performance
```

### Teste 4: Escalabilidade
```
Dados por nível:
├── 10 votos:  0.4-0.5 v/s, 0% perda
├── 25 votos:  0.8-1.0 v/s, 0% perda
├── 50 votos:  1.5-2.0 v/s, 0-2% perda
└── 100 votos: 2.0-2.5 v/s, 2-10% perda

Para TCC:
├── Gráfico: Throughput vs Carga
├── Gráfico: Perda vs Carga
└── Análise: Ponto de saturação
```

### Teste 5: Tolerância a Falhas
```
Dados:
├── Antes: 3/3 validadores, 3 votos
├── Durante: 2/3 validadores, 5 votos
├── Disponibilidade: 100%
└── Blocos durante falha: > 0

Para TCC:
├── Tabela: Antes/Durante/Após
├── Timeline: Eventos da falha
└── Análise: Tolerância f = (n-1)/2
```

---

## 🎯 Mapeamento para Capítulo 5

```
Capítulo 5: RESULTADOS

5.1 Ambiente Experimental
    └── Configuração dos testes

5.2 Validação Funcional
    ├── 5.2.1 Consenso Básico ← Teste 1
    └── 5.2.2 Segurança ← Teste 2

5.3 Análise de Performance ← Teste 3
    ├── Throughput
    ├── Latência
    └── Batching

5.4 Testes de Escalabilidade ← Teste 4
    ├── Carga Progressiva
    └── Ponto de Saturação

5.5 Tolerância a Falhas ← Teste 5
    └── Crash Failure

5.6 Integridade da Blockchain ← Análise
    └── Verificação de Consistência

5.7 Discussão dos Resultados
    ├── Síntese
    ├── Comparação
    └── Limitações
```

---

## ⏱️ Timeline de Execução

```
00:00 ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
      │ Início - Pré-requisitos
      │
00:30 ├─ Teste 1: Validação Funcional
      │
01:30 ├─ Teste 2: Segurança
      │
02:30 ├─ Teste 3: Performance
      │
03:30 ├─ Teste 4: Escalabilidade (4 níveis)
      │   ├─ Nível 1: 10 votos
      │   ├─ Nível 2: 25 votos
      │   ├─ Nível 3: 50 votos
      │   └─ Nível 4: 100 votos
      │
07:30 ├─ Teste 5: Tolerância a Falhas
      │
08:30 ├─ Análise de Blockchain
      │
09:00 └─ Geração de Relatório
      │
10:00 ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
      FIM - Todos os dados coletados ✅
```

---

## 📁 Estrutura de Resultados

```
simulation/
└── results_tcc/
    ├── report_20260301_143022.txt              # Relatório consolidado
    │
    ├── test1_20260301_143022.log               # Logs individuais
    ├── test2_20260301_143022.log
    ├── test3_20260301_143022.log
    ├── test4_20260301_143022.log
    ├── test5_20260301_143022.log
    │
    ├── test1_data_20260301_143022/             # Blockchain de cada teste
    │   ├── data1/
    │   ├── data2/
    │   └── data3/
    ├── test2_data_20260301_143022/
    ├── test3_data_20260301_143022/
    ├── test4_data_20260301_143022/
    ├── test5_data_20260301_143022/
    │
    └── blockchain_analysis_20260301.txt        # Análise de integridade
```

---

## ✅ Checklist de Validação

Após execução, verificar:

- [ ] **Teste 1**: 5/5 votos finalizados
- [ ] **Teste 2**: 2º voto rejeitado
- [ ] **Teste 3**: Throughput > 1.5 v/s
- [ ] **Teste 4**: Dados de 4 níveis coletados
- [ ] **Teste 5**: Sistema continuou operando
- [ ] **Análise**: 0 duplicações detectadas
- [ ] **Relatório**: Gerado com sucesso
- [ ] **Logs**: Todos os 5 testes presentes
- [ ] **Blockchain**: Dados persistidos

---

**Pronto para executar!** 🚀

```bash
cd simulation/scripts
./run_all_tcc_tests.sh
```

Aguarde ~10 minutos e todos os dados estarão prontos para o TCC!
