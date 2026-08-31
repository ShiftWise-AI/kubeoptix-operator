# shiftwise-operator

Operador Kubernetes/OpenShift em **Go** para a plataforma ShiftWise AI KubeOptix. Expõe o CRD `ShiftWise` (`shiftwises.shiftwise.ai`) e será responsável por reconciliar Harvester, Analyzer, Core AI, Configurations, Reporter, Dashboard e PostgreSQL.

O namespace padrão é **`shiftwise-ai`**. A imagem do manager é construída com **UBI 9** (`ubi9/go-toolset` + `ubi9/ubi-minimal`).

O reconciler materializa o inventário Helm, nesta ordem:

1. Harvester (PVC `harvester-app-data`, SA `shiftwisea-ai-user`, ClusterRoleBinding `cluster-reader`, Service, Route, ImageStream, BuildConfig, ValidatingAdmissionPolicy, StatefulSet)
2. Configurations (PostgreSQL `kubeoptix-db` + API, PDB, ImageStream, BuildConfig)
3. Analyzer (Secret `llm`, Service, Route, ImageStream, BuildConfig, scale policy, StatefulSet)
4. Core AI (Service, ImageStream, BuildConfig, StatefulSet)
5. Reporter (SA compartilhada, ClusterRoleBinding, Service, PDB, ImageStream, BuildConfig, scale policy, StatefulSet)
6. Dashboard (ConfigMap `.env.openshift`, Service, Route, ImageStream, BuildConfig binário, scale policy, StatefulSet)

Imagens padrão vêm do ImageStream interno (`image-registry.openshift-image-registry.svc:5000/<ns>/<stream>:latest`). Sobrescreva com `spec.images`.

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
config/samples/        CR de exemplo + Secret
config/manifests/      YAML único para oc apply
Containerfile          build UBI (Podman)
```

## Pré-requisitos

- Go 1.24+
- Podman
- `oc` autenticado em um cluster OpenShift

## Build da imagem

```bash
make image-build IMG=quay.io/parraes/shiftwise-operator:0.2.0
make image-push  IMG=quay.io/parraes/shiftwise-operator:0.2.0
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
oc apply -f config/samples/kubeoptix-credentials-example.yaml
oc apply -f config/samples/shiftwise.ai_v1alpha1_shiftwise.yaml
oc get shiftwises -n shiftwise-ai
```

`spec.targetNamespace` padrão: `shiftwise-ai`. Desabilite um componente com `spec.components.<nome>.enabled: false`.

Crie os Secrets de exemplo (troque os placeholders) e o Secret `github-auth` (basic-auth) **antes** de disparar os BuildConfigs:

```bash
oc apply -f config/samples/kubeoptix-credentials-example.yaml
```

O Secret `kubeoptix-db` precisa de `POSTGRESQL_USER`, `POSTGRESQL_PASSWORD` e `POSTGRESQL_DATABASE`. O Secret `llm` alimenta o Analyzer (`CURSOR_*` e `LLM_*`). O Dashboard usa BuildConfig do tipo Binary: após o operator criar o BC, rode `oc start-build kubeoptix-dashboard --from-dir=<repo-dashboard> -n shiftwise-ai`.

## Desenvolvimento local

```bash
make test
make run
```

`WATCH_NAMESPACE` vazio (padrão) observa todos os namespaces. Defina o valor para restringir o cache a um único projeto.
