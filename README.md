# ShiftWise Operator

OpenShift operator for the **KubeOptix** platform. Install it from OperatorHub, create a `ShiftWise` instance, and the Dashboard becomes available through a Route.

Components (Harvester, Analyzer, Core AI, Configurations, Reporter, Dashboard, and PostgreSQL) come up in the `shiftwise-ai` project. Only the Dashboard has a public route. Images come from Quay; database credentials are generated automatically.

---

## 1. Catalog on the cluster

With `oc` authenticated as an administrator (cluster-admin), run the installation script — it creates the `shiftwise-ai` namespace, builds and publishes the operator/bundle/catalog images to the internal registry, grants the required RBAC permissions, and applies the `CatalogSource`:

```bash
./hack/install-catalog.sh
```

> **Important:** simply running `oc apply -f config/olm/catalogsource.yaml` is **not enough**. The `CatalogSource` references an image in the internal registry (`shiftwise-ai/shiftwise-operator-catalog`) that must exist and be built/pushed beforehand, and the catalog pod (which runs in `openshift-marketplace`) needs explicit permission (`system:image-puller`) to pull images from the `shiftwise-ai` namespace. Without this the pod stays in `ImagePullBackOff` with an `authentication required` error and the operator never shows up in OperatorHub. `hack/install-catalog.sh` takes care of all of this automatically.

If the images have already been published before and you only want to re-apply the `CatalogSource`/RBAC (without rebuilding):

```bash
./hack/install-catalog.sh --skip-build
```

At the end, the script confirms that the `CatalogSource` is **READY** and that the package shows up in the `PackageManifest`:

```bash
oc get catalogsource shiftwise-operator-catalog -n openshift-marketplace
oc get packagemanifest -n openshift-marketplace | grep shiftwise
```

**Screenshot 1 — CatalogSource READY on OpenShift**

![CatalogSource READY](docs/images/01-catalogsource.png)

---

## 2. Install from OperatorHub

1. In the console, open **Operators → OperatorHub**.
2. In the source filter, check **ShiftWise Operator Catalog**.
3. Search for **ShiftWise Operator** and open the tile.
4. Click **Install** and confirm. The suggested namespace is `shiftwise-ai`.

**Screenshot 2 — OperatorHub, searching for ShiftWise Operator**

![OperatorHub](docs/images/02-operatorhub.png)

**Screenshot 3 — Operator install screen**

![Install](docs/images/03-install.png)

---

## 3. Find the installed operator

**Operators → Installed Operators**. In the project selector, use `shiftwise-ai` or **All Projects** and open **ShiftWise Operator**.

**Screenshot 4 — Installed Operators**

![Installed Operators](docs/images/04-installed-operators.png)

---

## 4. Create a ShiftWise instance

On the operator page, confirm the project is **`shiftwise-ai`**. Then: **ShiftWise → Create ShiftWise** tab.

The only field that matters is **storage** (size of the shared volume). The operator fills in the rest.

**Screenshot 5 — Create ShiftWise form**

![Create ShiftWise](docs/images/05-create-shiftwise.png)

Wait for the instance to become **Ready**.

**Screenshot 6 — Instance Ready**

![ShiftWise Ready](docs/images/06-shiftwise-ready.png)

Via CLI:

```bash
oc apply -f config/samples/shiftwise.ai_v1alpha1_shiftwise.yaml
oc get shiftwises -A
```

---

## 5. Open the Dashboard

**Networking → Routes**, project `shiftwise-ai`. The route is `kubeoptix-dashboard`.

**Screenshot 7 — Dashboard Route**

![Route Dashboard](docs/images/07-dashboard-route.png)

```bash
oc get route kubeoptix-dashboard -n shiftwise-ai
```

---

## Troubleshooting: operator does not show up in OperatorHub

If the `ShiftWise Operator` does not appear in the OperatorHub search, check the catalog pod:

```bash
oc get pods -n openshift-marketplace -l olm.catalogSource=shiftwise-operator-catalog
oc describe pod -n openshift-marketplace -l olm.catalogSource=shiftwise-operator-catalog
```

Most common causes (all resolved by `./hack/install-catalog.sh`):

- **`ImagePullBackOff` / `authentication required`**: the catalog image does not exist in the internal registry (the `shiftwise-ai` namespace was not created or the images were not published), or the pod's service account in `openshift-marketplace` lacks the `system:image-puller` role on the `shiftwise-ai` namespace.
- **`CatalogSource` in `TRANSIENT_FAILURE`**: a direct consequence of the catalog pod failing to start; fix the image pull issue above and restart the pod (`oc delete pod -n openshift-marketplace -l olm.catalogSource=shiftwise-operator-catalog`).
- **Empty `PackageManifest`** (`oc get packagemanifest -n openshift-marketplace | grep shiftwise`): wait a few seconds after the `CatalogSource` becomes `READY` — OLM sync is not instantaneous.

The screenshots above should live in `docs/images/` under the names referenced in each section.
