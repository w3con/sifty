"""Policy tests for the leak scanner's two rule groups.

These are unit tests over leak_census: they prove task identifiers and work commands are not
forms, Russian wording is allowed in work records and tooling but not in application source, and
credential rules still apply regardless of the formatting area. No repository data is touched.
"""
import os
import sys
import unittest

SCANNER_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, SCANNER_DIR)
import leak_census  # noqa: E402

CENSUS = leak_census.parse_census()


def findings(text, apply_formatting=False):
    return leak_census.scan_text(
        text, CENSUS, apply_formatting=apply_formatting)


def irreversible(text):
    return [f for f in findings(text) if f.cls == "irreversible"]


class FormattingAreaTest(unittest.TestCase):
    def test_task_identifiers_and_work_commands_are_ordinary_content(self):
        line = "// OPS-4 uses bin/work-test and bin/work-commit for this step"
        self.assertEqual(findings(line, apply_formatting=True), [])

    def test_conventional_scope_may_be_uppercase(self):
        line = "docs(OPS-4): record the checkpoint"
        self.assertEqual(findings(line, apply_formatting=True), [])

    def test_russian_is_allowed_outside_application_source(self):
        text = "Проверить и записать результат"
        for path in ("ai/tasks/ops/OPS-4_demo/log.md",
                     "work/OPS-4_demo/log.md",
                     "bin/work-note",
                     ".githooks/pre-commit"):
            self.assertFalse(
                leak_census.is_formatting_required(path, CENSUS), path)
            self.assertEqual(findings(text, apply_formatting=False), [])

    def test_russian_is_refused_in_application_source(self):
        path = "sifty/internal/assess/assess.go"
        self.assertTrue(leak_census.is_formatting_required(path, CENSUS))
        found = findings("// Проверить и записать результат", apply_formatting=True)
        self.assertIn("Russian-language text", {f.form for f in found})

    def test_scanner_sources_are_formatting_exempt(self):
        for path in ("sifty/scripts/leakcheck/leak_census.py",
                     "scripts/leakcheck/census.md"):
            self.assertTrue(leak_census.formatting_exempt(path), path)


class CredentialRuleTest(unittest.TestCase):
    def test_environment_assignments_are_not_automatically_secrets(self):
        for line in ("WORK_SESSION=1", "SIFTY_PROD_PORT=8080",
                     "WORK_PUSH=0", "API_KEY=your-key-here",
                     "password = <changeme>", "api_key = ${SIFTY_API_KEY}"):
            self.assertEqual(irreversible(line), [], line)

    def test_real_secret_assignment_fails_everywhere(self):
        value = "9f8d7a6b5c4e" + "3f2a1b0c9d8e"
        line = "api_key = " + value
        for path in ("sifty/app.go", "tests/test_app.py",
                     "sifty/scripts/leakcheck/example.py",
                     "ai/tasks/demo/log.md", "bin/tool.py"):
            self.assertTrue(irreversible(line), path)

    def test_dotted_unquoted_secret_assignment_is_still_refused(self):
        value = "actual" + "." + "long" + "-" + "secret" + "123"
        for line in ("PASSWORD=" + value,
                     "SECRET_KEY=" + value,
                     "password: " + value):
            self.assertIn("secret assigned a real value",
                          {f.form for f in irreversible(line)}, line)

    def test_code_property_and_function_expressions_pass(self):
        for line in ("password: decodeURIComponent(proxy.password)",
                     "password: props.password,",
                     "password: config.proxy.password }"):
            self.assertEqual(irreversible(line), [], line)

    def test_credential_url_fails(self):
        line = "postgres" + "://app:" + "s3cr3t" + "value" + "@db.internal/app"
        self.assertIn("credentials in a URL",
                      {f.form for f in irreversible(line)})

    def test_pem_block_fails_but_pattern_definition_does_not(self):
        block = ("-----BEGIN " + "RSA PRIVATE KEY" + "-----\n"
                 + "MIIE" + "A" * 40 + "\n"
                 + "-----END " + "RSA PRIVATE KEY" + "-----")
        self.assertIn("private key (PEM block)",
                      {f.form for f in irreversible(block)})
        # The engine's own regex definition must not match itself.
        with open(os.path.join(SCANNER_DIR, "leak_census.py"),
                  encoding="utf-8") as fh:
            source = fh.read()
        self.assertEqual(
            [f for f in findings(source) if f.form == "private key (PEM block)"], [])

    def test_diagnostics_redact_the_matched_value(self):
        token = "ghp_" + "a1B2c3D4e5F6g7H8i9J0k1L2m3N4o5P6q7R8"
        finding = leak_census.Finding(
            "personal access token", "irreversible", 1, "token: " + token, "")
        rendered = leak_census.format_findings(
            [finding], "path/" + token + "/file")
        self.assertNotIn(token, rendered)


if __name__ == "__main__":
    unittest.main()
