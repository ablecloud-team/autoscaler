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

import "testing"

func workerFixture() *Cluster {
	yes, no := true, false
	return &Cluster{VirtualMachines: []*VirtualMachine{
		{ID: "control-id", Name: "control-name", State: "Running", ControlNode: &yes},
		{ID: "etcd-id", Name: "etcd-name", State: "Running", ControlNode: &no, EtcdNode: true},
		{ID: "external-id", Name: "external-name", State: "Running", ControlNode: &no, ExternalNode: true},
		{ID: "worker-id", Name: "worker-name", State: "Running", ControlNode: &no},
		{ID: "starting-id", Name: "starting-name", State: "Starting", ControlNode: &no},
		{ID: "removed-id", Name: "removed-name", State: "Expunging", ControlNode: &no},
		{ID: "untyped-id", Name: "untyped-name", State: "Running"},
	}}
}
func TestMoldWorkerInstancesExcludeOtherRolesAndRemovedVMs(t *testing.T) {
	c := workerFixture()
	workers := c.WorkerVirtualMachines()
	if len(workers) != 2 || workers[0].ID != "worker-id" || workers[1].ID != "starting-id" {
		t.Fatalf("wrong workers: %+v", workers)
	}
}
func TestMoldWorkerIdentityLookup(t *testing.T) {
	cases := []struct{ name, providerID, systemUUID, want string }{
		{"worker-name", "", "", "worker-id"}, {"worker-id", "", "", "worker-id"},
		{"worker-name", "external-cloudstack://worker-id", "", "worker-id"}, {"external-cloudstack://worker-id", "external-cloudstack://worker-id", "", "worker-id"}, {"worker-name", "cloudstack://worker-id", "", "worker-id"}, {"worker-name", "worker-id", "", "worker-id"},
		{"", "", "WORKER-ID", "worker-id"}, {"starting-name", "", "", "starting-id"},
		{"control-name", "", "", ""}, {"etcd-name", "", "", ""}, {"external-name", "", "", ""},
		{"removed-name", "", "", ""}, {"untyped-name", "", "", ""}, {"foreign-node", "", "foreign-id", ""},
		{"worker-name", "cloudstack://foreign-id", "", ""}, {"worker-name", "other://worker-id", "", ""},
		{"starting-name", "cloudstack://worker-id", "", ""}, {"control-name", "cloudstack://worker-id", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name+tc.providerID+tc.systemUUID, func(t *testing.T) {
			vm := workerFixture().FindWorkerVM(tc.name, tc.providerID, tc.systemUUID)
			got := ""
			if vm != nil {
				got = vm.ID
			}
			if got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}
func TestMoldDuplicateWorkerIdentityFailsClosed(t *testing.T) {
	c := workerFixture()
	no := false
	c.VirtualMachines = append(c.VirtualMachines, &VirtualMachine{ID: "worker-id", Name: "duplicate-name", State: "Running", ControlNode: &no})
	if c.FindWorkerVM("", "cloudstack://worker-id", "") != nil {
		t.Fatal("duplicate identity was accepted")
	}
}
