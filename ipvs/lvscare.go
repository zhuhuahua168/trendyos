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

package ipvs

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/fanux/sealos/pkg/logger"
)

type LvscareImage struct {
	Image string
	Tag   string
}

func (l *LvscareImage) toImageName() string {
	return l.Image + ":" + l.Tag
}

// return lvscare static pod yaml
//func LvsStaticPodYaml(vip string, masters []string, image LvscareImage) string {
//	if vip == "" || len(masters) == 0 {
//		return ""
//	}
//	args := []string{"care", "--vs", vip + ":6443", "--health-path", "/healthz", "--health-schem", "https"}
//	for _, m := range masters {
//		if strings.Contains(m, ":") {
//			m = strings.Split(m, ":")[0]
//		}
//		args = append(args, "--rs")
//		args = append(args, m+":6443")
//	}
//	flag := true
//	pod := componentPod(v1.Container{
//		Name:            "kube-sealyun-lvscare",
//		Image:           image.toImageName(),
//		Command:         []string{"/usr/bin/lvscare"},
//		Args:            args,
//		ImagePullPolicy: v1.PullIfNotPresent,
//		SecurityContext: &v1.SecurityContext{Privileged: &flag},
//	})
//	logger.Info("lvscare static pod yaml %s", pod)
//	yaml, err := podToYaml(pod)
//	if err != nil {
//		logger.Error("encode lvscare static pod yaml failed %s", err)
//		return ""
//	}
//
//	return string(yaml)
//}

func LvsStaticPodYaml(vip string, masters []string, image LvscareImage) string {
	logger.Info("lvscare static pod yaml %s", image)
	if vip == "" || len(masters) == 0 {
		return ""
	}
	args := []string{"care", "--vs", vip + ":6443", "--health-path", "/healthz", "--health-schem", "https"}
	for _, m := range masters {
		if strings.Contains(m, ":") {
			m = strings.Split(m, ":")[0]
		}
		args = append(args, "--rs")
		args = append(args, m+":6443")
	}
	// Pod 的 YAML 模板
	podTemplate := `apiVersion: v1  
kind: Pod  
metadata:  
  creationTimestamp: null  
  labels:  
    component: kube-sealyun-lvscare  
    tier: control-plane  
  name: kube-sealyun-lvscare  
  namespace: kube-system  
spec:  
  containers:  
  - args:  
%s  
    command:  
    - /usr/bin/lvscare  
    image: fanux/lvscare:latest  
    imagePullPolicy: IfNotPresent  
    name: kube-sealyun-lvscare  
    resources: {}  
    securityContext:  
      privileged: true  
    volumeMounts:  
    - mountPath: /lib/modules  
      name: lib-modules  
      readOnly: true  
  hostNetwork: true  
  priorityClassName: system-cluster-critical  
  volumes:  
  - hostPath:  
      path: /lib/modules  
      type: ""  
    name: lib-modules  
status: {}`

	// 构建 args 的 YAML 部分
	argsYAML := ""
	for i, arg := range args {
		argsYAML += fmt.Sprintf("    - %s", arg)
		if i < len(args)-1 {
			argsYAML += "\n"
		}
	}

	// 替换模板中的 %s
	podYAML := fmt.Sprintf(podTemplate, argsYAML)

	return podYAML
}

func podToYaml(pod v1.Pod) ([]byte, error) {
	codecs := scheme.Codecs
	gv := v1.SchemeGroupVersion
	const mediaType = runtime.ContentTypeYAML
	info, ok := runtime.SerializerInfoForMediaType(codecs.SupportedMediaTypes(), mediaType)
	if !ok {
		return []byte{}, errors.Errorf("unsupported media type %q", mediaType)
	}
	logger.Info("start pod to yaml")
	encoder := codecs.EncoderForVersion(info.Serializer, gv)
	logger.Info("end pod to yaml")
	yamlData, err := runtime.Encode(encoder, &pod)
	if err != nil {
		return []byte{}, errors.Wrapf(err, "failed to encode pod to yaml")
	}
	return yamlData, nil
}

// componentPod returns a Pod object from the container and volume specifications
func componentPod(container v1.Container) v1.Pod {
	hostPathType := v1.HostPathUnset
	mountName := "lib-modules"
	volumes := []v1.Volume{
		{Name: mountName, VolumeSource: v1.VolumeSource{
			HostPath: &v1.HostPathVolumeSource{
				Path: "/lib/modules",
				Type: &hostPathType,
			},
		}},
	}
	container.VolumeMounts = []v1.VolumeMount{
		{Name: mountName, ReadOnly: true, MountPath: "/lib/modules"},
	}

	return v1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      container.Name,
			Namespace: metav1.NamespaceSystem,
			// The component and tier labels are useful for quickly identifying the control plane Pods when doing a .List()
			// against Pods in the kube-system namespace. Can for example be used together with the WaitForPodsWithLabel function
			Labels: map[string]string{"component": container.Name, "tier": "control-plane"},
		},
		Spec: v1.PodSpec{
			Containers:        []v1.Container{container},
			PriorityClassName: "system-cluster-critical",
			HostNetwork:       true,
			Volumes:           volumes,
		},
	}
}
