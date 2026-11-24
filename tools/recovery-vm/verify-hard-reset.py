#!/usr/bin/env python3
"""Hard-power-off recovery test for explicitly designated disposable libvirt VMs."""
import argparse
import base64
import json
import os
from pathlib import Path, PurePosixPath, PureWindowsPath
import re
import shlex
import subprocess
import tempfile
import time
import uuid

CASES = (
    "prepared_sql_committed", "backup_renamed", "stage_published",
    "backup_removed", "terminal_manifest_written", "rollback:rollback_restored",
)


def run(args, timeout=30):
    return subprocess.check_output(args, text=True, stderr=subprocess.STDOUT,
                                   timeout=timeout).strip()


def validate_domain(name, expected, confirmation):
    if not re.fullmatch(r"ghostftp-durability-[a-zA-Z0-9_-]+", name):
        raise ValueError("only a dedicated ghostftp-durability-* domain is permitted")
    canonical = str(uuid.UUID(expected))
    if confirmation != canonical:
        raise ValueError("--confirm-reset must equal the exact disposable domain UUID")
    return canonical


def guest_path(platform, root, run_id):
    cls = PureWindowsPath if platform == "windows" else PurePosixPath
    base = cls(root)
    if not base.is_absolute() or ".." in base.parts or len(base.parts) < 2:
        raise ValueError("guest root must be a dedicated absolute non-root directory")
    if platform == "windows" and base.drive.startswith("\\\\"):
        raise ValueError("UNC guest root not supported")
    return base / ("ghostftp-vm-" + run_id)


def psquote(value):
    return "'" + str(value).replace("'", "''") + "'"


def guest_command(platform, operation, root, binary, point, dbus):
    values = {
        "GHOSTFTP_VM_TEST_OPT_IN": "disposable-guest",
        "GHOSTFTP_VM_RECOVERY_ROOT": str(root),
        "GHOSTFTP_VM_RECOVERY_MODE": operation,
        "GHOSTFTP_VM_RECOVERY_POINT": point,
    }
    args = ["vm_recovery_hard_reset_probe", "--ignored", "--nocapture"]
    if platform == "linux":
        if operation == "boot-id":
            return "cat /proc/sys/kernel/random/boot_id"
        if operation == "checkpoint":
            return shlex.join(["cat", "--", str(root / "vault" / "checkpoint")])
        if dbus:
            values["DBUS_SESSION_BUS_ADDRESS"] = dbus
        return shlex.join(["env", *[k + "=" + v for k, v in values.items()], binary, *args])
    if operation == "boot-id":
        script = "(Get-CimInstance Win32_OperatingSystem).LastBootUpTime.ToUniversalTime().Ticks"
    elif operation == "checkpoint":
        script = "Get-Content -Raw -LiteralPath " + psquote(root / "vault" / "checkpoint")
    else:
        script = "; ".join("$env:" + key + "=" + psquote(value)
                           for key, value in values.items())
        script += "; & " + psquote(binary) + " " + " ".join(map(psquote, args))
        script += "; exit $LASTEXITCODE"
    encoded = base64.b64encode(("$ErrorActionPreference='Stop'; " + script)
                              .encode("utf-16-le")).decode("ascii")
    return "powershell.exe -NoProfile -NonInteractive -EncodedCommand " + encoded


def await_value(operation, accepted, seconds, child=None):
    deadline = time.monotonic() + seconds
    while time.monotonic() < deadline:
        if child is not None and child.poll() is not None:
            raise RuntimeError("guest stopped before reaching the selected checkpoint")
        try:
            value = operation()
            if accepted(value):
                return value
        except (subprocess.CalledProcessError, subprocess.TimeoutExpired):
            pass
        time.sleep(1)
    raise TimeoutError("guest checkpoint or new boot was not observed")


def exercise(args, report):
    expected = validate_domain(args.domain, args.uuid, args.confirm_reset)
    if args.ssh_target.startswith("-") or not args.ssh_target:
        raise ValueError("invalid SSH target")
    root = guest_path(args.platform, args.guest_root, report["run_id"])
    report.update(domain_uuid=expected, guest_root=str(root))
    virsh = ["virsh", "-c", args.libvirt_uri]
    ssh = ["ssh", "-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=yes",
           "-o", "ConnectTimeout=5", args.ssh_target]
    os.environ["LC_ALL"] = "C"

    def check_domain():
        if run(virsh + ["domuuid", args.domain]) != expected:
            raise RuntimeError("libvirt domain UUID mismatch")
        if run(virsh + ["domname", expected]) != args.domain:
            raise RuntimeError("libvirt domain name mismatch")
        if run(virsh + ["domstate", expected]) != "running":
            raise RuntimeError("disposable domain is not running")

    def command(operation):
        return ssh + [guest_command(args.platform, operation, root,
                                    args.test_binary, args.case, args.linux_dbus_address)]

    check_domain()
    boot = run(command("boot-id"), 15)
    if not boot:
        raise RuntimeError("guest boot identity missing")
    report["boot_before"] = boot
    prepared = run(command("prepare"), args.timeout)
    if "GHOSTFTP_VM_PREPARE_OK" not in prepared:
        raise RuntimeError("guest preparation did not run the expected test")
    # Keep child output in a temporary file instead of a blocking stdout pipe.
    with tempfile.TemporaryFile() as log:
        child = subprocess.Popen(command("crash"), stdout=log, stderr=log)
        try:
            await_value(lambda: run(command("checkpoint"), 15),
                        lambda value: value == args.case, args.timeout, child)
            report["checkpoint_observed"] = True
            check_domain()
            # This is the ONLY destructive host action. All identities were
            # checked again immediately before issuing the forced power-off.
            run(virsh + ["destroy", expected])
            report["hard_power_off"] = True
            if run(virsh + ["domstate", expected]) != "shut off":
                raise RuntimeError("domain did not reach powered-off state")
            run(virsh + ["start", expected], 60)
            report["boot_after"] = await_value(
                lambda: run(command("boot-id"), 15),
                lambda value: bool(value) and value != boot, args.timeout)
            check_domain()
            verified = run(command("verify"), args.timeout)
            report["verification_output"] = verified
            if "GHOSTFTP_VM_VERIFY_OK" not in verified:
                raise RuntimeError("guest did not confirm recovery verification")
            report["verified"] = True
            run(command("cleanup"), args.timeout)
            report["test_credentials_cleaned"] = True
        finally:
            if child.poll() is None:
                child.kill()
            child.wait()
            log.seek(0)
            report["guest_crash_log"] = log.read(65536).decode("utf-8", errors="replace")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--domain", required=True)
    parser.add_argument("--uuid", required=True)
    parser.add_argument("--confirm-reset", required=True)
    parser.add_argument("--libvirt-uri", default="qemu:///system")
    parser.add_argument("--ssh-target", required=True)
    parser.add_argument("--platform", choices=["linux", "windows"], required=True)
    parser.add_argument("--guest-root", required=True)
    parser.add_argument("--test-binary", required=True)
    parser.add_argument("--linux-dbus-address")
    parser.add_argument("--case", choices=CASES, required=True)
    parser.add_argument("--timeout", type=int, default=300)
    parser.add_argument("--report", type=Path, required=True)
    args = parser.parse_args()
    if not 30 <= args.timeout <= 1800:
        parser.error("timeout must be between 30 and 1800 seconds")
    report = {"run_id": str(uuid.uuid4()), "platform": args.platform,
              "case": args.case, "verified": False}
    started = time.monotonic()
    # Refuse to overwrite another run's evidence; reserve path before reset.
    with args.report.open("x", encoding="utf-8") as output:
        try:
            exercise(args, report)
        except Exception as error:
            report["error"] = str(error)
            raise
        finally:
            report["elapsed_seconds"] = round(time.monotonic() - started, 3)
            json.dump(report, output, indent=2)
            output.write("\n")


if __name__ == "__main__":
    main()
