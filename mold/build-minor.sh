#!/usr/bin/env bash
# Licensed under the Apache License, Version 2.0.
set -euo pipefail
MINOR=${1:?minor required}
OUTPUT=${2:?output required}
ROOT=$(git rev-parse --show-toplevel)
BASE=$(python3 - "$ROOT/mold/baselines.json" "$MINOR" <<'PY'
import json,sys
print(json.load(open(sys.argv[1]))[sys.argv[2]]['source_sha'])
PY
)
WORKTREE="$OUTPUT/source-$MINOR"
mkdir -p "$OUTPUT"
git fetch https://github.com/kubernetes/autoscaler.git "$BASE" --no-tags
git worktree add --detach "$WORKTREE" "$BASE" >/dev/null
trap 'git -C "$ROOT" worktree remove --force "$WORKTREE"' EXIT
SERVICE=cluster-autoscaler/cloudprovider/cloudstack/service
cp "$ROOT/$SERVICE/client.go" "$ROOT/$SERVICE/mold_signing_test.go" "$WORKTREE/$SERVICE/"
mkdir -p "$WORKTREE/$SERVICE/testdata"
cp "$ROOT/$SERVICE/testdata/mold-signing.json" "$WORKTREE/$SERVICE/testdata/"
cd "$WORKTREE/cluster-autoscaler"
go test -tags cloudstack -vet=atomic,bool,buildtags,directive,errorsas,ifaceassert,nilfunc,slog,stringintconv,tests ./cloudprovider/cloudstack/...
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -buildvcs=true -trimpath -ldflags='-s -w' -tags cloudstack -o "$OUTPUT/cluster-autoscaler" .
go version -m "$OUTPUT/cluster-autoscaler" > "$OUTPUT/go-modules.txt"
grep -F "vcs.revision=$BASE" "$OUTPUT/go-modules.txt" >/dev/null || { echo "ERROR: binary baseline source SHA is missing" >&2; exit 1; }
python3 - "$ROOT" "$MINOR" "$OUTPUT" <<'PY'
import json,sys,pathlib,subprocess,hashlib,os
root=pathlib.Path(sys.argv[1]);minor=sys.argv[2];out=pathlib.Path(sys.argv[3]);base=json.loads((root/'mold/baselines.json').read_text())[minor]
sha=subprocess.check_output(['git','-C',str(root),'rev-parse','HEAD'],text=True).strip()
files=['client.go','mold_signing_test.go','testdata/mold-signing.json']
patches={name:hashlib.sha256((root/'cluster-autoscaler/cloudprovider/cloudstack/service'/name).read_bytes()).hexdigest() for name in files}
(out/'provenance.json').write_text(json.dumps({'source_repository':os.environ.get('REPOSITORY','dhslove/autoscaler'),'source_sha':sha,'build_run':os.environ.get('RUN_URL','local'),'original_baseline':base['release'],'binary_source_sha':base['source_sha'],'baseline_status':base['status'],'kubernetes_minor':minor,'architecture':'amd64','api_signature':'HMAC-SHA256','customization_files':patches},indent=2)+'\n')
PY
