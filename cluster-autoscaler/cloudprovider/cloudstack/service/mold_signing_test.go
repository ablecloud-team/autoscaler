/*
Copyright The Kubernetes Authors.

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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

func TestMoldSigningVector(t *testing.T) {
	var v struct {
		Secret, Canonical, Signature string
		Params                       map[string]string
	}
	b, err := os.ReadFile("testdata/mold-signing.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	args := map[string]string{}
	for k, x := range v.Params {
		if k != "apiKey" && k != "command" && k != "response" {
			args[k] = x
		}
	}
	c := &client{config: &Config{APIKey: v.Params["apiKey"], SecretKey: v.Secret}}
	query := c.createQueryString(v.Params["command"], args)
	if canonical := strings.ToLower(strings.Split(query, "&signature=")[0]); canonical != v.Canonical {
		t.Fatalf("canonical %q", canonical)
	}
	p, err := url.ParseQuery(query)
	if err != nil {
		t.Fatal(err)
	}
	if p.Get("signature") != v.Signature {
		t.Fatalf("signature %s", p.Get("signature"))
	}
}
func TestMalformedMoldResponsesDoNotPanic(t *testing.T) {
	for _, body := range []string{`bad-json`, `{"response":[]}`, `{"response":{"jobid":23}}`, `{"response":{"jobid":""}}`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) }))
		c := NewAPIClient(&Config{Endpoint: server.URL, APIKey: "dummy", SecretKey: "dummy"})
		if _, err := c.NewRequest("scaleKubernetesCluster", nil, nil); err == nil {
			t.Errorf("expected failure for %s", body)
		}
		c.Close()
		server.Close()
	}
}
