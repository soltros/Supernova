#!/usr/bin/env python3
"""Exercise disposable Docker deployments; never mounts the user's data."""
import argparse
import json
import math
import os
from pathlib import Path
import struct
import subprocess
import tempfile
import time
import urllib.error
import urllib.request
import uuid
import wave


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--backend-image', default='supernova-review-backend')
    parser.add_argument('--frontend-image', default='supernova-review-frontend')
    parser.add_argument('--output', default='/tmp/supernova-deployment-results.json')
    args = parser.parse_args()
    containers = []
    network = 'supernova-review-' + uuid.uuid4().hex[:12]
    results = {}

    def docker(*args, check=True):
        return subprocess.run(['docker', *args], text=True, capture_output=True, check=check).stdout.strip()

    docker('network', 'create', network)
    try:
        with tempfile.TemporaryDirectory(prefix='supernova-deployment-') as root:
            root = Path(root)
            (root / 'music').mkdir()
            (root / 'data').mkdir()
            with wave.open(str(root / 'music' / 'sample.wav'), 'wb') as sound:
                sound.setparams((1, 2, 8000, 0, 'NONE', 'not compressed'))
                sound.writeframes(b''.join(struct.pack('<h', int(1000 * math.sin(i))) for i in range(8000)))

            def backend(db='supernova.db'):
                name = network + '-backend-' + str(len(containers))
                cid = docker('run', '-d', '--name', name, '--network', network, '--network-alias', 'backend',
                             '--user', f'{os.getuid()}:{os.getgid()}', '--read-only', '--cap-drop', 'ALL',
                             '--security-opt', 'no-new-privileges', '--memory', '256m', '--cpus', '1', '--pids-limit', '128',
                             '--tmpfs', '/tmp:rw,nosuid,nodev,size=64m', '-v', f'{root}:/review:rw',
                             '-e', 'JWT_SECRET=' + 'test-only-' * 8, '-e', 'SUBSONIC_CREDENTIAL_KEY=' + 'test-key-' * 8,
                             '-e', 'REGISTRATION_INVITE_CODE=test-invite', '-e', 'DB_PATH=/review/data/' + db,
                             '-e', 'MEDIA_PATH=/review/music', '-e', 'ART_CACHE_PATH=/review/data/art', args.backend_image)
                containers.append(cid)
                return cid

            first = backend()
            front = docker('run', '-d', '--network', network, '--memory', '128m', '--pids-limit', '128',
                           '-p', '127.0.0.1::80', args.frontend_image)
            containers.append(front)
            port = json.loads(docker('inspect', front))[0]['NetworkSettings']['Ports']['80/tcp'][0]['HostPort']
            base = 'http://127.0.0.1:' + port

            def request(path, body=None, token=None, method=None):
                headers = {'Content-Type': 'application/json'}
                if token:
                    headers['Authorization'] = 'Bearer ' + token
                req = urllib.request.Request(base + path, data=None if body is None else json.dumps(body).encode(), headers=headers, method=method)
                try:
                    with urllib.request.urlopen(req, timeout=15) as response:
                        data = response.read()
                        return response.status, json.loads(data) if 'application/json' in response.headers.get('Content-Type', '') else data
                except urllib.error.HTTPError as error:
                    return error.code, error.read()

            def ready():
                deadline = time.monotonic() + 60
                while time.monotonic() < deadline:
                    try:
                        if request('/api/health')[0] == 200:
                            return
                    except (OSError, ValueError):
                        pass
                    time.sleep(.2)
                raise RuntimeError('deployment did not become healthy: ' + docker('logs', containers[-2], check=False))

            ready()
            assert request('/')[0] == 200
            code, owner = request('/api/auth/register', {'username': 'owner', 'password': 'test-password'})
            assert code == 200, (code, owner)
            token = owner['token']
            code, member = request('/api/auth/register', {'username': 'member', 'password': 'test-password', 'invite_code': 'test-invite'})
            assert code == 200
            assert request('/api/plugins/deduper/preview', token=member['token'])[0] == 403
            assert request('/api/plugins/deduper/preview', token=token)[0] == 200
            assert request('/api/scan', {}, token=member['token'])[0] == 403
            code, playlist = request('/api/playlists', {'name': 'Recovery playlist'}, token=token)
            assert code == 201, (code, playlist)
            assert request('/api/playlists/' + playlist['id'] + '/tracks', token=member['token'])[0] != 200
            results['authorization_and_proxy'] = 'passed'
            docker('exec', first, '/app/supernova-backup', '/review/data/supernova.db', '/review/data/snapshot.db')
            results['live_wal_backup'] = 'passed'
            # Trigger a large new scan, then terminate while it may be processing.
            fixture = (root / 'music' / 'sample.wav').read_bytes()
            for i in range(300):
                (root / 'music' / f'song-{i}.wav').write_bytes(fixture)
            request('/api/scan', {}, token=token)
            started = time.monotonic()
            docker('stop', '-t', '20', first)
            elapsed = time.monotonic() - started
            state = json.loads(docker('inspect', first))[0]['State']
            assert state['ExitCode'] == 0 and not state['OOMKilled'], state
            assert elapsed < 20, elapsed
            results['interrupted_scan_shutdown_seconds'] = round(elapsed, 3)
            restored = backend('snapshot.db')
            ready()
            code, playlists = request('/api/playlists', token=token)
            assert code == 200 and any(p['id'] == playlist['id'] for p in playlists), (code, playlists)
            results['restore_accounts_sessions_playlists'] = 'passed'
            docker('stop', '-t', '20', restored)
            # Restart the original database after interruption and let recovery run.
            docker('start', first)
            ready()
            code, playlists = request('/api/playlists', token=token)
            assert code == 200 and any(p['id'] == playlist['id'] for p in playlists)
            results['restart_after_interrupted_scan'] = 'passed'
            results['backend_limits'] = {'memory_mib': 256, 'cpus': 1, 'pids': 128, 'read_only_root': True, 'non_root': True}
            Path(args.output).write_text(json.dumps(results, indent=2) + '\n')
            print(json.dumps(results, indent=2))
    finally:
        for cid in reversed(containers):
            docker('rm', '-f', cid, check=False)
        docker('network', 'rm', network, check=False)


if __name__ == '__main__':
    main()
