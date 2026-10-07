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

"""Apply the Mold shared cloud-config identity contract across pinned minors."""
import pathlib,sys
root=pathlib.Path(sys.argv[1]);p=root/'cluster-autoscaler/cloudprovider/cloudstack/cloudstack_manager.go';t=p.read_text()
if '\t"fmt"' not in t:t=t.replace('import (\n','import (\n\t"fmt"\n',1)
needle='APIURL      string `gcfg:"api-url"`'
if t.count(needle)!=1:raise ValueError('unexpected cloud-config APIURL field')
t=t.replace(needle,needle+'\n\t\tClusterUID  string `gcfg:"cluster-uuid"`')
needle='\tcfg.asg.cluster = &service.Cluster{'
if t.count(needle)!=1:raise ValueError('unexpected manager cluster binding')
t=t.replace(needle,'\tif cfg.clusterUID != "" && cfg.clusterUID != clusterConfig.clusterID {\n\t\tcfg.service.Close()\n\t\treturn nil, fmt.Errorf("cloud-config cluster-uuid does not match --nodes cluster ID")\n\t}\n\n'+needle)
p.write_text(t)
