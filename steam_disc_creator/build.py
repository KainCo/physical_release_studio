#!/usr/bin/env python3
# Steam Disc Creator - Build-Skript (nur Python-Standardbibliothek + Go-Werkzeugkette)
# SPDX-License-Identifier: GPL-3.0-or-later
"""
Baut dist/index.html (eine einzige, eigenstaendige Seite) und dist/Launcher.exe.

    python3 build.py                  Tests ausfuehren, Launcher kompilieren, Seite erzeugen
    python3 build.py --skip-tests     ohne Tests (schneller)
    python3 build.py --launcher PFAD  fertige (z. B. signierte) Launcher.exe einbetten, statt neu zu kompilieren

Signieren: erst normal bauen, dist/Launcher.exe mit signtool o. ae. signieren, dann
    python3 build.py --skip-tests --launcher dist/Launcher.exe
"""
import argparse
import base64
import hashlib
import os
import shutil
import subprocess
import sys

ROOT = os.path.dirname(os.path.abspath(__file__))
SRC = os.path.join(ROOT, 'src')
LAUNCHER = os.path.join(ROOT, 'launcher')
TOOLS = os.path.join(ROOT, 'tools')
DIST = os.path.join(ROOT, 'dist')


def run(cmd, cwd, env=None):
    print('$', ' '.join(cmd), '  [' + os.path.relpath(cwd, ROOT) + ']')
    e = dict(os.environ)
    if env:
        e.update(env)
    subprocess.run(cmd, cwd=cwd, env=e, check=True)


def read(path, mode='r'):
    with open(path, mode, **({} if 'b' in mode else {'encoding': 'utf-8'})) as f:
        return f.read()


def once(text, marker, value):
    if text.count(marker) != 1:
        sys.exit('Fehler: Platzhalter %r kommt %d-mal vor (erwartet: 1)' % (marker, text.count(marker)))
    return text.replace(marker, value)


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument('--skip-tests', action='store_true', help='Tests nicht ausfuehren')
    ap.add_argument('--launcher', help='vorhandene Launcher.exe einbetten (z. B. signiert)')
    ap.add_argument('--icons', action='store_true', help='Symbol (autorun.ico, .syso) neu erzeugen')
    args = ap.parse_args()

    os.makedirs(DIST, exist_ok=True)
    go_env = {'GOFLAGS': '-mod=mod', 'GOPROXY': 'off', 'CGO_ENABLED': '0'}

    if args.icons:
        run(['go', 'run', './mkrsrc', '-ico', '../src/autorun.ico', '-syso', '../launcher/rsrc_windows_amd64.syso'], TOOLS, go_env)

    if not args.skip_tests:
        run(['go', 'vet', './...'], LAUNCHER, go_env)
        run(['go', 'test', './...'], LAUNCHER, go_env)
        run(['node', '--test', 'test/core.test.mjs', 'test/e2e.test.mjs'], ROOT)

    exe_path = os.path.join(DIST, 'Launcher.exe')
    if args.launcher:
        src = os.path.abspath(args.launcher)
        if src != os.path.abspath(exe_path):
            shutil.copyfile(src, exe_path)
        print('Verwende vorhandene', args.launcher)
    else:
        run(['go', 'build', '-trimpath', '-ldflags', '-s -w -H=windowsgui', '-o', exe_path, '.'], LAUNCHER,
            dict(go_env, GOOS='windows', GOARCH='amd64'))
    run(['go', 'run', './checkpe', exe_path], TOOLS, go_env)

    core = read(os.path.join(SRC, 'core.js'))
    if '</script' in core.lower():
        sys.exit('Fehler: core.js enthaelt "</script" und kann nicht inline eingebettet werden')
    exe = read(exe_path, 'rb')
    ico = read(os.path.join(SRC, 'autorun.ico'), 'rb')

    legal = read(os.path.join(SRC, 'legal-note.js'))
    if '</script' in legal.lower():
        sys.exit('Fehler: legal-note.js enthaelt "</script" und kann nicht inline eingebettet werden')
    fonts = []
    for family, fn, weight in (('Inter', 'Inter-subset.woff', '400 800'), ('JetBrains Mono', 'JetBrainsMono-subset.woff', '400 700')):
        data = base64.b64encode(read(os.path.join(SRC, 'fonts', fn), 'rb')).decode('ascii')
        fonts.append("@font-face{font-family:'%s';src:url(data:font/woff;base64,%s) format('woff');font-weight:%s;font-style:normal;font-display:swap}" % (family, data, weight))

    html = read(os.path.join(SRC, 'index.template.html'))
    html = once(html, '/*__FONTS_CSS__*/', '\n'.join(fonts))
    html = once(html, '/*__LEGAL_JS__*/', legal)
    html = once(html, '/*__CORE_JS__*/', core)
    html = once(html, '__LAUNCHER_B64__', base64.b64encode(exe).decode('ascii'))
    html = once(html, '__ICO_B64__', base64.b64encode(ico).decode('ascii'))
    with open(os.path.join(DIST, 'index.html'), 'w', encoding='utf-8', newline='\n') as f:
        f.write(html)

    shutil.copyfile(os.path.join(SRC, 'datenschutz.html'), os.path.join(DIST, 'datenschutz.html'))

    sums = []
    for name in ('index.html', 'datenschutz.html', 'Launcher.exe'):
        sums.append('%s  %s' % (hashlib.sha256(read(os.path.join(DIST, name), 'rb')).hexdigest(), name))
    with open(os.path.join(DIST, 'SHA256SUMS.txt'), 'w', newline='\n') as f:
        f.write('\n'.join(sums) + '\n')
    print('\n'.join(sums))
    print('Fertig: dist/index.html (%.1f MB), dist/Launcher.exe (%.1f MB)' % (
        os.path.getsize(os.path.join(DIST, 'index.html')) / 1e6, len(exe) / 1e6))


if __name__ == '__main__':
    main()
