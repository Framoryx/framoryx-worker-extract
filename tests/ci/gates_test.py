import importlib.util
import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch


ROOT = Path(__file__).resolve().parents[2]
SPEC = importlib.util.spec_from_file_location("compose_check", ROOT / "scripts/ci/check-compose.py")
COMPOSE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(COMPOSE)


class GoReportTests(unittest.TestCase):
    def test_report_evidence(self):
        valid_package = {"Name": "stdlib", "Version": "1.26.11"}
        cases = {
            "válido": ({"Results": [{"Type": "gobinary", "Packages": [valid_package]}]}, 0),
            "raiz inválida": ([], 1),
            "sem resultados": ({}, 1),
            "resultados inválidos": ({"Results": {}}, 1),
            "resultados vazios": ({"Results": []}, 1),
            "resultado inválido": ({"Results": ["inválido"]}, 1),
            "tipo errado": ({"Results": [{"Type": "gomod", "Packages": [valid_package]}]}, 1),
            "sem pacotes": ({"Results": [{"Type": "gobinary", "Packages": None}]}, 1),
            "pacote inválido": ({"Results": [{"Type": "gobinary", "Packages": ["inválido"]}]}, 1),
            "sem stdlib": ({"Results": [{"Type": "gobinary", "Packages": [{"Name": "outro", "Version": "1.0.0"}]}]}, 1),
        }
        for name, (data, expected) in cases.items():
            with self.subTest(name=name), tempfile.TemporaryDirectory() as directory:
                path = Path(directory) / "report.json"
                path.write_text(json.dumps(data), encoding="utf-8")
                self.assertEqual(self.run_gate(path).returncode, expected)

    def test_invalid_versions(self):
        for version in [None, "", " ", "unknown", " UNKNOWN ", "(devel)", 123]:
            with self.subTest(version=version), tempfile.TemporaryDirectory() as directory:
                package = {"Name": "stdlib", "Version": version}
                path = Path(directory) / "report.json"
                path.write_text(json.dumps({"Results": [{"Type": "gobinary", "Packages": [package]}]}))
                self.assertEqual(self.run_gate(path).returncode, 1)

    def test_unreadable_reports(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "report.json"
            self.assertEqual(self.run_gate(path).returncode, 1)
            for content in ["", "{inválido"]:
                with self.subTest(content=content):
                    path.write_text(content, encoding="utf-8")
                    self.assertEqual(self.run_gate(path).returncode, 1)

    def run_gate(self, path):
        return subprocess.run([sys.executable, str(ROOT / "scripts/ci/check-go-report.py"), str(path)], capture_output=True, text=True)


class ComposeGateTests(unittest.TestCase):
    def test_rejected_or_invalid_otlp_response(self):
        cases = [
            ("metrics", "rejectedDataPoints", {"partialSuccess": {"rejectedDataPoints": "1"}}),
            ("traces", "rejectedSpans", {"partialSuccess": {"rejectedSpans": "1"}}),
            ("metrics", "rejectedDataPoints", []),
        ]
        for signal, field, response in cases:
            with self.subTest(signal=signal, response=response), patch.object(COMPOSE, "request_json", return_value=response):
                with self.assertRaises(RuntimeError):
                    COMPOSE.send_signal("http://localhost", signal, {}, field)

    def test_cleanup_runs_after_diagnostics_timeout(self):
        commands = []
        with tempfile.TemporaryDirectory() as directory:
            with patch.object(COMPOSE, "save_diagnostics", side_effect=subprocess.TimeoutExpired("diagnóstico", 1)):
                with patch.object(COMPOSE, "command", side_effect=lambda *args, **kwargs: commands.append(args)):
                    status = COMPOSE.finish_validation(["docker", "compose", "--project-name", "framoryx-validation-test"], Path(directory), 0)
            self.assertEqual(status, 1)
            self.assertEqual(len(commands), 1)
            self.assertEqual(commands[0][-3:], ("down", "--volumes", "--remove-orphans"))
            self.assertFalse(json.loads((Path(directory) / "result.json").read_text())["success"])

    def test_cleanup_failure_blocks_success(self):
        with tempfile.TemporaryDirectory() as directory, patch.object(COMPOSE, "save_diagnostics"):
            with patch.object(COMPOSE, "command", side_effect=subprocess.CalledProcessError(1, "cleanup")):
                self.assertEqual(COMPOSE.finish_validation(["docker", "compose", "--project-name", "framoryx-validation-test"], Path(directory), 0), 1)


if __name__ == "__main__":
    unittest.main()
