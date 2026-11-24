# Framoryx Worker Extract

Worker Go para a extração de frames e geração de arquivos ZIP. O repositório contém o domínio, logs JSON e exportadores de métricas e traces por OTLP/HTTP. A orquestração e os adapters são entregues em incrementos próprios.

O executável atual inicializa a observabilidade e aguarda um sinal de encerramento. O consumidor de mensagens e os adapters de processamento, armazenamento e coordenação ainda não estão conectados ao bootstrap. Subir o container não inicia uma extração.

## Executar localmente com Grafana

Pré-requisitos: Docker com daemon ativo, BuildKit habilitado e Docker Compose v2.20 ou superior. Em macOS, o runtime precisa oferecer containers Linux. Reserve memória para o worker e para o backend de observabilidade; o LGTM reúne vários processos no mesmo container.

Na raiz deste repositório:

```sh
docker compose config --quiet
docker compose up --build --detach --wait --wait-timeout 240
docker compose ps
```

Abra o Grafana em <http://localhost:3000>. O acesso administrativo anônimo é habilitado exclusivamente para este ambiente local; as portas são publicadas em `127.0.0.1`.

O Compose contém dois serviços:

| Serviço | Responsabilidade |
| --- | --- |
| `worker` | Compilar e executar o worker, enviando métricas e traces para `http://grafana:4318` |
| `grafana` | Executar o Grafana LGTM com Collector OTLP, Prometheus, Tempo, Loki e Pyroscope |

O worker aguarda a prontidão do LGTM antes de iniciar. O healthcheck é o fornecido pela imagem LGTM; o worker ainda não oferece uma verificação de prontidão do consumidor. Durante a parada pelo Compose, o worker recebe `SIGTERM` antes do backend e dispõe de 20 segundos para concluir seu encerramento, que atualmente reserva até 10 segundos à telemetria.

Para alterar as portas no host:

```sh
GRAFANA_PORT=3300 OTLP_HTTP_PORT=14318 docker compose up --build --detach --wait --wait-timeout 240
```

Essas variáveis alteram somente as portas publicadas. A comunicação interna continua em `http://grafana:4318`. O receptor OTLP/HTTP também fica acessível no host pela porta `4318`, ou pelo valor de `OTLP_HTTP_PORT`.

## Consultar telemetria e logs

No Grafana, abra **Explore** e escolha:

- **Prometheus** para métricas. O prefixo original é `framoryx.app`; a tradução para Prometheus pode substituir pontos por sublinhados e acrescentar sufixos de tipo/unidade. A consulta `{__name__=~"framoryx[._]app[._].*"}` encontra os dois formatos.
- **Tempo** para traces, filtrando pelo atributo `service.name=framoryx-worker-extract`.

O intervalo de exportação de métricas no Compose é de cinco segundos. O bootstrap ocioso não registra extrações ou spans de processamento, portanto consultas vazias são esperadas até que a instrumentação seja utilizada por uma execução real. Registrar os instrumentos não equivale a produzir pontos de métricas. O ambiente não gera dados fictícios.

Os logs JSON do worker são escritos em stdout e podem ser acompanhados com:

```sh
docker compose logs --follow worker
```

Este Compose não encaminha automaticamente stdout ao Loki. A presença da fonte Loki no Grafana não significa que os logs do worker já foram ingeridos. Também não há coleta automática de profiles; a auto-instrumentação eBPF está desabilitada.

## Encerrar e preservar os dados

```sh
docker compose down
```

O volume `grafana-data` preserva dados e configurações do LGTM entre recriações. Para descartar exclusivamente os volumes deste projeto junto com seus containers:

```sh
docker compose down --volumes
```

## Imagem do worker

```sh
docker build --tag framoryx-worker-extract:local .
```

O Dockerfile compila com Go `1.26.11`, em estágio separado, e produz um binário Linux sem CGO. O ambiente suporta `linux/amd64` e `linux/arm64`, escolhidos pelo build. A imagem final usa `scratch`, inclui certificados de autoridades certificadoras e executa como `65532:65532`. O processo recebe sinais diretamente pelo entrypoint. Não há shell ou FFmpeg nessa imagem; o adapter de processamento ainda depende de uma implementação futura.

O contexto de build é uma lista permitida de `go.mod`, `go.sum`, `cmd/` e `internal/`, além dos próprios arquivos de build. O Compose executa o worker com filesystem somente leitura, sem capabilities e sem permissão para elevar privilégios. As imagens de build e observabilidade têm versões e digests fixados; atualizar a versão exige atualizar também o digest. O LGTM mantém o usuário padrão do fornecedor para inicializar o volume `/data`.

O LGTM foi escolhido como ferramenta local de desenvolvimento. Este Compose não define implantação, autenticação, retenção ou topologia de produção. Referências: [Grafana LGTM](https://github.com/grafana/docker-otel-lgtm/tree/v0.41.1), [build em múltiplos estágios](https://docs.docker.com/build/building/multi-stage/) e [ordem de inicialização e encerramento do Compose](https://docs.docker.com/compose/how-tos/startup-order/).

Se aparecer `Cannot connect to the Docker daemon`, inicie o runtime Docker e selecione seu contexto antes de subir o ambiente. Se `docker compose` não for reconhecido, instale o plugin Compose pelo procedimento do seu runtime. Somente o cliente `docker` não é suficiente para construir ou executar os containers.
