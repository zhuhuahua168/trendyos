// Copyright © 2021 sealos.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package install

import (
	"fmt"
	"path"
)

// SendPackage is
func (s *SealosInstaller) SendPackage() {
	pkg := path.Base(PkgURL)
	// rm old trendyos in package avoid old version problem. if trendyos not exist in package then skip rm
	kubeHook := fmt.Sprintf("cd /root && rm -rf kube && tar zxvf %s  && cd /root/kube/shell && bash init.sh && rm -rf /opt/cni/bin/* ", pkg)
	deletekubectl := `sed -i '/kubectl/d;/trendyos/d' /root/.bashrc `
	completion := "echo 'command -v kubectl &>/dev/null && source <(kubectl completion bash)' >> /root/.bashrc && echo '[ -x /usr/bin/trendyos ] && source <(trendyos completion bash)' >> /root/.bashrc && source /root/.bashrc"
	kubeHook = kubeHook + " && " + deletekubectl + " && " + completion
	PkgURL = SendPackage(PkgURL, s.Hosts, "/root", nil, &kubeHook)
}

// Sendtrendyos is send the exec trendyos to /usr/bin/trendyos
func (s *SealosInstaller) SendSealos() {
	// send trendyos first to avoid old version
	sealos := FetchSealosAbsPath()
	beforeHook := "ps -ef |grep -v 'grep'|grep trendyos >/dev/null || rm -rf /usr/bin/trendyos"
	afterHook := "chmod a+x /usr/bin/trendyos"
	SendPackage(sealos, s.Hosts, "/usr/bin", &beforeHook, &afterHook)
}

// SendPackage is send new pkg to all nodes.
func (u *SealosUpgrade) SendPackage() {
	all := append(u.Masters, u.Nodes...)
	pkg := path.Base(u.NewPkgURL)
	// rm old trendyos in package avoid old version problem. if trendyos not exist in package then skip rm
	var kubeHook string
	if For120(Version) {
		// TODO update need load modprobe -- br_netfilter modprobe -- bridge.
		// https://github.com/fanux/cloud-kernel/issues/23
		kubeHook = fmt.Sprintf("cd /root && rm -rf kube && tar zxvf %s  && cd /root/kube/shell  && (ctr -n=k8s.io image import ../images/images.tar || true) && cp -f ../bin/* /usr/bin/ ", pkg)
	} else {
		kubeHook = fmt.Sprintf("cd /root && rm -rf kube && tar zxvf %s  && cd /root/kube/shell  && (docker load -i ../images/images.tar || true) && cp -f ../bin/* /usr/bin/ ", pkg)
	}
	PkgURL = SendPackage(pkg, all, "/root", nil, &kubeHook)
}
