# Testes Experimentais do TCC

Esta pasta contém apenas os testes usados para responder às três questões investigativas do TCC.

## Questões e Scripts

| Questão | Propriedade avaliada | Script |
| --- | --- | --- |
| Q1 | Integridade: votos válidos são finalizados corretamente | `scripts/q1_integridade.sh` |
| Q2 | Unicidade: voto duplicado é rejeitado | `scripts/q2_unicidade.sh` |
| Q3 | Tolerância a falhas: operação após `crash-stop` e sincronização do nó reiniciado | `scripts/q3_falha_sincronizacao.sh` |

## Como Executar

A partir da raiz do projeto:

```bash
./simulation/scripts/q1_integridade.sh
./simulation/scripts/q2_unicidade.sh
./simulation/scripts/q3_falha_sincronizacao.sh
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

Q3 passa quando um validador fica offline, os demais avançam a cadeia, o validador retorna, sincroniza os blocos perdidos e termina com a mesma altura e o mesmo arquivo de blocos dos demais validadores.
