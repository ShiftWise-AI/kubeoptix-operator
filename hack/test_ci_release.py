import importlib.util
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

import yaml


SPEC = importlib.util.spec_from_file_location("ci_release", Path(__file__).with_name("ci-release.py"))
release = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(release)
SOURCE = "a" * 40


class ReleaseTests(unittest.TestCase):
    def test_initial_version(self):
        self.assertEqual(release.next_version([]), "1.0.0")

    def test_patch_increment_uses_numeric_order(self):
        self.assertEqual(release.next_version(["v1.0.9", "v1.0.10", "v0.9.99"]), "1.0.11")

    def test_ignores_non_release_tags(self):
        self.assertEqual(release.next_version(["v1.0.5", "v2.0.0-rc1", "v01.0.9", "latest"]), "1.0.6")

    def test_increments_highest_major_and_minor(self):
        self.assertEqual(release.next_version(["v1.99.9", "v2.1.3"]), "2.1.4")

    def test_rejects_invalid_source_before_running_commands(self):
        with patch.object(release, "run") as run:
            with self.assertRaises(ValueError):
                release.verify_source("main")
            run.assert_not_called()

    def test_verifies_same_repository_merge(self):
        pr = {"merged_at": "2026-10-22", "merge_commit_sha": SOURCE,
              "base": {"ref": "main", "repo": {"full_name": release.REPOSITORY}},
              "head": {"ref": "develop", "repo": {"full_name": release.REPOSITORY}}}
        with patch.object(release, "run", side_effect=["", json.dumps([pr])]):
            release.verify_source(SOURCE)
        pr["head"]["repo"]["full_name"] = "fork/operator"
        with patch.object(release, "run", side_effect=["", json.dumps([pr])]):
            with self.assertRaises(ValueError):
                release.verify_source(SOURCE)

    def test_rejects_commit_without_merged_pr(self):
        with patch.object(release, "run", side_effect=["", "[]"]):
            with self.assertRaises(ValueError):
                release.verify_source(SOURCE)

    def reserve(self, tags, annotations, commit=SOURCE):
        def command(*args):
            if args[1:3] == ("tag", "--list"):
                return "\n".join(tags)
            if args[1] == "for-each-ref":
                return annotations.get(args[-1].split("/")[-1], "")
            if args[1] == "rev-parse":
                return commit
            return ""

        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "output"
            with patch.dict(os.environ, GITHUB_OUTPUT=str(output)), \
                    patch.object(release, "verify_source"), \
                    patch.object(release, "run", side_effect=command) as run:
                release.reserve(SOURCE)
            return run, output.read_text()

    def test_reserves_next_tag(self):
        run, output = self.reserve(["v1.0.5"], {})
        run.assert_any_call("git", "tag", "-a", "v1.0.6", SOURCE, "-m", f"Release-Source: {SOURCE}")
        run.assert_any_call("git", "push", "origin", "refs/tags/v1.0.6")
        self.assertIn("version=1.0.6\n", output)

    def test_retry_reuses_reservation_without_pushing(self):
        run, output = self.reserve(["v1.0.6", "v1.0.7"], {"v1.0.6": f"Release-Source: {SOURCE}"})
        self.assertIn("version=1.0.6\n", output)
        self.assertFalse(any(call.args[1] == "push" for call in run.call_args_list))

    def test_rejects_multiple_reservations(self):
        with self.assertRaisesRegex(ValueError, "Multiple releases"):
            self.reserve(["v1.0.6", "v1.0.7"], dict.fromkeys(["v1.0.6", "v1.0.7"], f"Release-Source: {SOURCE}"))

    def test_rejects_reservation_pointing_to_another_commit(self):
        with self.assertRaisesRegex(ValueError, "unexpected source"):
            self.reserve(["v1.0.6"], {"v1.0.6": f"Release-Source: {SOURCE}"}, commit="b" * 40)

    def test_reservation_round_trip_with_local_remote(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            remote = root / "remote.git"
            checkout = root / "checkout"
            subprocess.check_call(["git", "init", "--bare", "--quiet", str(remote)])
            subprocess.check_call(["git", "init", "--quiet", "--initial-branch=main", str(checkout)])

            def command(*args):
                return subprocess.check_output(args, cwd=checkout, text=True).strip()

            command("git", "config", "user.name", "Release test")
            command("git", "config", "user.email", "release@example.test")
            command("git", "commit", "--quiet", "--allow-empty", "-m", "Initial source")
            source = command("git", "rev-parse", "HEAD")
            command("git", "tag", "v1.0.5")
            command("git", "remote", "add", "origin", str(remote))
            command("git", "push", "--quiet", "origin", "main", "--tags")
            output = root / "output"
            with patch.dict(os.environ, GITHUB_OUTPUT=str(output)), \
                    patch.object(release, "verify_source"), \
                    patch.object(release, "run", side_effect=command):
                release.reserve(source)
                release.reserve(source)
            self.assertEqual(command("git", "tag", "--list"), "v1.0.5\nv1.0.6")
            self.assertEqual(command("git", "rev-parse", "v1.0.6^{commit}"), source)
            self.assertIn("refs/tags/v1.0.6", command("git", "ls-remote", "--tags", "origin"))
            self.assertEqual(output.read_text().count("version=1.0.6\n"), 2)


class WorkflowTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        workflow = Path(__file__).resolve().parents[1] / ".github/workflows/publish-quay.yml"
        cls.workflow = yaml.load(workflow.read_text(), Loader=yaml.BaseLoader)
        cls.publish = cls.workflow["jobs"]["publish"]

    def step(self, name):
        return next(step for step in self.publish["steps"] if step.get("name") == name)

    def test_only_main_push_and_manual_resume_trigger_releases(self):
        triggers = self.workflow["on"]
        self.assertEqual(triggers["push"], {"branches": ["main"]})
        self.assertIn("source", triggers["workflow_dispatch"]["inputs"])
        self.assertIn("github.ref == 'refs/heads/main'", self.publish["if"])

    def test_releases_use_one_lossless_queue_and_scoped_permissions(self):
        self.assertNotIn("concurrency", self.workflow)
        self.assertEqual(self.publish["concurrency"], {
            "group": "operator-production-release", "cancel-in-progress": "false", "queue": "max"})
        self.assertEqual(self.publish["environment"], "quay-release")
        self.assertEqual(self.publish["permissions"]["contents"], "write")
        self.assertEqual(self.workflow["permissions"]["contents"], "read")

    def test_reservation_runs_trusted_main_code_after_credential_check(self):
        self.assertEqual(self.step("Check out source")["with"]["ref"], "main")
        names = [step.get("name") for step in self.publish["steps"]]
        self.assertLess(names.index("Check credentials before reserving a release"),
                        names.index("Reserve or resume version"))
        self.assertEqual(self.step("Reserve or resume version")["env"]["SOURCE"],
                         "${{ inputs.source || github.sha }}")

    def test_pr_tests_are_mandatory(self):
        self.assertNotIn("if", self.workflow["jobs"]["test"])
        gate = self.workflow["jobs"]["ci-required"]["steps"][0]["run"]
        self.assertNotIn('.test.result == "skipped"', gate)
        self.assertIn('.test.result == "success"', gate)

    def test_component_version_is_independent_from_operator_release(self):
        build = self.step("Build Operator image without publishing")["with"]
        self.assertIn("COMPONENT_IMAGE_TAG=${{ env.COMPONENT_IMAGE_TAG }}", build["build-args"])
        self.assertNotIn("COMPONENT_IMAGE_TAG=${{ steps.release.outputs.version }}", build["build-args"])
        self.assertEqual(self.publish["env"]["COMPONENT_IMAGE_TAG"], "1.0.5")

    def test_all_release_images_support_reuse_and_provenance(self):
        for kind, name in (("operator", "Operator"), ("bundle", "OLM bundle"), ("catalog", "OLM catalog")):
            build = self.step(f"Build {name} image without publishing")
            self.assertEqual(build["if"], f"steps.existing.outputs.{kind} != 'true'")
            self.assertIn("org.opencontainers.image.revision=${{ steps.release.outputs.source }}",
                          build["with"]["labels"])
            self.assertNotIn("github.sha", build["with"]["tags"])

    def test_mutable_aliases_require_current_source_and_highest_version(self):
        script = self.step("Update current release aliases without rolling back")["run"]
        self.assertIn('"$(git rev-parse origin/main)" != "$SOURCE"', script)
        self.assertIn('"$highest" != "v$VERSION"', script)

    def test_every_workflow_shell_step_has_valid_bash_syntax(self):
        for job in self.workflow["jobs"].values():
            for step in job.get("steps", []):
                if "run" in step:
                    with self.subTest(step=step.get("name", "unnamed")):
                        subprocess.run(["bash", "-n"], input=step["run"], text=True, check=True)

    def run_alias_update(self, current, highest):
        helpers = (
            'git() { case "$1" in '
            'fetch) return 0 ;; '
            'tag) printf "%s\\n" "$HIGHEST" ;; '
            'rev-parse) printf "%s\\n" "$CURRENT" ;; '
            '*) return 1 ;; esac; }\n'
            'docker() { if [[ "$1 $2" == "buildx imagetools" ]]; then '
            'case "$*" in '
            '*-catalog:stable*) printf "%s\\n" "$CATALOG_DIGEST" ;; '
            '*) printf "%s\\n" "$OPERATOR_DIGEST" ;; esac; '
            'else printf "%s\\n" "$*" >> "$DOCKER_LOG"; fi; }\n'
        )
        with tempfile.TemporaryDirectory() as directory:
            log = Path(directory) / "docker.log"
            environment = dict(os.environ, SOURCE=SOURCE, CURRENT=current, HIGHEST=highest,
                               VERSION="1.0.6", OPERATOR_DIGEST="sha256:" + "b" * 64,
                               CATALOG_DIGEST="sha256:" + "c" * 64, DOCKER_LOG=str(log))
            script = self.step("Update current release aliases without rolling back")["run"]
            subprocess.run(["bash", "-e", "-o", "pipefail", "-c", helpers + script],
                           env=environment, capture_output=True, text=True, check=True)
            return log.read_text() if log.exists() else ""

    def test_current_release_updates_both_aliases(self):
        commands = self.run_alias_update(SOURCE, "v1.0.6")
        self.assertIn("push quay.io/parraes/shiftwise-operator:latest\n", commands)
        self.assertIn("push quay.io/parraes/shiftwise-operator-catalog:stable\n", commands)

    def test_historical_source_does_not_update_aliases(self):
        self.assertEqual(self.run_alias_update("d" * 40, "v1.0.6"), "")

    def test_older_version_does_not_update_aliases(self):
        self.assertEqual(self.run_alias_update(SOURCE, "v1.0.7"), "")


if __name__ == "__main__":
    unittest.main()