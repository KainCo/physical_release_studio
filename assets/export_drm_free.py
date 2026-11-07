import csv
import json
import time
from pathlib import Path

import requests

CURATOR_ID = "45614612"
BASE = f"https://store.steampowered.com/curator/{CURATOR_ID}"
INFO_URL = f"{BASE}/ajaxgetcreatorhomeinfo/"
DETAILS_URL = "https://store.steampowered.com/api/appdetails/"
OUT = Path("KainCo_DRM_Free_Steam.csv")

HEADERS = {
    "User-Agent": "Mozilla/5.0 (compatible; KainCoDRMFreeDatabase/1.0)",
    "Accept": "application/json,text/plain,*/*",
    "Referer": f"{BASE}/",
}

def get_json(session, url, params=None, retries=4):
    for attempt in range(retries):
        try:
            response = session.get(url, params=params, timeout=35)
            response.raise_for_status()
            return response.json()
        except (requests.RequestException, ValueError):
            if attempt == retries - 1:
                raise
            time.sleep(2 ** attempt)

def main():
    session = requests.Session()
    session.headers.update(HEADERS)

    print("Lade App-IDs des Steam-Kurators ...")
    info = get_json(session, INFO_URL, {"get_appids": 1, "l": "english"})
    if info.get("success") != 1:
        raise RuntimeError(f"Steam lieferte keine gültigen Kuratorendaten: {info}")

    appids = list(dict.fromkeys(str(x) for x in info.get("appids", [])))
    if not appids:
        raise RuntimeError("Steam hat keine App-IDs zurückgegeben.")

    print(f"Kurator: {info.get('name', 'unbekannt')}")
    print(f"Gefundene App-IDs: {len(appids)}")

    rows = []
    failed = 0
    for index, appid in enumerate(appids, start=1):
        print(f"[{index}/{len(appids)}] AppID {appid}")
        try:
            payload = get_json(
                session, DETAILS_URL,
                {"appids": appid, "cc": "de", "l": "german"}
            )
            item = payload.get(appid, {})
            if item.get("success"):
                data = item.get("data", {})
                rows.append({
                    "title": data.get("name", ""),
                    "appid": appid,
                    "type": data.get("type", ""),
                    "developers": "; ".join(data.get("developers", [])),
                    "publishers": "; ".join(data.get("publishers", [])),
                    "notes": "In der Steam-Kurator-Liste enthalten; DRM-Status nicht unabhängig geprüft."
                })
            else:
                failed += 1
                rows.append({
                    "title": "Steam-Daten nicht verfügbar",
                    "appid": appid,
                    "type": "",
                    "developers": "",
                    "publishers": "",
                    "notes": "AppID vom Kurator geliefert; Steam App Details nicht verfügbar."
                })
        except Exception as exc:
            failed += 1
            rows.append({
                "title": "Abfrage fehlgeschlagen",
                "appid": appid,
                "type": "",
                "developers": "",
                "publishers": "",
                "notes": str(exc)[:300]
            })
        time.sleep(0.25)

    with OUT.open("w", newline="", encoding="utf-8-sig") as handle:
        writer = csv.writer(handle, delimiter=";")
        writer.writerow([
            "Nr", "Title", "AppID", "DRM_Free_According_To_Curator",
            "Type", "Developer", "Publisher", "Notes"
        ])
        for number, row in enumerate(rows, start=1):
            writer.writerow([
                number, row["title"], row["appid"], "Ja",
                row["type"], row["developers"], row["publishers"], row["notes"]
            ])

    summary = {
        "curator_id": CURATOR_ID,
        "curator_name": info.get("name"),
        "appids_found": len(appids),
        "rows_exported": len(rows),
        "details_unavailable_or_failed": failed,
        "csv_file": str(OUT),
    }
    Path("export_summary.json").write_text(
        json.dumps(summary, ensure_ascii=False, indent=2), encoding="utf-8"
    )
    print(json.dumps(summary, ensure_ascii=False, indent=2))

if __name__ == "__main__":
    main()
