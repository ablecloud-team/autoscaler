#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
# Licensed under the Apache License, Version 2.0.
# https://www.apache.org/licenses/LICENSE-2.0
"""Scan repository headers and verify exact inherited-license exceptions."""
import argparse
import hashlib
import json
import pathlib
import re
import subprocess

def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()

def member(root, name):
    p = root / name
    if not name or pathlib.PurePosixPath(name).is_absolute() or p.is_symlink() or p.resolve() != p.absolute() or root not in p.resolve().parents:
        raise ValueError("Unsafe license coverage path: " + name)
    if not p.is_file():
        raise ValueError("Missing covered file: " + name)
    return p

def verify_coverage(root, ledger):
    paths = set()
    notices = {row["licenseFile"]: row["licenseSha256"] for row in ledger["files"]}
    if ledger["licenseNotices"] != notices:
        raise ValueError("License notice inventory differs from reviewed coverage")
    for row in ledger["files"]:
        name = row["path"]
        if name in paths or any(x in name for x in "*?[]") or "\\n" in name or "\\r" in name or "\\\\" in name:
            raise ValueError("Duplicate or wildcard coverage path: " + name)
        paths.add(name)
        if digest(member(root, name)) != row["sha256"]:
            raise ValueError("Covered source changed; review its inherited license: " + name)
        notice = member(root, row["licenseFile"])
        if notice.name.upper() not in {"LICENSE", "LICENSE.TXT", "LICENSE.MD", "LICENSE.RST", "LICENCE", "LICENCE.TXT", "COPYING", "COPYING.TXT", "LICENSE-MIT", "LICENSE-APACHE"}:
            raise ValueError("Coverage must reference an actual license notice: " + row["licenseFile"])
        if digest(notice) != row["licenseSha256"]:
            raise ValueError("License notice changed: " + row["licenseFile"])
        if row["reason"] not in ("inherited-project-metadata", "preserved-third-party-notice", "upstream-oci-project-license"):
            raise ValueError("Unreviewed license coverage reason: " + name)
        if row["reason"] == "inherited-project-metadata" and pathlib.PurePosixPath(name).suffix in (".go", ".py", ".sh", ".java", ".c", ".cc", ".rs", ".js", ".ts"):
            raise ValueError("First-party source requires a recognized header: " + name)
        if row["reason"] == "upstream-oci-project-license" and (not name.startswith("cluster-autoscaler/cloudprovider/oci/") or "/vendor-internal/" in name or not name.endswith(".go")):
            raise ValueError("OCI exception cannot cover another component: " + name)
        if row["reason"] == "preserved-third-party-notice" and row["licenseFile"] == "LICENSE":
            raise ValueError("External source must preserve its own license: " + name)
        if row["licenseFile"] != "LICENSE" and notice.parent not in member(root, name).parents:
            raise ValueError("License notice is outside component ancestry: " + name)
    return paths

def scan(root, rat_jar, output):
    ledger = json.loads(member(root, "mold/license-coverage.json").read_text())
    paths = verify_coverage(root, ledger)
    output.mkdir(parents=True, exist_ok=True)
    excludes = output / "inherited-license-paths.txt"
    excludes.write_text("\n".join(sorted(paths)) + "\n")
    report = output / "rat-report.txt"
    with report.open("w") as stream:
        process = subprocess.run(["java", "-jar", str(rat_jar), "--input-exclude-file", str(excludes), "--", str(root)], stdout=stream, stderr=subprocess.STDOUT)
    text = report.read_text()
    count = re.search(r"^\s*[!]?[ ]*Unapproved:\s*(\d+)\s+A count of unapproved licenses\.", text, re.M)
    if process.returncode or not count or int(count.group(1)) != 0:
        raise ValueError("Repository license headers failed; see " + str(report))
    inventory = []
    for path in sorted(root.rglob("*")):
        if path.is_file():
            if path.is_symlink():
                raise ValueError("Unreviewed symlink in source archive")
            name = path.relative_to(root).as_posix()
            inventory.append({"path": name, "sha256": digest(path), "inheritedLicenseVerified": name in paths})
    result = {"schemaVersion": 1, "trackedArchiveFiles": len(inventory), "verifiedInheritedLicenseFiles": len(paths), "ratUnapproved": 0, "licenseNotices": ledger["licenseNotices"], "files": inventory}
    (output / "license-coverage-report.json").write_text(json.dumps(result, indent=2) + "\n")
    print(json.dumps({k:v for k,v in result.items() if k != "files"}))

if __name__ == "__main__":
    p = argparse.ArgumentParser()
    p.add_argument("--source", type=pathlib.Path, required=True)
    p.add_argument("--rat-jar", type=pathlib.Path, required=True)
    p.add_argument("--output", type=pathlib.Path, required=True)
    a = p.parse_args()
    scan(a.source.resolve(), a.rat_jar.resolve(), a.output.resolve())
