import argparse
import json
import os
import shlex
import subprocess
import sys
import tempfile
import time
import urllib.parse
import urllib.request
import uuid
from pathlib import Path


SERVICE = "framoryx-compose-validation"
SPAN_NAME = "Validação de conectividade do ambiente local"
FOLDER_API = "/apis/folder.grafana.app/v1/namespaces/default/folders"
FOLDER_UID = "framoryx-validation"
FOLDER_TITLE = "Validação do ambiente local"


def command(*arguments, timeout=300):
    print("+ " + shlex.join(arguments), flush=True)
    return subprocess.check_output(arguments, text=True, stderr=subprocess.STDOUT, timeout=timeout)


def request_json(url, payload=None):
    data = json.dumps(payload).encode() if payload is not None else None
    headers = {"Accept": "application/json"}
    if data is not None:
        headers["Content-Type"] = "application/json"
    request = urllib.request.Request(url, data=data, headers=headers)
    with urllib.request.urlopen(request, timeout=10) as response:
        return json.load(response)


def eventually(check, description, timeout=60):
    deadline = time.monotonic() + timeout
    last_error = "resposta ainda não disponível"
    while True:
        try:
            if check():
                print(description, flush=True)
                return
        except (OSError, json.JSONDecodeError) as error:
            last_error = str(error)
        if time.monotonic() >= deadline:
            raise RuntimeError(f"Tempo esgotado: {description}; {last_error}")
        time.sleep(1)


def require(condition, message):
    if not condition:
        raise RuntimeError(message)


def resource():
    return {"attributes": [{"key": "service.name", "value": {"stringValue": SERVICE}}]}


def metric_payload():
    now = time.time_ns()
    metric = {
        "name": "framoryx.app.validation.connectivity",
        "sum": {
            "aggregationTemporality": 2,
            "isMonotonic": True,
            "dataPoints": [{"startTimeUnixNano": str(now - 1_000_000), "timeUnixNano": str(now), "asInt": "1"}],
        },
    }
    return {"resourceMetrics": [{"resource": resource(), "scopeMetrics": [{"scope": {"name": SERVICE}, "metrics": [metric]}]}]}


def trace_payload(trace_id):
    now = time.time_ns()
    span = {
        "traceId": trace_id,
        "spanId": uuid.uuid4().hex[:16],
        "name": SPAN_NAME,
        "kind": 1,
        "startTimeUnixNano": str(now - 1_000_000),
        "endTimeUnixNano": str(now),
        "status": {"code": 1},
    }
    return {"resourceSpans": [{"resource": resource(), "scopeSpans": [{"scope": {"name": SERVICE}, "spans": [span]}]}]}


def contains_span(value):
    if isinstance(value, dict):
        return value.get("name") == SPAN_NAME or any(contains_span(item) for item in value.values())
    if isinstance(value, list):
        return any(contains_span(item) for item in value)
    return False


def send_signal(endpoint, signal, payload, rejected_field):
    response = request_json(endpoint + "/v1/" + signal, payload)
    require(isinstance(response, dict), "Resposta OTLP inválida.")
    rejected = response.get("partialSuccess", {}).get(rejected_field, 0)
    require(int(rejected) == 0, f"O receptor OTLP rejeitou dados de {signal}.")


def check_ingestion(grafana, otlp):
    trace_id = uuid.uuid4().hex
    send_signal(otlp, "metrics", metric_payload(), "rejectedDataPoints")
    send_signal(otlp, "traces", trace_payload(trace_id), "rejectedSpans")
    query = urllib.parse.urlencode({"query": '{__name__=~"framoryx[._]app[._]validation[._]connectivity.*"}'})
    metrics_url = grafana + "/api/datasources/proxy/uid/prometheus/api/v1/query?" + query
    traces_url = grafana + "/api/datasources/proxy/uid/tempo/api/traces/" + trace_id
    eventually(lambda: metric_is_visible(metrics_url), "Métrica de validação consultada pelo Grafana.")
    eventually(lambda: contains_span(request_json(traces_url)), "Trace de validação consultado pelo Grafana.")


def metric_is_visible(url):
    response = request_json(url)
    if response.get("status") != "success":
        return False
    return any(float(sample["value"][1]) == 1 for sample in response.get("data", {}).get("result", []))


def worker_id(compose):
    identifier = command(*compose, "ps", "--all", "--quiet", "worker").strip()
    require(bool(identifier), "Container do worker não encontrado.")
    return identifier


def inspect_worker(compose):
    identifier = worker_id(compose)
    container = json.loads(command("docker", "inspect", identifier))[0]
    host = container["HostConfig"]
    require(container["State"]["Running"], "O worker não permaneceu em execução.")
    require(container["Config"]["User"] == "65532:65532", "O worker não está usando o usuário esperado.")
    require(host["ReadonlyRootfs"], "O filesystem do worker permite escrita.")
    require("ALL" in host["CapDrop"], "As capabilities do worker não foram removidas.")
    require(any(value.startswith("no-new-privileges") for value in host["SecurityOpt"]), "Elevação de privilégios permitida.")
    logs = command("docker", "logs", identifier)
    records = [json.loads(line) for line in logs.splitlines() if line.strip()]
    require(any(record.get("msg", "").startswith("Rastreabilidade inicializada") for record in records), "Bootstrap do worker não confirmado.")


def stop_worker(compose):
    command(*compose, "stop", "worker")
    container = json.loads(command("docker", "inspect", worker_id(compose)))[0]
    state = container["State"]
    require(not state["Running"] and state["ExitCode"] == 0 and not state["OOMKilled"], "Encerramento do worker falhou.")
    print("Worker encerrado por SIGTERM com código zero.", flush=True)


def check_persistence(compose, grafana):
    eventually(lambda: request_json(grafana + FOLDER_API).get("kind") == "FolderList", "API de pastas do Grafana pronta.")
    request_json(grafana + FOLDER_API, {"metadata": {"name": FOLDER_UID}, "spec": {"title": FOLDER_TITLE}})
    previous = command(*compose, "ps", "--quiet", "grafana").strip()
    command(*compose, "up", "--detach", "--wait", "--wait-timeout", "240", "--force-recreate", "grafana", timeout=600)
    current = command(*compose, "ps", "--quiet", "grafana").strip()
    require(bool(current) and current != previous, "O container Grafana não foi recriado.")
    folder_url = grafana + FOLDER_API + "/" + FOLDER_UID
    eventually(lambda: request_json(folder_url).get("spec", {}).get("title") == FOLDER_TITLE, "Configuração do Grafana preservada após recriar o container.")


def save_diagnostics(compose, directory):
    for name, arguments in [("containers.json", ["ps", "--all", "--format", "json"]), ("containers.log", ["logs", "--no-color"])]:
        try:
            output = command(*compose, *arguments, timeout=30)
        except subprocess.CalledProcessError as error:
            output = error.output
        (directory / name).write_text(output, encoding="utf-8")


def validate_environment(compose, no_build):
    command(*compose, "config", "--quiet")
    build_option = "--no-build" if no_build else "--build"
    command(*compose, "up", "--detach", "--wait", "--wait-timeout", "240", build_option, timeout=900)
    grafana = "http://127.0.0.1:" + os.environ.get("GRAFANA_PORT", "3000")
    otlp = "http://127.0.0.1:" + os.environ.get("OTLP_HTTP_PORT", "4318")
    eventually(lambda: request_json(grafana + "/api/health").get("database") == "ok", "Grafana pronto para consultas.")
    eventually(lambda: bootstrap_ready(compose), "Bootstrap e permissões do worker verificados.")
    check_ingestion(grafana, otlp)
    stop_worker(compose)
    check_persistence(compose, grafana)


def bootstrap_ready(compose):
    try:
        inspect_worker(compose)
    except RuntimeError:
        return False
    return True


def main():
    parser = argparse.ArgumentParser(description="Validar o ambiente Compose em um projeto isolado e descartável.")
    parser.add_argument("--no-build", action="store_true", help="Usar a imagem local já construída e analisada.")
    parser.add_argument("--report-dir", type=Path, help="Diretório para preservar os diagnósticos.")
    options = parser.parse_args()
    directory = options.report_dir or Path(tempfile.mkdtemp(prefix="framoryx-compose-"))
    directory.mkdir(parents=True, exist_ok=True)
    project = "framoryx-validation-" + uuid.uuid4().hex[:12]
    compose = ["docker", "compose", "--project-name", project]
    status = 0
    try:
        validate_environment(compose, options.no_build)
    except (OSError, ValueError, RuntimeError, subprocess.SubprocessError) as error:
        print(f"Validação Compose reprovada: {error}", file=sys.stderr)
        if isinstance(error, subprocess.CalledProcessError):
            print(error.output, file=sys.stderr)
        status = 1
    finally:
        status = finish_validation(compose, directory, status)
    return status


def finish_validation(compose, directory, status):
    try:
        try:
            save_diagnostics(compose, directory)
        finally:
            command(*compose, "down", "--volumes", "--remove-orphans", timeout=120)
    except (OSError, subprocess.SubprocessError) as error:
        print(f"Falha ao coletar diagnósticos ou limpar o projeto isolado: {error}", file=sys.stderr)
        status = 1
    (directory / "result.json").write_text(json.dumps({"project": compose[-1], "success": status == 0}), encoding="utf-8")
    return status


if __name__ == "__main__":
    sys.exit(main())
