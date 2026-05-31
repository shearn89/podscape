package model

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type stubResolver map[string]string // namespace/replicaset -> deployment

func (s stubResolver) ResolveReplicaSet(ns, name string) (string, bool) {
	d, ok := s[ns+"/"+name]
	return d, ok
}

func ptrTrue() *bool { b := true; return &b }

func TestFromPod_OwnerKinds(t *testing.T) {
	cases := []struct {
		name      string
		kind      string
		ownerName string
		resolver  OwnerResolver
		wantKind  Kind
		wantName  string
	}{
		{"daemonset", "DaemonSet", "kube-proxy", nil, KindDaemonSet, "kube-proxy"},
		{"statefulset", "StatefulSet", "etcd", nil, KindStatefulSet, "etcd"},
		{"job", "Job", "migration-1", nil, KindJob, "migration-1"},
		{
			"replicaset resolves to deployment",
			"ReplicaSet", "api-7c8b",
			stubResolver{"default/api-7c8b": "api"},
			KindDeployment, "api",
		},
		{
			"replicaset unresolved stays a replicaset",
			"ReplicaSet", "orphan-1",
			stubResolver{}, KindReplicaSet, "orphan-1",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "pod-1",
					OwnerReferences: []metav1.OwnerReference{{
						Kind: tc.kind, Name: tc.ownerName, Controller: ptrTrue(),
					}},
				},
			}
			got := FromPod(p, tc.resolver)
			if got.Owner.Kind != tc.wantKind || got.Owner.Name != tc.wantName {
				t.Errorf("owner = %v/%v, want %v/%v", got.Owner.Kind, got.Owner.Name, tc.wantKind, tc.wantName)
			}
		})
	}
}

func TestFromPod_StandaloneWhenNoOwner(t *testing.T) {
	p := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "loose"}}
	got := FromPod(p, nil)
	if got.Owner.Kind != KindStandalone || got.Owner.Name != "loose" {
		t.Errorf("expected standalone Pod, got %v/%v", got.Owner.Kind, got.Owner.Name)
	}
}

func TestFromPod_CapturesSchedulingProfile(t *testing.T) {
	p := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "kube-system", Name: "coredns-1"},
		Spec: corev1.PodSpec{
			PriorityClassName: "system-cluster-critical",
			NodeSelector:      map[string]string{"kubernetes.io/os": "linux"},
			Tolerations: []corev1.Toleration{{
				Key: "critical", Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoSchedule,
			}},
			Affinity: &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{{
							MatchExpressions: []corev1.NodeSelectorRequirement{{
								Key: "topology.kubernetes.io/zone", Operator: corev1.NodeSelectorOpIn, Values: []string{"eu-west-1a"},
							}},
						}},
					},
				},
			},
		},
	}
	got := FromPod(p, nil)
	if got.Profile.PriorityClass != "system-cluster-critical" {
		t.Errorf("priority class lost: %q", got.Profile.PriorityClass)
	}
	if len(got.Profile.Tolerations) != 1 {
		t.Errorf("tolerations lost: %v", got.Profile.Tolerations)
	}
	if got.Profile.Affinity == nil || len(got.Profile.Affinity.RequiredNodeMatch) != 1 {
		t.Fatalf("affinity not captured: %v", got.Profile.Affinity)
	}
	want := "topology.kubernetes.io/zone in [eu-west-1a]"
	if got.Profile.Affinity.RequiredNodeMatch[0] != want {
		t.Errorf("affinity rendered as %q, want %q", got.Profile.Affinity.RequiredNodeMatch[0], want)
	}
}

func TestFromNode_DerivesGroupAndInstanceType(t *testing.T) {
	n := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "ip-10-0-1-1",
			Labels: map[string]string{"karpenter.sh/nodepool": "system", "node.kubernetes.io/instance-type": "m5.large"},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionTrue}},
		},
	}
	got := FromNode(n)
	if got.Group.Key != "karpenter:system" {
		t.Errorf("group key %q", got.Group.Key)
	}
	if got.InstanceType != "m5.large" {
		t.Errorf("instance type lost: %q", got.InstanceType)
	}
	if !got.Ready {
		t.Errorf("ready flag should be true")
	}
}
