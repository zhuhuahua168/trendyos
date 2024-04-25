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
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"strings"

	"github.com/fanux/sealos/pkg/logger"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type LvscareImage struct {
	Image string
	Tag   string
}

func (l *LvscareImage) toImageName() string {
	return l.Image + ":" + l.Tag
}

// return lvscare static pod yaml
func LvsStaticPodYaml(vip string, masters []string, image LvscareImage) string {
	logger.Info("start lvscare static pod yaml")
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
	flag := true
	pod := componentPod(v1.Container{
		Name:            "kube-sealyun-lvscare",
		Image:           image.toImageName(),
		Command:         []string{"/usr/bin/lvscare"},
		Args:            args,
		ImagePullPolicy: v1.PullIfNotPresent,
		SecurityContext: &v1.SecurityContext{Privileged: &flag},
	})
	yaml, err := podToYaml(pod)
	logger.Info(" end podtoyaml")
	if err != nil {
		logger.Error("decode lvscare static pod yaml failed %s", err)
		return ""
	}
	return string(yaml)
}

//func podToYaml(pod v1.Pod) ([]byte, error) {
//	logger.Info(" pod to yaml")
//	codecs := scheme.Codecs
//	gv := v1.SchemeGroupVersion
//	logger.Info(" start yaml")
//	const mediaType = runtime.ContentTypeYAML
//	logger.Info(" end yaml")
//	info, ok := runtime.SerializerInfoForMediaType(codecs.SupportedMediaTypes(), mediaType)
//	logger.Info(" info yaml")
//	if !ok {
//		return []byte{}, errors.Errorf("unsupported media type %q", mediaType)
//	}
//
//	encoder := codecs.EncoderForVersion(info.Serializer, gv)
//	logger.Info(" end ok yaml")
//	logger.Info(encoder)
//	logger.Info(" start runtime")
//	yamlpod, _ := runtime.Encode(encoder, &pod)
//	logger.Info(" print yaml")
//	logger.Info(yamlpod)
//	return yamlpod, nil
//}

func podToYaml(pod v1.Pod) ([]byte, error) {
	logger.Info(" pod to yaml")
	var scheme = runtime.NewScheme()
	var codecs = serializer.NewCodecFactory(scheme)
	podYAML, err := runtime.Encode(codecs.LegacyCodec(v1.SchemeGroupVersion), &pod)
	logger.Info(" print podYAML", podYAML)
	if err != nil {
		logger.Info("pod to yaml failed %s", err)
	}
	return podYAML, nil
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
