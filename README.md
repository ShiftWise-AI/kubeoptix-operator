# ShiftWise Operator

OpenShift operator for the **KubeOptix** platform. Install it from OperatorHub, create a `ShiftWise` instance, and the Dashboard becomes available through a Route.

Components (Harvester, Analyzer, Core AI, Configurations, Reporter, Dashboard, and PostgreSQL) come up in the `shiftwise-ai` project. Only the Dashboard has a public route. Images come from Quay; database credentials are generated automatically.

---

## 1. Catalog on the cluster

The operator, bundle, and catalog images are already built and published on `quay.io/parraes` (public, no authentication required). With `oc` authenticated as an administrator (cluster-admin), just apply the `CatalogSource` pointing at those images:

```bash
./hack/install-catalog.sh
```

This is equivalent to `oc apply -f config/olm/catalogsource.yaml`, plus waiting for the catalog pod to become **READY** and for the package to show up in `PackageManifest`. No image build, no internal registry, and no extra RBAC are needed — OLM (in `openshift-marketplace`) and the operator Deployment (in `openshift-operators`) pull the images directly from quay.io.

To install a specific version:

```bash
./hack/install-catalog.sh --version 1.0.1
```

Maintainers who need to publish a new version to quay.io (requires `podman login quay.io` with push access to the `parraes` org) can build and push before applying the `CatalogSource`:

```bash
./hack/install-catalog.sh --build --version 1.0.1
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

- **`ImagePullBackOff` / `authentication required`**: the `CatalogSource` (or the CSV it produced) is pointing at an image tag that does not exist on `quay.io/parraes`, or at a stale internal-registry reference from a previous local build. Re-run `./hack/install-catalog.sh --version <version>` with a version that is actually published on quay.io.
- **`CatalogSource` in `TRANSIENT_FAILURE`**: a direct consequence of the catalog pod failing to start; fix the image reference above and restart the pod (`oc delete pod -n openshift-marketplace -l olm.catalogSource=shiftwise-operator-catalog`).
- **Subscription stuck in `BundleUnpacking`**: OLM could not pull the bundle image referenced by the catalog; confirm the tag exists on `quay.io/parraes/shiftwise-operator-bundle`.
- **Empty `PackageManifest`** (`oc get packagemanifest -n openshift-marketplace | grep shiftwise`): wait a few seconds after the `CatalogSource` becomes `READY` — OLM sync is not instantaneous.

The screenshots above should live in `docs/images/` under the names referenced in each section.
