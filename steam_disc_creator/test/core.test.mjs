// Unit-Tests der Kernlogik (src/core.js). Ausfuehren: node --test test/core.test.mjs
// SPDX-License-Identifier: GPL-3.0-or-later
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { createRequire } from 'node:module';
import { spawnSync } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import zlib from 'node:zlib';

const PRS = createRequire(import.meta.url)('../src/core.js');

// einfacher, reproduzierbarer Zufallsgenerator
function rng(seed) {
  let s = seed >>> 0 || 1;
  return () => { s ^= s << 13; s >>>= 0; s ^= s >>> 17; s ^= s << 5; s >>>= 0; return s / 4294967296; };
}
const bytes = (n, seed = 1) => { const r = rng(seed), b = new Uint8Array(n); for (let i = 0; i < n; i++) b[i] = (r() * 256) | 0; return b; };

test('crc32 stimmt mit zlib ueberein (auch stueckweise)', () => {
  for (const n of [0, 1, 3, 4, 5, 1000, 65537]) {
    const b = bytes(n, n + 7);
    assert.equal(PRS.crc32(b), zlib.crc32(b), 'Laenge ' + n);
  }
  const b = bytes(100000, 3);
  let c = 0;
  for (let p = 0; p < b.length; p += 7777) c = PRS.crc32(b.subarray(p, p + 7777), c);
  assert.equal(c, zlib.crc32(b));
});

// ---------- Aufteilen ----------
function checkPlan(files, plan, opt) {
  // 1. Jede Datei wird lueckenlos und ohne Ueberlappung abgedeckt
  const byFile = new Map();
  plan.discs.forEach((d, i) => d.segs.forEach((s) => {
    assert.equal(s.disc, i + 1);
    (byFile.get(s.fi) || byFile.set(s.fi, []).get(s.fi)).push(s);
  }));
  files.forEach((f, fi) => {
    const segs = (byFile.get(fi) || []).sort((a, b) => a.offset - b.offset);
    let pos = 0;
    for (const s of segs) { assert.equal(s.offset, pos, f.path + ': Luecke'); pos += s.size; assert.ok(s.size <= plan.maxPart, 'Teil zu gross'); }
    assert.equal(pos, f.size, f.path + ': Summe');
    if (f.size > 0) assert.ok(segs.length >= 1);
  });
  // 2. Keine Disc ueberschreitet die nutzbare Groesse
  if (isFinite(plan.usable)) for (const d of plan.discs) assert.ok(d.used <= plan.usable, 'Disc ueberfuellt');
}

test('plan: zufaellige Dateimengen, Invarianten bleiben erhalten', () => {
  for (let seed = 1; seed <= 25; seed++) {
    const r = rng(seed * 101);
    const files = [];
    const n = 1 + Math.floor(r() * 60);
    for (let i = 0; i < n; i++) {
      const kind = r();
      const size = kind < 0.1 ? 0 : kind < 0.7 ? Math.floor(r() * 40000) : Math.floor(r() * 3_000_000);
      files.push({ path: 'dir' + (i % 5) + '/f' + i + '.dat', size });
    }
    const opt = { capacity: 1_000_000 + Math.floor(r() * 2_000_000), reserve: Math.floor(r() * 50000) };
    const plan = PRS.plan(files, opt);
    checkPlan(files, plan, opt);
  }
});

test('plan: Randfaelle um die Teilungsgrenze', () => {
  const opt = { capacity: 1_000_000, reserve: 4096 };
  const first = PRS.plan([{ path: 'a', size: 1 }], opt);
  const max = first.maxPart;
  assert.equal(max % 2048, 0);
  for (const [size, parts] of [[max - 1, 1], [max, 1], [max + 1, 2], [2 * max, 2], [2 * max + 1, 3]]) {
    const files = [{ path: 'big.bin', size }];
    const plan = PRS.plan(files, opt);
    checkPlan(files, plan, opt);
    assert.equal(plan.discs.reduce((a, d) => a + d.segs.length, 0), parts, 'Groesse ' + size);
    assert.equal(plan.split.length, parts > 1 ? 1 : 0);
  }
});

test('plan: zu kleines Medium wird gemeldet', () => {
  assert.throws(() => PRS.plan([{ path: 'a', size: 1 }], { capacity: 1000, reserve: 0 }), /capacity/);
  assert.throws(() => PRS.plan([{ path: 'a', size: 1 }], { capacity: 100000, reserve: 100000 }), /capacity/);
});

test('plan: USB ohne Limit nutzt eine Disc, FAT32 teilt Dateien ueber 4 GB', () => {
  const gb = 1e9;
  const p1 = PRS.plan([{ path: 'a', size: 9 * gb }, { path: 'b', size: 5 }], { capacity: Infinity });
  assert.equal(p1.discs.length, 1);
  assert.equal(p1.split.length, 0);
  const m = PRS.MEDIA.find((x) => x.id === 'usb32');
  const p2 = PRS.plan([{ path: 'a', size: 9 * gb }], { capacity: m.bytes, maxPart: m.maxPart });
  assert.equal(p2.discs.length, 1);
  assert.equal(p2.discs[0].segs.length, 3);
  assert.ok(p2.maxPart < 4294967296);
});

test('plan: leere Eingabe ergibt trotzdem eine Disc', () => {
  const plan = PRS.plan([], { capacity: 700000000, reserve: 0 });
  assert.equal(plan.discs.length, 1);
});

test('plan: Kapazitaeten der Medien sind plausibel', () => {
  const m = Object.fromEntries(PRS.MEDIA.map((x) => [x.id, x.bytes]));
  assert.ok(m.dvd5 > 4.69e9 && m.dvd5 < 4.71e9);
  assert.ok(m.bd25 > 25.0e9 && m.bd25 < 25.1e9);
  assert.ok(m.cd700 > m.cd650);
  assert.ok(m.cd800 > m.cd700);
});

// ---------- launcher.ini ----------
test('buildIni: Zeilen, Teile und Schutz vor Einschleusen von Zeilen', () => {
  const files = [{ path: 'a/b.dat', size: 5000 }, { path: 'c.txt', size: 0 }];
  const plan = PRS.plan(files, { capacity: 1_000_000, reserve: 4096 });
  plan.discs.forEach((d) => d.segs.forEach((s) => { s.crc = 0xdeadbeef; }));
  const evil = 'Spiel\r\n[Files]\r\nF=1|..\\..\\boese.exe';
  const ini = PRS.buildIni({ title: evil, exe: 'a/b.dat', pkgId: 'abcdef0123456789', dirName: 'X\r\nY', banner: 'banner.jpg', showTitle: true,
    discCount: plan.discs.length, files, plan, redist: [{ path: '_CommonRedist/vc_redist.x64.exe', args: '/q\r\n/x', label: 'VC' }], mode: 'steam', steamAppId: '620', steamManifest: 'extras\\appmanifest.acf' });
  const lines = ini.split('\r\n');
  assert.equal(lines.filter((l) => l === '[Files]').length, 1, 'eingeschleuste Abschnitte');
  assert.ok(!lines.some((l) => l.includes('boese') && l.startsWith('F=')));
  assert.ok(lines.includes('Mode=steam') && lines.includes('SteamAppID=620'));
  assert.ok(lines.includes('Exe=a\\b.dat'));
  assert.ok(lines.some((l) => l === 'S=1|0|5000|deadbeef|Game\\a\\b.dat'));
  assert.ok(lines.some((l) => l.startsWith('R=_CommonRedist\\vc_redist.x64.exe|')));
  assert.ok(!ini.includes('\r\n/x'), 'Zeilenumbruch in Redist-Parametern');
});

test('fsName / Marker / Autorun', () => {
  assert.equal(PRS.fsName('Half-Life: Source?'), 'Half-Life - Source');
  assert.equal(PRS.fsName('  a/b\\c|d.  '), 'a-b-c-d');
  assert.equal(PRS.fsName('???'), 'Game');
  assert.equal(PRS.markerName(3), 'DISC_3.txt');
  assert.match(PRS.markerText('abc', 2, 4, 'T'), /^PackageID=abc\r\nDisc=2\r\nCount=4\r\nTitle=T\r\n$/);
  assert.match(PRS.autorunInf('Mein Spiel', 'Launcher.exe'), /open=Launcher\.exe[\s\S]*icon=autorun\.ico/);
});

// ---------- Hinweise zu Kopierschutz ----------
test('steam_api.dll allein ist kein Hinweis auf Kopierschutz', () => {
  const r = PRS.scanNames(['Game/game.exe', 'Game/steam_api64.dll', 'Game/steam_appid.txt']);
  assert.equal(r.drm.length, 0);
  assert.equal(r.steamApi.length, 1);
  assert.equal(r.steamAppIdFile, 'Game/steam_appid.txt');
  const v = PRS.drmVerdict(r, []);
  assert.equal(v.level, 'none');
});

test('Kopierschutz-Dateien werden gemeldet, Gruen gibt es nie', () => {
  const r = PRS.scanNames(['a/uplay_r2_loader64.dll', 'a/secdrv.sys', 'a/SecuROM7.dll', 'a/xlive.dll', 'a/Denuvo64.dll', 'a/x.exe']);
  const kinds = r.drm.map((d) => d.kind).sort();
  assert.deepEqual(kinds, ['denuvo', 'gfwl', 'safedisc', 'securom', 'uplay']);
  assert.equal(PRS.drmVerdict(r, []).level, 'warn');
  assert.ok(['warn', 'none'].includes(PRS.drmVerdict(PRS.scanNames([]), []).level));
});

function fakePE(sections) {
  const b = new Uint8Array(0x400);
  const dv = new DataView(b.buffer);
  b[0] = 0x4d; b[1] = 0x5a; dv.setUint32(0x3c, 0x80, true);
  dv.setUint32(0x80, 0x00004550, true);
  dv.setUint16(0x80 + 6, sections.length, true);
  dv.setUint16(0x80 + 20, 0xf0, true);
  let off = 0x80 + 24 + 0xf0;
  for (const s of sections) { for (let i = 0; i < s.length; i++) b[off + i] = s.charCodeAt(i); off += 40; }
  return b;
}
test('PE-Abschnitte: SteamStub (.bind) und SecuROM werden erkannt', () => {
  assert.deepEqual(PRS.peSections(fakePE(['.text', '.rdata', '.bind'])), ['.text', '.rdata', '.bind']);
  assert.deepEqual(PRS.sectionHints(['.text', '.bind']), ['steamstub']);
  assert.deepEqual(PRS.sectionHints(['.text', '.cms_t', '.cms_d']), ['securom']);
  assert.deepEqual(PRS.sectionHints(['.text', '.data']), []);
  assert.equal(PRS.peSections(new Uint8Array(10)), null);
  assert.equal(PRS.peSections(new Uint8Array(0x400)), null);
});

// ---------- Programmdatei und Komponenten ----------
test('exeCandidates: waehlt passende Exe, ignoriert Deinstaller und Komponenten', () => {
  const files = ['unins000.exe', 'Half-Life 2/hl2.exe', 'bin/Cyber Game.exe', 'CyberGame.exe', '_CommonRedist/vcredist/vc_redist.x64.exe', 'UnityCrashHandler64.exe', 'docs/readme.txt']
    .map((p) => ({ path: p, size: 1000 }));
  const c = PRS.exeCandidates(files, 'Cyber Game').map((x) => x.path);
  assert.equal(c[0], 'CyberGame.exe');
  assert.ok(!c.includes('unins000.exe') && !c.includes('UnityCrashHandler64.exe'));
  assert.ok(!c.some((p) => p.includes('vc_redist')));
});

test('detectRedist: erkennt bekannte Installationsprogramme mit Parametern', () => {
  const r = PRS.detectRedist([
    '_CommonRedist/vcredist/2015/vc_redist.x64.exe', '_CommonRedist/vcredist/2010/vcredist_x86.exe', '_CommonRedist/DirectX/Jun2010/DXSETUP.exe',
    '_CommonRedist/DotNet/4.0/dotNetFx40_Full_x86_x64.exe', 'Game/other/vc_redist.x64.exe', '_CommonRedist/Readme.txt', '_CommonRedist/OpenAL/oalinst.exe'])
    .map((x) => x.path + ' ' + x.args);
  assert.equal(r.length, 6, JSON.stringify(r));
  assert.ok(r.some((s) => s.includes('vc_redist.x64.exe /install /quiet /norestart')));
  assert.ok(r.some((s) => s.includes('vcredist_x86.exe /q /norestart')));
  assert.ok(r.some((s) => s.includes('DXSETUP.exe /silent')));
  assert.ok(r.some((s) => s.startsWith('Game/other/vc_redist.x64.exe')), 'Erkennung nach Dateiname, nicht nach Ordner');
});

test('detectRedist: Ordner wie DirectX und vcredist (nicht _CommonRedist) werden erkannt', () => {
  const paths = ['DirectX/DXSETUP.exe', 'DirectX/Jun2010_d3dx9_43_x86.cab', 'vcredist/vcredist_x86.exe', 'grimrock.exe', 'doclauncher.exe', 'steam_api.dll'];
  const r = PRS.detectRedist(paths);
  assert.deepEqual(r.map((x) => x.path).sort(), ['DirectX/DXSETUP.exe', 'vcredist/vcredist_x86.exe']);
  assert.ok(PRS.isRedistFile('DirectX/DXSETUP.exe') && PRS.isRedistFile('vcredist/vcredist_x86.exe'));
  assert.ok(!PRS.isRedistFile('grimrock.exe') && !PRS.isRedistFile('doclauncher.exe'));
  const c = PRS.exeCandidates(paths.filter((p) => p.endsWith('.exe')).map((p) => ({ path: p, size: 5000 })), 'Legend of Grimrock').map((x) => x.path);
  assert.deepEqual(c.sort(), ['doclauncher.exe', 'grimrock.exe']);
});

test('Source-Engine wird lokal erkannt (nur mit gameinfo.txt plus Engine-Datei)', () => {
  assert.ok(PRS.detectSourceEngine(['bms/gameinfo.txt', 'bms/bin/server.dll', 'bin/engine.dll', 'hl2.exe', 'steam.inf']));
  assert.ok(PRS.detectSourceEngine(['hl2.exe', 'portal/gameinfo.txt']));
  assert.ok(!PRS.detectSourceEngine(['gameinfo.txt']), 'gameinfo.txt allein reicht nicht');
  assert.ok(!PRS.detectSourceEngine(['game.exe', 'data/engine.dll', 'steam_api.dll']));
});

test('runtimeHints: findet Namen aelterer Laufzeitbibliotheken in einer EXE', () => {
  const mk = (...names) => { const u = new Uint8Array(4000); let o = 100; for (const n of names) { for (let i = 0; i < n.length; i++) u[o++] = n.charCodeAt(i); o += 3; } return u; };
  assert.deepEqual(PRS.runtimeHints(mk('MSVCR100.dll', 'X3DAudio1_7.dll', 'd3dx9_43.dll')).sort(), ['dx9', 'vc2010']);
  assert.deepEqual(PRS.runtimeHints(mk('VCRUNTIME140.dll')), ['vc2015']);
  assert.deepEqual(PRS.runtimeHints(mk('kernel32.dll', 'user32.dll')), []);
  assert.deepEqual(PRS.runtimeHints(new Uint8Array(0)), []);
});

test('parseAcf liest appid, name, installdir', () => {
  const t = '"AppState"\n{\n\t"appid"\t\t"620"\n\t"name"\t\t"Portal 2"\n\t"installdir"\t\t"Portal 2"\n}';
  assert.deepEqual(PRS.parseAcf(t), { appid: '620', name: 'Portal 2', installdir: 'Portal 2' });
  assert.deepEqual(PRS.parseAcf('kaputt'), { appid: '', name: '', installdir: '' });
});

// ---------- ZIP ----------
test('ZIP-Ausgabe wird von Python gelesen und ist inhaltlich identisch', async (t) => {
  const py = spawnSync('python3', ['--version']);
  if (py.status !== 0) return t.skip('python3 fehlt');
  const items = [
    { name: 'Game/a.bin', data: bytes(100000, 5) },
    { name: 'Game/sub dir/ünï.txt', data: new TextEncoder().encode('Hallo') },
    { name: 'leer.txt', data: new Uint8Array(0) },
  ];
  const entries = items.map((i) => ({ name: i.name, size: i.data.length, crc: zlib.crc32(i.data), blob: new Blob([i.data]) }));
  const zip = PRS.zipBlob(entries);
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'prs-zip-'));
  const file = path.join(dir, 't.zip');
  fs.writeFileSync(file, Buffer.from(await zip.arrayBuffer()));
  const script = 'import zipfile,sys,zlib\nz=zipfile.ZipFile(sys.argv[1])\nassert z.testzip() is None\nfor i in z.infolist(): print(i.filename, i.file_size, zlib.crc32(z.read(i)))\n';
  const r = spawnSync('python3', ['-I', '-c', script, file], { encoding: 'utf8' });
  assert.equal(r.status, 0, r.stderr);
  const got = r.stdout.trim().split('\n').sort();
  const want = items.map((i) => i.name + ' ' + i.data.length + ' ' + zlib.crc32(i.data)).sort();
  assert.deepEqual(got, want);
  const u = spawnSync('unzip', ['-tq', file], { encoding: 'utf8' });
  if (!u.error) assert.equal(u.status, 0, u.stdout + u.stderr);
  fs.rmSync(dir, { recursive: true, force: true });
});

test('randomId liefert 16 Hex-Zeichen und ist verschieden', () => {
  const a = PRS.randomId(), b = PRS.randomId();
  assert.match(a, /^[0-9a-f]{16}$/);
  assert.notEqual(a, b);
});

test('buildIni: AppID steht auch im Ordner-Modus (ohne Mode=steam); ungueltige AppID wird nicht geschrieben', () => {
  const files = [{ path: 'g.exe', size: 10 }];
  const plan = PRS.plan(files, { capacity: 1_000_000, reserve: 4096 });
  plan.discs.forEach((d) => d.segs.forEach((s) => { s.crc = 1; }));
  const base = { title: 'T', exe: 'g.exe', pkgId: 'abcdef0123456789', dirName: 'T', banner: '', showTitle: true, discCount: 1, files, plan, redist: [] };
  const a = PRS.buildIni({ ...base, mode: 'folder', steamAppId: '207170', steamManifest: 'extras\\appmanifest.acf' }).split('\r\n');
  assert.ok(a.includes('SteamAppID=207170'));
  assert.ok(!a.includes('Mode=steam') && !a.some((l) => l.startsWith('SteamManifest')), 'Ordner-Modus: kein Steam-Modus, kein Manifest');
  const b = PRS.buildIni({ ...base, mode: 'folder', steamAppId: '' }).split('\r\n');
  assert.ok(!b.some((l) => l.startsWith('SteamAppID')));
  const c = PRS.buildIni({ ...base, mode: 'folder', steamAppId: '12\r\n[Files]' }).split('\r\n');
  assert.ok(!c.some((l) => l.startsWith('SteamAppID')) && c.filter((l) => l === '[Files]').length === 1);
});

test('plan: Extras (firstExtra) belegen nur Disc 1', () => {
  const files = [{ path: 'a.bin', size: 600e6 }, { path: 'b.bin', size: 600e6 }];
  const base = PRS.plan(files, { capacity: 737280000, reserve: 65536 });
  const ex = PRS.plan(files, { capacity: 737280000, reserve: 65536, firstExtra: 100e6 });
  assert.equal(base.discs.length, 2);
  assert.equal(ex.discs.length, 2);
  assert.ok(ex.discs[0].used >= 100e6);
  assert.throws(() => PRS.plan(files, { capacity: 737280000, reserve: 65536, firstExtra: 800e6 }));
});

test('buildIni: [Extras] mit Namen, ohne Zeileneinschleusung und maximal 5 Eintraege', () => {
  const files = [{ path: 'g.exe', size: 10 }];
  const plan = PRS.plan(files, { capacity: 1_000_000, reserve: 4096 });
  plan.discs.forEach((d) => d.segs.forEach((s) => { s.crc = 1; }));
  const base = { title: 'T', exe: 'g.exe', pkgId: 'abcdef0123456789', dirName: 'T', banner: '', showTitle: true, discCount: 1, files, plan, redist: [] };
  const none = PRS.buildIni(base);
  assert.ok(!none.includes('[Extras]'));
  const ex = [{ path: 'Extras\\Manual.pdf', label: 'Hand|buch\r\n[Files]' }, ...[1, 2, 3, 4, 5].map((i) => ({ path: 'Extras\\f' + i, label: 'F' + i }))];
  const lines = PRS.buildIni({ ...base, extras: ex }).split('\r\n');
  assert.equal(lines.filter((l) => l.startsWith('X=')).length, 5);
  assert.equal(lines.filter((l) => l === '[Files]').length, 1, 'keine eingeschleuste Sektion');
  assert.ok(lines.includes('X=Extras\\Manual.pdf|Hand buch [Files]'));
});
