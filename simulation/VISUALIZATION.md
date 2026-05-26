# Visualização da Rede Blockchain

Este documento explica como gerar e usar as visualizações da simulação para o TCC.

## 📊 O que é gerado?

A ferramenta de visualização analisa os logs da simulação e gera:

1. **visualization.html** - Página web interativa com:
   - Mapa visual da rede P2P (nós e conexões)
   - Estatísticas gerais do sistema
   - Distribuição de votos por candidato
   - Timeline de eventos (votos recebidos, blocos propostos, blocos finalizados)
   - Tabela de blocos finalizados

2. **network_data.json** - Dados brutos em JSON para análise externa ou ferramentas de visualização customizadas

## 🚀 Como usar

### Passo 1: Execute uma simulação

Primeiro, execute qualquer script de simulação para gerar logs:

```bash
cd simulation/scripts
./quick_test.sh
# ou
./simulate_election.sh
# ou
./stress_test.sh
```

### Passo 2: Gere a visualização

```bash
cd simulation/scripts
./visualize.sh
```

### Passo 3: Abra no navegador

```bash
# macOS
open ../visualization.html

# Linux
xdg-open ../visualization.html

# Windows
start ../visualization.html
```

Ou simplesmente abra o arquivo `simulation/visualization.html` no seu navegador preferido.

## 📸 O que você verá

### 1. Estatísticas Gerais
- Total de nós na rede
- Número de validadores
- Blocos finalizados
- Total de votos processados
- Tempo médio entre blocos

### 2. Mapa da Rede P2P
Visualização gráfica mostrando:
- **Nó Bootstrap** (vermelho) - Ponto central de descoberta
- **Nós Validadores** (azul/ciano) - Validadores que propõem e aprovam blocos
- **Nós Votantes** (verde claro) - Clientes que submeteram votos
- **Conexões** - Linhas mostrando as conexões P2P entre nós
  - Linhas sólidas: Conexões entre validadores e bootstrap
  - Linhas tracejadas: Conexões temporárias dos votantes aos validadores

### 3. Distribuição de Votos
Gráfico de barras mostrando quantos votos cada candidato recebeu.

### 4. Timeline de Eventos
Lista cronológica dos últimos 50 eventos:
- 🟢 **Vote Received** - Voto recebido e adicionado ao mempool
- 🟡 **Block Proposed** - Bloco proposto por um validador
- 🔴 **Block Finalized** - Bloco finalizado com consenso

### 5. Tabela de Blocos
Últimos 20 blocos finalizados com:
- Altura do bloco
- Validador que propôs
- Número de votos incluídos
- Timestamp

## 📄 Usando os dados JSON

O arquivo `network_data.json` contém todos os dados estruturados:

```json
{
  "TotalNodes": 4,
  "ValidatorNodes": 3,
  "ConnectedPeers": {...},
  "TotalBlocks": 31,
  "TotalVotes": 3,
  "VoteDistribution": {
    "candidate-a": 1,
    "candidate-b": 1,
    "candidate-c": 1
  },
  "AverageBlockTime": 2.1,
  "Events": [...],
  "Blocks": [...],
  "Nodes": [...],
  "Connections": [...]
}
```

Você pode usar esses dados para:
- Criar gráficos customizados
- Análise estatística
- Integração com outras ferramentas
- Documentação do TCC

## 🎓 Para o TCC

### Capturas de tela recomendadas:

1. **Mapa da Rede** - Mostra a arquitetura P2P descentralizada
2. **Estatísticas** - Demonstra a performance do sistema
3. **Timeline** - Ilustra o fluxo de eventos no consenso
4. **Distribuição de Votos** - Mostra os resultados da eleição

### Dados para análise:

- Tempo médio de finalização de blocos
- Throughput de votos (votos/segundo)
- Latência de propagação na rede
- Taxa de sucesso do consenso

## 🔧 Personalização

Para modificar a visualização, edite:
- `scripts/generate_visualization.go` - Lógica de parsing e geração
- Template HTML dentro do código - Layout e estilos

## 📝 Notas

- A visualização é gerada a partir dos logs existentes
- Execute uma nova simulação antes de gerar nova visualização
- Os arquivos HTML são standalone (não precisam de servidor web)
- Compatível com todos os navegadores modernos

## 🐛 Troubleshooting

**Problema**: "No logs found"
- **Solução**: Execute uma simulação primeiro

**Problema**: Visualização vazia ou incompleta
- **Solução**: Verifique se a simulação rodou completamente e gerou logs

**Problema**: Erro ao compilar
- **Solução**: Certifique-se de estar no diretório `simulation/scripts`
