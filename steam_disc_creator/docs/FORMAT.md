# Paketformat

Ein Paket besteht aus einem Ordner pro Disc (`Disc1`, `Disc2`, …). Jede Disc enthält dieselben Verwaltungsdateien
und einen Teil der Spieldateien unter `Game\`.

| Datei | Zweck |
|---|---|
| `Launcher.exe` | Installer, Starter und (als Kopie `uninstall.exe` im Spielordner) Deinstaller |
| `launcher.ini` | Beschreibung des Pakets (siehe unten), auf jeder Disc identisch |
| `DISC_N.txt` | Marker: `PackageID`, `Disc`, `Count`, `Title`; damit erkennt der Installer die Disc N |
| `autorun.inf`, `autorun.ico` | AutoPlay |
| `README.txt` | Hinweise für Endnutzer (DE/EN) |
| `banner.jpg` / `banner.png` | optionales Bild (vom Nutzer gewählt) |
| `extras\appmanifest.acf` | optional, nur im Steam-Modus, nur auf Disc 1 |
| `Game\…` | Spieldateien; geteilte Dateien als `Name.ext.part1`, `.part2`, … |

## launcher.ini

UTF-8 mit BOM, Zeilenende CRLF. Pfade mit `\`.

```
[Game]
Title=Anzeigename
Exe=Ordner\Programm.exe            ; Datei, die "Spiel starten" öffnet (im Steam-Modus nur Anzeige)
PackageID=0123456789abcdef         ; 8–32 Hex-Zeichen, pro Paket zufällig
DirName=Vorschlag Ordnername
Banner=banner.jpg
ShowTitle=1
Mode=steam                         ; nur im Steam-Modus
SteamAppID=620                     ; Pflicht im Steam-Modus, sonst optional: damit wird steam_appid.txt angelegt, falls sie fehlt
SteamManifest=extras\appmanifest.acf   ; optional, nur im Steam-Modus
[Disc]
Count=3
[Files]
F=<Größe>|<Pfad im Spielordner>
S=<Disc>|<Offset>|<Länge>|<CRC32 hex>|<Pfad auf der Disc>   ; ein S pro Teil, direkt nach seinem F
[Redist]
R=<Pfad>|<Parameter>|<Anzeigename>
```

Der Installer prüft beim Einlesen: gültige, nicht ausbrechende Pfade (kein `..`, keine Laufwerksbuchstaben),
lückenlose Teile, Summe = Dateigröße. Jeder Teil wird beim Kopieren mit CRC32 geprüft.

## Installations- und Deinstallationsablauf

1. Installiert-Erkennung: Registry-Eintrag `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\DiscPkg_<PackageID>`
   (+ Programmdatei vorhanden) → Oberfläche zeigt „Spiel starten“, „Neu installieren“, „Deinstallieren“.
2. Installation: Discs werden bei Bedarf angefordert (Suche auf allen Laufwerken und in Geschwisterordnern `DiscN`).
3. Danach: `uninstall.exe`, Registry-Eintrag, optionale Verknüpfungen, optionale Komponenten (`[Redist]`, mit Admin-Abfrage).
4. Alles, was angelegt wurde, steht in `uninstall.lst` (`H` Kopfzeile, `T` Titel, `P` Paket-ID, `E` Programmdatei, `M` Modus `steam`,
   `F` Datei, `D` Ordner, `L` Verknüpfung, `X` Datei außerhalb des Zielordners).
   Die Deinstallation löscht nur, was dort steht. Fremde Dateien im Spielordner (z. B. Spielstände) bleiben, außer der Nutzer
   stimmt zu und der Installer hat den Ordner selbst angelegt.
5. Steam-Modus: Ziel ist `<Bibliothek>\steamapps\common\<Name>`. Neben `common` wird `appmanifest_<AppID>.acf` angelegt: die mitgelieferte
   Datei (`SteamManifest`) oder, wenn keine dabei ist, ein schlichtes, vom Installer erzeugtes Manifest (`appid`, `name`, `installdir`,
   `StateFlags` 4, `SizeOnDisk`, `buildid` 0). Eine vorhandene Datei wird nie überschrieben und bei der Deinstallation nie gelöscht.
   „Spiel starten“ öffnet `steam://rungameid/<AppID>`. Bei der Deinstallation erscheint der Hinweis, das Spiel auch in Steam zu entfernen.
6. `steam_appid.txt` (nur AppID, ohne Zeilenumbruch) wird in beiden Modi neben der Programmdatei angelegt, wenn `SteamAppID` bekannt ist und
   die Datei fehlt. Eine vorhandene Datei bleibt unverändert; eine vom Installer angelegte wird bei der Deinstallation entfernt.
7. Zusatzkomponenten (`[Redist]`) erkennt die Webseite am Dateinamen (`vcredist*.exe`, `vc_redist*.exe`, `dxsetup.exe`, `dotnetfx*.exe` u. a.),
   unabhängig vom Ordner. Der Launcher bietet sie bei der Installation an und später über den Button „Zusatzkomponenten installieren“.

## [Extras] (optional)

`X=<Pfad relativ zum Disc-Stamm>|<Name der Schaltfläche>` – bis zu 5 Einträge. Der Launcher zeigt für jeden Eintrag, dessen Datei im Stamm der Disc vorhanden ist, eine Schaltfläche auf der Startseite und öffnet die Datei mit dem Standardprogramm. Pfade mit `..` oder absolute Pfade werden ignoriert.
