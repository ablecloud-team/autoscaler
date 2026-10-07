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

import "strings"

// WorkerVirtualMachines returns only typed, managed worker instances. Missing
// role metadata fails closed until the Mold node-role API contract is present.
func (cluster *Cluster) WorkerVirtualMachines() []*VirtualMachine {
	var workers []*VirtualMachine
	if cluster == nil {
		return workers
	}
	for _, vm := range cluster.VirtualMachines {
		if vm == nil || vm.ID == "" || vm.ControlNode == nil || *vm.ControlNode || vm.EtcdNode || vm.ExternalNode {
			continue
		}
		if vm.State == "Destroyed" || vm.State == "Expunging" {
			continue
		}
		workers = append(workers, vm)
	}
	return workers
}

// FindWorkerVM supports real Node names and the UUID identities used by the
// autoscaler core for synthetic cloud instances. An explicit provider ID is
// authoritative; it never falls back to a matching name after a mismatch.
func (cluster *Cluster) FindWorkerVM(name, providerID, systemUUID string) *VirtualMachine {
	identity := strings.TrimPrefix(strings.TrimPrefix(providerID, "external-cloudstack://"), "cloudstack://")
	if providerID != "" && strings.Contains(identity, "://") {
		return nil
	}
	var found *VirtualMachine
	for _, vm := range cluster.WorkerVirtualMachines() {
		match := false
		if providerID != "" {
			match = strings.EqualFold(vm.ID, identity)
		} else {
			match = (name != "" && (strings.EqualFold(vm.Name, name) || strings.EqualFold(vm.ID, name))) || (systemUUID != "" && strings.EqualFold(vm.ID, systemUUID))
		}
		if !match {
			continue
		}
		if found != nil {
			return nil
		}
		found = vm
	}
	if found != nil {
		// Reject a conflicting identity that names another VM in this cluster.
		for _, vm := range cluster.VirtualMachines {
			if vm != nil && vm.ID != found.ID && name != "" && (strings.EqualFold(vm.Name, name) || strings.EqualFold(vm.ID, name)) {
				return nil
			}
		}
	}
	return found
}
