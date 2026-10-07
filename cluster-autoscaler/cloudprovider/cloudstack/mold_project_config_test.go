/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cloudstack

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCloudConfigProjectIDReachesApiClient(t *testing.T) {
	file := filepath.Join(t.TempDir(), "cloud-config")
	body := "[Global]\napi-url=http://127.0.0.1:1/api\napi-key=fixture\nsecret-key=fixture\nproject-id=d629aa4e-eb4a-43fb-abf6-45ad86cae21c\n"
	if err := os.WriteFile(file, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := createConfig(withConfigFile(file))
	if err != nil {
		t.Fatal(err)
	}
	defer cfg.service.Close()
	if cfg.acsConfig.ProjectID != "d629aa4e-eb4a-43fb-abf6-45ad86cae21c" {
		t.Fatal("project setting was discarded")
	}
}
