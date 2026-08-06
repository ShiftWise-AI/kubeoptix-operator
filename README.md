# kubeoptix-operator

A Kubernetes operator for managing ShiftWise-AI applications. It introduces the `ShiftWiseApp` custom resource, which automatically provisions a `Deployment` and a `Service` for each application you declare.

## Custom Resource: ShiftWiseApp

```yaml
apiVersion: shiftwise.ai/v1alpha1
kind: ShiftWiseApp
metadata:
  name: my-app
  namespace: default
spec:
  image: myapp:latest   # required
  replicas: 2           # default: 1
  port: 8080            # default: 8080
  serviceType: ClusterIP # ClusterIP | NodePort | LoadBalancer (default: ClusterIP)
  env:
    - name: ENV_VAR
      value: "value"
  resources:
    requests:
      cpu: "100m"
      memory: "128Mi"
    limits:
      cpu: "500m"
      memory: "512Mi"
```

## Development

```bash
# Build
make build

# Run tests
make test

# Build Docker image
make docker-build IMG=ghcr.io/shiftwise-ai/kubeoptix-operator:latest

# Install CRDs
make install

# Deploy operator
make deploy

# Apply a sample
make sample
```