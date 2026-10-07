package operands

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ReadyCount(ctx context.Context, c client.Client, s Settings) (ready, desired int, err error) {
	names := enabledComponents(s)
	desired = len(names)
	for _, name := range names {
		sts := &appsv1.StatefulSet{}
		if getErr := c.Get(ctx, client.ObjectKey{Name: name, Namespace: s.Namespace}, sts); getErr != nil {
			if apierrors.IsNotFound(getErr) {
				continue
			}
			return ready, desired, fmt.Errorf("read readiness of StatefulSet %s/%s: %w", s.Namespace, name, getErr)
		}
		if sts.Status.ReadyReplicas >= 1 {
			ready++
		}
	}
	return ready, desired, nil
}

func Phase(ready, desired int, recErr error) string {
	if recErr != nil {
		return "Error"
	}
	if desired == 0 {
		return "Pending"
	}
	if ready == desired {
		return "Ready"
	}
	if ready == 0 {
		return "Initializing"
	}
	return "Progressing"
}

func ReadyString(ready, desired int) string {
	return fmt.Sprintf("%d/%d", ready, desired)
}
