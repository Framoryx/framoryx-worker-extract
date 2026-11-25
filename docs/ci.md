# Integração contínua do worker

O workflow **CI do worker** roda em pull requests destinados a `main`, pushes em `main` e disparos manuais. Os jobs têm nomes próprios no gráfico do GitHub Actions. Além das verificações Go e de segurança, a CI constrói imagens locais e valida um ambiente Compose isolado no runner, sem publicar imagens em registro nem implantar serviços fora desse teste.

| Job visível | Verificação |
|---|---|
| Validação do workflow | actionlint e sintaxe dos scripts de CI |
| Verificação dos gates de CI | Testes de relatórios inválidos/sem versão, rejeição OTLP e limpeza mesmo sob falha de diagnóstico |
| Formatação Go | `gofmt` sem alterações automáticas |
| Análise estática Go | `go vet ./...` |
| Testes e concorrência | Testes com `-race`, ordem aleatória e perfil de cobertura de `internal` |
| Cobertura mínima de 80% | Lê o artefato dos testes e bloqueia perfil ausente, inválido ou abaixo do mínimo |
| Compilação do worker | Compila `./cmd` e preserva o executável |
| Segredos no histórico Git | Gitleaks CLI no histórico completo, com valores ocultos no relatório |
| Vulnerabilidades das dependências | Trivy bloqueia achados HIGH/CRITICAL, inclusive sem correção disponível |
| Vulnerabilidades do executável | Trivy examina o binário compilado, incluindo módulos e versão da biblioteca padrão Go |
| Configuração de infraestrutura | `docker compose config --quiet` e Trivy nas configurações reconhecidas do checkout |
| Build da imagem Linux/amd64 e Linux/arm64 | Matriz que exporta e confere a plataforma e o filesystem de cada imagem |
| Vulnerabilidades da imagem Linux/amd64 e Linux/arm64 | Matriz Trivy sobre o arquivo exato produzido no build, com evidência de análise Go |
| Integração real com Compose e Grafana | Carrega a imagem AMD64 construída e testa startup, ingestão OTLP, persistência e SIGTERM |
| Verificações internas obrigatórias | Exige sucesso de todos os jobs anteriores |

Cobertura depende dos testes; análise do executável depende da compilação; análise das imagens e integração Compose dependem do build das imagens. As demais verificações executam em paralelo. O job final depende de todos. Falha, cancelamento ou job ignorado impedem a aprovação final; não há `continue-on-error`. As matrizes usam `fail-fast: false` para preservar o resultado de ambas as plataformas. Execuções anteriores da mesma referência podem ser canceladas quando chegar uma atualização.

## Cobertura e evidências

O perfil é produzido com `go test -count=1 -race -shuffle=on -coverpkg=./internal/... -coverprofile=coverage.out ./...`. A instrumentação inclui a lógica chamada pelos testes externos em `tests/`. O mínimo é calculado pelos statements efetivamente cobertos, sem arredondar um resultado abaixo de 80% para aprovação.

Relatórios, executável e imagens ficam em artefatos por sete dias. O perfil ausente não equivale a cobertura zero ou sucesso: o job reprova. A análise de infraestrutura verifica o Dockerfile reconhecido pelo Trivy; a configuração Compose é validada pelo próprio CLI. Isso não equivale a um scan de segurança do Compose pelo Trivy.

A análise do executável usa `trivy rootfs` sobre o diretório do artefato, restaura sua permissão de execução após o download e exige que o relatório reconheça um binário Go com a versão da biblioteca padrão. O mesmo validador, `scripts/ci/check-go-report.py`, é aplicado aos relatórios das imagens. Um relatório sem essa evidência reprova em vez de aparentar ausência de vulnerabilidades. [Cobertura oficial de módulos e binários Go no Trivy](https://trivy.dev/docs/latest/coverage/language/golang/).

## Imagem exata e teste do ambiente

O Dockerfile usa um estágio de build nativo com `BUILDPLATFORM` e compila Go para o alvo escolhido, sem executar instruções na imagem alvo. Cada job constrói somente uma plataforma, sem QEMU. O builder usa o driver `docker-container`, necessário ao exporter Docker para arquivo, com BuildKit fixado por digest. [Compilação para múltiplas plataformas](https://docs.docker.com/build/building/multi-platform/) e [exporters Docker e OCI](https://docs.docker.com/build/exporters/oci-docker/).

Os artefatos `imagem-worker-amd64` e `imagem-worker-arm64` contêm `image.tar`, `image.tar.sha256` e a inspeção da imagem carregada desse arquivo. Antes do upload são conferidos Linux, arquitetura esperada e filesystem não vazio. O checksum é validado novamente em cada consumidor do artefato. Os jobs são isolados e podem usar a mesma tag local `framoryx-worker-extract:local` sem compartilhar a imagem entre plataformas.

O scan usa `trivy image --input image.tar`, restringe o scanner a vulnerabilidades e bloqueia HIGH/CRITICAL, inclusive sem correção. Ele analisa o arquivo transferido, sem reconstruir a imagem nem resolver uma tag remota. [Análise oficial de imagens em arquivo no Trivy](https://trivy.dev/docs/latest/target/container_image/#tar-files).

O teste Compose carrega somente o artefato AMD64 e executa `scripts/ci/check-compose.py --no-build --report-dir reports/compose`. O script utiliza seu próprio projeto e volume, verifica prontidão do backend e worker, ingestão OTLP em fonte de validação, persistência de configuração Grafana e parada por SIGTERM; ao final, limpa somente os recursos isolados do teste. As evidências de `reports/compose` são preservadas também quando houver falha. Esse teste não comprova processamento de vídeo nem ingestão automática dos logs do worker.

## SonarQube Cloud

A ligação do SonarQube Cloud ao GitHub é uma integração externa a este workflow. Ela deve produzir seu próprio check de análise/Quality Gate. A CI não executa um segundo scanner, não exige um token adicional e não apresenta um job fictício de Sonar aprovado.

O nome e o contexto reais desse check devem ser identificados em uma execução da integração antes de torná-lo obrigatório na proteção de `main`. O check **Verificações internas obrigatórias** cobre somente os jobs da tabela; seu sucesso não comprova aprovação do Sonar. A proteção deve exigir ambos quando o check externo estiver disponível. A configuração de proteção de branch é uma operação separada da criação do workflow.

O plano Free oferece análise de pull request quando o destino é a branch principal; análise de outras branches depende do plano. A integração e seu Quality Gate precisam ser verificados na organização existente, sem assumir a criação de uma conta, projeto ou plano pago. [Planos oficiais do SonarQube Cloud](https://docs.sonarsource.com/sonarqube-cloud/administering-sonarcloud/managing-subscription/subscription-plans).

## Versões e permissões

O Go segue a versão exata de `go.mod`. As actions estão fixadas por SHA: checkout v7.0.1, setup-go v7.1.0, upload-artifact v7.0.2 e download-artifact v8.0.2. Gitleaks 8.30.1, Trivy 0.76.2 e actionlint 1.7.12 são instalados de releases oficiais com SHA-256 fixado antes da extração. O builder de imagem usa BuildKit v0.35.1 com o digest de seu índice multiplataforma. A atualização de versões exige atualizar também essas referências, checksums e digests.

O workflow usa apenas `contents: read`, checkout sem credenciais persistidas e o evento `pull_request`, sem execução privilegiada por `pull_request_target`. Relatórios de Gitleaks ocultam integralmente os valores detectados. Nenhuma exceção de vulnerabilidade ou segredo foi criada nesta entrega; eventuais exceções precisam de análise específica.

Referências operacionais: [setup-go](https://github.com/actions/setup-go), [artefatos do GitHub Actions](https://github.com/actions/upload-artifact), [Gitleaks CLI](https://github.com/gitleaks/gitleaks), [Trivy filesystem](https://trivy.dev/docs/latest/target/filesystem/) e [Trivy configuração](https://trivy.dev/docs/latest/scanner/misconfiguration/).
