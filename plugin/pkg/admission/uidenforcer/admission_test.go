/*
Copyright 2014 The Kubernetes Authors All rights reserved.

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

package uidenforcer

import (
	"strconv"
	"testing"

	"k8s.io/kubernetes/pkg/admission"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/cache"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/fake"
)

func validPod(name string, numContainers int) api.Pod {
	pod := api.Pod{ObjectMeta: api.ObjectMeta{Name: name, Namespace: "test"},
		Spec: api.PodSpec{},
	}
	pod.Spec.Containers = make([]api.Container, 0, numContainers)
	for i := 0; i < numContainers; i++ {
		pod.Spec.Containers = append(pod.Spec.Containers, api.Container{
			Image: "foo:V" + strconv.Itoa(i),
		})
	}
	return pod
}

func TestNoRunUser(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	store := cache.NewStore(cache.MetaNamespaceKeyFunc)
	testPod := validPod("test", 2)

	namespace := &api.Namespace{
		ObjectMeta: api.ObjectMeta{
			Name: testPod.Namespace,
		},
	}
	store.Add(namespace)

	handler := &plugin{
		clientset: clientset,
		store:     store,
	}

	err := handler.Admit(admission.NewAttributesRecord(&testPod, api.Kind("Pod"), testPod.Namespace, testPod.Name, api.Resource("pods"), "", admission.Update, nil))
	if err == nil {
		t.Errorf("Expected admission to fail but it passed!")
	}
}

func TestPodWithUID(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	store := cache.NewStore(cache.MetaNamespaceKeyFunc)
	testPod := validPod("test", 2)

	namespace := &api.Namespace{
		ObjectMeta: api.ObjectMeta{
			Name: testPod.Namespace,
			Annotations: map[string]string{
				"RunAsUser": "100",
			},
		},
	}
	store.Add(namespace)

	handler := &plugin{
		clientset: clientset,
		store:     store,
	}

	err := handler.Admit(admission.NewAttributesRecord(&testPod, api.Kind("Pod"), testPod.Namespace, testPod.Name, api.Resource("pods"), "", admission.Update, nil))
	if err != nil {
		t.Errorf("%+v", err)
	}

	if testPod.Spec.SecurityContext == nil {
		t.Errorf("Pod Security Context is unexpectedly nil")
	}

	if *testPod.Spec.SecurityContext.RunAsUser != 100 {
		t.Errorf("Pod Security Context doesn't have RunAsUser set")
	}

	if *testPod.Spec.SecurityContext.FSGroup != 100 {
		t.Errorf("Pod Security Context doesn't have FSGroup set")
	}

	for _, v := range testPod.Spec.Containers {
		if v.SecurityContext != nil {
			if *v.SecurityContext.RunAsUser != 100 {
				t.Errorf("WTF!")
			}
		} else {
			t.Errorf("Uh, no SecurityContext!")
		}
	}
}
