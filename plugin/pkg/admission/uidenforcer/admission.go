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
	"io"
	"strconv"

	"k8s.io/kubernetes/pkg/admission"
	"k8s.io/kubernetes/pkg/api"
	apierrors "k8s.io/kubernetes/pkg/api/errors"
	client "k8s.io/kubernetes/pkg/client/unversioned"
)

func init() {
	admission.RegisterPlugin("UidEnforcer", func(client client.Interface, config io.Reader) (admission.Interface, error) {
		return NewUidEnforcer(client), nil
	})
}

// plugin contains the client used by the uidenforcer admin controller
type plugin struct {
	*admission.Handler
	client client.Interface
}

// NewSecurityContextDeny creates a new instance of the SecurityContextDeny admission controller
func NewUidEnforcer(client client.Interface) admission.Interface {
	return &plugin{
		Handler: admission.NewHandler(admission.Create, admission.Update),
		client:  client,
	}
}

// Admit will deny pods that have a RunAsUser set that isn't the uid of the user requesting it
func (p *plugin) Admit(a admission.Attributes) (err error) {
	if a.GetResource() != string(api.ResourcePods) {
		return nil
	}

	pod, ok := a.GetObject().(*api.Pod)
	if !ok {
		return apierrors.NewBadRequest("Resource was marked with kind Pod but was unable to be converted")
	}
	user := a.GetUserInfo()
	if user == nil {
		return apierrors.NewBadRequest("uidenforcer admission controller can not be used if there is no user set")
	}

	for i := 0; i < len(pod.Spec.Containers); i++ {
		container := &pod.Spec.Containers[i]
		uid, ok := strconv.ParseInt(user.GetUID(), 10, 32)
		if ok == nil {
			if container.SecurityContext == nil {
				container.SecurityContext = &api.SecurityContext{
					RunAsUser: &uid,
				}
			} else {
				container.SecurityContext.RunAsUser = &uid
			}
		} else {
			return apierrors.NewBadRequest("Requesting user's uid is not an integer")
		}
	}
	return nil
}
