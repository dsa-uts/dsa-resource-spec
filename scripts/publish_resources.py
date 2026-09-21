"""Append immutable releases to main without overwriting concurrent source commits."""

import argparse
import contextlib
from pathlib import Path
import sys
import tempfile

from publishing import PublishError, cli, git, pin_images, read_index, release_path, run, snapshot, write_json


@contextlib.contextmanager
def publication_tree(root):
    git(root, 'fetch', 'origin', 'main')
    with tempfile.TemporaryDirectory(prefix='resource-publication-') as temp:
        tree = Path(temp) / 'main'
        git(root, 'worktree', 'add', '--detach', tree, 'origin/main')
        try:
            yield tree
        finally:
            git(root, 'worktree', 'remove', '--force', tree)


def append_releases(tree, sources, resources, source_commit):
    index = read_index(tree)
    added = []
    for resource_id, source in sources.items():
        version = source['version']
        versions = index['resources'].setdefault(resource_id, {})
        if version in versions:
            if versions[version]['source-hash'] != source['source-hash']:
                raise PublishError(f'{resource_id}/{version}: published version has different source')
            continue
        relative = release_path(resource_id, version)
        target = tree / relative
        if target.exists() or target.is_symlink() or target.parent.is_symlink():
            raise PublishError(f'Refusing to overwrite {relative}')
        write_json(target, resources[resource_id])
        versions[version] = {'path': relative, 'source-commit': source_commit, 'source-hash': source['source-hash']}
        added.append(f'{resource_id}/{version}')
    if added:
        write_json(tree / 'release/index.json', index)
        read_index(tree)
    return added


def publish(root):
    source_commit = git(root, 'rev-parse', 'HEAD')
    if git(root, 'status', '--porcelain', '--untracked-files=no'):
        raise PublishError('Publish requires an unchanged source checkout')
    sources = snapshot(root)
    # Resolve each tag once, after builds, and keep those exact results across push retries.
    resources, cache = {}, {}
    for attempt in range(5):
        with publication_tree(root) as tree:
            index = read_index(tree)
            for resource_id, source in sources.items():
                existing = index['resources'].get(resource_id, {}).get(source['version'])
                if existing:
                    if existing['source-hash'] != source['source-hash']:
                        raise PublishError(f'{resource_id}/{source["version"]}: published version has different source')
                    continue
                if resource_id not in resources:
                    resources[resource_id] = pin_images(cli('show', root, resource_id), cache)
            added = append_releases(tree, sources, resources, source_commit)
            if not added:
                print('All resource versions are already published.', flush=True)
                return
            git(tree, 'add', '--', 'release')
            git(tree, '-c', 'user.name=github-actions[bot]', '-c',
                'user.email=41898282+github-actions[bot]@users.noreply.github.com',
                'commit', '-m', f'Publish resources: {", ".join(added)}')
            result = run('git', 'push', 'origin', 'HEAD:refs/heads/main', cwd=tree, check=False)
            if not result.returncode:
                print(f'Published {", ".join(added)} from {source_commit}', flush=True)
                return
            # Retry only a non-fast-forward race. Permission and policy failures need intervention.
            if not any(reason in result.stderr for reason in ('fetch first', 'non-fast-forward')):
                raise PublishError(result.stderr)
            print(f'main advanced; retrying publication ({attempt + 1}/5)', flush=True)
    raise PublishError('main kept advancing; rerun this workflow to publish')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--root', type=Path, default=Path('.'))
    args = parser.parse_args()
    publish(args.root.resolve())


if __name__ == '__main__':
    try:
        main()
    except (PublishError, ValueError, OSError) as error:
        sys.exit(str(error))
