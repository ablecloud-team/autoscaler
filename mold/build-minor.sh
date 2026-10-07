#!/usr/bin/env bash
# Copyright The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail
MINOR=${1:?minor required}
OUTPUT=${2:?output required}
ROOT=$(git rev-parse --show-toplevel)
BASE=$(python3 - "$ROOT/mold/baselines.json" "$MINOR" <<'PY'
import json,sys
print(json.load(open(sys.argv[1]))[sys.argv[2]]['source_sha'])
PY
)
mkdir -p "$OUTPUT"
WORKTREE=$(mktemp -d "$OUTPUT/source-$MINOR.XXXXXX")
git fetch https://github.com/kubernetes/autoscaler.git "$BASE" --no-tags
# Go's VCS discovery requires a .git directory; a linked worktree .git file
# omits source metadata even with -buildvcs=true in the pinned toolchain.
trap 'rm -rf -- "$WORKTREE"' EXIT
git clone --shared --no-checkout "$ROOT" "$WORKTREE" >/dev/null
git -C "$WORKTREE" checkout --detach "$BASE" >/dev/null
SERVICE=cluster-autoscaler/cloudprovider/cloudstack/service
cp "$ROOT/$SERVICE/client.go" "$ROOT/$SERVICE/mold_signing_test.go" "$WORKTREE/$SERVICE/"
cp "$ROOT/$SERVICE/cks.go" "$ROOT/$SERVICE/mold_worker_identity.go" "$ROOT/$SERVICE/mold_worker_identity_test.go" "$WORKTREE/$SERVICE/"
cp "$ROOT/cluster-autoscaler/cloudprovider/cloudstack/cloudstack_option.go" "$ROOT/cluster-autoscaler/cloudprovider/cloudstack/mold_project_config_test.go" "$WORKTREE/cluster-autoscaler/cloudprovider/cloudstack/"
cp "$ROOT/$SERVICE/mold_project_scope_test.go" "$WORKTREE/$SERVICE/"
python3 "$ROOT/mold/apply-cloud-config.py" "$WORKTREE"
python3 "$ROOT/mold/apply-worker-identity.py" "$WORKTREE"
gofmt -w "$WORKTREE/$SERVICE/cks.go" "$WORKTREE/$SERVICE/mold_worker_identity.go" "$WORKTREE/$SERVICE/mold_worker_identity_test.go" "$WORKTREE/cluster-autoscaler/cloudprovider/cloudstack/cloudstack_node_group.go" "$WORKTREE/cluster-autoscaler/cloudprovider/cloudstack/cloudstack_manager.go" "$WORKTREE/cluster-autoscaler/cloudprovider/cloudstack/"*test.go
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
files=['client.go','mold_signing_test.go','testdata/mold-signing.json','cks.go','mold_worker_identity.go','mold_worker_identity_test.go']
patches={name:hashlib.sha256((root/'cluster-autoscaler/cloudprovider/cloudstack/service'/name).read_bytes()).hexdigest() for name in files}
for name in ['mold/apply-worker-identity.py','mold/worker-identity-provider-test.go.tmpl','mold/apply-cloud-config.py','cluster-autoscaler/cloudprovider/cloudstack/cloudstack_option.go','cluster-autoscaler/cloudprovider/cloudstack/mold_project_config_test.go','cluster-autoscaler/cloudprovider/cloudstack/service/mold_project_scope_test.go']:
 patches[name]=hashlib.sha256((root/name).read_bytes()).hexdigest()
(out/'provenance.json').write_text(json.dumps({'source_repository':os.environ.get('REPOSITORY','dhslove/autoscaler'),'source_sha':sha,'build_run':os.environ.get('RUN_URL','local'),'original_baseline':base['release'],'binary_source_sha':base['source_sha'],'baseline_status':base['status'],'kubernetes_minor':minor,'architecture':'amd64','api_signature':'HMAC-SHA256','customization_files':patches},indent=2)+'\n')
PY
