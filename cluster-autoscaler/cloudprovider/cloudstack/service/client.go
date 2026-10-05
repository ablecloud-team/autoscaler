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
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	klog "k8s.io/klog/v2"
)

// APIClient is an interface to communicate to CloudStack via HTTP calls
type APIClient interface {
	// NewRequest makes an API request to configured management server
	NewRequest(api string, args map[string]string, out interface{}) (map[string]interface{}, error)

	// Close terminates the client
	Close()
}

// Config contains the parameters used to configure a new APIClient
type Config struct {
	APIKey       string
	SecretKey    string
	Endpoint     string
	Timeout      int
	PollInterval int
}

// client implements the APIClient interface
type client struct {
	config    *Config
	client    *http.Client
	interrupt chan struct{}
}

// QueryAsyncJobResponse is the response returned by the queryAsyncJobResult API
type QueryAsyncJobResponse struct {
	JobResponse *JobResponse `json:"queryasyncjobresultresponse"`
}

// JobResponse contains the async job details
type JobResponse struct {
	JobStatus float64     `json:"jobstatus"`
	JobID     string      `json:"jobid"`
	JobResult interface{} `json:"jobresult"`
}

func (client *client) getResponseData(data map[string]interface{}) map[string]interface{} {
	for k := range data {
		if strings.HasSuffix(k, "response") {
			if response, ok := data[k].(map[string]interface{}); ok {
				return response
			}
		}
	}
	return nil
}

func (client *client) pollAsyncJob(jobID string, out interface{}) (map[string]interface{}, error) {
	timeout := time.NewTimer(time.Duration(client.config.Timeout) * time.Second)
	ticker := time.NewTicker(time.Duration(client.config.PollInterval) * time.Second)

	defer ticker.Stop()
	defer timeout.Stop()

	for {
		select {
		case <-client.interrupt:
			return nil, fmt.Errorf("Client interrupted")

		case <-timeout.C:
			return nil, fmt.Errorf("Timed out getting result for jobid : %s", jobID)

		case <-ticker.C:
			result, err := client.newRequest("queryAsyncJobResult", map[string]string{
				"jobid": jobID,
			}, false, out)
			if err != nil {
				return result, err
			}

			status, ok := result["jobstatus"].(float64)
			if !ok {
				return nil, errors.New("missing async job status")
			}
			switch {
			case status == 0:
				klog.Info("Still waiting for job " + jobID + " to complete")
				continue
			case status == 1:
				data, err := json.Marshal(result["jobresult"])
				if err != nil {
					return nil, err
				}
				if out != nil {
					if err := json.Unmarshal(data, out); err != nil {
						return nil, err
					}
				}
				jobResult, ok := result["jobresult"].(map[string]interface{})
				if !ok {
					return nil, errors.New("invalid async job result")
				}
				return jobResult, nil
			case status > 1:
				err := fmt.Errorf("API failed for job %s : %v", jobID, result)
				klog.Error(err)
				return result, fmt.Errorf("API failed for job %s : %v", jobID, result)
			}
		}
	}
}

// NewRequest makes an API request to configured management server
func (client *client) NewRequest(api string, args map[string]string, out interface{}) (map[string]interface{}, error) {
	return client.newRequest(api, args, true, out)
}

// Close terminates the client
func (client *client) Close() {
	close(client.interrupt)
}

func (client *client) createQueryString(api string, args map[string]string) string {
	params := make(url.Values)
	for key, value := range args {
		params.Add(key, value)
	}

	params.Add("command", api)
	params.Add("response", "json")

	params.Add("apiKey", client.config.APIKey)
	encodedParams := encodeMoldValues(params)

	mac := hmac.New(sha256.New, []byte(client.config.SecretKey))
	mac.Write([]byte(strings.Replace(strings.ToLower(encodedParams), "+", "%20", -1)))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	encodedParams = fmt.Sprintf("%s&signature=%s", encodedParams, url.QueryEscape(signature))

	return encodedParams
}

// encodeMoldValues matches Java URLEncoder in Mold ApiServer: sort keys,
// encode values only, preserve '*', encode '~', and use %20 for spaces.
func encodeMoldValues(params url.Values) string {
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		value := url.QueryEscape(params.Get(key))
		value = strings.NewReplacer("+", "%20", "%2A", "*", "~", "%7E").Replace(value)
		parts = append(parts, key+"="+value)
	}
	return strings.Join(parts, "&")
}

func createMaskedURL(url string) string {
	maskedKeys := []string{"apiKey", "signature"}
	for _, x := range maskedKeys {
		r, _ := regexp.Compile(x + "=.*?(&|$)")
		url = r.ReplaceAllString(url, x+"=***$1")
	}
	return url
}

func (client *client) newRequest(api string, args map[string]string, async bool, out interface{}) (map[string]interface{}, error) {
	params := client.createQueryString(api, args)

	requestURL := fmt.Sprintf("%s?%s", client.config.Endpoint, params)
	maskedURL := createMaskedURL(requestURL)
	klog.Info("NewAPIRequest API request URL:", maskedURL)

	response, err := client.client.Get(requestURL)
	if err != nil {
		return nil, err
	}
	klog.Info("NewAPIRequest response status code:", response.StatusCode)

	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("invalid API response (HTTP %d): %w", response.StatusCode, err)
	}

	if data != nil && async {
		if jobResponse := client.getResponseData(data); jobResponse != nil && jobResponse["jobid"] != nil {
			jobID, ok := jobResponse["jobid"].(string)
			if !ok || jobID == "" {
				return nil, errors.New("invalid async job id")
			}
			return client.pollAsyncJob(jobID, out)
		}
	}

	if apiResponse := client.getResponseData(data); apiResponse != nil {
		if _, ok := apiResponse["errorcode"]; ok {
			return nil, fmt.Errorf("(HTTP %v, error code %v) %v", apiResponse["errorcode"], apiResponse["cserrorcode"], apiResponse["errortext"])
		}
		if out != nil {
			if err := json.Unmarshal(body, out); err != nil {
				return nil, err
			}
		}
		return apiResponse, nil
	}

	return nil, errors.New("failed to decode response")
}

func newHTTPClient() *http.Client {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
		Transport: &http.Transport{
			Proxy:           http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	return client
}

// NewAPIClient returns a new APIClient
func NewAPIClient(config *Config) APIClient {
	if config.Timeout <= 0 {
		config.Timeout = 3600
	}
	if config.PollInterval <= 0 {
		config.PollInterval = 10
	}
	httpClient := newHTTPClient()
	httpClient.Timeout = time.Duration(time.Duration(config.Timeout) * time.Second)

	return &client{
		config:    config,
		client:    httpClient,
		interrupt: make(chan struct{}),
	}
}
