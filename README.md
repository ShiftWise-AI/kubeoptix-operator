# shiftwise-operator

Operador Kubernetes em Lua para a plataforma ShiftWise AI KubeOptix. Ele expõe o CRD `ShiftWise` (`shiftwises.shiftwise.ai`) e reconcilia a plataforma completa: Harvester, Analyzer, Core AI, Configurations, Reporter, Dashboard e PostgreSQL.

## Estrutura

- `lua/shiftwise_operator.lua`: reconciliador Lua que usa `kubectl` in-cluster e `dkjson`
- `config/crd/bases/shiftwise.ai_shiftwises.yaml`: definição do CRD
- `config/manifests/shiftwise-operator.yaml`: deployment e RBAC do operator
- `config/manifests/shiftwise-sample.yaml`: exemplo de CR
- `config/manifests/kubeoptix-credentials-example.yaml`: Secret externo requerido pelo exemplo

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
oc apply -f config/manifests/kubeoptix-credentials-example.yaml
oc apply -f config/manifests/shiftwise-sample.yaml
```

## Customização do CR

Defina `spec.targetNamespace`, as imagens em `spec.images`, o Secret em `spec.credentials.existingSecret` e o armazenamento em `spec.storage`. Todos os componentes são habilitados por padrão; desabilite um deles com `spec.components.<nome>.enabled: false`.

O volume compartilhado precisa suportar `ReadWriteMany`, pois Harvester, Analyzer, Core AI e Reporter compartilham `/app/data`. Use `spec.storage.existingClaim` quando o PVC já existir; caso contrário, o operador cria `kubeoptix-data` com 10Gi por padrão.

O Secret referenciado precisa conter `POSTGRESQL_USER`, `POSTGRESQL_PASSWORD`, `POSTGRESQL_DATABASE` e, quando o Analyzer usar LLM remoto, `LLM_API_KEY`. Nunca use o arquivo de exemplo em produção sem trocar os valores.

## Observações

- O controlador registra eventos de reconciliação em JSON no stdout e atualiza `.status.phase` e `.status.readyComponents` em cada ciclo de 30 segundos.
- A prontidão dos seis serviços é obtida a partir de `Deployment.status.readyReplicas`; isso oferece um sinal operacional sem depender de CRDs opcionais de monitoramento.
- O Harvester recebe RBAC de leitura para os recursos que coleta, inclusive nós, workloads, rotas e recursos de monitoramento.
- O operador usa imagens configuráveis. As referências `quay.io/shiftwise-ai/...` no exemplo devem existir e ser acessíveis ao cluster antes da instalação.

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