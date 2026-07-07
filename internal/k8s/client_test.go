package k8s

import (
	"context"
	"errors"
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
)

func TestDefaultContainer(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "multi", Namespace: "e2e"},
		Spec: corev1.PodSpec{Containers: []corev1.Container{
			{Name: "main"}, {Name: "sidecar"},
		}},
	}
	c := &client{typed: fake.NewSimpleClientset(pod)}

	got, err := c.defaultContainer(context.Background(), "e2e", "multi")
	if err != nil {
		t.Fatal(err)
	}
	if got != "main" {
		t.Errorf("expected first container %q, got %q", "main", got)
	}

	if _, err := c.defaultContainer(context.Background(), "e2e", "missing"); err == nil {
		t.Error("expected error for missing pod")
	}
}

func TestIsCRDNotRegistered(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"unrelated", errors.New("connection refused"), false},
		{"discovery message", errors.New("the server could not find the requested resource"), true},
		{"wrapped discovery", fmt.Errorf("listing: %w", errors.New("could not find the requested resource")), true},
		{"api not found", kerrors.NewNotFound(schema.GroupResource{Group: "deckhouse.io", Resource: "nodegroups"}, ""), true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsCRDNotRegistered(tc.err); got != tc.want {
				t.Errorf("IsCRDNotRegistered(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
