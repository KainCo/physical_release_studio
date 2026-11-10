// Ende-zu-Ende-Test: Der Web-Creator (src/core.js) schreibt ein Paket, der Go-Installer (launcher/)
// installiert es im Linux-Testmodus; danach wird Byte fuer Byte verglichen und deinstalliert.
// Ausfuehren: node --test test/e2e.test.mjs   (benoetigt Go)
// SPDX-License-Identifier: GPL-3.0-or-later
import { test, before, after } from 'node:test';
import assert from 'node:assert/strict';
import { createRequire } from 'node:module';
import { spawnSync } from 'node:child_process';
import crypto from 'node:crypto';
import fs from 'node:fs';
import fsp from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const HERE = path.dirname(fileURLToPath(import.meta.url));
const ROOT = path.join(HERE, '..');
const PRS = createRequire(import.meta.url)('../src/core.js');

let TMP, BIN, GO_OK = false;

before(() => {
  TMP = fs.mkdtempSync(path.join(os.tmpdir(), 'prs-e2e-'));
  BIN = path.join(TMP, 'launcher-cli');
  const r = spawnSync('go', ['build', '-o', BIN, '.'], { cwd: path.join(ROOT, 'launcher'), encoding: 'utf8', env: { ...process.env, GOFLAGS: '-mod=mod', GOPROXY: 'off', CGO_ENABLED: '0' } });
  GO_OK = r.status === 0;
  if (!GO_OK) console.log('Go-Build nicht moeglich, Test wird uebersprungen:', r.stderr || r.error);
});
after(() => { if (TMP) fs.rmSync(TMP, { recursive: true, force: true }); });

// Minimaler Ersatz fuer FileSystemDirectoryHandle (wie ihn showDirectoryPicker liefert), schreibt in einen echten Ordner.
class FakeDir {
  constructor(p) { this.p = p; }
  async getDirectoryHandle(name, o = {}) {
    const q = path.join(this.p, name);
    if (o.create) await fsp.mkdir(q, { recursive: true }); else await fsp.access(q);
    return new FakeDir(q);
  }
  async getFileHandle(name, o = {}) {
    const q = path.join(this.p, name);
    if (o.create) await fsp.writeFile(q, '', { flag: 'a' }); else await fsp.access(q);
    return {
      createWritable: async () => {
        const fd = await fsp.open(q, 'w');
        return { write: async (b) => { await fd.write(b); }, close: async () => { await fd.close(); } };
      },
    };
  }
}

function rng(seed) {
  let s = seed >>> 0 || 1;
  return () => { s ^= s << 13; s >>>= 0; s ^= s >>> 17; s ^= s << 5; s >>>= 0; return s / 4294967296; };
}
const bytes = (n, seed) => { const r = rng(seed), b = Buffer.alloc(n); for (let i = 0; i < n; i++) b[i] = (r() * 256) | 0; return b; };
const sha = (p) => crypto.createHash('sha256').update(fs.readFileSync(p)).digest('hex');

// Spiel-Ordner mit schwierigen Dateien anlegen
const CAP = 1_000_000, RESERVE = 8192; // Platz fuer die Zusatzdateien (Marker, INI, LICENSE.txt ...)
const MAXPART = PRS.plan([{ path: 'x', size: 1 }], { capacity: CAP, reserve: RESERVE }).maxPart;
const GAME_FILES = [
  ['game.exe', 5000], ['data/big.pak', 2_600_000], ['data/exact.bin', MAXPART], ['data/exactplus1.bin', MAXPART + 1],
  ['data/sub dir/mit Leerzeichen.txt', 123], ['data/Ünïcödé ß 日本語.dat', 4097], ['leer.txt', 0], ['a/b/c/d/e/tief.bin', 70000],
  ['_CommonRedist/vcredist/2015/vc_redist.x64.exe', 3000],
];

async function makePackage(name, { mode = 'folder', appId = '', acf = null } = {}) {
  const base = path.join(TMP, name);
  const game = path.join(base, 'src');
  GAME_FILES.forEach(([p, n], i) => { const f = path.join(game, ...p.split('/')); fs.mkdirSync(path.dirname(f), { recursive: true }); fs.writeFileSync(f, bytes(n, i + 11)); });
  const files = [];
  for (const [p] of GAME_FILES) files.push({ path: p, size: fs.statSync(path.join(game, ...p.split('/'))).size, blob: await fs.openAsBlob(path.join(game, ...p.split('/'))) });
  files.sort((a, b) => (a.path < b.path ? -1 : a.path > b.path ? 1 : 0));

  const plan = PRS.plan(files.map((f) => ({ path: f.path, size: f.size })), { capacity: CAP, reserve: RESERVE });
  const out = path.join(base, 'out');
  const discDirs = [];
  for (let i = 0; i < plan.discs.length; i++) { fs.mkdirSync(path.join(out, 'Disc' + (i + 1)), { recursive: true }); discDirs.push(new FakeDir(path.join(out, 'Disc' + (i + 1)))); }
  await PRS.writeDataToDirectory(discDirs, files, plan);

  const pkgId = PRS.randomId();
  const title = 'Test Spiel: Ünïcode';
  const redist = PRS.detectRedist(files.map((f) => f.path));
  const ini = PRS.buildIni({ title, exe: 'game.exe', pkgId, dirName: PRS.fsName(title), banner: '', showTitle: true, discCount: plan.discs.length,
    files: files.map((f) => ({ path: f.path, size: f.size })), plan, redist, mode, steamAppId: appId, steamManifest: acf ? 'extras\\appmanifest.acf' : '' });
  for (let i = 0; i < plan.discs.length; i++) {
    const cache = new Map();
    const extra = PRS.extraFiles({ ini, pkgId, discCount: plan.discs.length, title, exeName: 'Launcher.exe', icoBytes: Buffer.from('ICO'), licenseBytes: Buffer.from('LIC'), launcherBytes: Buffer.from('MZ-test'),
      readmeName: 'README.txt', readme: PRS.readmeText(title, plan.discs.length), bannerName: '', bannerBytes: null, acfBytes: acf }, i + 1);
    for (const x of extra) await PRS.writeBytesToDirectory(discDirs[i], x.path, x.bytes, cache);
  }
  return { base, game, out, plan, files, pkgId, discs: plan.discs.map((_, i) => path.join(out, 'Disc' + (i + 1))), title };
}

function runCli(args, env = {}) {
  const r = spawnSync(BIN, args, { encoding: 'utf8', env: { PATH: process.env.PATH, ...env } });
  return { code: r.status, out: (r.stdout || '') + (r.stderr || '') };
}
function envFor(base) {
  return { PRS_STATE_DIR: path.join(base, 'state'), PRS_SPECIAL: path.join(base, 'special'), PRS_STEAM: path.join(base, 'steam') };
}
function listAll(dir) {
  const out = [];
  (function walk(d) { for (const e of fs.readdirSync(d, { withFileTypes: true })) { const p = path.join(d, e.name); if (e.isDirectory()) walk(p); else out.push(path.relative(dir, p)); } })(dir);
  return out.sort();
}

test('Paket wird auf mehrere Discs verteilt, geteilte Dateien entstehen', async (t) => {
  if (!GO_OK) return t.skip('Go fehlt');
  const p = await makePackage('plan');
  assert.ok(p.plan.discs.length >= 4, 'erwartet mehrere Discs, bekam ' + p.plan.discs.length);
  assert.ok(p.plan.split.length >= 2);
  // jede Disc hat Marker, ini, autorun
  for (let i = 0; i < p.discs.length; i++) {
    for (const f of ['launcher.ini', 'autorun.inf', 'autorun.ico', 'Launcher.exe', 'README.txt', 'LICENSE.txt', PRS.markerName(i + 1)]) assert.ok(fs.existsSync(path.join(p.discs[i], f)), f + ' fehlt auf Disc ' + (i + 1));
    // Kein Disc-Inhalt ueberschreitet die Kapazitaet
    const used = listAll(p.discs[i]).reduce((a, f) => a + PRS.alloc(fs.statSync(path.join(p.discs[i], f)).size), 0);
    assert.ok(used <= CAP, `Disc ${i + 1}: ${used} > ${CAP}`);
  }
});

test('Installation von mehreren Discs ist Byte fuer Byte identisch; Deinstallation raeumt auf', async (t) => {
  if (!GO_OK) return t.skip('Go fehlt');
  const p = await makePackage('install');
  const dest = path.join(p.base, 'Installiert', 'Spiel');
  const env = envFor(p.base);
  // Discs in umgekehrter Reihenfolge uebergeben: die Reihenfolge darf keine Rolle spielen
  const r = runCli(['install', dest, ...p.discs.slice().reverse()], env);
  assert.equal(r.code, 0, r.out);
  assert.match(r.out, /OK/);
  for (const f of p.files) {
    const target = path.join(dest, ...f.path.split('/'));
    assert.ok(fs.existsSync(target), f.path + ' fehlt');
    assert.equal(fs.statSync(target).size, f.size, f.path + ' Groesse');
    assert.equal(sha(target), sha(path.join(p.game, ...f.path.split('/'))), f.path + ' Inhalt');
  }
  // Installiert-Erkennung (Ersatz fuer die Registry) und Deinstallations-Protokoll
  assert.equal(fs.readFileSync(path.join(env.PRS_STATE_DIR, p.pkgId), 'utf8'), dest);
  assert.ok(fs.existsSync(path.join(dest, 'uninstall.lst')));
  assert.ok(fs.existsSync(path.join(dest, 'uninstall.exe')));
  // Es liegen keine Teildateien (.partN) im Zielordner
  assert.ok(!listAll(dest).some((f) => /\.part\d+$/.test(f)));

  const u = runCli(['uninstall', dest], env);
  assert.equal(u.code, 0, u.out);
  assert.ok(!fs.existsSync(dest), 'Zielordner sollte entfernt sein, uebrig: ' + (fs.existsSync(dest) ? listAll(dest).join(', ') : ''));
  assert.ok(!fs.existsSync(path.join(env.PRS_STATE_DIR, p.pkgId)), 'Registrierung bleibt zurueck');
});

test('Deinstallation loescht keine fremden Dateien im Spielordner', async (t) => {
  if (!GO_OK) return t.skip('Go fehlt');
  const p = await makePackage('extra');
  const dest = path.join(p.base, 'Installiert', 'Spiel');
  const env = envFor(p.base);
  assert.equal(runCli(['install', dest, ...p.discs], env).code, 0);
  fs.writeFileSync(path.join(dest, 'meine_spielstaende.sav'), 'wichtig');
  const u = runCli(['uninstall', dest], env); // ohne PRS_DELETE_EXTRA: nicht alles loeschen
  assert.equal(u.code, 0, u.out);
  assert.equal(fs.readFileSync(path.join(dest, 'meine_spielstaende.sav'), 'utf8'), 'wichtig');
  assert.deepEqual(listAll(dest), ['meine_spielstaende.sav'], 'nur eigene Dateien duerfen uebrig bleiben');
});

test('Beschaedigte Disc wird erkannt, nichts wird als fertig gemeldet', async (t) => {
  if (!GO_OK) return t.skip('Go fehlt');
  const p = await makePackage('corrupt');
  const dest = path.join(p.base, 'Installiert', 'Spiel');
  // ein Byte in einem Teil der grossen Datei kippen
  let flipped = null;
  for (const d of p.discs) {
    const f = path.join(d, 'Game', 'data', 'big.pak.part2');
    if (fs.existsSync(f)) { const b = fs.readFileSync(f); b[1000] ^= 0xff; fs.writeFileSync(f, b); flipped = f; }
  }
  assert.ok(flipped, 'Teil big.pak.part2 nicht gefunden');
  const r = runCli(['install', dest, ...p.discs], envFor(p.base));
  assert.equal(r.code, 1);
  assert.match(r.out, /big\.pak/);
  assert.ok(!/\bOK\b/.test(r.out));
});

test('Fehlende Disc wird gemeldet', async (t) => {
  if (!GO_OK) return t.skip('Go fehlt');
  const p = await makePackage('missing');
  const dest = path.join(p.base, 'Installiert', 'Spiel');
  const r = runCli(['install', dest, ...p.discs.slice(0, 1)], envFor(p.base));
  assert.equal(r.code, 1);
  assert.match(r.out, /nicht gefunden/);
});

test('Steam-Modus: Manifest landet in steamapps, wird beim Deinstallieren wieder entfernt', async (t) => {
  if (!GO_OK) return t.skip('Go fehlt');
  const acf = Buffer.from('"AppState"\n{\n\t"appid"\t\t"620"\n\t"name"\t\t"Test"\n\t"installdir"\t\t"Spiel"\n}\n');
  const p = await makePackage('steam', { mode: 'steam', appId: '620', acf });
  const lib = path.join(p.base, 'SteamLibrary');
  const dest = path.join(lib, 'steamapps', 'common', 'Spiel');
  const env = envFor(p.base);
  const r = runCli(['install', dest, ...p.discs], env);
  assert.equal(r.code, 0, r.out);
  const manifest = path.join(lib, 'steamapps', 'appmanifest_620.acf');
  assert.ok(fs.existsSync(manifest), 'appmanifest fehlt');
  assert.equal(fs.readFileSync(manifest, 'utf8'), acf.toString());
  assert.equal(sha(path.join(dest, 'game.exe')), sha(path.join(p.game, 'game.exe')));

  const u = runCli(['uninstall', dest], env);
  assert.equal(u.code, 0, u.out);
  assert.ok(!fs.existsSync(manifest), 'appmanifest sollte entfernt sein');
});

test('Steam-Modus: vorhandenes appmanifest wird nie ueberschrieben', async (t) => {
  if (!GO_OK) return t.skip('Go fehlt');
  const acf = Buffer.from('"AppState"\n{\n\t"appid"\t\t"620"\n}\n');
  const p = await makePackage('steam2', { mode: 'steam', appId: '620', acf });
  const lib = path.join(p.base, 'SteamLibrary');
  const dest = path.join(lib, 'steamapps', 'common', 'Spiel');
  fs.mkdirSync(path.join(lib, 'steamapps'), { recursive: true });
  fs.writeFileSync(path.join(lib, 'steamapps', 'appmanifest_620.acf'), 'ORIGINAL');
  assert.equal(runCli(['install', dest, ...p.discs], envFor(p.base)).code, 0);
  assert.equal(fs.readFileSync(path.join(lib, 'steamapps', 'appmanifest_620.acf'), 'utf8'), 'ORIGINAL');
  assert.equal(runCli(['uninstall', dest], envFor(p.base)).code, 0);
  assert.equal(fs.readFileSync(path.join(lib, 'steamapps', 'appmanifest_620.acf'), 'utf8'), 'ORIGINAL', 'fremdes Manifest darf nicht geloescht werden');
});

test('Der Installer legt niemals eine steam_appid.txt an (Ordner- und Steam-Modus, mit AppID)', async (t) => {
  if (!GO_OK) return t.skip('Go fehlt');
  for (const mode of ['folder', 'steam']) {
    const p = await makePackage('appid-' + mode, { mode, appId: '207170' });
    const dest = mode === 'steam' ? path.join(p.base, 'Lib', 'steamapps', 'common', 'Spiel') : path.join(p.base, 'Installiert', 'Spiel');
    const r = runCli(['install', dest, ...p.discs], envFor(p.base));
    assert.equal(r.code, 0, r.out);
    assert.ok(!listAll(dest).some((f) => /steam_appid\.txt$/i.test(f)), mode + ': steam_appid.txt darf nicht angelegt werden');
  }
});

test('Steam-Modus ohne Manifest-Datei: Installer erzeugt ein Manifest und entfernt es wieder', async (t) => {
  if (!GO_OK) return t.skip('Go fehlt');
  const p = await makePackage('steam3', { mode: 'steam', appId: '207170' });
  const lib = path.join(p.base, 'SteamLibrary');
  const dest = path.join(lib, 'steamapps', 'common', 'Legend of Grimrock');
  const env = envFor(p.base);
  const r = runCli(['install', dest, ...p.discs], env);
  assert.equal(r.code, 0, r.out);
  const manifest = path.join(lib, 'steamapps', 'appmanifest_207170.acf');
  assert.ok(fs.existsSync(manifest), 'Manifest fehlt');
  const info = PRS.parseAcf(fs.readFileSync(manifest, 'utf8'));
  assert.equal(info.appid, '207170');
  assert.equal(info.installdir, 'Legend of Grimrock');
  assert.equal(runCli(['uninstall', dest], env).code, 0);
  assert.ok(!fs.existsSync(manifest), 'erzeugtes Manifest sollte entfernt sein');
});

test('Manipulierte launcher.ini kann nicht aus dem Zielordner ausbrechen', async (t) => {
  if (!GO_OK) return t.skip('Go fehlt');
  const p = await makePackage('evil');
  const dest = path.join(p.base, 'Installiert', 'Spiel');
  for (const d of p.discs) {
    const f = path.join(d, 'launcher.ini');
    fs.writeFileSync(f, fs.readFileSync(f, 'utf8').replace('F=5000|game.exe', 'F=5000|..\\..\\ausbruch.exe'));
  }
  const r = runCli(['install', dest, ...p.discs], envFor(p.base));
  assert.equal(r.code, 1, r.out);
  assert.ok(!fs.existsSync(path.join(p.base, 'ausbruch.exe')));
  assert.ok(!fs.existsSync(path.join(p.base, 'Installiert', 'ausbruch.exe')));
});
