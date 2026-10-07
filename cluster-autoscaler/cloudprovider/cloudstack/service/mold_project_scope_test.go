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

package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

const fixtureProjectID = "d629aa4e-eb4a-43fb-abf6-45ad86cae21c"

func TestProjectScopeIsSignedAndCallerArgumentsRemainUnchanged(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		q := r.URL.Query()
		if q.Get("projectid") != fixtureProjectID || len(q["projectid"]) != 1 {
			t.Error("project scope missing or duplicated")
		}
		expected := "apikey=fixture&command=listkubernetesclusters&id=cluster-1&projectid=" + fixtureProjectID + "&response=json"
		mac := hmac.New(sha256.New, []byte("fixture"))
		mac.Write([]byte(expected))
		if q.Get("signature") != base64.StdEncoding.EncodeToString(mac.Sum(nil)) {
			t.Error("project scope was not covered by SHA256 signature")
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"listkubernetesclustersresponse": map[string]interface{}{"count": 1}})
	}))
	defer server.Close()
	c := NewAPIClient(&Config{APIKey: "fixture", SecretKey: "fixture", Endpoint: server.URL, ProjectID: fixtureProjectID})
	defer c.Close()
	args := map[string]string{"id": "cluster-1"}
	if _, err := c.NewRequest("listKubernetesClusters", args, nil); err != nil {
		t.Fatal(err)
	}
	if len(args) != 1 {
		t.Fatal("caller arguments were mutated")
	}
	for _, key := range []string{"projectid", "projectId"} {
		if _, err := c.NewRequest("listKubernetesClusters", map[string]string{key: "other-project"}, nil); err == nil {
			t.Fatal("foreign project was sent")
		}
	}
	if requests != 1 {
		t.Fatal("foreign project reached the API")
	}
}

func TestProjectScopeSurvivesMutationAndAsyncPolling(t *testing.T) {
	commands := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		cmd := q.Get("command")
		commands = append(commands, cmd)
		if q.Get("projectid") != fixtureProjectID {
			t.Error("async request lost project")
		}
		if cmd == "queryAsyncJobResult" {
			json.NewEncoder(w).Encode(map[string]interface{}{"queryasyncjobresultresponse": map[string]interface{}{"jobstatus": 1, "jobresult": map[string]interface{}{"kubernetescluster": map[string]interface{}{"id": "cluster-1"}}}})
		} else {
			json.NewEncoder(w).Encode(map[string]interface{}{cmd + "response": map[string]interface{}{"jobid": "job-1"}})
		}
	}))
	defer server.Close()
	for _, cmd := range []string{"scaleKubernetesCluster", "removeNodesFromKubernetesCluster"} {
		c := NewAPIClient(&Config{APIKey: "fixture", SecretKey: "fixture", Endpoint: server.URL, ProjectID: fixtureProjectID, Timeout: 5, PollInterval: 1})
		if _, err := c.NewRequest(cmd, map[string]string{"id": "cluster-1"}, nil); err != nil {
			t.Fatal(err)
		}
		c.Close()
	}
	if len(commands) != 4 {
		t.Fatalf("expected 2 mutations and 2 polls, got %v", commands)
	}
}
