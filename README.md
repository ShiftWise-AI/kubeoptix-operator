# ShiftWise Operator

Operador OpenShift da plataforma **KubeOptix**. Instale pelo OperatorHub, crie um `ShiftWise` e o Dashboard fica disponível numa Route.

Componentes (Harvester, Analyzer, Core AI, Configurations, Reporter, Dashboard e PostgreSQL) sobem no projeto `shiftwise-ai`. Só o Dashboard tem rota pública. Imagens vêm do Quay; as credenciais do banco são geradas automaticamente.

---

## 1. Catalogo no cluster

Com `oc` autenticado como administrador (cluster-admin), rode o script de instalação — ele cria o namespace `shiftwise-ai`, builda e publica as imagens do operator/bundle/catalogo no registry interno, concede as permissões de RBAC necessárias e aplica o `CatalogSource`:

```bash
./hack/install-catalog.sh
```

> **Importante:** só rodar `oc apply -f config/olm/catalogsource.yaml` **não é suficiente**. O `CatalogSource` referencia uma imagem no registry interno (`shiftwise-ai/shiftwise-operator-catalog`) que precisa existir e ser buildada/enviada antes, e o pod do catálogo (que roda em `openshift-marketplace`) precisa de permissão explícita (`system:image-puller`) para puxar imagens do namespace `shiftwise-ai`. Sem isso o pod fica em `ImagePullBackOff` com erro `authentication required` e o operator nunca aparece no OperatorHub. O `hack/install-catalog.sh` cuida de tudo isso automaticamente.

Se as imagens já foram publicadas anteriormente e você só quer reaplicar o `CatalogSource`/RBAC (sem rebuild):

```bash
./hack/install-catalog.sh --skip-build
```

Ao final, o script confirma que o `CatalogSource` está **READY** e que o pacote aparece no `PackageManifest`:

```bash
oc get catalogsource shiftwise-operator-catalog -n openshift-marketplace
oc get packagemanifest -n openshift-marketplace | grep shiftwise
```

**Print 1 — CatalogSource READY no OpenShift**

![CatalogSource READY](docs/images/01-catalogsource.png)

---

## 2. Instalar pelo OperatorHub

1. Na console, abra **Operators → OperatorHub**.
2. No filtro de fontes, marque **ShiftWise Operator Catalog**.
3. Busque **ShiftWise Operator** e abra o tile.
4. Clique em **Install** e confirme. O namespace sugerido é `shiftwise-ai`.

**Print 2 — OperatorHub, busca do ShiftWise Operator**

![OperatorHub](docs/images/02-operatorhub.png)

**Print 3 — Tela de instalação do operator**

![Install](docs/images/03-install.png)

---

## 3. Achar o operator instalado

**Operators → Installed Operators**. No seletor de projeto, use `shiftwise-ai` ou **All Projects** e abra **ShiftWise Operator**.

**Print 4 — Installed Operators**

![Installed Operators](docs/images/04-installed-operators.png)

---

## 4. Criar uma instância ShiftWise

Na página do operator, confirme que o projeto é **`shiftwise-ai`**. Depois: aba **ShiftWise → Create ShiftWise**.

O único campo que importa é **storage** (tamanho do volume compartilhado). O resto o operator preenche.

**Print 5 — Formulário Create ShiftWise**

![Create ShiftWise](docs/images/05-create-shiftwise.png)

Aguarde a instância ficar **Ready**.

**Print 6 — Instância Ready**

![ShiftWise Ready](docs/images/06-shiftwise-ready.png)

Pela CLI:

```bash
oc apply -f config/samples/shiftwise.ai_v1alpha1_shiftwise.yaml
oc get shiftwises -A
```

---

## 5. Abrir o Dashboard

**Networking → Routes**, projeto `shiftwise-ai`. A rota é `kubeoptix-dashboard`.

**Print 7 — Route do Dashboard**

![Route Dashboard](docs/images/07-dashboard-route.png)

```bash
oc get route kubeoptix-dashboard -n shiftwise-ai
```

---

## Troubleshooting: operator não aparece no OperatorHub

Se o `ShiftWise Operator` não aparece na busca do OperatorHub, verifique o pod do catálogo:

```bash
oc get pods -n openshift-marketplace -l olm.catalogSource=shiftwise-operator-catalog
oc describe pod -n openshift-marketplace -l olm.catalogSource=shiftwise-operator-catalog
```

Causas mais comuns (todas resolvidas por `./hack/install-catalog.sh`):

- **`ImagePullBackOff` / `authentication required`**: a imagem do catálogo não existe no registry interno (namespace `shiftwise-ai` não criado ou imagens não publicadas), ou o service account do pod em `openshift-marketplace` não tem a role `system:image-puller` no namespace `shiftwise-ai`.
- **`CatalogSource` em `TRANSIENT_FAILURE`**: consequência direta do pod do catálogo não subir; corrija o pull de imagem acima e reinicie o pod (`oc delete pod -n openshift-marketplace -l olm.catalogSource=shiftwise-operator-catalog`).
- **`PackageManifest` vazio** (`oc get packagemanifest -n openshift-marketplace | grep shiftwise`): aguarde alguns segundos após o `CatalogSource` ficar `READY` — a sincronização do OLM não é instantânea.

Os prints acima devem ficar em `docs/images/` com os nomes indicados em cada seção.
