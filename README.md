# shiftwise-operator

Operador Kubernetes/OpenShift em **Go** para a plataforma ShiftWise AI KubeOptix. Expõe o CRD `ShiftWise` (`shiftwises.shiftwise.ai`) e será responsável por reconciliar Harvester, Analyzer, Core AI, Configurations, Reporter, Dashboard e PostgreSQL.

O namespace padrão é **`shiftwise-ai`**. A imagem do manager é construída com **UBI 9** (`ubi9/go-toolset` + `ubi9/ubi-minimal`).

O reconciler materializa o inventário, nesta ordem:

1. Harvester (PVC `harvester-app-data`, SA `shiftwisea-ai-user`, ClusterRoleBinding `cluster-reader`, Service, StatefulSet)
2. Configurations (PostgreSQL `kubeoptix-db` + API, PDB, StatefulSet)
3. Analyzer (Service, scale policy, StatefulSet)
4. Core AI (Service, StatefulSet)
5. Reporter (SA compartilhada, ClusterRoleBinding, Service, PDB, scale policy, StatefulSet)
6. Dashboard (ConfigMap `.env.openshift`, Service, Route pública, scale policy, StatefulSet)

Imagens vêm do Quay (`quay.io/parraes/kubeoptix-<componente>:0.2.1`). O PostgreSQL usa `registry.redhat.io`; usuário, senha e database do Secret `kubeoptix-db` são gerados automaticamente.

## Estrutura

```
api/v1alpha1/          CRD ShiftWise (Go types)
cmd/main.go            entrypoint do controller-runtime
internal/controller/   reconciler
internal/constants/    namespace, nomes e imagens padrão
config/crd/            CustomResourceDefinition
config/rbac/           ServiceAccount, ClusterRole, leader election
config/manager/        Deployment no namespace shiftwise-ai
config/default/        kustomize (instalação completa)
config/samples/        CR de exemplo
config/manifests/      YAML único para oc apply
Containerfile          build UBI (Podman)
```

## Pré-requisitos

- Go 1.24+
- Podman
- `oc` autenticado em um cluster OpenShift

## Build da imagem

```bash
make image-build IMG=quay.io/parraes/shiftwise-operator:0.2.1
make image-push  IMG=quay.io/parraes/shiftwise-operator:0.2.1
```

## Instalação no OpenShift

```bash
make deploy
# equivalente:
oc apply -k config/default
```

Ou com o manifesto único:

```bash
oc apply -f config/manifests/shiftwise-operator.yaml
```

O operador sobe no projeto `shiftwise-ai`.

## Instância ShiftWise

```bash
oc apply -f config/samples/shiftwise.ai_v1alpha1_shiftwise.yaml
oc get shiftwises -n shiftwise-ai
```

O único campo opcional do CR é `spec.storage` (tamanho, StorageClass, claim existente). Componentes, imagens e credenciais PostgreSQL são preenchidos pelo operator.

## Desenvolvimento local

```bash
make test
make run
```

`WATCH_NAMESPACE` vazio (padrão) observa todos os namespaces. Defina o valor para restringir o cache a um único projeto.
