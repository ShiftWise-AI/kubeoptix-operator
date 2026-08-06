# shiftwise-operator

Operator para OpenShift baseado nos charts Helm de:

- kubeoptix-harvester
- kubeoptix-analyzer

O operator expõe o CRD `ShiftWise` (`shiftwises.shiftwise.ai`) e reconcilia os dois componentes via Helm Operator.

## Estrutura

- `watches.yaml`: mapeia CRD `ShiftWise` para o chart umbrella `helm-charts/shiftwise`
- `helm-charts/shiftwise`: chart principal que agrega os dois subcharts
- `config/crd/bases/shiftwise.ai_shiftwises.yaml`: definição do CRD
- `config/manifests/shiftwise-operator.yaml`: deployment e RBAC do operator
- `config/manifests/shiftwise-sample.yaml`: exemplo de CR

## Build da imagem do operator

Exemplo com Podman:

```bash
cd /home/parraes/redhat/shiftwise-ai/kubeoptix-operator
podman build -t quay.io/parraes/shiftwise-operator-system:0.1.0 .
podman push quay.io/parraes/shiftwise-operator-system:0.1.0
```

O manifest já está configurado para `quay.io/parraes/shiftwise-operator-system:0.1.0`.

## Instalação no OpenShift

```bash
oc apply -f config/manifests/install-crd.yaml
oc apply -f config/manifests/shiftwise-operator.yaml
```

## Criando instância ShiftWise

```bash
oc create namespace shiftwise-ai
oc apply -f config/manifests/shiftwise-sample.yaml
```

## Customização do CR

Tudo que estiver em `spec.kubeoptix-harvester` e `spec.kubeoptix-analyzer` é passado como values Helm para os respectivos subcharts.

Exemplos comuns:

- `spec.kubeoptix-harvester.build.source.gitUri`
- `spec.kubeoptix-harvester.build.source.gitRef`
- `spec.kubeoptix-harvester.image.repository`
- `spec.kubeoptix-analyzer.build.source.gitUri`
- `spec.kubeoptix-analyzer.image.repository`

## Observações

- O token GitHub foi removido dos values e deve ser injetado via CR/Secret.
- Por padrão, o analyzer está configurado para não recriar ServiceAccount e ClusterRoleBinding, evitando conflito com o harvester no mesmo namespace.

## Item 2: Bundle OLM (CSV + canal/pacote)

Arquivos gerados:

- `bundle/manifests/shiftwise-operator.v0.1.0.clusterserviceversion.yaml`
- `bundle/manifests/shiftwise.ai_shiftwises.yaml`
- `bundle/metadata/annotations.yaml`
- `bundle/metadata/package.yaml`
- `bundle.Dockerfile`
- `catalog/shiftwise-operator/catalog.yaml`

### Publicar imagem do bundle

```bash
cd /home/parraes/redhat/shiftwise-ai/kubeoptix-operator
podman build -f bundle.Dockerfile -t quay.io/parraes/shiftwise-operator-bundle:v0.1.0 .
podman push quay.io/parraes/shiftwise-operator-bundle:v0.1.0
```

Se alterar o nome/tag da imagem do bundle, atualize `catalog/shiftwise-operator/catalog.yaml`.

### Publicar imagem do catálogo (FBC)

```bash
cd /home/parraes/redhat/shiftwise-ai/kubeoptix-operator
podman build -t quay.io/parraes/shiftwise-operator-catalog:v0.1.0 -f - . <<'EOF'
FROM scratch
COPY catalog/shiftwise-operator/catalog.yaml /configs/
EOF
podman push quay.io/parraes/shiftwise-operator-catalog:v0.1.0
```

### Registrar no OpenShift (OperatorHub interno)

```bash
oc apply -f - <<'EOF'
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
	name: shiftwise-operator-catalog
	namespace: openshift-marketplace
spec:
	sourceType: grpc
	image: quay.io/parraes/shiftwise-operator-catalog:v0.1.0
	displayName: ShiftWise Operator Catalog
	publisher: ShiftWise
	updateStrategy:
		registryPoll:
			interval: 30m
EOF
```

Depois disso, o pacote `shiftwise-operator` no canal `stable` fica disponível no OperatorHub do cluster.