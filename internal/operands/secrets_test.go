package operands

import (
	"bytes"
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	shiftwisev1alpha1 "github.com/ShiftWise-AI/kubeoptix-operator/api/v1alpha1"
)

type failingRandomReader struct {
	calls int
	err   error
}

func (r *failingRandomReader) Read([]byte) (int, error) { r.calls++; return 0, r.err }

func TestRandomAlphanumReadFailure(t *testing.T) {
	failure := errors.New("random source unavailable")
	for _, source := range []io.Reader{
		&failingRandomReader{err: failure},
		io.MultiReader(bytes.NewReader([]byte{1, 2, 3}), &failingRandomReader{err: failure}),
	} {
		got, err := randomAlphanum(24, source)
		if !errors.Is(err, failure) || got != "" {
			t.Fatalf("expected empty result and source error, got length %d, error %v", len(got), err)
		}
	}
	got, err := randomAlphanum(24, bytes.NewReader([]byte{1, 2, 3}))
	if !errors.Is(err, io.ErrUnexpectedEOF) || got != "" {
		t.Fatalf("expected empty result on short read, got length %d, error %v", len(got), err)
	}
}

func TestReconcileGeneratedSecrets(t *testing.T) {
	for _, kind := range []struct {
		name, field string
		template    func(Settings, map[string]string) *corev1.Secret
	}{
		{"postgres", "POSTGRESQL_PASSWORD", postgresSecret},
		{"oauth", "session_secret", dashboardOAuthSecret},
	} {
		t.Run(kind.name, func(t *testing.T) {
			for _, scenario := range []string{"create", "random failure", "partial random failure", "existing", "get failure", "create failure", "concurrent create"} {
				t.Run(scenario, func(t *testing.T) {
					ctx := context.Background()
					scheme := runtime.NewScheme()
					if err := corev1.AddToScheme(scheme); err != nil {
						t.Fatal(err)
					}
					if err := shiftwisev1alpha1.AddToScheme(scheme); err != nil {
						t.Fatal(err)
					}
					owner := newShiftWise()
					owner.Name = "a" // The removed OAuth fallback also panicked for short instance names.
					owner.UID = "owner-uid"
					secret := kind.template(FromCR(owner), map[string]string{"test": "secret"})
					key := client.ObjectKeyFromObject(secret)
					failure := errors.New("random source unavailable")
					failingSource := &failingRandomReader{err: failure}
					var source io.Reader = bytes.NewReader(bytes.Repeat([]byte{42}, 24))
					var wantErr error
					existing := secret.DeepCopy()
					existing.Data = map[string][]byte{kind.field: []byte("existing-credential")}
					existing.StringData = nil
					builder := fake.NewClientBuilder().WithScheme(scheme)
					switch scenario {
					case "random failure":
						source = failingSource
						wantErr = failure
					case "partial random failure":
						source = io.MultiReader(bytes.NewReader([]byte{1, 2, 3}), failingSource)
						wantErr = failure
					case "existing":
						builder = builder.WithObjects(existing)
						source = failingSource
					case "get failure":
						wantErr = apierrors.NewForbidden(schema.GroupResource{Resource: "secrets"}, key.Name, errors.New("denied"))
						builder = builder.WithInterceptorFuncs(interceptor.Funcs{Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
							return wantErr
						}})
						source = failingSource
					case "create failure":
						wantErr = apierrors.NewForbidden(schema.GroupResource{Resource: "secrets"}, key.Name, errors.New("denied"))
						builder = builder.WithInterceptorFuncs(interceptor.Funcs{Create: func(context.Context, client.WithWatch, client.Object, ...client.CreateOption) error { return wantErr }})
					case "concurrent create":
						builder = builder.WithInterceptorFuncs(interceptor.Funcs{Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
							if err := c.Create(ctx, existing); err != nil {
								return err
							}
							return apierrors.NewAlreadyExists(schema.GroupResource{Resource: "secrets"}, obj.GetName())
						}})
					}
					c := builder.Build()
					err := reconcileGeneratedSecret(ctx, c, scheme, owner, secret, kind.field, source)
					if !errors.Is(err, wantErr) {
						t.Fatalf("error = %v, want %v", err, wantErr)
					}
					if scenario == "existing" || scenario == "get failure" {
						if failingSource.calls != 0 {
							t.Fatal("randomness requested before confirming Secret is missing")
						}
					}
					if scenario == "get failure" {
						return
					}
					got := &corev1.Secret{}
					getErr := c.Get(ctx, key, got)
					if wantErr != nil {
						if !apierrors.IsNotFound(getErr) {
							t.Fatalf("failed reconciliation must not create a secret: %v", getErr)
						}
						if errors.Is(err, failure) && !strings.Contains(err.Error(), key.Name) {
							t.Fatal("generation error lacks secret context")
						}
						return
					}
					if getErr != nil {
						t.Fatal(getErr)
					}
					if scenario == "existing" || scenario == "concurrent create" {
						if !reflect.DeepEqual(got.Data, existing.Data) || !reflect.DeepEqual(got.StringData, existing.StringData) {
							t.Fatal("existing credentials were changed")
						}
						return
					}
					// The fake client retains StringData instead of converting it into Data.
					credential := got.StringData[kind.field]
					if len(credential) != 24 {
						t.Fatalf("credential length = %d, want 24", len(credential))
					}
					for _, ch := range credential {
						if !strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", ch) {
							t.Fatal("credential contains a non-alphanumeric character")
						}
					}
					if got.Type != corev1.SecretTypeOpaque || got.Labels["test"] != "secret" {
						t.Fatal("secret metadata changed")
					}
					if len(got.OwnerReferences) != 1 || got.OwnerReferences[0].UID != owner.UID {
						t.Fatal("missing owner reference")
					}
					if kind.name == "postgres" {
						s := FromCR(owner)
						if got.StringData["POSTGRESQL_USER"] != s.PostgresUser || got.StringData["POSTGRESQL_DATABASE"] != s.PostgresDatabase {
							t.Fatal("postgres connection fields changed")
						}
					}
				})
			}
		})
	}
}
