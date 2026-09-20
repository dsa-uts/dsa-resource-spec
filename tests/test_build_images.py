import importlib.util
import json
from pathlib import Path
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location("build_images", Path(__file__).parents[1] / "scripts/build_images.py")
build_images = importlib.util.module_from_spec(spec)
spec.loader.exec_module(build_images)


class ImageBuildTests(unittest.TestCase):
    def test_no_images_needs_no_publication_services(self):
        for resources in ([], [{"id": "sample", "path": "exercises/sample"}]):
            with self.subTest(resources=resources):
                manifest = {"resources": resources, "sandbox-images": {}}
                with patch.object(build_images, "run", return_value=json.dumps(manifest)) as run:
                    build_images.main()
                run.assert_called_once_with("resource-spec", "manifest", ".")

    def test_build_and_push_images(self):
        manifest = {"resources": [{"id": "sample", "path": "exercises/sample"}],
                    "sandbox-images": {"sandbox": {
                        "image": "ghcr.io/example/sandbox",
                        "platforms": ["linux/amd64", "linux/arm64"],
                        "dockerfile": "sandbox/Dockerfile", "context": "sandbox"}}}
        calls = []

        def fake(*args):
            calls.append(args)
            if args == ("resource-spec", "manifest", "."):
                return json.dumps(manifest)
            if args[:3] == ("docker", "buildx", "build"):
                return ""
            raise AssertionError(args)

        with patch.object(build_images, "run", side_effect=fake):
            build_images.main()
        self.assertEqual(calls, [
            ("resource-spec", "manifest", "."),
            ("docker", "buildx", "build", "--push", "--tag", "ghcr.io/example/sandbox:latest",
             "--platform", "linux/amd64,linux/arm64", "--file", "sandbox/Dockerfile",
             "--cache-from", "type=gha,scope=sandbox", "--cache-to", "type=gha,mode=max,scope=sandbox",
             "sandbox"),
        ])


if __name__ == "__main__":
    unittest.main()
