import importlib.util
import json
from pathlib import Path
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location("publish", Path(__file__).parents[1] / "scripts/publish.py")
publish = importlib.util.module_from_spec(spec)
spec.loader.exec_module(publish)


class PublicationTests(unittest.TestCase):
    manifest = {"resources": [{"id": "sample", "path": "sample/resource.yaml"}]}

    def runner(self, *args):
        if args[1] == "inspect":
            return json.dumps({"resource": {"version": "v1.0.0"}, "workflows": {}})
        if args[1] == "compare":
            return "-1" if args[3] == "v2.0.0" else "0" if args[3] in ("v1.0.0", "v1.0.0+build") else "1"
        raise AssertionError(args)

    def test_empty_manifest_needs_no_publication_services(self):
        empty = {"resources": [], "sandbox-images": {}}
        with patch.object(publish, "run", return_value=json.dumps(empty)) as run:
            publish.main()
        run.assert_called_once_with("resource-spec", "manifest", ".")

    def test_published_release_is_untouched(self):
        with patch.object(publish, "run", side_effect=self.runner):
            self.assertEqual([], publish.pending_resources(self.manifest, [
                {"tag_name": "sample/v1.0.0", "draft": False}]))

    def test_new_version(self):
        with patch.object(publish, "run", side_effect=self.runner):
            pending = publish.pending_resources(self.manifest, [{"tag_name": "sample/v0.9.0", "draft": False}])
            self.assertEqual(pending[0][0], "sample/v1.0.0")

    def test_downgrade_and_metadata_only_change(self):
        for tag in ["sample/v2.0.0", "sample/v1.0.0+build"]:
            with self.subTest(tag=tag), patch.object(publish, "run", side_effect=self.runner):
                with self.assertRaisesRegex(RuntimeError, "version must exceed"):
                    publish.pending_resources(self.manifest, [{"tag_name": tag, "draft": False}])

    def test_existing_draft_requires_inspection(self):
        with patch.object(publish, "run", side_effect=self.runner):
            with self.assertRaisesRegex(RuntimeError, "existing draft"):
                publish.pending_resources(self.manifest, [{"tag_name": "sample/v1.0.0", "draft": True}])


    def test_old_published_version_cannot_be_selected_again(self):
        with patch.object(publish, "run", side_effect=self.runner):
            with self.assertRaisesRegex(RuntimeError, "version must exceed"):
                publish.pending_resources(self.manifest, [
                    {"tag_name": "sample/v1.0.0", "draft": False},
                    {"tag_name": "sample/v2.0.0", "draft": False}])

    def test_full_flow_build_resolve_archive_draft_publish(self):
        calls = []
        definition = {"resource": {"version": "v1.0.0"}, "workflows": {
            "test": {"jobs": {"run": {"sandbox-image": "ghcr.io/example/image:release"}}}}}
        manifest = {**self.manifest, "sandbox-images": {"default": {
            "image": "ghcr.io/example/image", "platforms": ["linux/amd64"],
            "dockerfile": "sandbox/Dockerfile", "context": "sandbox"}}}
        def fake(*args):
            calls.append(args)
            if args == ("git", "status", "--porcelain"):
                return ""
            if args == ("git", "rev-parse", "HEAD"):
                return "a" * 40
            if args[:2] == ("resource-spec", "manifest"):
                return json.dumps(manifest)
            if args[:2] == ("resource-spec", "inspect"):
                return json.dumps(definition)
            if args[:4] == ("gh", "api", "--paginate", "--slurp"):
                return "[[]]"
            if args[:2] == ("gh", "api"):
                if args[2].endswith("commits/main"):
                    return json.dumps({"sha": "a" * 40})
                if "matching-refs" in args[2]:
                    return "[]"
            if args[:4] == ("docker", "buildx", "imagetools", "inspect"):
                return "sha256:" + "b" * 64
            if args[:3] == ("docker", "buildx", "build"):
                return ""
            if args[:2] == ("resource-spec", "archive"):
                mapping = json.loads(Path(args[3]).read_text())
                self.assertEqual(mapping, {"ghcr.io/example/image:release": "ghcr.io/example/image@sha256:" + "b" * 64})
                Path(args[5]).write_bytes(b"test archive")
                return ""
            if args[:2] == ("gh", "release"):
                return ""
            raise AssertionError(args)
        with patch.object(publish, "run", side_effect=fake), patch.dict(publish.os.environ, {
            "GITHUB_REPOSITORY": "example/resources", "IMMUTABLE_RELEASES_ENABLED": "true"}):
            publish.main()
        create = next(i for i, c in enumerate(calls) if c[:3] == ("gh", "release", "create"))
        edit = next(i for i, c in enumerate(calls) if c[:3] == ("gh", "release", "edit"))
        self.assertLess(create, edit)
        self.assertIn("--draft", calls[create])
        self.assertIn("--draft=false", calls[edit])
        build = next(c for c in calls if c[:3] == ("docker", "buildx", "build"))
        self.assertIn("ghcr.io/example/image:latest", build)
        self.assertIn("--cache-from", build)
        self.assertIn("--cache-to", build)


if __name__ == "__main__":
    unittest.main()
