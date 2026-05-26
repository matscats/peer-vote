# Como Usar os Resultados no Capítulo 5 do TCC

Este guia orienta como incorporar os resultados experimentais obtidos no Capítulo 5 (Resultados) do TCC seguindo as normas ABNT.

## 1. Estrutura Sugerida para o Capítulo 5

```
5 RESULTADOS EXPERIMENTAIS

5.1 Ambiente de Testes
5.2 Metodologia de Avaliação
5.3 Teste de Validação Funcional
5.4 Teste de Segurança
5.5 Teste de Performance
5.6 Teste de Escalabilidade
5.7 Teste de Tolerância a Falhas
5.8 Discussão dos Resultados
```

## 2. Mapeamento: Resultados → Seções do TCC

### 5.1 Ambiente de Testes
**Fonte:** Seção "Anexos > A. Ambiente de Teste" do RESULTADOS_TESTES_TCC.md

**O que incluir:**
- Configuração de hardware (se relevante)
- Sistema operacional e versões
- Topologia da rede (3 validadores, P2P local)
- Ferramentas utilizadas (Go, libp2p)

**Exemplo de texto:**
```
Os testes foram executados em ambiente local simulando uma rede P2P 
com três validadores. O sistema foi implementado em Go 1.21+ utilizando 
a biblioteca libp2p para comunicação peer-to-peer. A blockchain foi 
implementada de forma customizada com consenso PoA (Proof of Authority).
```

### 5.2 Metodologia de Avaliação
**Fonte:** Seção "Sumário Executivo" + PLANO_TESTES_TCC.md

**O que incluir:**
- Justificativa dos 5 testes escolhidos
- Métricas avaliadas (throughput, latência, disponibilidade)
- Procedimento de execução
- Critérios de sucesso/falha

**Exemplo de texto:**
```
Para validar o sistema proposto, foram definidos cinco testes experimentais 
cobrindo aspectos funcionais, de segurança, performance, escalabilidade e 
tolerância a falhas. Cada teste foi executado de forma automatizada através 
de scripts bash, com resultados coletados em arquivos de log para análise 
posterior.
```

### 5.3 Teste de Validação Funcional
**Fonte:** Seção "1. Teste de Validação Funcional" do RESULTADOS_TESTES_TCC.md

**O que incluir:**
- Tabela 5.1: Configuração do teste
- Figura 5.1: Diagrama de sequência (opcional)
- Tabela 5.2: Resultados obtidos
- Análise: 100% de sucesso

**Exemplo de tabela:**
```latex
\begin{table}[htb]
\centering
\caption{Resultados do Teste de Validação Funcional}
\label{tab:teste1}
\begin{tabular}{|l|r|}
\hline
\textbf{Métrica} & \textbf{Valor} \\
\hline
Votos submetidos & 5 \\
Votos finalizados & 5 \\
Taxa de sucesso & 100\% \\
Blocos finalizados & 17 \\
\hline
\end{tabular}
\fonte{Elaborado pelo autor.}
\end{table}
```

### 5.4 Teste de Segurança
**Fonte:** Seção "2. Teste de Segurança" do RESULTADOS_TESTES_TCC.md

**O que incluir:**
- Descrição do cenário de double voting
- Logs de detecção (pode usar \texttt{} para código)
- Análise crítica: detecção correta, mas rejeição tardia
- Recomendação de melhoria

**Ponto importante:** Seja honesto sobre a limitação identificada. Isso demonstra maturidade científica.

**Exemplo de texto:**
```
O teste identificou uma limitação no pipeline de validação: embora o 
sistema detecte corretamente a votação dupla e impeça sua inclusão na 
blockchain, o voto duplicado é inicialmente aceito antes da validação 
completa. Recomenda-se implementar validação prévia no mempool para 
rejeitar votos duplicados mais cedo no processo.
```

### 5.5 Teste de Performance
**Fonte:** Seção "3. Teste de Performance" do RESULTADOS_TESTES_TCC.md

**O que incluir:**
- Tabela 5.3: Métricas de performance
- Gráfico 5.1: Distribuição de votos por candidato (pizza ou barras)
- Análise: 0% de perda, throughput de 1.61 v/s

**Exemplo de figura:**
```latex
\begin{figure}[htb]
\centering
\caption{Distribuição de votos por candidato no teste de performance}
\label{fig:distribuicao}
% Incluir gráfico de barras ou pizza
\fonte{Elaborado pelo autor.}
\end{figure}
```

### 5.6 Teste de Escalabilidade
**Fonte:** Seção "4. Teste de Escalabilidade" do RESULTADOS_TESTES_TCC.md

**O que incluir:**
- Tabela 5.4: Resultados por nível de carga (10, 25, 50, 100)
- Gráfico 5.2: Curva de escalabilidade (votos finalizados vs carga)
- Gráfico 5.3: Taxa de perda vs carga
- Análise: Saturação em ~25 votos

**Este é um dos resultados mais importantes!** Mostra limitações reais do sistema.

**Exemplo de análise:**
```
A Figura 5.2 evidencia comportamento não-linear do sistema sob carga 
crescente. Observa-se operação ideal até 10 votos simultâneos (0% de perda), 
início de saturação em 25 votos (40% de perda) e saturação completa a partir 
de 50 votos (50% de perda estável). O throughput máximo observado foi de 
2.38 votos/s com 100 votos submetidos.
```

### 5.7 Teste de Tolerância a Falhas
**Fonte:** Seção "5. Teste de Tolerância a Falhas" do RESULTADOS_TESTES_TCC.md

**O que incluir:**
- Figura 5.4: Timeline do teste (3 fases)
- Tabela 5.5: Comparação Fase 1 vs Fase 3
- Análise: 100% disponibilidade mantida

**Exemplo de texto:**
```
O sistema demonstrou tolerância a falhas conforme esperado pela teoria 
de consenso bizantino: com n=3 validadores, o sistema tolera f=(n-1)/3=1 
falha mantendo a maioria de 2/3 validadores. Após o crash do Node 2, 
o sistema continuou processando votos normalmente, finalizando 5 votos 
em 10 blocos sem degradação de performance.
```

### 5.8 Discussão dos Resultados
**Fonte:** Seção "6. Discussão Geral dos Resultados" do RESULTADOS_TESTES_TCC.md

**O que incluir:**
- Tabela 5.6: Comparação com requisitos
- Pontos fortes identificados
- Limitações identificadas
- Comparação com trabalhos relacionados (se houver)

**Estrutura sugerida:**
```
5.8.1 Validação dos Requisitos
5.8.2 Pontos Fortes do Sistema
5.8.3 Limitações Identificadas
5.8.4 Comparação com Trabalhos Relacionados
```

## 3. Tabelas e Figuras: Boas Práticas ABNT

### 3.1 Numeração
- Tabelas: numeração sequencial (Tabela 5.1, 5.2, ...)
- Figuras: numeração sequencial (Figura 5.1, 5.2, ...)
- Sempre referenciar no texto antes de apresentar

### 3.2 Legendas
- **Tabelas:** Legenda ACIMA da tabela
- **Figuras:** Legenda ABAIXO da figura
- Sempre incluir fonte (mesmo que seja "Elaborado pelo autor")

### 3.3 Exemplo de Referência no Texto
```
Os resultados do teste de performance (Tabela 5.3) demonstram throughput 
de 1.61 votos/s com 0% de perda. A distribuição uniforme de votos entre 
os três candidatos (Figura 5.1) confirma a aleatoriedade do processo.
```

## 4. Gráficos Recomendados

### 4.1 Gráfico de Escalabilidade (Essencial)
- **Tipo:** Linha
- **Eixo X:** Votos submetidos (10, 25, 50, 100)
- **Eixo Y:** Votos finalizados
- **Curvas:** Ideal (linha tracejada) vs Obtido (linha sólida)

### 4.2 Taxa de Perda vs Carga
- **Tipo:** Barras
- **Eixo X:** Carga (10, 25, 50, 100)
- **Eixo Y:** Perda (%)

### 4.3 Timeline de Tolerância a Falhas
- **Tipo:** Diagrama temporal
- **Fases:** Normal → Falha → Degradado
- **Métricas:** Validadores ativos, votos processados

### 4.4 Distribuição de Votos (Opcional)
- **Tipo:** Pizza ou barras
- **Dados:** candidate-a (34%), candidate-b (34%), candidate-c (32%)

## 5. Linguagem e Estilo

### 5.1 Tempo Verbal
- Use **pretérito perfeito** para descrever o que foi feito:
  - "Foram executados cinco testes..."
  - "O sistema processou 50 votos..."
  - "Observou-se saturação em 25 votos..."

### 5.2 Voz Passiva
- Prefira voz passiva para manter impessoalidade:
  - ✅ "Foi identificada uma limitação..."
  - ❌ "Eu identifiquei uma limitação..."

### 5.3 Precisão Numérica
- Use 2 casas decimais para métricas: 1.61 v/s (não 1.6 ou 1.607)
- Use percentuais inteiros quando apropriado: 100% (não 100.0%)

### 5.4 Termos Técnicos
- Mantenha termos em inglês quando consagrados: throughput, double voting
- Itálico para termos estrangeiros: \textit{throughput}
- Defina siglas na primeira ocorrência: PoA (\textit{Proof of Authority})

## 6. Checklist de Qualidade

Antes de finalizar o Capítulo 5, verifique:

- [ ] Todas as tabelas estão numeradas e referenciadas no texto
- [ ] Todas as figuras estão numeradas e referenciadas no texto
- [ ] Todas as tabelas/figuras têm fonte ("Elaborado pelo autor")
- [ ] Números estão formatados consistentemente (casas decimais)
- [ ] Siglas foram definidas na primeira ocorrência
- [ ] Texto está em voz passiva e pretérito perfeito
- [ ] Limitações foram discutidas honestamente
- [ ] Resultados foram comparados com requisitos (Capítulo 3)
- [ ] Há conexão clara com o próximo capítulo (Conclusão)

## 7. Transição para o Capítulo 6 (Conclusão)

O último parágrafo do Capítulo 5 deve preparar o leitor para a conclusão:

**Exemplo:**
```
Os resultados experimentais apresentados neste capítulo validaram os 
principais requisitos do sistema proposto, demonstrando viabilidade 
técnica para aplicações reais em eleições de pequeno e médio porte. 
As limitações identificadas, especialmente quanto à escalabilidade, 
serão discutidas no próximo capítulo juntamente com propostas de 
trabalhos futuros.
```

## 8. Arquivos de Referência

Para escrever o Capítulo 5, consulte:

1. **RESULTADOS_TESTES_TCC.md** - Resultados completos e análises
2. **PLANO_TESTES_TCC.md** - Metodologia e objetivos dos testes
3. **GUIA_ESCRITA_TCC.md** - Regras ABNT e padrões de escrita
4. **Logs originais** - simulation/results_tcc/*.log (para detalhes)

## 9. Exemplo de Seção Completa

```latex
\section{TESTE DE PERFORMANCE}

Para avaliar a capacidade de processamento do sistema, foi conduzido 
um teste com 50 votos submetidos simultaneamente a uma rede de 3 
validadores. O objetivo foi medir throughput, latência e taxa de 
perda de votos sob carga moderada.

\subsection{Configuração do Teste}

A Tabela~\ref{tab:config-perf} apresenta os parâmetros utilizados 
no teste de performance.

\begin{table}[htb]
\centering
\caption{Configuração do teste de performance}
\label{tab:config-perf}
\begin{tabular}{|l|l|}
\hline
\textbf{Parâmetro} & \textbf{Valor} \\
\hline
Validadores & 3 nós \\
Votos submetidos & 50 \\
Candidatos & 3 \\
Tempo de processamento & 30s \\
\hline
\end{tabular}
\fonte{Elaborado pelo autor.}
\end{table}

\subsection{Resultados Obtidos}

A Tabela~\ref{tab:result-perf} apresenta as métricas coletadas 
durante o teste.

\begin{table}[htb]
\centering
\caption{Resultados do teste de performance}
\label{tab:result-perf}
\begin{tabular}{|l|r|}
\hline
\textbf{Métrica} & \textbf{Valor} \\
\hline
Votos finalizados & 50 \\
Blocos finalizados & 14 \\
Throughput & 1.61 votos/s \\
Latência end-to-end & 31s \\
Perda de votos & 0\% \\
\hline
\end{tabular}
\fonte{Elaborado pelo autor.}
\end{table}

\subsection{Análise}

O sistema demonstrou excelente performance sob carga moderada, 
finalizando 100\% dos votos submetidos. O throughput de 1.61 
votos/s é adequado para eleições de pequeno e médio porte. 
A latência end-to-end de 31 segundos é aceitável considerando 
o overhead do consenso distribuído e propagação P2P.

A distribuição uniforme de votos entre os três candidatos 
(Figura~\ref{fig:dist-votos}) confirma a aleatoriedade do 
processo de teste, validando a representatividade dos resultados.
```

## 10. Dicas Finais

1. **Seja honesto:** Discuta limitações abertamente
2. **Seja preciso:** Use dados exatos dos logs
3. **Seja visual:** Gráficos comunicam melhor que tabelas
4. **Seja conectado:** Relacione resultados com objetivos (Cap. 1) e requisitos (Cap. 3)
5. **Seja crítico:** Analise, não apenas descreva

---

**Próximos Passos:**
1. Ler RESULTADOS_TESTES_TCC.md completamente
2. Criar esboço do Capítulo 5 seguindo a estrutura sugerida
3. Gerar gráficos a partir dos dados
4. Escrever seção por seção
5. Revisar com checklist de qualidade
