package model

import (
	corev1 "k8s.io/api/core/v1"
)

type Taint struct {
	Key    string
	Value  string
	Effect corev1.TaintEffect
}

func (t Taint) String() string {
	if t.Value == "" {
		return t.Key + ":" + string(t.Effect)
	}
	return t.Key + "=" + t.Value + ":" + string(t.Effect)
}

type Node struct {
	Name         string
	InstanceType string
	Taints       []Taint
	Labels       map[string]string
	Allocatable  corev1.ResourceList
	Capacity     corev1.ResourceList
	Group        NodeGroup
	Ready        bool
}
