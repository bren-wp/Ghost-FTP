#!/usr/bin/env python3
"""Fail-closed contract for Linux AskPass executable identity and capability emission."""

from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


def read(path: str) -> str:
    target = ROOT / path
    if not target.is_file():
        raise AssertionError(f"missing required AskPass security source: {path}")
    return target.read_text(encoding="utf-8")


def go_function_body(source: str, signature: str) -> str:
    start = source.find(signature)
    if start < 0:
        raise AssertionError(f"missing Go function signature: {signature}")
    brace = source.find("{", start)
    if brace < 0:
        raise AssertionError(f"missing Go function body: {signature}")
    depth = 0
    for pos in range(brace, len(source)):
        char = source[pos]
        if char == "{":
            depth += 1
        elif char == "}":
            depth -= 1
            if depth == 0:
                return source[brace + 1 : pos]
    raise AssertionError(f"unterminated Go function body: {signature}")


class AskPassExecutableIdentityContractTests(unittest.TestCase):
    def test_linux_askpass_uses_proc_only_as_identity_oracle(self):
        source = read("internal/platform/askpass_executable_linux.go")
        stable = go_function_body(source, "func StableAskPassExecutable(exePath string) (string, error)")
        identity = go_function_body(source, "func trustedAskPassExecutableIdentity(exePath, runningPath string) (string, bool)")
        same = go_function_body(source, "func SameExecutableIdentity(_ string, expected string) bool")

        self.assertIn('const linuxRunningExecutable = "/proc/self/exe"', source)
        self.assertIn("linuxtrust.TrustedExecutable(exePath)", identity)
        self.assertIn("os.SameFile(trustedInfo, runningInfo)", identity)
        self.assertIn("return trustedPath, nil", stable)
        self.assertNotIn("return linuxRunningExecutable", stable)
        self.assertNotIn('filepath.Join("/proc"', source)
        self.assertIn("linuxtrust.TrustedExecutable(strings.TrimSpace(expected))", same)
        self.assertIn("os.SameFile(currentInfo, expectedInfo)", same)

    def test_application_does_not_fail_startup_when_askpass_is_unavailable(self):
        source = read("cmd/ghostftp/main.go")
        self.assertIn("askpassExe, _ := platform.StableAskPassExecutable(exe)", source)
        self.assertIn("api.New(dataDir, askpassExe)", source)
        self.assertNotIn("showError(\"Ghost FTP could not establish a safe authentication helper identity.\")", source)

    def test_sftp_blocks_capability_environment_before_token_generation(self):
        source = read("internal/remote/sftp.go")
        body = go_function_body(source, "func (s *SFTP) askpassEnvironment() ([]string, error)")
        no_helper = body.find('strings.TrimSpace(s.exePath) == ""')
        token_generation = body.find("rand.Read(tokenBytes)")
        env_append = body.find('"GhostFTP_ASKPASS_TOKEN="+token')
        self.assertGreaterEqual(no_helper, 0)
        self.assertGreater(token_generation, no_helper)
        self.assertGreater(env_append, token_generation)
        self.assertIn("return nil, errors.New", body[no_helper:token_generation])

    def test_regressions_cover_trusted_identity_and_env_boundary(self):
        platform_tests = read("internal/platform/askpass_executable_linux_test.go")
        remote_tests = read("internal/remote/sftp_askpass_boundary_test.go")
        cmd_tests = read("cmd/ghostftp/askpass_identity_linux_test.go")
        for name in (
            "TestTrustedAskPassExecutableIdentityAcceptsTrustedSameImage",
            "TestTrustedAskPassExecutableIdentityRejectsUserControlledPath",
            "TestTrustedAskPassExecutableIdentityRejectsDifferentInode",
        ):
            self.assertIn(name, platform_tests)
        for name in (
            "TestAskpassEnvironmentRejectsCredentialWithoutTrustedHelper",
            "TestAskpassEnvironmentWithoutCredentialsDoesNotRequireHelper",
        ):
            self.assertIn(name, remote_tests)
        self.assertIn("TestValidAskpassInvocationRejectsDifferentExecutableIdentity", cmd_tests)


if __name__ == "__main__":
    suite = unittest.defaultTestLoader.loadTestsFromTestCase(AskPassExecutableIdentityContractTests)
    result = unittest.TextTestRunner(verbosity=2).run(suite)
    if not result.wasSuccessful():
        raise SystemExit(1)
    print("LINUX_ASKPASS_EXECUTABLE_TOCTOU=BLOCKED")
    print("LINUX_ASKPASS_PROC_EXECUTION=BLOCKED")
    print("LINUX_ASKPASS_UNTRUSTED_HELPER_ENV=BLOCKED")
