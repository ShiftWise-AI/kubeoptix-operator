#!/usr/bin/env python3
"""Generate the OLM bundle CSV and file-based catalog for the current VERSION."""
from __future__ import annotations

import base64
import json
import os
import shutil
from copy import deepcopy
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[1]
VERSION = os.environ.get("VERSION", "1.0.0")
PREVIOUS_VERSION = os.environ.get("PREVIOUS_VERSION", "")
QUAY_ORG = os.environ.get("QUAY_ORG", "parraes")
PACKAGE = "shiftwise-operator"
CSV_NAME = f"{PACKAGE}.v{VERSION}"
# OPERATOR_IMG/BUNDLE_IMG can be overridden with full image references (e.g.
# by hack/install-catalog.sh --build); otherwise they default to the public
# quay.io images already published for the current VERSION.
OPERATOR_IMG = os.environ.get("OPERATOR_IMG", f"quay.io/{QUAY_ORG}/shiftwise-operator:{VERSION}")
BUNDLE_IMG = os.environ.get("BUNDLE_IMG", f"quay.io/{QUAY_ORG}/shiftwise-operator-bundle:v{VERSION}")
SA = "shiftwise-operator-controller-manager"


def load_yaml(path: Path):
    with path.open() as fh:
        return yaml.safe_load(fh)


def dump_yaml(path: Path, data) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w") as fh:
        yaml.safe_dump(data, fh, sort_keys=False, width=10**9)


def icon() -> dict:
    png = (ROOT / "config/manifests/logo.png").read_bytes()
    return {"mediatype": "image/png", "base64data": base64.b64encode(png).decode("ascii")}


def sample() -> dict:
    return load_yaml(ROOT / "config/samples/shiftwise.ai_v1alpha1_shiftwise.yaml")


def deployment_spec() -> dict:
    dep = load_yaml(ROOT / "config/manager/manager.yaml")
    spec = deepcopy(dep["spec"])
    spec.pop("paused", None)
    # config/manager/manager.yaml pins a static image reference used for
    # `oc apply -k config/default`-style deploys; the CSV must always ship
    # the image actually built/published for this VERSION (OPERATOR_IMG).
    for container in spec["template"]["spec"]["containers"]:
        if container.get("name") == "manager":
            container["image"] = OPERATOR_IMG
    return spec


def cluster_rules() -> list:
    return load_yaml(ROOT / "config/rbac/role.yaml")["rules"]


def leader_rules() -> list:
    return load_yaml(ROOT / "config/rbac/leader_election_role.yaml")["rules"]


def previous_version() -> str:
    if PREVIOUS_VERSION:
        return PREVIOUS_VERSION

    catalog_path = ROOT / "catalog/shiftwise-operator/catalog.yaml"
    if not catalog_path.exists():
        return ""
    for document in yaml.safe_load_all(catalog_path.open()):
        if (
            document
            and document.get("schema") == "olm.channel"
            and document.get("package") == PACKAGE
            and document.get("name") == "stable"
        ):
            entries = document.get("entries", [])
            if entries:
                prefix = f"{PACKAGE}.v"
                name = entries[-1]["name"]
                return name.removeprefix(prefix)
    return ""


def csv() -> dict:
    previous = previous_version()
    return {
        "apiVersion": "operators.coreos.com/v1alpha1",
        "kind": "ClusterServiceVersion",
        "metadata": {
            "name": CSV_NAME,
            "namespace": "placeholder",
            "annotations": {
                "alm-examples": json.dumps([sample()], indent=2),
                "capabilities": "Basic Install",
                "categories": "AI/Machine Learning,OpenShift Optional",
                "containerImage": OPERATOR_IMG,
                "createdAt": "2026-08-31T00:00:00Z",
                "description": "ShiftWise operator that reconciles the KubeOptix platform on OpenShift.",
                "operators.operatorframework.io/suggested-namespace": "shiftwise-ai",
                "repository": "https://github.com/ShiftWise-AI/kubeoptix-operator",
                "support": "ShiftWise AI",
            },
        },
        "spec": {
            "displayName": "ShiftWise Operator",
            **(
                {"replaces": f"{PACKAGE}.v{previous}"}
                if previous
                else {}
            ),
            "description": (
                "Operador Kubernetes/OpenShift que reconcilia o CR ShiftWise na plataforma "
                "KubeOptix (Harvester, Analyzer, Core AI, Configurations, Reporter, Dashboard "
                "e PostgreSQL)."
            ),
            "keywords": ["shiftwise", "kubeoptix", "openshift", "kubernetes", "aiops"],
            "maintainers": [{"name": "ShiftWise AI"}],
            "provider": {"name": "ShiftWise AI"},
            "maturity": "alpha",
            "minKubeVersion": "1.26.0",
            "version": VERSION,
            "links": [
                {
                    "name": "ShiftWise Operator",
                    "url": "https://github.com/ShiftWise-AI/kubeoptix-operator",
                }
            ],
            "icon": [icon()],
            "installModes": [
                {"type": "OwnNamespace", "supported": True},
                {"type": "SingleNamespace", "supported": True},
                {"type": "MultiNamespace", "supported": False},
                {"type": "AllNamespaces", "supported": True},
            ],
            "customresourcedefinitions": {
                "owned": [
                    {
                        "name": "shiftwises.shiftwise.ai",
                        "version": "v1alpha1",
                        "kind": "ShiftWise",
                        "displayName": "ShiftWise",
                        "description": "Instância da plataforma KubeOptix gerenciada pelo ShiftWise Operator.",
                    }
                ]
            },
            "relatedImages": [{"name": "manager", "image": OPERATOR_IMG}],
            "install": {
                "strategy": "deployment",
                "spec": {
                    "clusterPermissions": [
                        {"serviceAccountName": SA, "rules": cluster_rules()}
                    ],
                    "permissions": [{"serviceAccountName": SA, "rules": leader_rules()}],
                    "deployments": [
                        {
                            "name": "shiftwise-operator-controller-manager",
                            "spec": deployment_spec(),
                        }
                    ],
                },
            },
        },
    }


def catalog(csv_obj: dict) -> list:
    spec = csv_obj["spec"]
    previous = previous_version()
    catalog_path = ROOT / "catalog/shiftwise-operator/catalog.yaml"
    previous_entries = []
    previous_bundles = []
    if catalog_path.exists():
        for document in yaml.safe_load_all(catalog_path.open()):
            if not document or document.get("package") != PACKAGE:
                continue
            if document.get("schema") == "olm.channel" and document.get("name") == "stable":
                previous_entries.extend(
                    entry
                    for entry in document.get("entries", [])
                    if entry.get("name") != CSV_NAME
                )
            elif document.get("schema") == "olm.bundle" and document.get("name") != CSV_NAME:
                previous_bundles.append(document)

    current_entry = {"name": CSV_NAME}
    if previous:
        current_entry["replaces"] = f"{PACKAGE}.v{previous}"

    current_bundle = {
        "schema": "olm.bundle",
        "package": PACKAGE,
        "name": CSV_NAME,
        "image": BUNDLE_IMG,
        "properties": [
            {
                "type": "olm.package",
                "value": {"packageName": PACKAGE, "version": VERSION},
            },
            {
                "type": "olm.gvk",
                "value": {
                    "group": "shiftwise.ai",
                    "kind": "ShiftWise",
                    "version": "v1alpha1",
                },
            },
            {
                "type": "olm.csv.metadata",
                "value": {
                    "annotations": csv_obj["metadata"]["annotations"],
                    "apiServiceDefinitions": {},
                    "crdDescriptions": spec["customresourcedefinitions"],
                    "description": spec["description"],
                    "displayName": spec["displayName"],
                    "icon": spec["icon"],
                    "installModes": spec["installModes"],
                    "keywords": spec["keywords"],
                    "links": spec["links"],
                    "maintainers": spec["maintainers"],
                    "maturity": spec["maturity"],
                    "provider": spec["provider"],
                    "version": spec["version"],
                },
            },
        ],
    }

    return [
        {
            "schema": "olm.package",
            "name": PACKAGE,
            "defaultChannel": "stable",
            # OperatorHub tiles use the package icon (object), not CSV spec.icon (list).
            "icon": spec["icon"][0],
        },
        {
            "schema": "olm.channel",
            "package": PACKAGE,
            "name": "stable",
            "entries": [*previous_entries, current_entry],
        },
        *previous_bundles,
        current_bundle,
    ]


def main() -> None:
    csv_obj = csv()
    dump_yaml(ROOT / "config/manifests/bases/shiftwise-operator.clusterserviceversion.yaml", csv_obj)

    bundle_manifests = ROOT / "bundle/manifests"
    bundle_manifests.mkdir(parents=True, exist_ok=True)
    current_csv_name = f"{CSV_NAME}.clusterserviceversion.yaml"
    for existing_csv in bundle_manifests.glob(f"{PACKAGE}.v*.clusterserviceversion.yaml"):
        if existing_csv.name != current_csv_name:
            existing_csv.unlink()

    dump_yaml(ROOT / f"bundle/manifests/{CSV_NAME}.clusterserviceversion.yaml", csv_obj)

    crd_src = ROOT / "config/crd/bases/shiftwise.ai_shiftwises.yaml"
    crd_dst = ROOT / "bundle/manifests/shiftwise.ai_shiftwises.yaml"
    crd_dst.parent.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(crd_src, crd_dst)

    annotations = {
        "annotations": {
            "operators.operatorframework.io.bundle.mediatype.v1": "registry+v1",
            "operators.operatorframework.io.bundle.manifests.v1": "manifests/",
            "operators.operatorframework.io.bundle.metadata.v1": "metadata/",
            "operators.operatorframework.io.bundle.package.v1": PACKAGE,
            "operators.operatorframework.io.bundle.channels.v1": "stable",
            "operators.operatorframework.io.bundle.channel.default.v1": "stable",
        }
    }
    dump_yaml(ROOT / "bundle/metadata/annotations.yaml", annotations)

    catalog_docs = catalog(csv_obj)
    catalog_path = ROOT / "catalog/shiftwise-operator/catalog.yaml"
    catalog_path.parent.mkdir(parents=True, exist_ok=True)
    with catalog_path.open("w") as fh:
        yaml.safe_dump_all(catalog_docs, fh, sort_keys=False, width=10**9)

    print(f"generated bundle and catalog for {CSV_NAME}")


if __name__ == "__main__":
    main()
