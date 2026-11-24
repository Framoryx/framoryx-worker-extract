import json
import sys
from pathlib import Path


def has_go_version(result):
    if not isinstance(result, dict) or result.get("Type") != "gobinary":
        return False
    packages = result.get("Packages")
    if not isinstance(packages, list):
        return False
    for package in packages:
        if not isinstance(package, dict) or package.get("Name") != "stdlib":
            continue
        version = package.get("Version")
        if isinstance(version, str) and version.strip().lower() not in ("", "unknown", "(devel)"):
            return True
    return False


def check_report(path):
    report = json.loads(Path(path).read_text(encoding="utf-8"))
    results = report.get("Results") if isinstance(report, dict) else None
    if not isinstance(results, list) or not any(has_go_version(result) for result in results):
        raise ValueError("O scanner não reconheceu o binário Go e a versão da biblioteca padrão.")


def main():
    if len(sys.argv) != 2:
        print("Uso: python3 scripts/ci/check-go-report.py RELATORIO.json", file=sys.stderr)
        return 1
    try:
        check_report(sys.argv[1])
    except (OSError, ValueError) as error:
        print(f"Verificação de evidência Go reprovada: {error}", file=sys.stderr)
        return 1
    print("Evidência de binário Go e versão da biblioteca padrão confirmada.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
