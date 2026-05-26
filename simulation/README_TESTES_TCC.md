# Guia Rápido - Execução dos Testes do TCC

## 🚀 Execução Rápida

### Opção 1: Executar Todos os Testes (Recomendado)

```bash
cd simulation/scripts
./run_all_tcc_tests.sh
```

**Tempo total**: ~10 minutos  
**Resultado**: Relatório consolidado em `simulation/results_tcc/`

---

### Opção 2: Executar Testes Individuais

```bash
cd simulation/scripts

# 1. Build (primeira vez)
./build.sh

# 2. Gerar votantes (primeira vez)
./generate_voters.sh 100
./update_eligibility.sh

# 3. Executar testes individuais
./quick_test.sh                          # Teste 1: Validação Funcional
./test_validation_failure.sh             # Teste 2: Segurança
./stress_test_advanced.sh 50 0 30        # Teste 3: Performance
./progressive_load_test.sh               # Teste 4: Escalabilidade
./byzantine_failure_test.sh crash        # Teste 5: Tolerância a Falhas

# 4. Análise de blocos
go run analyze_blocks.go
```

---

## 📊 Resultados

Após a execução, os resultados estarão em:

```
simulation/
├── results_tcc/
│   ├── report_YYYYMMDD_HHMMSS.txt           # Relatório consolidado
│   ├── test1_YYYYMMDD_HHMMSS.log            # Log do teste 1
│   ├── test2_YYYYMMDD_HHMMSS.log            # Log do teste 2
│   ├── test3_YYYYMMDD_HHMMSS.log            # Log do teste 3
│   ├── test4_YYYYMMDD_HHMMSS.log            # Log do teste 4
│   ├── test5_YYYYMMDD_HHMMSS.log            # Log do teste 5
│   ├── blockchain_analysis_YYYYMMDD.txt     # Análise da blockchain
│   ├── test1_data_YYYYMMDD/                 # Blockchain do teste 1
│   ├── test2_data_YYYYMMDD/                 # Blockchain do teste 2
│   ├── test3_data_YYYYMMDD/                 # Blockchain do teste 3
│   ├── test4_data_YYYYMMDD/                 # Blockchain do teste 4
│   └── test5_data_YYYYMMDD/                 # Blockchain do teste 5
└── logs/
    ├── bootstrap.log                         # Logs do último teste
    ├── node1.log
    ├── node2.log
    └── node3.log
```

---

## 📝 Extração de Dados para o TCC

### Métricas Principais

Para cada teste, extrair do relatório consolidado:

**Teste 1 - Validação Funcional**:
- Votos finalizados / Votos submetidos
- Blocos finalizados
- Validação de round-robin

**Teste 2 - Segurança**:
- Primeiro voto: aceito (sim/não)
- Segundo voto: rejeitado (sim/não)
- Mensagem de erro

**Teste 3 - Performance**:
- Throughput (votos/segundo)
- Latência end-to-end
- Perda de votos (%)
- Votos por bloco (média)

**Teste 4 - Escalabilidade**:
- Throughput por nível de carga
- Ponto de saturação
- Perda de votos por nível

**Teste 5 - Tolerância a Falhas**:
- Blocos finalizados durante falha
- Votos processados durante falha
- Disponibilidade (%)

### Gráficos a Gerar

1. **Teste 3**: Throughput ao longo do tempo
2. **Teste 3**: Histograma de votos por bloco
3. **Teste 4**: Throughput vs Carga (linha)
4. **Teste 4**: Perda de votos vs Carga (linha)
5. **Teste 5**: Timeline da falha

---

## 🔧 Troubleshooting

### Erro: "Build failed"
```bash
# Verificar instalação do Go
go version

# Reinstalar dependências
cd ../..
go mod tidy
go mod download
```

### Erro: "Port already in use"
```bash
# Matar processos antigos
pkill -9 -f 'bin/bootstrap'
pkill -9 -f 'bin/node'

# Verificar portas
lsof -i :4000
lsof -i :4001
lsof -i :4002
lsof -i :4003
```

### Erro: "No voters found"
```bash
cd simulation/scripts
./generate_voters.sh 100
./update_eligibility.sh
```

### Logs não aparecem
```bash
# Verificar se diretório existe
ls -la ../logs/

# Criar se necessário
mkdir -p ../logs
```

---

## 📚 Documentação Completa

Para detalhes completos sobre cada teste, consulte:
- `simulation/PLANO_TESTES_TCC.md` - Plano detalhado de testes
- `GUIA_ESCRITA_TCC.md` - Guia de escrita do TCC
- `TESTING_METHODOLOGY.md` - Metodologia de testes

---

## ✅ Checklist Pré-Execução

Antes de executar os testes para coleta final de dados:

- [ ] Go instalado e funcionando (`go version`)
- [ ] Binários compilados (`./build.sh`)
- [ ] Votantes gerados (`./generate_voters.sh 100`)
- [ ] Nenhum processo antigo rodando (`pkill -f bin/`)
- [ ] Diretórios limpos (`rm -rf ../data/* ../logs/*`)
- [ ] Espaço em disco suficiente (>500MB)

---

## 🎯 Para o TCC

Após executar os testes:

1. **Revisar** o relatório consolidado
2. **Extrair** métricas para tabelas
3. **Gerar** gráficos a partir dos dados
4. **Analisar** logs para insights
5. **Escrever** Capítulo 5 usando os resultados

Boa sorte! 🚀
