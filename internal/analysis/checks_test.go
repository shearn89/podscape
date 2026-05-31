package analysis

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"

	"github.com/shearn89/podscape/internal/model"
)

func TestDaemonSetMissingPriorityClass_FlagsOnlyDaemonSetPodsWithoutPC(t *testing.T) {
	pods := []model.Pod{
		{
			Namespace: "kube-system", Name: "kp-1",
			Owner:   model.WorkloadKey{Namespace: "kube-system", Kind: model.KindDaemonSet, Name: "kube-proxy"},
			Profile: model.SchedulingProfile{}, // no PC
		},
		{
			Namespace: "kube-system", Name: "kp-2",
			Owner: model.WorkloadKey{Namespace: "kube-system", Kind: model.KindDaemonSet, Name: "kube-proxy"},
		},
		{
			Namespace: "kube-system", Name: "csi-1",
			Owner:   model.WorkloadKey{Namespace: "kube-system", Kind: model.KindDaemonSet, Name: "ebs-csi"},
			Profile: model.SchedulingProfile{PriorityClass: "system-node-critical"},
		},
		{
			// Not a daemonset — shouldn't appear.
			Namespace: "default", Name: "api-1",
			Owner: model.WorkloadKey{Namespace: "default", Kind: model.KindDeployment, Name: "api"},
		},
	}
	got := DaemonSetMissingPriorityClass(pods)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding (deduped per DS), got %d: %+v", len(got), got)
	}
	if got[0].Code != "DS_NO_PRIORITY" {
		t.Errorf("wrong code: %q", got[0].Code)
	}
	if !strings.Contains(got[0].Message, "kube-proxy") {
		t.Errorf("message should name the DS: %q", got[0].Message)
	}
}

func TestTolerationWithoutMatchingTaint_FlagsPodsOnUntaintedNode(t *testing.T) {
	nodes := map[string]model.Node{
		"plain":   {Name: "plain"},
		"tainted": {Name: "tainted", Taints: []model.Taint{{Key: "critical", Effect: corev1.TaintEffectNoSchedule}}},
	}
	pods := []model.Pod{
		{
			Namespace: "default", Name: "wrong-pool", NodeName: "plain",
			Owner: model.WorkloadKey{Kind: model.KindDeployment, Name: "api"},
			Profile: model.SchedulingProfile{Tolerations: []model.Toleration{
				{Key: "critical", Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoSchedule},
			}},
		},
		{
			Namespace: "default", Name: "right-pool", NodeName: "tainted",
			Owner: model.WorkloadKey{Kind: model.KindDeployment, Name: "api"},
			Profile: model.SchedulingProfile{Tolerations: []model.Toleration{
				{Key: "critical", Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoSchedule},
			}},
		},
	}
	got := TolerationWithoutMatchingTaint(pods, nodes)
	if len(got) != 1 || got[0].Pod != "default/wrong-pool" {
		t.Fatalf("expected one finding for wrong-pool, got %+v", got)
	}
}

func TestTolerationWithoutMatchingTaint_SkipsKubeletInjected(t *testing.T) {
	nodes := map[string]model.Node{"n": {Name: "n"}}
	pods := []model.Pod{{
		Namespace: "default", Name: "p", NodeName: "n",
		Profile: model.SchedulingProfile{Tolerations: []model.Toleration{
			{Key: "node.kubernetes.io/not-ready", Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoExecute},
			{Key: "node.kubernetes.io/unreachable", Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoExecute},
		}},
	}}
	got := TolerationWithoutMatchingTaint(pods, nodes)
	if len(got) != 0 {
		t.Errorf("kubelet-injected tolerations should be ignored, got %+v", got)
	}
}

func TestTolerationWithoutMatchingTaint_SkipsDaemonSets(t *testing.T) {
	nodes := map[string]model.Node{"n": {Name: "n"}}
	pods := []model.Pod{{
		Namespace: "kube-system", Name: "ds", NodeName: "n",
		Owner: model.WorkloadKey{Kind: model.KindDaemonSet, Name: "kp"},
		Profile: model.SchedulingProfile{Tolerations: []model.Toleration{
			{Key: "critical", Operator: corev1.TolerationOpExists},
		}},
	}}
	got := TolerationWithoutMatchingTaint(pods, nodes)
	if len(got) != 0 {
		t.Errorf("daemonset tolerations are noisy, should skip: %+v", got)
	}
}

func TestRequiredAntiAffinityViolations_DetectsCoLocation(t *testing.T) {
	selector := []model.PodAffinityTerm{{
		TopologyKey: corev1.LabelHostname,
		MatchLabels: map[string]string{"app": "api"},
	}}
	pods := []model.Pod{
		{
			Namespace: "default", Name: "api-1", NodeName: "n1",
			Labels:  map[string]string{"app": "api"},
			Profile: model.SchedulingProfile{Affinity: &model.AffinitySummary{RequiredPodAntiAffinity: selector}},
		},
		{
			Namespace: "default", Name: "api-2", NodeName: "n1", // co-located, violation
			Labels:  map[string]string{"app": "api"},
			Profile: model.SchedulingProfile{Affinity: &model.AffinitySummary{RequiredPodAntiAffinity: selector}},
		},
		{
			Namespace: "default", Name: "api-3", NodeName: "n2", // separate, fine
			Labels:  map[string]string{"app": "api"},
			Profile: model.SchedulingProfile{Affinity: &model.AffinitySummary{RequiredPodAntiAffinity: selector}},
		},
	}
	got := RequiredAntiAffinityViolations(pods)
	if len(got) != 1 {
		t.Fatalf("expected 1 deduped violation (api-1 ↔ api-2 on n1), got %d: %+v", len(got), got)
	}
	if got[0].Code != "ANTIAFFINITY_VIOLATED" || got[0].Node != "n1" {
		t.Errorf("wrong finding: %+v", got[0])
	}
}

func TestRequiredAntiAffinityViolations_IgnoresSoftAntiAffinity(t *testing.T) {
	pods := []model.Pod{
		{Namespace: "default", Name: "a", NodeName: "n1", Labels: map[string]string{"app": "x"},
			Profile: model.SchedulingProfile{Affinity: &model.AffinitySummary{HasPodAntiAffinity: true}}},
		{Namespace: "default", Name: "b", NodeName: "n1", Labels: map[string]string{"app": "x"}},
	}
	got := RequiredAntiAffinityViolations(pods)
	if len(got) != 0 {
		t.Errorf("soft anti-affinity should not be flagged: %+v", got)
	}
}

func TestRequiredAntiAffinityViolations_RespectsNamespaceScope(t *testing.T) {
	selector := []model.PodAffinityTerm{{
		TopologyKey: corev1.LabelHostname,
		MatchLabels: map[string]string{"app": "api"},
	}}
	pods := []model.Pod{
		{Namespace: "ns-a", Name: "a", NodeName: "n1", Labels: map[string]string{"app": "api"},
			Profile: model.SchedulingProfile{Affinity: &model.AffinitySummary{RequiredPodAntiAffinity: selector}}},
		{Namespace: "ns-b", Name: "b", NodeName: "n1", Labels: map[string]string{"app": "api"}},
	}
	got := RequiredAntiAffinityViolations(pods)
	if len(got) != 0 {
		t.Errorf("default term-namespace scope is the source pod's NS — cross-NS pods shouldn't match: %+v", got)
	}
}

func TestRunChecks_OrdersBySeverity(t *testing.T) {
	pods := []model.Pod{
		{Namespace: "kube-system", Name: "ds-1",
			Owner: model.WorkloadKey{Namespace: "kube-system", Kind: model.KindDaemonSet, Name: "kp"}},
		{Namespace: "default", Name: "a", NodeName: "n1", Labels: map[string]string{"app": "x"},
			Profile: model.SchedulingProfile{Affinity: &model.AffinitySummary{
				RequiredPodAntiAffinity: []model.PodAffinityTerm{{
					TopologyKey: corev1.LabelHostname, MatchLabels: map[string]string{"app": "x"},
				}},
			}}},
		{Namespace: "default", Name: "b", NodeName: "n1", Labels: map[string]string{"app": "x"}},
	}
	got := RunChecks([]model.Node{{Name: "n1"}}, pods)
	if len(got) < 2 {
		t.Fatalf("expected at least 2 findings, got %d", len(got))
	}
	if got[0].Severity != SeverityError {
		t.Errorf("expected error first (anti-affinity), got severity %v / code %v", got[0].Severity, got[0].Code)
	}
}
