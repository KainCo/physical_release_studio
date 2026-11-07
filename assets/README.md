# KainCo DRM-Free Steam Database

Exportiert die App-IDs, die dem Steam-Kurator **DRM-Free Complete List** zugeordnet sind.

- Curator-ID: `45614612`
- Quelle für die App-IDs: öffentlicher Steam-Curator-Endpunkt `ajaxgetcreatorhomeinfo`
- Metadaten: Steam Store App Details
- Ausgabe: `KainCo_DRM_Free_Steam.csv` (Semikolon-getrennt, UTF-8 mit BOM)

## Nutzung über iPhone / Browser

1. Dieses ZIP in ein neues GitHub-Repository namens `KainCo-DRM-Free-Database` hochladen.
2. Im Repository auf **Actions** gehen.
3. Den Workflow **Export DRM-Free Steam curator list** auswählen.
4. **Run workflow** antippen.
5. Nach Abschluss den erzeugten Eintrag unter **Artifacts** herunterladen und entpacken.

Der Workflow braucht keine Steam-Anmeldung und keinen Steam-API-Key.

## Lokal (optional)

```bash
pip install -r requirements.txt
python export_drm_free.py
```

## Hinweise zur Datenqualität

Die Kuratoren-Zugehörigkeit wird als Quellenhinweis exportiert, ist aber keine unabhängige technische DRM-Prüfung. Steam App Details werden einzeln abgefragt, mit Pausen und Wiederholungsversuchen. Nicht verfügbare Metadaten bleiben als markierte Zeilen erhalten, damit App-IDs nicht stillschweigend verloren gehen.

GitHub Actions kann je nach Repository-Sichtbarkeit und Kontingent Nutzungsgrenzen unterliegen.
