package operands

import (
	"context"
	"fmt"
	"io"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// reconcileGeneratedSecret preserves existing credentials and only uses randomness
// when a Secret needs to be created. The source must be crypto/rand.Reader in production.
func reconcileGeneratedSecret(ctx context.Context, c client.Client, scheme *runtime.Scheme, owner client.Object, secret *corev1.Secret, field string, source io.Reader) error {
	key := client.ObjectKeyFromObject(secret)
	if err := c.Get(ctx, key, &corev1.Secret{}); err == nil {
		return nil
	} else if !apierrors.IsNotFound(err) {
		return fmt.Errorf("get secret %s: %w", key, err)
	}
	value, err := randomAlphanum(24, source)
	if err != nil {
		return fmt.Errorf("generate credential for secret %s: %w", key, err)
	}
	if secret.StringData == nil {
		secret.StringData = make(map[string]string)
	}
	secret.StringData[field] = value
	return applyIfMissing(ctx, c, scheme, owner, secret)
}

func randomAlphanum(n int, source io.Reader) (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	// ReadFull propagates reader errors and never returns a partially generated secret.
	if _, err := io.ReadFull(source, b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(b), nil
}
