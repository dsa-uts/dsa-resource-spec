"""Shared release operations. YAML and resource validation belong to the Go CLI."""

import contextlib
import json
import os
from pathlib import Path
import re
import subprocess
import tarfile
import tempfile


class PublishError(RuntimeError):
    pass


def run(*args, cwd=None, capture=True, check=True):
    result = subprocess.run(
        [str(arg) for arg in args], cwd=cwd, text=True,
        stdout=subprocess.PIPE if capture else None,
        stderr=subprocess.PIPE if capture else None,
    )
    if check and result.returncode:
        raise PublishError(f"Command failed ({result.returncode}): {' '.join(map(str, args))}\n{result.stderr or ''}")
    return result


def git(root, *args):
    return run('git', *args, cwd=root).stdout.strip()


def cli(command, root, *args):
    return json.loads(run(os.environ.get('RESOURCE_SPEC', 'resource-spec'), command, Path(root).resolve(), *args).stdout)


def snapshot(root):
    catalog = cli('catalog', root)
    hashes = cli('sources', root)
    return {item['id']: {**item, 'source-hash': hashes[item['id']]} for item in catalog['resources']}


def write_json(path, value):
    path = Path(path)
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, ensure_ascii=False, indent=2, sort_keys=True) + '\n', encoding='utf-8')


@contextlib.contextmanager
def revision(root, ref):
    """Read a committed snapshot without switching the working tree or running its code."""
    with tempfile.TemporaryDirectory(prefix='resource-source-') as temp:
        archive = Path(temp) / 'source.tar'
        run('git', 'archive', '--format=tar', f'--output={archive}', ref, cwd=root)
        directory = Path(temp) / 'source'
        directory.mkdir()
        with tarfile.open(archive) as stream:
            stream.extractall(directory, filter='data')
        yield directory


VERSION = re.compile(r'v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-([0-9A-Za-z.-]+))?(?:\+([0-9A-Za-z.-]+))?')
ID = re.compile(r'[a-z][a-z0-9-]*')
DIGEST = re.compile(r'sha256:[0-9a-f]{64}')
SHA = re.compile(r'[0-9a-f]{40,64}')


def version_key(version):
    match = VERSION.fullmatch(version)
    if not match:
        raise PublishError(f'Invalid version: {version}')
    major, minor, patch, pre, _ = match.groups()
    parts = tuple((0, int(p)) if p.isdecimal() else (1, p) for p in pre.split('.')) if pre else ()
    return int(major), int(minor), int(patch), pre is None, parts


def release_path(resource_id, version):
    if not ID.fullmatch(resource_id):
        raise PublishError(f'Invalid resource ID: {resource_id}')
    version_key(version)
    return f'release/{resource_id}/{version}.json'


def read_index(root):
    root = Path(root)
    path = root / 'release/index.json'
    if not path.exists():
        if (root / 'release').exists() and any((root / 'release').rglob('*.json')):
            raise PublishError('Release JSON exists without release/index.json')
        return {'resources': {}}
    if path.is_symlink() or (root / 'release').is_symlink():
        raise PublishError('Release paths must not be symlinks')
    index = json.loads(path.read_text())
    if set(index) != {'resources'} or not isinstance(index['resources'], dict):
        raise PublishError('Invalid release index')
    expected = {path}
    for resource_id, versions in index['resources'].items():
        if not isinstance(versions, dict):
            raise PublishError('Invalid release versions')
        for version, entry in versions.items():
            relative = release_path(resource_id, version)
            if (set(entry) != {'path', 'source-commit', 'source-hash'}
                    or entry['path'] != relative
                    or not SHA.fullmatch(entry['source-commit'])
                    or not DIGEST.fullmatch(entry['source-hash'])):
                raise PublishError(f'Invalid release entry: {resource_id}/{version}')
            file = root / relative
            if file.is_symlink() or file.parent.is_symlink() or not file.is_file():
                raise PublishError(f'Missing or unsafe release file: {relative}')
            resource = json.loads(file.read_text())
            if resource['metadata']['id'] != resource_id or resource['metadata']['version'] != version:
                raise PublishError(f'Release metadata mismatch: {relative}')
            for workflow in resource['workflows'].values():
                for job in workflow['jobs'].values():
                    if '@' not in job['sandbox-image'] or not DIGEST.fullmatch(job['sandbox-image'].rsplit('@', 1)[1]):
                        raise PublishError(f'Unpinned image in {relative}')
            expected.add(file)
    actual = set((root / 'release').rglob('*.json'))
    if actual != expected:
        raise PublishError('Release index does not match release JSON files')
    return index


def check_versions(before, after, index):
    for resource_id, current in after.items():
        previous = before.get(resource_id)
        if previous:
            if current['version'] == previous['version']:
                if current['source-hash'] != previous['source-hash']:
                    raise PublishError(f'{resource_id}: definition/materials changed; bump resource.version')
            elif version_key(current['version']) <= version_key(previous['version']):
                raise PublishError(f'{resource_id}: resource.version must increase')
        released = index['resources'].get(resource_id, {})
        existing = released.get(current['version'])
        if existing:
            if current['source-hash'] != existing['source-hash']:
                raise PublishError(f'{resource_id}/{current["version"]}: published version cannot be reused')
        elif released and version_key(current['version']) <= max(map(version_key, released)):
            raise PublishError(f'{resource_id}: new version must exceed published versions')


def digest(reference, missing_ok=False):
    result = run('regctl', 'image', 'digest', reference, check=False)
    if result.returncode:
        # Authentication/network failures must never look like a missing image.
        if missing_ok and re.search(r'(?i)(manifest unknown|name unknown|not found|\b404\b)', result.stderr):
            return None
        raise PublishError(f'Cannot resolve {reference}: {result.stderr}')
    value = result.stdout.strip()
    if not DIGEST.fullmatch(value):
        raise PublishError(f'Invalid registry digest for {reference}: {value}')
    return value


def pin_images(resource, cache):
    for workflow in resource['workflows'].values():
        for job in workflow['jobs'].values():
            reference = job['sandbox-image']
            if '@' in reference:
                continue
            if reference not in cache:
                cache[reference] = digest(reference)
            # Only remove the tag; retain a registry's optional port.
            repository = reference.rsplit(':', 1)[0]
            job['sandbox-image'] = f'{repository}@{cache[reference]}'
    return resource
