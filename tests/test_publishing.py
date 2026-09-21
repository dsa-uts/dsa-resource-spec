"""Exercise real CLI loading and Git publication; registry/build boundaries are simulated."""

import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / 'scripts'))
import build_images
import check_resources
import publish_resources
import publishing as p

D1 = 'sha256:' + '1' * 64
D2 = 'sha256:' + '2' * 64
IMAGE = 'ghcr.io/example/default'

# A process-level registry substitute: commands, authentication errors and stored tags
# behave independently of the Python publication code.
FAKE_TOOL = r'''
import json, os, pathlib, sys
state_path = pathlib.Path(os.environ['FAKE_REGISTRY'])
state = json.loads(state_path.read_text())
args = sys.argv[1:]
state.setdefault('calls', []).append([pathlib.Path(sys.argv[0]).name, *args])
state_path.write_text(json.dumps(state))
if pathlib.Path(sys.argv[0]).name == 'docker':
    output = args[args.index('--output') + 1]
    destination = next(part[5:] for part in output.split(',') if part.startswith('dest='))
    layout = pathlib.Path(destination)
    layout.mkdir()
    (layout / 'index.json').write_text(json.dumps({'manifests': [{'digest': state['built']}]}))
    sys.exit(1 if state.get('build-fails') else 0)
if args[:2] == ['image', 'digest']:
    ref = args[2]
    if state.get('auth-fails') and not ref.startswith('ocidir://'):
        sys.exit('unauthorized: authentication required')
    value = state.get('built') if ref.startswith('ocidir://') else state['images'].get(ref)
    if not value:
        sys.exit('manifest unknown: not found')
    print(value)
elif args[:2] == ['image', 'copy']:
    src, dst = args[2:]
    if state.get('copy-fails'):
        sys.exit('upload failed')
    value = state.get('built') if src.startswith('ocidir://') else src.rsplit('@', 1)[1]
    state['images'][dst] = value
    state_path.write_text(json.dumps(state))
else:
    sys.exit('unexpected command')
'''


class PublicationTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.tools = tempfile.TemporaryDirectory()
        cls.binary = os.environ.get('RESOURCE_SPEC')
        if not cls.binary:
            cls.binary = str(Path(cls.tools.name) / 'resource-spec')
            subprocess.run(['go', 'build', '-o', cls.binary, './cmd/resource-spec'], cwd=ROOT, check=True)

    @classmethod
    def tearDownClass(cls):
        cls.tools.cleanup()

    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.directory = Path(self.temp.name)
        self.root = self.directory / 'source'
        shutil.copytree(ROOT / 'testdata/cli/valid/basic/input', self.root)
        self.remote = self.directory / 'remote.git'
        self.bin = self.directory / 'bin'
        self.bin.mkdir()
        for tool in ('docker', 'regctl'):
            file = self.bin / tool
            file.write_text(f'#!{sys.executable}\n' + FAKE_TOOL)
            file.chmod(0o755)
        self.state_path = self.directory / 'registry.json'
        self.set_state({'images': {IMAGE + ':latest': D1}, 'built': D1})
        env = patch.dict(os.environ, {
            'RESOURCE_SPEC': self.binary,
            'PATH': str(self.bin) + os.pathsep + os.environ['PATH'],
            'FAKE_REGISTRY': str(self.state_path),
            'GIT_CONFIG_GLOBAL': os.devnull,
            'GIT_CONFIG_NOSYSTEM': '1',
        })
        env.start()
        self.addCleanup(env.stop)
        p.git(self.root, 'init', '-b', 'main')
        p.git(self.root, 'config', 'user.name', 'Test')
        p.git(self.root, 'config', 'user.email', 'test@example.com')
        self.initial = self.commit()
        p.run('git', 'init', '--bare', '--initial-branch=main', self.remote)
        p.git(self.root, 'remote', 'add', 'origin', self.remote)
        p.git(self.root, 'push', '-u', 'origin', 'main')

    def commit(self):
        p.git(self.root, 'add', '.')
        p.git(self.root, 'commit', '-m', 'Test change')
        return p.git(self.root, 'rev-parse', 'HEAD')

    def set_state(self, state):
        self.state_path.write_text(json.dumps(state))

    def state(self):
        return json.loads(self.state_path.read_text())

    def remote_file(self, path):
        return p.git(self.remote, 'show', f'main:{path}')

    def bump(self, version):
        file = self.root / 'sample/resource.yaml'
        file.write_text(file.read_text().replace('v1.0.0', version))

    def setup_build(self):
        (self.root / 'sandbox').mkdir()
        (self.root / 'sandbox/Dockerfile').write_text('FROM scratch\n')
        (self.root / 'manifest.yaml').write_text('''resources:
  - id: sample
    path: sample
sandbox-images:
  default:
    context: sandbox
    dockerfile: sandbox/Dockerfile
    image: ghcr.io/example/default
    platforms: [linux/amd64, linux/arm64]
''')

    def test_source_changes_require_version_bump(self):
        for relative in ('sample/resource.yaml', 'sample/description.md', 'sample/expected.txt'):
            with self.subTest(relative=relative):
                file = self.root / relative
                original = file.read_bytes()
                file.write_bytes(original + b'\n')
                with self.assertRaisesRegex(p.PublishError, 'bump resource.version'):
                    check_resources.check(self.root, self.initial)
                file.write_bytes(original)
        (self.root / 'sample/expected.txt').write_text('changed\n')
        self.bump('v1.0.1')
        check_resources.check(self.root, self.initial)

    def test_shared_materials_and_executable_bits_are_tracked(self):
        shutil.move(self.root / 'sample/expected.txt', self.root / 'expected.txt')
        file = self.root / 'sample/resource.yaml'
        file.write_text(file.read_text().replace('path: expected.txt', 'path: ../expected.txt'))
        base = self.commit()
        (self.root / 'expected.txt').write_text('shared material changed\n')
        with self.assertRaisesRegex(p.PublishError, 'bump resource.version'):
            check_resources.check(self.root, base)
        base = self.commit()
        (self.root / 'expected.txt').chmod(0o755)
        with self.assertRaisesRegex(p.PublishError, 'bump resource.version'):
            check_resources.check(self.root, base)

    def test_sandbox_and_unreferenced_changes_do_not_require_bump(self):
        self.setup_build()
        (self.root / 'sample/unused.txt').write_text('not a resource input')
        check_resources.check(self.root, self.initial)

    def test_semver_must_increase(self):
        for version in ('v0.9.9', 'v1.0.0-rc.1', 'v1.0.0+build.2'):
            with self.subTest(version=version):
                file = self.root / 'sample/resource.yaml'
                original = file.read_text()
                self.bump(version)
                with self.assertRaisesRegex(p.PublishError, 'must increase'):
                    check_resources.check(self.root, self.initial)
                file.write_text(original)
        self.assertLess(p.version_key('v1.0.0-rc.2'), p.version_key('v1.0.0-rc.10'))
        self.assertLess(p.version_key('v1.0.0-rc.10'), p.version_key('v1.0.0'))

    def test_publish_snapshot_and_rerun_without_registry_access(self):
        publish_resources.publish(self.root)
        path = 'release/sample/v1.0.0.json'
        resource = json.loads(self.remote_file(path))
        for job in resource['workflows']['main']['jobs'].values():
            self.assertEqual(job['sandbox-image'], IMAGE + '@' + D1)
        index = json.loads(self.remote_file('release/index.json'))
        self.assertEqual(index['resources']['sample']['v1.0.0']['source-commit'], self.initial)
        self.assertEqual(len(self.state()['calls']), 1)  # Same tag is resolved once.
        head = p.git(self.remote, 'rev-parse', 'main')
        self.set_state({'images': {}, 'auth-fails': True})
        publish_resources.publish(self.root)
        self.assertEqual(p.git(self.remote, 'rev-parse', 'main'), head)
        self.assertNotIn('calls', self.state())

    def test_resolution_failure_publishes_nothing(self):
        self.set_state({'images': {}})
        with self.assertRaisesRegex(p.PublishError, 'Cannot resolve'):
            publish_resources.publish(self.root)
        self.assertEqual(p.git(self.remote, 'rev-parse', 'main'), self.initial)

    def test_pinned_image_needs_no_resolution(self):
        file = self.root / 'sample/resource.yaml'
        file.write_text(file.read_text().replace(IMAGE + ':latest', IMAGE + '@' + D1))
        self.commit()
        p.git(self.root, 'push', 'origin', 'main')
        self.set_state({'images': {}, 'auth-fails': True})
        publish_resources.publish(self.root)
        self.assertNotIn('calls', self.state())

    def test_published_bytes_and_index_cannot_change(self):
        publish_resources.publish(self.root)
        p.git(self.root, 'pull', '--ff-only')
        base = p.git(self.root, 'rev-parse', 'HEAD')
        file = self.root / 'release/sample/v1.0.0.json'
        original = file.read_bytes()
        file.write_bytes(original + b'\n')
        with self.assertRaisesRegex(p.PublishError, 'Published JSON changed'):
            check_resources.check(self.root, base)
        file.unlink()
        with self.assertRaises(p.PublishError):
            check_resources.check(self.root, base)
        file.write_bytes(original)
        index = p.read_index(self.root)
        index['resources']['sample']['v1.0.0']['source-commit'] = '0' * 40
        p.write_json(self.root / 'release/index.json', index)
        with self.assertRaisesRegex(p.PublishError, 'Published index entry changed'):
            check_resources.check(self.root, base)

    def test_publish_rejects_reused_version(self):
        publish_resources.publish(self.root)
        (self.root / 'sample/expected.txt').write_text('changed')
        self.commit()
        with self.assertRaisesRegex(p.PublishError, 'different source'):
            publish_resources.publish(self.root)

    def test_queued_snapshots_both_publish(self):
        self.bump('v1.1.0')
        second = self.commit()
        p.git(self.root, 'push', 'origin', 'main')
        p.git(self.root, 'checkout', '--detach', self.initial)
        publish_resources.publish(self.root)
        p.git(self.root, 'checkout', '--detach', second)
        self.set_state({'images': {IMAGE + ':latest': D2}})
        publish_resources.publish(self.root)
        index = json.loads(self.remote_file('release/index.json'))
        self.assertEqual(set(index['resources']['sample']), {'v1.0.0', 'v1.1.0'})
        first = json.loads(self.remote_file('release/sample/v1.0.0.json'))
        second_json = json.loads(self.remote_file('release/sample/v1.1.0.json'))
        self.assertEqual(first['workflows']['main']['jobs']['build']['sandbox-image'], IMAGE + '@' + D1)
        self.assertEqual(second_json['workflows']['main']['jobs']['build']['sandbox-image'], IMAGE + '@' + D2)
        self.assertIn('v1.1.0', self.remote_file('sample/resource.yaml'))

    def test_push_race_preserves_new_main_and_original_snapshot(self):
        original_run = publish_resources.run
        raced = False

        def run_with_race(*args, **kwargs):
            nonlocal raced
            if args[:3] == ('git', 'push', 'origin') and not raced:
                raced = True
                self.bump('v1.1.0')
                self.commit()
                p.git(self.root, 'push', 'origin', 'main')
                self.set_state({'images': {IMAGE + ':latest': D2}})
            return original_run(*args, **kwargs)

        with patch.object(publish_resources, 'run', side_effect=run_with_race):
            publish_resources.publish(self.root)
        self.assertTrue(raced)
        self.assertIn('v1.1.0', self.remote_file('sample/resource.yaml'))
        resource = json.loads(self.remote_file('release/sample/v1.0.0.json'))
        self.assertEqual(resource['workflows']['main']['jobs']['build']['sandbox-image'], IMAGE + '@' + D1)
        index = json.loads(self.remote_file('release/index.json'))
        self.assertEqual(index['resources']['sample']['v1.0.0']['source-commit'], self.initial)

    def test_build_unchanged_skips_push(self):
        self.setup_build()
        build_images.build_images(self.root, gha_cache=True)
        calls = self.state()['calls']
        self.assertEqual(sum(c[0] == 'docker' for c in calls), 1)
        self.assertFalse(any(c[1:3] == ['image', 'copy'] for c in calls))

    def test_build_changed_adds_utc_retention_tag_then_latest(self):
        self.setup_build()
        self.set_state({'images': {IMAGE + ':latest': D1}, 'built': D2})
        build_images.build_images(self.root)
        state = self.state()
        self.assertEqual(state['images'][IMAGE + ':latest'], D2)
        copies = [call for call in state['calls'] if call[1:3] == ['image', 'copy']]
        self.assertEqual(len(copies), 2)
        self.assertRegex(copies[0][-1], r':\d{14}-sha256-' + '2' * 64 + '$')
        self.assertEqual(copies[1][-1], IMAGE + ':latest')
        self.assertEqual(len(copies[0][-1].rsplit(':', 1)[1]), 86)

    def test_first_build_pushes_both_tags(self):
        self.setup_build()
        self.set_state({'images': {}, 'built': D1})
        build_images.build_images(self.root)
        self.assertEqual(self.state()['images'][IMAGE + ':latest'], D1)

    def test_build_auth_and_copy_errors_stop_publication(self):
        self.setup_build()
        for flag in ('build-fails', 'auth-fails', 'copy-fails'):
            with self.subTest(flag=flag):
                self.set_state({'images': {IMAGE + ':latest': D1}, 'built': D2, flag: True})
                with self.assertRaises(p.PublishError):
                    build_images.build_images(self.root)
                self.assertEqual(self.state()['images'][IMAGE + ':latest'], D1)

    def test_repository_port_survives_tag_resolution(self):
        value = p.cli('show', self.root, 'sample')
        for workflow in value['workflows'].values():
            for job in workflow['jobs'].values():
                job['sandbox-image'] = 'registry.example:5000/sandbox:release'
        self.set_state({'images': {'registry.example:5000/sandbox:release': D1}})
        p.pin_images(value, {})
        self.assertEqual(value['workflows']['main']['jobs']['build']['sandbox-image'],
                         'registry.example:5000/sandbox@' + D1)


if __name__ == '__main__':
    unittest.main()
