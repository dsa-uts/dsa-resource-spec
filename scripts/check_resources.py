"""Validate source changes and protect published resources (no registry access)."""

import argparse
from pathlib import Path
import sys

from publishing import PublishError, check_versions, git, read_index, revision, snapshot


def check(root, base):
    current = snapshot(root)
    index = read_index(root)
    with revision(root, base) as old:
        previous_index = read_index(old)
        for resource_id, versions in previous_index['resources'].items():
            for version, entry in versions.items():
                if index['resources'].get(resource_id, {}).get(version) != entry:
                    raise PublishError(f'Published index entry changed: {resource_id}/{version}')
                relative = entry['path']
                if (old / relative).read_bytes() != (Path(root) / relative).read_bytes():
                    raise PublishError(f'Published JSON changed: {relative}')
        check_versions(snapshot(old), current, previous_index)
    # Also check versions published after the comparison base, if any.
    check_versions({}, current, index)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--base', required=True, help='Base commit to compare (PR merge-base or push parent)')
    parser.add_argument('--root', type=Path, default=Path('.'))
    args = parser.parse_args()
    check(args.root.resolve(), git(args.root, 'rev-parse', '--verify', f'{args.base}^{{commit}}'))


if __name__ == '__main__':
    try:
        main()
    except (PublishError, ValueError, OSError) as error:
        sys.exit(str(error))
