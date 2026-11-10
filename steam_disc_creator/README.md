# Steam Disc Creator

**Dieses Tool umgeht keinen Kopierschutz und ist nicht dafür gedacht, einen Kopierschutz zu umgehen oder zu entfernen.**
**This tool does not circumvent any copy protection and is not intended to circumvent or remove any copy protection.**

Nur für private Sicherungskopien von Spielen, die man rechtmäßig besitzt. Steam ist eine Marke der Valve Corporation; das Projekt ist unabhängig und nicht mit Valve oder anderen Firmen verbunden.

Webtool, das aus einem Spielordner ein Paket für **CD, DVD, Blu-ray oder USB** macht: Es teilt die Dateien bei Bedarf auf
mehrere Discs auf, und ein kleiner Windows-Installer (`Launcher.exe`, Win98-Optik, DE/EN/PL/FR/ES/IT) fügt sie wieder zusammen.

- Alles läuft **lokal im Browser**. Keine Uploads, keine externen Skripte, Schriften oder Bilder.
- Installer: Zielordner wählen, Discs nacheinander einlegen, Prüfsummen (CRC32), Deinstaller, Eintrag unter „Apps & Features“.
- Erkennt eine vorhandene Installation und bietet dann **„Spiel starten“** statt „Spiel installieren“.
- Optional: Visual C++, DirectX, .NET u. a. mitinstallieren (Erkennung am Dateinamen, in jedem Ordner) – bei der Installation und später über den Button „Zusatzkomponenten installieren“.
- Im Steam-Modus erzeugt der Installer bei Bedarf ein einfaches `appmanifest`, damit Steam das Spiel als installiert erkennt. Es wird **keine** `steam_appid.txt` angelegt.
- Der Installer und die Dateien auf der Disc tragen keine Namen oder Herausgeber-Angaben; auf jeder Disc liegt eine `LICENSE.txt` (GPL-3.0 und Go-Lizenz).
- Zwei Arten: **Normaler Ordner** (DRM-freie Spiele) oder **Steam-Bibliothek** (Spiel braucht Steam; Installation nach `steamapps\common`).
- Eigenes Banner-Bild einfügen, Medien- und Kapazitätsberechnung (CD 650/700, DVD, DVD DL, BD, BD DL/XL, USB, FAT32, eigene Größe).
- **DRM-Hinweise nur lokal** (Dateinamen, SteamStub-/SecuROM-Abschnitte). Ein Hinweis ist kein Beweis, ein fehlender Hinweis auch nicht;
  `steam_api.dll` zählt bewusst nicht. Dazu ein Link zu PCGamingWiki, der erst nach Klick geöffnet wird. Das Tool verändert keine Spieldateien
  und entfernt keinen Kopierschutz.

## Benutzen

`dist/index.html` im Browser öffnen oder über GitHub Pages bereitstellen (Chrome/Edge schreiben Ordner direkt, andere Browser erzeugen ZIP-Dateien).
Auf der Disc mit UDF brennen. Beim Einlegen von Disc 1 startet der Installer.

## Bauen

Benötigt Go ≥ 1.21, Node ≥ 20, Python 3. Keine Pakete von außen.

```
python3 build.py                  # Tests, Launcher.exe, dist/index.html
python3 build.py --skip-tests
python3 build.py --launcher dist/Launcher.exe   # eine (z. B. signierte) EXE einbetten
```

Signieren: erst bauen, `dist/Launcher.exe` signieren, dann mit `--launcher` erneut einbetten. Ohne Signatur zeigt Windows
„Herausgeber nicht angegeben“, und manche Virenscanner schlagen bei unbekannten Programmen an.

Tests: `go test ./...` (in `launcher/`), `node --test test/core.test.mjs test/e2e.test.mjs` (Kernlogik + Ende-zu-Ende: Web-Paket → Go-Installer → Byte-Vergleich → Deinstallation).
Die Tests laufen unter Linux; der Installer hat dort einen Kommandozeilen-Testmodus.

## Aufbau

```
src/        index.template.html, core.js (Kernlogik), datenschutz.html, autorun.ico
launcher/   Go-Installer/Launcher (nur Standardbibliothek, Win32 per syscall, kein Netzwerkcode)
tools/      mkrsrc (Symbol + Manifest), checkpe (EXE-Prüfung)
test/       Node-Tests
docs/       FORMAT.md
dist/       Build-Ausgabe (index.html, Launcher.exe, SHA256SUMS.txt)
```

## Bekannte Grenzen

- Die Windows-Oberfläche des Launchers wurde unter Linux kompiliert und ihre Logik getestet, aber **nicht auf echtem Windows**. Bitte vor dem Verteilen einmal mit echten Discs oder USB prüfen.
- Steam-Modus: dass Steam das kopierte Spiel samt Manifest ohne erneute Prüfung akzeptiert, ist ungetestet.
- Übersetzungen PL/FR/ES/IT sind maschinell erstellt, nicht muttersprachlich geprüft.
- Nur Windows 10/11 (64 Bit) für den Installer.
- Nutze das Tool nur für Spiele, die du rechtmäßig besitzt, und gib Medien nur mit Erlaubnis der Rechteinhaber weiter.

Lizenz: GPL-3.0-or-later (siehe `LICENSE`).

## Schriften
Inter und JetBrains Mono (SIL OFL 1.1) sind als Subset eingebettet; Lizenztexte: src/fonts/*-OFL.txt.
