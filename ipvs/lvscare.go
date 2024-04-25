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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"strings"

	"github.com/fanux/sealos/pkg/logger"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
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
	//flag := true
	pod := componentPod(corev1.Container{
		Name:    "kube-sealyun-lvscare",
		Image:   image.toImageName(),
		Command: []string{"/usr/bin/lvscare"},
		//Args:            args,
		Args:            []string{"care", "--vs", "10.103.97.2" + ":6443", "--health-path", "/healthz", "--health-schem", "https", "--rs", "192.168.20.66" + ":6443"},
		ImagePullPolicy: corev1.PullIfNotPresent,
		//SecurityContext: &corev1.SecurityContext{Privileged: &flag},
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
//	logger.Info("Encoder: ", encoder)
//	if encoder == nil {
//		return []byte{}, errors.New("Encoder is nil")
//	}
//	logger.Info(" end ok yaml")
//	logger.Info(encoder)
//	logger.Info(" start runtime")
//	yamlpod, err := runtime.Encode(encoder, &pod)
//	if err != nil {
//		// 处理错误
//		logger.Info("runtime error:", err)
//		return nil, err
//	}
//	logger.Info(" print yaml")
//	logger.Info(yamlpod)
//	return yamlpod, nil
//}

//func podToYaml(pod v1.Pod) ([]byte, error) {
//	logger.Info(" pod to yaml")
//	var scheme = runtime.NewScheme()
//	var codecs = serializer.NewCodecFactory(scheme)
//	podYAML, err := runtime.Encode(codecs.LegacyCodec(v1.SchemeGroupVersion), &pod)
//	logger.Info(" print podYAML", podYAML)
//	if err != nil {
//		logger.Info("pod to yaml failed %s", err)
//	}
//	return podYAML, nil
//}

func podToYaml(pod corev1.Pod) ([]byte, error) {
	// 使用 Kubernetes 客户端库预定义的 Scheme
	codecs := serializer.NewCodecFactory(scheme.Scheme)

	// 获取 YAML 编码器的信息
	info, ok := runtime.SerializerInfoForMediaType(codecs.SupportedMediaTypes(), runtime.ContentTypeYAML)
	if !ok {
		return nil, fmt.Errorf("unsupported media type")
	}

	// 创建编码器
	encoder := codecs.EncoderForVersion(info.Serializer, corev1.SchemeGroupVersion)
	logger.Info(" start runtime podtoyaml")
	// 将 Pod 对象编码为 YAML 格式的字节流
	yamlBytes, err := runtime.Encode(encoder, &pod)
	if err != nil {
		return nil, err
	}

	return yamlBytes, nil
}

// componentPod returns a Pod object from the container and volume specifications
func componentPod(container corev1.Container) corev1.Pod {
	hostPathType := corev1.HostPathUnset
	mountName := "lib-modules"
	volumes := []corev1.Volume{
		{Name: mountName, VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: "/lib/modules",
				Type: &hostPathType,
			},
		}},
	}
	container.VolumeMounts = []corev1.VolumeMount{
		{Name: mountName, ReadOnly: true, MountPath: "/lib/modules"},
	}

	return corev1.Pod{
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
		Spec: corev1.PodSpec{
			Containers:        []corev1.Container{container},
			PriorityClassName: "system-cluster-critical",
			HostNetwork:       true,
			Volumes:           volumes,
		},
	}
}
