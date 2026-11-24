#!/usr/bin/env bash
set -euo pipefail

profile="${1:-coverage.out}"
if [[ ! -s "$profile" ]]; then
  printf 'Relatório de cobertura ausente ou vazio: %s\n' "$profile" >&2
  exit 1
fi

awk '
  NR == 1 {
    if ($0 !~ /^mode: (set|count|atomic)$/) {
      print "Cabeçalho do relatório de cobertura inválido." > "/dev/stderr"
      invalid = 1
      exit 1
    }
    next
  }
  {
    if (NF != 3 || $1 !~ /:[0-9]+\.[0-9]+,[0-9]+\.[0-9]+$/ || $2 !~ /^[0-9]+$/ || $3 !~ /^[0-9]+$/) {
      print "Registro do relatório de cobertura inválido." > "/dev/stderr"
      invalid = 1
      exit 1
    }
    if ($1 in blocks && blocks[$1] != $2) {
      print "Blocos repetidos com quantidade de statements inconsistente." > "/dev/stderr"
      invalid = 1
      exit 1
    }
    blocks[$1] = $2
    if ($3 > 0) hits[$1] = 1
  }
  END {
    if (invalid) exit 1
    for (block in blocks) {
      statements += blocks[block]
      if (hits[block]) covered += blocks[block]
    }
    if (statements <= 0) {
      print "Relatório de cobertura sem statements mensuráveis." > "/dev/stderr"
      exit 1
    }
    printf "Cobertura total: %.2f%% (%d de %d statements).\n", covered * 100 / statements, covered, statements
    if (covered * 100 < statements * 80) {
      print "Cobertura abaixo do mínimo obrigatório de 80%." > "/dev/stderr"
      exit 1
    }
  }
' "$profile"
