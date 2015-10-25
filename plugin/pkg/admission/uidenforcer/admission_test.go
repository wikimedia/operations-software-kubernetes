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
	"k8s.io/kubernetes/pkg/auth/user"
	"k8s.io/kubernetes/pkg/client/unversioned/testclient"
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

func TestFailWithNoUserInfo(t *testing.T) {
	client := testclient.NewSimpleFake()

	enforcer := NewUidEnforcer(client)
	testPod := validPod("test", 2)
	err := enforcer.Admit(admission.NewAttributesRecord(&testPod, "Pod", "test", "testPod", "pods", "", admission.Update, nil))
	if err == nil {
		t.Errorf("Expected an error since the pod did not specify resource limits in its update call")
	}
}

func TestPodWithUID(t *testing.T) {
	client := testclient.NewSimpleFake()

	enforcer := NewUidEnforcer(client)
	testPod := validPod("test", 2)
	userInfo := &user.DefaultInfo{
		Name:   "test",
		UID:    "50",
		Groups: nil,
	}

	err := enforcer.Admit(admission.NewAttributesRecord(&testPod, "Pod", "test", "testPod", "pods", "", admission.Update, userInfo))
	if err != nil {
		t.Errorf("%+v", err)
	}

	for _, v := range testPod.Spec.Containers {
		if v.SecurityContext != nil {
			if *v.SecurityContext.RunAsUser != 50 {
				t.Errorf("WTF!")
			}
		} else {
			t.Errorf("Uh, no SecurityContext!")
		}
	}
}
