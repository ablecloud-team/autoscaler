#!/usr/bin/env python3
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

"""Apply shared worker identity bodies without replacing minor-specific interfaces."""
import pathlib,re,sys
root=pathlib.Path(sys.argv[1])
p=root/'cluster-autoscaler/cloudprovider/cloudstack/cloudstack_node_group.go'
t=p.read_text()
def method(text,name,body):
 pattern=r'(?ms)^(func \(asg \*asg\) '+name+r'\([^\n]+\{)\n.*?^}'
 result,n=re.subn(pattern,lambda m:m.group(1)+'\n'+body+'\n}',text)
 if n!=1:raise ValueError('unexpected minor interface: '+name)
 return result
old=re.search(r'(?ms)^func \(asg \*asg\) DeleteNodes\([^\n]+\{\n.*?^}',t).group(0)
minimum='asg.MinSize(context.TODO())' if 'asg.MinSize(context.TODO())' in old else 'asg.MinSize()'
t=method(t,'Belongs',""" vm := asg.cluster.FindWorkerVM(node.Name, node.Spec.ProviderID, node.Status.NodeInfo.SystemUUID)
 return vm != nil, nil""")
t=method(t,'Nodes',""" var instances []cloudprovider.Instance
 for _, vm := range asg.cluster.WorkerVirtualMachines() {
  instances = append(instances, cloudprovider.Instance{Id: vm.ID})
 }
 return instances, nil""")
t=method(t,'DeleteNodes',""" nodeIDs := make([]string, 0, len(nodes))
 seen := make(map[string]bool)
 for _, node := range nodes {
  if node == nil { return fmt.Errorf("Cannot delete a nil Kubernetes node") }
  vm := asg.cluster.FindWorkerVM(node.Name, node.Spec.ProviderID, node.Status.NodeInfo.SystemUUID)
  if vm == nil || seen[vm.ID] { return fmt.Errorf("Cannot delete unknown, non-worker or duplicate node %s", node.Name) }
  seen[vm.ID] = true
  nodeIDs = append(nodeIDs, vm.ID)
 }
 if len(nodeIDs) == 0 { return fmt.Errorf("No managed worker nodes requested for deletion") }
 if asg.cluster.WorkerCount-len(nodeIDs) < MINIMUM {
  return fmt.Errorf("Goes below minsize. Can not delete %v nodes", len(nodeIDs))
 }
 cluster, err := asg.manager.removeNodesFromCluster(asg.cluster.ID, nodeIDs...)
 if err != nil { return err }
 asg.Copy(cluster)
 return nil""".replace('MINIMUM',minimum))
p.write_text(t)
p=root/'cluster-autoscaler/cloudprovider/cloudstack/cloudstack_manager.go';t=p.read_text();old='_, err := manager.asg.Belongs(node)';new='belongs, err := manager.asg.Belongs(node)'
if t.count(old)!=1:raise ValueError('unexpected clusterForNode interface')
t=t.replace(old,new).replace('\treturn manager.asg, nil','\tif !belongs { return nil, nil }\n\treturn manager.asg, nil');p.write_text(t)
