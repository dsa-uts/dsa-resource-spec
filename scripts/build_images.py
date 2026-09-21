"""Build every sandbox with cache; push only when its registry digest changes."""

import argparse
from datetime import datetime, timezone
import json
from pathlib import Path
import re
import sys
import tempfile

from publishing import DIGEST, PublishError, cli, digest, run


def build_images(root, gha_cache=False):
    images = cli('catalog', root)['sandbox-images']
    names = set()
    for image_id, build in images.items():
        image = build['image']
        if not re.fullmatch(r'ghcr\.io/[a-z0-9][a-z0-9._-]*(?:/[a-z0-9][a-z0-9._-]*)+', image) or image in names:
            raise PublishError(f'{image_id}: expected a unique, untagged GHCR repository, got {image}')
        names.add(image)
    for image_id, build in sorted(images.items()):
        image = build['image']
        with tempfile.TemporaryDirectory(prefix='resource-image-') as temp:
            layout = Path(temp) / 'image'
            args = [
                'docker', 'buildx', 'build', '--file', build['dockerfile'],
                '--platform', ','.join(build['platforms']),
                '--provenance=false', '--sbom=false',
                '--build-arg', 'SOURCE_DATE_EPOCH=0',
                '--output', f'type=oci,dest={layout},tar=false,rewrite-timestamp=true',
            ]
            if gha_cache:
                args += ['--cache-from', f'type=gha,version=2,scope=sandbox-{image_id}',
                         '--cache-to', f'type=gha,version=2,scope=sandbox-{image_id},mode=max']
            run(*args, build['context'], cwd=root, capture=False)
            # BuildKit's layout need not have a "latest" tag. Select its sole
            # exported manifest (or multi-platform index) explicitly by digest.
            manifests = json.loads((layout / 'index.json').read_text())['manifests']
            if len(manifests) != 1 or not DIGEST.fullmatch(manifests[0]['digest']):
                raise PublishError(f'{image_id}: expected one exported OCI image')
            local = f'ocidir://{layout}@{manifests[0]["digest"]}'
            built = digest(local)
            latest = f'{image}:latest'
            if digest(latest, missing_ok=True) == built:
                print(f'{image_id}: unchanged ({built})', flush=True)
                continue
            timestamp = datetime.now(timezone.utc).strftime('%Y%m%d%H%M%S')
            fixed = f'{image}:{timestamp}-{built.replace(":", "-")}'
            # Keep a retention tag before moving latest. A failed copy stops publication.
            run('regctl', 'image', 'copy', local, fixed, capture=False)
            if digest(fixed) != built:
                raise PublishError(f'Digest changed while pushing {fixed}')
            run('regctl', 'image', 'copy', f'{image}@{built}', latest, capture=False)
            if digest(latest) != built:
                raise PublishError(f'Unexpected latest digest: {latest}')
            print(f'{image_id}: published {fixed}', flush=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--root', type=Path, default=Path('.'))
    parser.add_argument('--gha-cache', action='store_true')
    args = parser.parse_args()
    build_images(args.root.resolve(), args.gha_cache)


if __name__ == '__main__':
    try:
        main()
    except (PublishError, ValueError, OSError) as error:
        sys.exit(str(error))
