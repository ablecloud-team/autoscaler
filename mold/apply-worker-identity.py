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
  instances = append(instances, cloudprovider.Instance{Id: "external-cloudstack://" + vm.ID})
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

# Adapt the baseline tests to Mold's explicit node-role contract. Keep each
# minor's API signatures and imports, and still run its complete provider suite.
p=root/'cluster-autoscaler/cloudprovider/cloudstack/cloudstack_manager_test.go'
t=p.read_text().replace('nodeIDs = []string{"m1", "vm2"}', 'nodeIDs = []string{"vm1", "vm2"}')
t=t.replace('func createClusterDetails()', 'func moldControlNode(value bool) *bool { return &value }\n\nfunc createClusterDetails()')
t=re.sub(r'(ID:\s*"(m1|vm[1-5])",)', lambda m: m[1]+'\n                ControlNode: moldControlNode('+('true' if m[2]=='m1' else 'false')+'),', t)
start=t.index('func createScaleDownClusterDetails()');end=t.index('type mockCKSService',start)
part=t[start:end];part=re.sub(r'\n\s*\{\n\s*ID:\s*"vm2",.*?\n\s*\},', '',part,flags=re.S)
t=t[:start]+part+t[end:];p.write_text(t)
p=root/'cluster-autoscaler/cloudprovider/cloudstack/cloudstack_cloud_provider_test.go';t=p.read_text()
for name in ['testNodeNotExistWithName','testNodeNotExistWithoutName']:
 start=t.index('func '+name+'(');end=t.index('\n}',start)+2
 part=t[start:end].replace('_, err :=', 'group, err :=').replace('assert.NotEqual(t, nil, err)', 'assert.NoError(t, err)\n\tassert.Nil(t, group)')
 t=t[:start]+part+t[end:]
start=t.index('func testNodeExistsWithoutName(');end=t.index('\n}',start)+2
part=t[start:end].replace('ObjectMeta: metav1.ObjectMeta{\n\t\t\tName: "vm1",\n\t\t}', 'Status: v1.NodeStatus{NodeInfo: v1.NodeSystemInfo{SystemUUID: "vm1"}}')
t=t[:start]+part+t[end:];p.write_text(t)
p=root/'cluster-autoscaler/cloudprovider/cloudstack/cloudstack_node_group_test.go';t=p.read_text()
t=t.replace('len(clusterDetails.VirtualMachines)', 'len(clusterDetails.WorkerVirtualMachines())').replace('clusterDetails.VirtualMachines[i].ID', '"external-cloudstack://"+clusterDetails.WorkerVirtualMachines()[i].ID')
# Delete workers vm1/vm2, rather than allowing the former control-plane m1 case.
t=t.replace('SystemUUID: "m1"', 'SystemUUID: "vm1"')
p.write_text(t)
extra=(pathlib.Path(__file__).resolve().parent/'worker-identity-provider-test.go.tmpl').read_text()
delete='asg.DeleteNodes(context.Background(), nodes)' if 'DeleteNodes(context.Background(),' in t else 'asg.DeleteNodes(nodes)'
extra=extra.replace('DELETE_CALL',delete)
if 'context.Background()' not in extra:extra=extra.replace('    "context"\n','')
(root/'cluster-autoscaler/cloudprovider/cloudstack/mold_worker_identity_provider_test.go').write_text(extra)
