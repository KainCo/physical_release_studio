// Steam Disc Creator – Kernlogik (läuft im Browser und unter Node.js für die Tests)
// SPDX-License-Identifier: GPL-3.0-or-later
(function (root) {
  'use strict';
  const PRS = {};

  // ---------- CRC32 (IEEE, wie in Go hash/crc32) ----------
  const T = new Int32Array(256 * 4);
  (function () {
    for (let i = 0; i < 256; i++) {
      let c = i;
      for (let k = 0; k < 8; k++) c = c & 1 ? (c >>> 1) ^ 0xedb88320 : c >>> 1;
      T[i] = c;
    }
    for (let i = 0; i < 256; i++) {
      for (let s = 1; s < 4; s++) T[s * 256 + i] = (T[(s - 1) * 256 + i] >>> 8) ^ T[T[(s - 1) * 256 + i] & 255];
    }
  })();
  PRS.crc32 = function (buf, crc) {
    crc = ~(crc || 0);
    let i = 0;
    const n = buf.length;
    for (; i + 4 <= n; i += 4) {
      crc ^= buf[i] | (buf[i + 1] << 8) | (buf[i + 2] << 16) | (buf[i + 3] << 24);
      crc = T[768 + (crc & 255)] ^ T[512 + ((crc >>> 8) & 255)] ^ T[256 + ((crc >>> 16) & 255)] ^ T[crc >>> 24];
    }
    for (; i < n; i++) crc = T[(crc ^ buf[i]) & 255] ^ (crc >>> 8);
    return ~crc >>> 0;
  };
  PRS.hex8 = (n) => (n >>> 0).toString(16).padStart(8, '0');

  // ---------- Medien ----------
  // Kapazitäten in Bytes (Datenbereich, 2048-Byte-Sektoren). USB-Medien haben kein festes Limit.
  PRS.MEDIA = [
    { id: 'cd650', bytes: 681984000, kind: 'disc' },
    { id: 'cd700', bytes: 737280000, kind: 'disc' },
    { id: 'cd800', bytes: 829440000, kind: 'disc' }, // 90 min = 405000 Sektoren
    { id: 'dvd5', bytes: 4707319808, kind: 'disc' },
    { id: 'dvd9', bytes: 8543666176, kind: 'disc' },
    { id: 'bd25', bytes: 25025314816, kind: 'disc' },
    { id: 'bd50', bytes: 50050629632, kind: 'disc' },
    { id: 'bd100', bytes: 100103356416, kind: 'disc' },
    { id: 'usb', bytes: Infinity, kind: 'usb' },
    { id: 'usb32', bytes: Infinity, maxPart: 4294901760, kind: 'usb' }, // FAT32: max. 4 GiB pro Datei
    { id: 'custom', bytes: 0, kind: 'disc' },
  ];
  const SECTOR = 2048;
  const FILE_OVERHEAD = 160; // Verzeichniseintrag (UDF + Joliet) pro Datei
  PRS.SAFETY = 0.01; // 1 % Sicherheitsreserve gegen Brennprogramm-Abweichungen
  PRS.alloc = (size) => Math.ceil(size / SECTOR) * SECTOR + FILE_OVERHEAD;

  // ---------- Aufteilen ----------
  // files: [{path, size}] ; opt: {capacity, reserve, maxPart}
  // Ergebnis: {discs:[{used, segs:[{fi, offset, size, disc, part, parts}]}], usable, split:[fi...]}
  PRS.plan = function (files, opt) {
    const capacity = opt.capacity;
    const reserve = opt.reserve || 0;
    const unlimited = !isFinite(capacity);
    const usable = unlimited ? Infinity : Math.floor(capacity * (1 - PRS.SAFETY)) - reserve;
    let maxPart = opt.maxPart || Infinity;
    if (!unlimited) {
      if (usable < 4 * SECTOR + FILE_OVERHEAD) throw new Error('capacity');
      const per = Math.floor((usable - FILE_OVERHEAD) / SECTOR) * SECTOR; // größte Dateilänge, die allein auf eine Disc passt
      maxPart = Math.min(maxPart, per);
    } else if (isFinite(maxPart)) {
      maxPart = Math.floor(maxPart / SECTOR) * SECTOR;
    }
    // 1. Dateien, die nicht in einen Teil passen, werden in Stücke geteilt
    const items = [];
    const split = [];
    files.forEach((f, fi) => {
      if (f.size <= maxPart) {
        items.push({ fi, offset: 0, size: f.size, part: 1, parts: 1 });
      } else {
        const parts = Math.ceil(f.size / maxPart);
        split.push(fi);
        for (let p = 0; p < parts; p++) {
          const off = p * maxPart;
          items.push({ fi, offset: off, size: Math.min(maxPart, f.size - off), part: p + 1, parts });
        }
      }
    });
    // 2. First-Fit-Decreasing: große Teile zuerst, möglichst wenige Discs
    const order = items.map((it, i) => i).sort((a, b) => {
      const d = PRS.alloc(items[b].size) - PRS.alloc(items[a].size);
      return d || items[a].fi - items[b].fi || items[a].part - items[b].part;
    });
    const discs = [];
    if (opt.firstExtra > 0) discs.push({ used: opt.firstExtra, segs: [] }); // Extras liegen nur auf Disc 1
    if (!unlimited && opt.firstExtra > usable) throw new Error('capacity');
    for (const idx of order) {
      const it = items[idx];
      const need = PRS.alloc(it.size);
      let d = null;
      if (!unlimited) {
        for (const c of discs) if (usable - c.used >= need) { d = c; break; }
      } else d = discs[0] || null;
      if (!d) { d = { used: 0, segs: [] }; discs.push(d); }
      d.used += need;
      d.segs.push(it);
    }
    if (!discs.length) discs.push({ used: 0, segs: [] });
    discs.forEach((d, i) => {
      d.segs.sort((a, b) => (files[a.fi].path < files[b.fi].path ? -1 : files[a.fi].path > files[b.fi].path ? 1 : a.part - b.part));
      d.segs.forEach((s) => { s.disc = i + 1; });
    });
    return { discs, usable, split, capacity, maxPart };
  };

  // Pfad des Teils auf der Disc: Game\<Pfad>[.partN]
  PRS.discPath = function (file, seg) {
    const p = 'Game\\' + file.path.split('/').join('\\');
    return seg.parts > 1 ? p + '.part' + seg.part : p;
  };

  // ---------- Konfiguration (launcher.ini) ----------
  const clean = (s) => String(s == null ? '' : s).replace(/[\r\n]+/g, ' ').trim();
  PRS.buildIni = function (o) {
    // o: {title, exe, pkgId, dirName, banner, showTitle, discCount, files, plan, redist, mode, steamAppId, steamManifest}
    const L = [];
    L.push('[Game]');
    L.push('Title=' + clean(o.title));
    L.push('Exe=' + o.exe.split('/').join('\\'));
    L.push('PackageID=' + o.pkgId);
    if (o.dirName) L.push('DirName=' + clean(o.dirName));
    if (o.banner) L.push('Banner=' + o.banner);
    L.push('ShowTitle=' + (o.showTitle ? 1 : 0));
    if (o.mode === 'steam') L.push('Mode=steam');
    // Die AppID wird in beiden Modi gespeichert (im Steam-Modus für das Manifest).
    if (/^\d{1,10}$/.test(o.steamAppId || '')) L.push('SteamAppID=' + o.steamAppId);
    if (o.mode === 'steam' && o.steamManifest) L.push('SteamManifest=' + o.steamManifest);
    L.push('[Disc]');
    L.push('Count=' + o.discCount);
    L.push('[Files]');
    const segsByFile = new Map();
    for (const d of o.plan.discs) for (const s of d.segs) {
      if (!segsByFile.has(s.fi)) segsByFile.set(s.fi, []);
      segsByFile.get(s.fi).push(s);
    }
    o.files.forEach((f, fi) => {
      L.push('F=' + f.size + '|' + f.path.split('/').join('\\'));
      for (const s of (segsByFile.get(fi) || []).sort((a, b) => a.offset - b.offset)) {
        L.push('S=' + s.disc + '|' + s.offset + '|' + s.size + '|' + PRS.hex8(s.crc) + '|' + PRS.discPath(f, s));
      }
    });
    if (o.extras && o.extras.length) {
      L.push('[Extras]');
      for (const x of o.extras.slice(0, 5)) L.push('X=' + x.path.split('/').join('\\') + '|' + clean(x.label).replace(/\|/g, ' '));
    }
    if (o.redist && o.redist.length) {
      L.push('[Redist]');
      for (const r of o.redist) L.push('R=' + r.path.split('/').join('\\') + '|' + clean(r.args) + '|' + clean(r.label));
    }
    return L.join('\r\n') + '\r\n';
  };

  PRS.estimateIniBytes = function (files) {
    let n = 400;
    for (const f of files) n += 2 * f.path.length + 90;
    return Math.ceil(n * 1.15);
  };

  PRS.markerName = (n) => 'DISC_' + n + '.txt';
  PRS.markerText = (pkgId, n, count, title) =>
    'PackageID=' + pkgId + '\r\nDisc=' + n + '\r\nCount=' + count + '\r\nTitle=' + clean(title) + '\r\n';

  PRS.autorunInf = (title, exeName) =>
    '[AutoRun]\r\nopen=' + exeName + '\r\nicon=autorun.ico\r\nlabel=' + clean(title).slice(0, 32) + '\r\n';

  // ---------- Dateinamen ----------
  PRS.fsName = function (s) {
    s = clean(s).replace(/[<>]/g, ' ').replace(/:/g, ' -').replace(/"/g, "'").replace(/[\/\\|]/g, '-').replace(/[?*]/g, '');
    s = s.replace(/\s+/g, ' ').replace(/[. ]+$/, '').trim();
    return s || 'Game';
  };

  // ---------- Analyse der Dateiliste ----------
  const base = (p) => p.split('/').pop().toLowerCase();
  const REDIST_DIR = /(^|\/)(_commonredist|redist|redistributables|commonredist|__installer)(\/|$)/i;
  PRS.isRedistPath = (p) => REDIST_DIR.test(p);

  const REDIST_RULES = [
    { re: /^vc_?redist.*\.exe$/i, label: 'Visual C++', args: (n) => (/^vc_redist/i.test(n) ? '/install /quiet /norestart' : '/q /norestart') },
    { re: /^dxsetup\.exe$/i, label: 'DirectX', args: () => '/silent' },
    { re: /^(dotnetfx|ndp)[^/]*\.exe$/i, label: '.NET Framework', args: () => '/q /norestart' },
    { re: /^xnafx[^/]*\.msi$/i, label: 'XNA Framework', args: () => '/qn /norestart' },
    { re: /^oalinst\.exe$/i, label: 'OpenAL', args: () => '/silent' },
    { re: /^physx[^/]*\.exe$/i, label: 'PhysX', args: () => '/s' },
    { re: /^physx[^/]*\.msi$/i, label: 'PhysX', args: () => '/qn /norestart' },
  ];
  // Erkennt bekannte Installationsprogramme für Zusatzkomponenten anhand des Dateinamens,
  // unabhängig vom Ordner (z. B. _CommonRedist, DirectX, vcredist).
  PRS.isRedistFile = (p) => { const b = p.split('/').pop(); return REDIST_RULES.some((r) => r.re.test(b)); };
  PRS.detectRedist = function (paths) {
    const out = [];
    for (const p of paths) {
      const b = p.split('/').pop();
      for (const r of REDIST_RULES) {
        if (r.re.test(b)) {
          out.push({ path: p, args: r.args(b), label: r.label + ' (' + b + ')', rule: r.label });
          break;
        }
      }
    }
    out.sort((a, b) => (a.path < b.path ? -1 : 1));
    return out;
  };

  const EXE_IGNORE = /^(unins|uninst|vc_?redist|vcredist|dxsetup|dxwebsetup|dotnet|ndp|crashhandler|crashreport|crashpad|unitycrash|ue4prereq|oalinst|physx|setup|installer|redist|easyanticheat|beservice|bssndrpt|cefsub|vulkan-?\d|notification_helper|errorreporter|bugreport|launcherhelper|steamerrorreporter)/i;
  // Kandidaten für die Programmdatei, beste zuerst.
  PRS.exeCandidates = function (files, folderName) {
    const fn = (folderName || '').toLowerCase().replace(/[^a-z0-9]/g, '');
    const out = [];
    for (const f of files) {
      if (!/\.exe$/i.test(f.path) || REDIST_DIR.test(f.path) || PRS.isRedistFile(f.path)) continue;
      const b = f.path.split('/').pop();
      if (EXE_IGNORE.test(b)) continue;
      const depth = f.path.split('/').length - 1;
      const stem = b.toLowerCase().replace(/\.exe$/, '').replace(/[^a-z0-9]/g, '');
      let score = Math.max(0, 3 - depth) * 2;
      if (fn && stem && (fn.includes(stem) || stem.includes(fn))) score += 6;
      if (/shipping|win64|x64|x86_64/i.test(b)) score -= 1;
      out.push({ path: f.path, size: f.size, score });
    }
    out.sort((a, b) => b.score - a.score || b.size - a.size || (a.path < b.path ? -1 : 1));
    return out;
  };

  // Hinweise aus Dateinamen. Reine Hinweise – kein Beweis für oder gegen DRM-Freiheit.
  const DRM_FILES = [
    { re: /^(uplay_r[12]_loader(64)?\.dll|upc_r[12]_loader(64)?\.dll|uplayservice\.exe|ubiorbitapi_r2_loader(64)?\.dll)$/i, kind: 'uplay' },
    { re: /^(originsdk(64)?\.dll|origin\.exe|eaapp.*\.exe)$/i, kind: 'origin' },
    { re: /^(secdrv\.sys|drvmgt\.dll|clcd(16|32)\.dll|clokspl\.exe|00000001\.tmp)$/i, kind: 'safedisc' },
    { re: /^(securom[^/]*\.(dll|exe)|paul\.dll)$/i, kind: 'securom' },
    { re: /^(sfdrv\d*\.sys|sfhlp\d*\.sys)$/i, kind: 'starforce' },
    { re: /^xlive\.dll$/i, kind: 'gfwl' },
    { re: /^(socialclubhelper\.exe|launcherpatcher\.exe)$/i, kind: 'rockstar' },
    { re: /denuvo/i, kind: 'denuvo' },
  ];
  PRS.scanNames = function (paths) {
    const r = { drm: [], steamApi: [], steamAppIdFile: null, installScript: false };
    const seen = new Set();
    for (const p of paths) {
      const b = base(p);
      if (/^steam_api(64)?\.dll$/.test(b) || /^steamclient(64)?\.dll$/.test(b) || /^steamworks.*\.dll$/.test(b)) r.steamApi.push(p);
      if (b === 'steam_appid.txt' && (!r.steamAppIdFile || p.length < r.steamAppIdFile.length)) r.steamAppIdFile = p;
      if (/^installscript.*\.vdf$/.test(b)) r.installScript = true;
      for (const rule of DRM_FILES) {
        if (rule.re.test(b) && !seen.has(rule.kind)) { seen.add(rule.kind); r.drm.push({ kind: rule.kind, path: p }); }
      }
    }
    return r;
  };

  // Erkennt Spiele auf der Source-Engine (Valve). Diese laufen in der Regel nur mit Steam.
  PRS.detectSourceEngine = function (paths) {
    let info = false, marker = false;
    for (const p of paths) {
      const b = base(p);
      if (b === 'gameinfo.txt') info = true;
      if (b === 'steam.inf' || b === 'tier0.dll' || b === 'vstdlib.dll' || b === 'hl2.exe' || (b === 'engine.dll' && /(^|\/)bin\//i.test(p))) marker = true;
    }
    return info && marker;
  };

  // Sucht in einer EXE nach Namen älterer Laufzeitbibliotheken (Hinweis, keine Garantie).
  const RUNTIME_RULES = [
    { kind: 'vc2005', re: /msvc[pr]80\.dll/i },
    { kind: 'vc2008', re: /msvc[pr]90\.dll/i },
    { kind: 'vc2010', re: /msvc[pr]100\.dll/i },
    { kind: 'vc2012', re: /msvc[pr]110\.dll/i },
    { kind: 'vc2013', re: /msvc[pr]120\.dll/i },
    { kind: 'vc2015', re: /vcruntime140(_1)?\.dll/i },
    { kind: 'dx9', re: /(d3dx9_\d+|d3dx10_\d+|d3dx11_\d+|x3daudio1_\d|xactengine\d_\d|xaudio2_\d|xinput1_[123])\.dll/i },
  ];
  PRS.runtimeHints = function (u8) {
    const txt = new TextDecoder('latin1').decode(u8);
    const out = [];
    for (const r of RUNTIME_RULES) if (r.re.test(txt)) out.push(r.kind);
    return out;
  };

  // PE-Abschnittsnamen aus den ersten Bytes einer EXE lesen.
  PRS.peSections = function (u8) {
    const dv = new DataView(u8.buffer, u8.byteOffset, u8.byteLength);
    if (u8.length < 0x40 || dv.getUint16(0, true) !== 0x5a4d) return null;
    const pe = dv.getUint32(0x3c, true);
    if (pe + 24 > u8.length || dv.getUint32(pe, true) !== 0x00004550) return null;
    const n = dv.getUint16(pe + 6, true);
    const optSize = dv.getUint16(pe + 20, true);
    let off = pe + 24 + optSize;
    const names = [];
    for (let i = 0; i < n && off + 40 <= u8.length; i++, off += 40) {
      let s = '';
      for (let k = 0; k < 8 && u8[off + k]; k++) s += String.fromCharCode(u8[off + k]);
      names.push(s);
    }
    return names;
  };
  // Abschnittsnamen, die auf Schutzhüllen hindeuten.
  PRS.sectionHints = function (names) {
    const out = [];
    for (const n of names) {
      if (n === '.bind') out.push('steamstub');
      else if (n === '.cms_t' || n === '.cms_d' || n === '.securom') out.push('securom');
    }
    return out.filter((v, i, a) => a.indexOf(v) === i);
  };

  // Gesamtergebnis: 'warn' (Spuren gefunden) oder 'none' (keine Spuren – kein Beweis!).
  PRS.drmVerdict = function (names, peHits) {
    const hits = [];
    for (const d of names.drm) hits.push({ kind: d.kind, path: d.path });
    for (const h of peHits) hits.push({ kind: h.kind, path: h.path });
    return { level: hits.length ? 'warn' : 'none', hits, steamApi: names.steamApi };
  };

  // ---------- Steam-Manifest (appmanifest_<id>.acf) ----------
  PRS.parseAcf = function (text) {
    const g = (k) => { const m = new RegExp('"' + k + '"\\s+"([^"]*)"', 'i').exec(text); return m ? m[1] : ''; };
    return { appid: g('appid'), name: g('name'), installdir: g('installdir') };
  };

  // ---------- ZIP (Store, ZIP64) ----------
  const enc = new TextEncoder();
  function dosTime(d) {
    return [((d.getHours() << 11) | (d.getMinutes() << 5) | (d.getSeconds() >> 1)) & 0xffff,
      (((d.getFullYear() - 1980) << 9) | ((d.getMonth() + 1) << 5) | d.getDate()) & 0xffff];
  }
  const u16 = (v) => { const b = new Uint8Array(2); new DataView(b.buffer).setUint16(0, v, true); return b; };
  const u32 = (v) => { const b = new Uint8Array(4); new DataView(b.buffer).setUint32(0, v >>> 0, true); return b; };
  const u64 = (v) => { const b = new Uint8Array(8); const dv = new DataView(b.buffer); dv.setUint32(0, v % 4294967296, true); dv.setUint32(4, Math.floor(v / 4294967296), true); return b; };
  const cat = (...a) => { let n = 0; for (const x of a) n += x.length; const o = new Uint8Array(n); let p = 0; for (const x of a) { o.set(x, p); p += x.length; } return o; };

  // entries: [{name (mit /), size, crc, blob}] -> Blob. blob = Blob/File oder Uint8Array.
  PRS.zipBlob = function (entries, BlobCtor) {
    const B = BlobCtor || root.Blob;
    const [tm, dt] = dosTime(new Date());
    const parts = [];
    const central = [];
    let pos = 0;
    for (const e of entries) {
      const name = enc.encode(e.name);
      const extraL = cat(u16(1), u16(16), u64(e.size), u64(e.size));
      const lh = cat(u32(0x04034b50), u16(45), u16(0x0800), u16(0), u16(tm), u16(dt), u32(e.crc), u32(0xffffffff), u32(0xffffffff),
        u16(name.length), u16(extraL.length), name, extraL);
      parts.push(lh, e.blob);
      const extraC = cat(u16(1), u16(24), u64(e.size), u64(e.size), u64(pos));
      central.push(cat(u32(0x02014b50), u16(45), u16(45), u16(0x0800), u16(0), u16(tm), u16(dt), u32(e.crc), u32(0xffffffff), u32(0xffffffff),
        u16(name.length), u16(extraC.length), u16(0), u16(0), u16(0), u32(0), u32(0xffffffff), name, extraC));
      pos += lh.length + e.size;
    }
    const cd = cat(...central);
    const eocd64Pos = pos + cd.length;
    const tail = cat(
      u32(0x06064b50), u64(44), u16(45), u16(45), u32(0), u32(0), u64(entries.length), u64(entries.length), u64(cd.length), u64(pos),
      u32(0x07064b50), u32(0), u64(eocd64Pos), u32(1),
      u32(0x06054b50), u16(0), u16(0), u16(0xffff), u16(0xffff), u32(0xffffffff), u32(0xffffffff), u16(0));
    return new B([...parts, cd, tail]);
  };

  // ---------- Paket zusammenstellen ----------
  PRS.randomId = function () {
    const a = new Uint8Array(8);
    (root.crypto || require('crypto').webcrypto).getRandomValues(a);
    return Array.from(a, (x) => x.toString(16).padStart(2, '0')).join('');
  };

  const SLICE = 8 * 1024 * 1024;
  // Liest ein Blob in Stücken, ruft cb(Uint8Array) für jedes auf.
  async function eachChunk(blob, start, end, cb, signal) {
    for (let p = start; p < end; p += SLICE) {
      if (signal && signal.aborted) throw new Error('abort');
      const buf = new Uint8Array(await blob.slice(p, Math.min(end, p + SLICE)).arrayBuffer());
      await cb(buf);
    }
  }
  PRS.eachChunk = eachChunk;

  // Berechnet die CRC32 aller Teile (für die ZIP-Ausgabe; die Ordnerausgabe rechnet beim Schreiben mit).
  PRS.computeCrcs = async function (files, plan, progress, signal) {
    let done = 0, total = 0;
    for (const d of plan.discs) for (const s of d.segs) total += s.size;
    for (const d of plan.discs) for (const s of d.segs) {
      let c = 0;
      await eachChunk(files[s.fi].blob, s.offset, s.offset + s.size, (buf) => { c = PRS.crc32(buf, c); done += buf.length; if (progress) progress(done, total); }, signal);
      s.crc = c;
    }
  };

  // Schreibt eine Disc als Ordner. dir = FileSystemDirectoryHandle (oder kompatibel).
  async function ensureDir(dir, parts, cache, key) {
    let h = dir, k = key;
    for (const p of parts) {
      k += '/' + p;
      if (!cache.has(k)) cache.set(k, await h.getDirectoryHandle(p, { create: true }));
      h = cache.get(k);
    }
    return h;
  }
  async function writeFile(dir, path, cache, key, producer) {
    const parts = path.split('\\');
    const name = parts.pop();
    const d = await ensureDir(dir, parts, cache, key);
    const fh = await d.getFileHandle(name, { create: true });
    const w = await fh.createWritable();
    try { await producer(w); } finally { await w.close(); }
  }
  PRS.writeDataToDirectory = async function (discDirs, files, plan, progress, signal) {
    let done = 0, total = 0;
    for (const d of plan.discs) for (const s of d.segs) total += s.size;
    const caches = discDirs.map(() => new Map());
    for (let i = 0; i < plan.discs.length; i++) {
      for (const s of plan.discs[i].segs) {
        const f = files[s.fi];
        let c = 0;
        await writeFile(discDirs[i], PRS.discPath(f, s), caches[i], 'd' + i, async (w) => {
          await eachChunk(f.blob, s.offset, s.offset + s.size, async (buf) => {
            c = PRS.crc32(buf, c);
            await w.write(buf);
            done += buf.length;
            if (progress) progress(done, total, f.path);
          }, signal);
        });
        s.crc = c;
        if (!s.size && progress) progress(done, total, f.path);
      }
    }
  };
  PRS.writeBlobToDirectory = async function (dir, path, blob, cache, signal) {
    await writeFile(dir, path, cache || new Map(), 'x', async (w) => { await eachChunk(blob, 0, blob.size, async (buf) => { await w.write(buf); }, signal); });
  };
  PRS.blobCrc = async function (blob, signal) {
    let c = 0; await eachChunk(blob, 0, blob.size, (buf) => { c = PRS.crc32(buf, c); }, signal); return c;
  };
  PRS.writeBytesToDirectory = async function (dir, path, bytes, cache) {
    await writeFile(dir, path, cache || new Map(), 'x', async (w) => { if (bytes.length) await w.write(bytes); });
  };

  // Zusatzdateien, die auf jede Disc gehören. Liefert [{path, bytes}].
  PRS.extraFiles = function (o, discNo) {
    const list = [
      { path: 'launcher.ini', bytes: enc.encode('﻿' + o.ini) },
      { path: PRS.markerName(discNo), bytes: enc.encode(PRS.markerText(o.pkgId, discNo, o.discCount, o.title)) },
      { path: 'autorun.inf', bytes: enc.encode(PRS.autorunInf(o.title, o.exeName)) },
      { path: 'autorun.ico', bytes: o.icoBytes },
      { path: o.exeName, bytes: o.launcherBytes },
      { path: o.readmeName, bytes: enc.encode(o.readme) },
    ];
    if (o.licenseBytes) list.push({ path: 'LICENSE.txt', bytes: o.licenseBytes });
    if (o.bannerName) list.push({ path: o.bannerName, bytes: o.bannerBytes });
    if (o.acfBytes) list.push({ path: 'extras\\appmanifest.acf', bytes: o.acfBytes });
    return list;
  };

  PRS.readmeText = function (title, count) {
    return [
      title,
      '='.repeat(Math.min(60, title.length)),
      '',
      'DE: Legen Sie Disc 1 ein. Startet der Installer nicht von selbst, öffnen Sie die Disc im Explorer',
      '    und starten Sie Launcher.exe. Bei Bedarf werden Sie aufgefordert, weitere Discs einzulegen (insgesamt ' + count + ').',
      '    Das Spiel wird nur installiert, wenn Sie es rechtmäßig besitzen. Brennen Sie mit dem Dateisystem UDF.',
      '',
      'EN: Insert disc 1. If the installer does not start automatically, open the disc in Explorer and run',
      '    Launcher.exe. You will be asked to insert further discs when needed (' + count + ' in total).',
      '    Only install the game if you own it legally. Burn with the UDF file system.',
      '',
      'Dieses Programm steht unter der GPL-3.0 / This program is licensed under the GPL-3.0 (siehe LICENSE.txt / see LICENSE.txt)',
      '',
    ].join('\r\n');
  };

  if (typeof module !== 'undefined' && module.exports) module.exports = PRS;
  else root.PRS = PRS;
})(typeof globalThis !== 'undefined' ? globalThis : this);
