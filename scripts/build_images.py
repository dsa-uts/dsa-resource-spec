"""Build and push manifest sandbox images from a trusted checkout.

Requires docker, resource-spec and authenticated registry access.
"""
import json
import subprocess


def run(*args):
    return subprocess.check_output(args, text=True).strip()


def build_images(images):
    for key, build in sorted(images.items()):
        run("docker", "buildx", "build", "--push", "--tag", build["image"] + ":latest",
            "--platform", ",".join(build["platforms"]), "--file", build["dockerfile"],
            "--cache-from", f"type=gha,scope={key}", "--cache-to", f"type=gha,mode=max,scope={key}",
            build["context"])


def main():
    manifest = json.loads(run("resource-spec", "manifest", "."))
    build_images(manifest["sandbox-images"])


if __name__ == "__main__":
    main()
