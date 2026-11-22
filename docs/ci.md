# Integração contínua do worker

O workflow **CI do worker** roda em pull requests destinados a `main`, pushes em `main` e disparos manuais. Os jobs têm nomes próprios no gráfico do GitHub Actions e executam verificações internas sem publicar imagens ou implantar serviços.

| Job visível | Verificação |
|---|---|
| Validação do workflow | actionlint e sintaxe dos scripts de CI |
| Formatação Go | `gofmt` sem alterações automáticas |
| Análise estática Go | `go vet ./...` |
| Testes e concorrência | Testes com `-race`, ordem aleatória e perfil de cobertura de `internal` |
| Cobertura mínima de 80% | Lê o artefato dos testes e bloqueia perfil ausente, inválido ou abaixo do mínimo |
| Compilação do worker | Compila `./cmd` e preserva o executável |
| Segredos no histórico Git | Gitleaks CLI no histórico completo, com valores ocultos no relatório |
| Vulnerabilidades das dependências | Trivy bloqueia achados HIGH/CRITICAL, inclusive sem correção disponível |
| Vulnerabilidades do executável | Trivy examina o binário compilado, incluindo módulos e versão da biblioteca padrão Go |
| Configuração de infraestrutura | Trivy verifica as configurações reconhecidas presentes no checkout |
| Verificações internas obrigatórias | Exige sucesso de todos os jobs anteriores |

Os jobs são independentes, exceto cobertura, que depende dos testes, análise do executável, que depende da compilação, e o job final, que depende de todos. Falha, cancelamento ou job ignorado impedem a aprovação final; não há `continue-on-error`. Execuções anteriores da mesma referência podem ser canceladas quando chegar uma atualização.

## Cobertura e evidências

O perfil é produzido com `go test -count=1 -race -shuffle=on -coverpkg=./internal/... -coverprofile=coverage.out ./...`. A instrumentação inclui a lógica chamada pelos testes externos em `tests/`. O mínimo é calculado pelos statements efetivamente cobertos, sem arredondar um resultado abaixo de 80% para aprovação.

Relatórios e executável ficam em artefatos por sete dias. O perfil ausente não equivale a cobertura zero ou sucesso: o job reprova. A análise de infraestrutura pode terminar sem arquivos aplicáveis enquanto não houver manifests reconhecidos pelo Trivy. A verificação de uma imagem final será acrescentada quando existir uma imagem definida para o projeto; esta CI não cria Dockerfile nem escolhe uma imagem de produção.

A análise do executável usa `trivy rootfs` sobre o diretório do artefato, restaura sua permissão de execução após o download e exige que o relatório reconheça um binário Go com a versão da biblioteca padrão. Um relatório sem essa evidência reprova em vez de aparentar ausência de vulnerabilidades. [Cobertura oficial de módulos e binários Go no Trivy](https://trivy.dev/docs/latest/coverage/language/golang/).

## SonarQube Cloud

A ligação do SonarQube Cloud ao GitHub é uma integração externa a este workflow. Ela deve produzir seu próprio check de análise/Quality Gate. A CI não executa um segundo scanner, não exige um token adicional e não apresenta um job fictício de Sonar aprovado.

O nome e o contexto reais desse check devem ser identificados em uma execução da integração antes de torná-lo obrigatório na proteção de `main`. O check **Verificações internas obrigatórias** cobre somente os jobs da tabela; seu sucesso não comprova aprovação do Sonar. A proteção deve exigir ambos quando o check externo estiver disponível. A configuração de proteção de branch é uma operação separada da criação do workflow.

O plano Free oferece análise de pull request quando o destino é a branch principal; análise de outras branches depende do plano. A integração e seu Quality Gate precisam ser verificados na organização existente, sem assumir a criação de uma conta, projeto ou plano pago. [Planos oficiais do SonarQube Cloud](https://docs.sonarsource.com/sonarqube-cloud/administering-sonarcloud/managing-subscription/subscription-plans).

## Versões e permissões

O Go segue a versão exata de `go.mod`. As actions estão fixadas por SHA: checkout v7.0.1, setup-go v7.1.0, upload-artifact v7.0.2 e download-artifact v8.0.2. Gitleaks 8.30.1, Trivy 0.76.2 e actionlint 1.7.12 são instalados de releases oficiais com SHA-256 fixado antes da extração. A atualização de versões exige atualizar também essas referências e checksums.

O workflow usa apenas `contents: read`, checkout sem credenciais persistidas e o evento `pull_request`, sem execução privilegiada por `pull_request_target`. Relatórios de Gitleaks ocultam integralmente os valores detectados. Nenhuma exceção de vulnerabilidade ou segredo foi criada nesta entrega; eventuais exceções precisam de análise específica.

Referências operacionais: [setup-go](https://github.com/actions/setup-go), [artefatos do GitHub Actions](https://github.com/actions/upload-artifact), [Gitleaks CLI](https://github.com/gitleaks/gitleaks), [Trivy filesystem](https://trivy.dev/docs/latest/target/filesystem/) e [Trivy configuração](https://trivy.dev/docs/latest/scanner/misconfiguration/).
