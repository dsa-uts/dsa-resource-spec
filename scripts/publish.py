"""Actions orchestration. Resource parsing/validation/packaging lives in Go.

No source writes, generated image tags, or image deletion. Requires gh, docker,
resource-spec and authenticated registry access. Run only on a trusted checkout.
"""
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile
from urllib.parse import quote


def run(*args):
    return subprocess.check_output(args, text=True).strip()


def api(path, *args):
    return json.loads(run("gh", "api", path, *args))


def pending_resources(manifest, releases):
    pending = []
    published = [r for r in releases if not r["draft"]]
    for entry in manifest["resources"]:
        directory = str(Path(entry["path"]).parent)
        definition = json.loads(run("resource-spec", "inspect", directory))
        version = definition["resource"]["version"]
        tag = f'{entry["id"]}/{version}'
        for release in published:
            prefix = entry["id"] + "/"
            if release["tag_name"].startswith(prefix):
                old = release["tag_name"][len(prefix):]
                comparison = int(run("resource-spec", "compare", version, old))
                if comparison < 0 or (comparison == 0 and old != version):
                    raise RuntimeError(f"{tag}: version must exceed published {old}")
        if any(r["tag_name"] == tag for r in published):
            continue
        if any(r["tag_name"] == tag for r in releases):
            raise RuntimeError(f"{tag}: existing draft requires inspection before retry")
        pending.append((tag, directory, definition))
    return pending


def build_images(images):
    for key, build in sorted(images.items()):
        run("docker", "buildx", "build", "--push", "--tag", build["image"] + ":latest",
            "--platform", ",".join(build["platforms"]), "--file", build["dockerfile"],
            "--cache-from", f"type=gha,scope={key}", "--cache-to", f"type=gha,mode=max,scope={key}",
            build["context"])


def resolve_images(pending):
    images = set()
    for _, _, definition in pending:
        for workflow in definition["workflows"].values():
            for job in workflow["jobs"].values():
                images.add(job["sandbox-image"])
    resolved = {}
    for image in sorted(images):
        if "@" in image:
            continue
        digest = run("docker", "buildx", "imagetools", "inspect", image, "--format", "{{.Manifest.Digest}}")
        if not re.fullmatch(r"sha256:[a-f0-9]{64}", digest):
            raise RuntimeError(f"invalid registry digest for {image}: {digest}")
        resolved[image] = image.rsplit(":", 1)[0] + "@" + digest
    return resolved


def publish_release(repo, commit, tag, resource_directory, mapping, directory):
    # Abort stale runs before creating a release. Existing releases remain untouched.
    if api(f"repos/{repo}/commits/main")["sha"] != commit:
        raise RuntimeError("main advanced; retry the latest workflow")
    refs = api(f"repos/{repo}/git/matching-refs/tags/{quote(tag, safe='')}")
    if any(ref["ref"] == "refs/tags/" + tag for ref in refs):
        raise RuntimeError(f"{tag}: existing tag without published release requires inspection")
    output = directory / (tag.replace("/", "-") + ".zip")
    run("resource-spec", "archive", "--images", str(mapping), "--output", str(output), resource_directory)
    if output.stat().st_size >= 2 * 1024**3:
        raise RuntimeError("release asset must be smaller than 2 GiB")
    run("gh", "release", "create", tag, str(output), "--repo", repo,
        "--target", commit, "--draft", "--title", tag,
        "--notes", f"Resource {tag}; source commit {commit}")
    run("gh", "release", "edit", tag, "--repo", repo, "--draft=false", "--latest=false")


def main():
    manifest = json.loads(run("resource-spec", "manifest", "."))
    if not manifest["resources"] and not manifest["sandbox-images"]:
        return
    repo = os.environ["GITHUB_REPOSITORY"]
    if run("git", "status", "--porcelain"):
        raise RuntimeError("publication requires a clean checkout")
    if os.environ.get("IMMUTABLE_RELEASES_ENABLED") != "true":
        raise RuntimeError("Enable release immutability in repository settings before publication")
    commit = run("git", "rev-parse", "HEAD")
    pages = json.loads(run("gh", "api", "--paginate", "--slurp", f"repos/{repo}/releases?per_page=100"))
    releases = [release for page in pages for release in page]
    pending = pending_resources(manifest, releases)
    build_images(manifest["sandbox-images"])
    resolved = resolve_images(pending)
    with tempfile.TemporaryDirectory() as temporary_directory:
        directory = Path(temporary_directory)
        mapping = directory / "images.json"
        mapping.write_text(json.dumps(resolved))
        for tag, resource_directory, _ in pending:
            publish_release(repo, commit, tag, resource_directory, mapping, directory)


if __name__ == "__main__":
    main()
