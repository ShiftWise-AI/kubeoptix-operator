#!/usr/bin/env bash
# Embed config/manifests/logo.png into the ClusterServiceVersion icon field.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOGO="${ROOT_DIR}/config/manifests/logo.png"
CSV="${ROOT_DIR}/config/manifests/bases/shiftwise-operator.clusterserviceversion.yaml"

[[ -f "${LOGO}" ]] || { echo "error: missing ${LOGO}" >&2; exit 1; }
[[ -f "${CSV}" ]] || { echo "error: missing ${CSV}" >&2; exit 1; }

python3 - "${LOGO}" "${CSV}" <<'PY'
import base64
import pathlib
import sys

logo_path = pathlib.Path(sys.argv[1])
csv_path = pathlib.Path(sys.argv[2])
b64 = base64.b64encode(logo_path.read_bytes()).decode("ascii")

lines = csv_path.read_text().splitlines(keepends=True)
out = []
replaced = False
for line in lines:
    if line.lstrip().startswith("base64data:"):
        indent = line[: len(line) - len(line.lstrip())]
        out.append(f"{indent}base64data: {b64}\n")
        replaced = True
    else:
        out.append(line)

if not replaced:
    sys.exit("error: CSV has no base64data field")

csv_path.write_text("".join(out))
print(f"updated icon in {csv_path} ({len(b64)} chars)")
PY
