# ShiftWise Operator

Operador OpenShift da plataforma **KubeOptix**. Instale pelo OperatorHub, crie um `ShiftWise` e o Dashboard fica disponível numa Route.

Componentes (Harvester, Analyzer, Core AI, Configurations, Reporter, Dashboard e PostgreSQL) sobem no projeto `shiftwise-ai`. Só o Dashboard tem rota pública. Imagens vêm do Quay; as credenciais do banco são geradas automaticamente.

---

## 1. Catalogo no cluster

Com `oc` autenticado como administrador:

```bash
oc apply -f config/olm/catalogsource.yaml
```

Aguarde o catalogo ficar **READY**:

```bash
oc get catalogsource shiftwise-operator-catalog -n openshift-marketplace
```

**Print 1 — CatalogSource READY no OpenShift**

![CatalogSource READY](docs/images/01-catalogsource.png)

---

## 2. Instalar pelo OperatorHub

1. Na consola, abra **Operators → OperatorHub**.
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

Os prints acima devem ficar em `docs/images/` com os nomes indicados em cada seção.
