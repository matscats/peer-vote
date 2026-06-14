# Testes Experimentais do TCC

Esta pasta contém apenas os testes usados para responder às três questões investigativas do TCC.

## Questões e Scripts

| Questão | Propriedade avaliada | Script |
| --- | --- | --- |
| Q1 | Integridade: votos válidos são finalizados corretamente | `scripts/q1_integridade.sh` |
| Q2 | Unicidade: voto duplicado é rejeitado | `scripts/q2_unicidade.sh` |
| Q3 | Tolerância a falhas: operação após `crash-stop` de um validador | `scripts/q3_tolerancia_falhas.sh` |

## Como Executar

A partir da raiz do projeto:

```bash
./simulation/scripts/q1_integridade.sh
./simulation/scripts/q2_unicidade.sh
./simulation/scripts/q3_tolerancia_falhas.sh
```

Cada script compila os binários necessários, limpa `simulation/data/` e `simulation/logs/`, inicia a rede local com três validadores e imprime um resumo final.

## Saídas Geradas

Os logs da execução ficam em:

```text
simulation/logs/
```

Os dados persistidos da blockchain ficam em:

```text
simulation/data/
```

Essas pastas são artefatos de execução e não precisam ser versionadas.

## Interpretação

Q1 passa quando os votos válidos submetidos são finalizados na blockchain.

Q2 passa quando o primeiro voto de um eleitor é finalizado e a tentativa de segundo voto é detectada como duplicada, mantendo apenas um voto finalizado para aquele eleitor.

Q3 passa quando um validador fica offline e os validadores remanescentes ainda apresentam progresso durante a falha, isto é, finalizam ao menos um novo bloco e ao menos um novo voto. Esse é o mesmo critério usado no experimento descrito no TCC.
