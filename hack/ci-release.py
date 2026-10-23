import argparse
import json
import os
import re
import subprocess


REPOSITORY = "ShiftWise-AI/kubeoptix-operator"
SEMVER = re.compile(r"(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)")


def run(*args):
    return subprocess.check_output(args, text=True).strip()


def next_version(tags):
    versions = [tuple(map(int, tag[1:].split("."))) for tag in tags
                if tag.startswith("v") and SEMVER.fullmatch(tag[1:])]
    if not versions:
        return "1.0.0"
    major, minor, patch = max(versions)
    return f"{major}.{minor}.{patch + 1}"


def verify_source(source):
    if not re.fullmatch(r"[0-9a-f]{40}", source):
        raise ValueError("Expected full source commit SHA")
    run("git", "merge-base", "--is-ancestor", source, "origin/main")
    prs = json.loads(run("gh", "api", f"repos/{REPOSITORY}/commits/{source}/pulls?per_page=100"))
    for pr in prs:
        base = pr.get("base", {})
        head = pr.get("head", {})
        if (pr.get("merged_at") and pr.get("merge_commit_sha") == source
                and base.get("ref") == "main"
                and (base.get("repo") or {}).get("full_name") == REPOSITORY
                and (head.get("repo") or {}).get("full_name") == REPOSITORY
                and (head.get("ref") == "develop"
                     or re.fullmatch(r"fix/[^\s]+", head.get("ref", "")))):
            return
    raise ValueError("Release source must be the merge commit of an allowed same-repository PR to main")


def reserve(source):
    run("git", "fetch", "origin", "main", "--tags")
    verify_source(source)
    tags = run("git", "tag", "--list", "v*").splitlines()
    marker = f"Release-Source: {source}"
    matches = [tag for tag in tags if SEMVER.fullmatch(tag[1:]) and
               marker in run("git", "for-each-ref", "--format=%(contents)", f"refs/tags/{tag}").splitlines()]
    if len(matches) > 1:
        raise ValueError("Multiple releases reserved for the same source")
    if matches:
        tag = matches[0]
        if run("git", "rev-parse", f"{tag}^{{commit}}") != source:
            raise ValueError("Reserved release has an unexpected source commit")
    else:
        tag = f"v{next_version(tags)}"
        run("git", "tag", "-a", tag, source, "-m", marker)
        run("git", "push", "origin", f"refs/tags/{tag}")
    run("git", "checkout", "--detach", tag)
    with open(os.environ["GITHUB_OUTPUT"], "a") as stream:
        stream.write(f"version={tag[1:]}\ntag={tag}\nsource={source}\n")


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--source", required=True)
    reserve(parser.parse_args().source)