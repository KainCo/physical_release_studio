//go:build windows

package main

type Strings struct {
	Install, Extras, NoExtras, Cancel, Browse, SubTitle          string
	Copying, InsertDisc, WaitHint, Running, RunningHint          string
	ErrNoINI, ErrNoInstaller, ErrNotFound, ErrOpen, ErrStart     string
	ErrNoSpace, ErrNotDisc, ErrStage, AskDelete, KeepTemp        string
	CloseRunning, PickFolder, ChooseMode, NativeInfo, NoSpaceMsg string
}

var en = Strings{
	Install: "INSTALL GAME", Extras: "EXTRAS", NoExtras: "No extras were included with this edition.",
	Cancel: "Cancel", Browse: "Choose folder...", SubTitle: "GOG OFFLINE INSTALLER",
	Copying: "Copying installation files...", InsertDisc: "Please insert disc %d of %d",
	WaitHint: "The disc is detected automatically. If you work with folders or mounted images, use \"Choose folder...\".",
	Running:  "Setup is running...", RunningHint: "Keep this window open until setup has finished. Afterwards you can delete the temporary files.",
	ErrNoINI:       "GOGAutoLauncher.ini was not found next to the launcher.",
	ErrNoInstaller: "No installer is defined in GOGAutoLauncher.ini (entry Installer= in section [Game]).",
	ErrNotFound:    "File not found:\r\n%s", ErrOpen: "The file could not be opened:\r\n%s", ErrStart: "The installer could not be started:\r\n%s",
	ErrNoSpace: "Not enough free space for the temporary installation files.\r\n\r\nRequired: %s\r\nAvailable: %s\r\n\r\nChoose another folder on a drive with enough free space?",
	ErrNotDisc: "This folder does not contain disc %d of this release.", ErrStage: "The installation files could not be prepared:\r\n%s",
	AskDelete:    "Setup has ended.\r\n\r\nDelete the temporary installation files (%s)?\r\n\r\nChoose Yes only if setup has finished completely.",
	KeepTemp:     "The temporary files remain in:\r\n%s",
	CloseRunning: "Setup may still be running. If you close the launcher now, the temporary files stay in:\r\n%s\r\n\r\nClose anyway?",
	PickFolder:   "Select the folder that contains the contents of disc %d",
	ChooseMode:   "How do you want to install?\r\n\r\nYES - Directly from the discs (recommended). Setup asks for the next disc when it needs it.\r\n\r\nNO - Copy everything to the hard drive first, then install without further disc changes. Needs about %s free space (available: %s).\r\n\r\nCANCEL - Go back.",
	NativeInfo:   "This edition has %d discs.\r\n\r\nSetup will ask you for disc 2 right at the start. Insert it, confirm with OK and follow the instructions. During installation Setup may ask for the discs again.\r\n\r\nContinue?",
	NoSpaceMsg:   "",
}

var de = Strings{
	Install: "SPIEL INSTALLIEREN", Extras: "EXTRAS", NoExtras: "Dieser Edition liegen keine Extras bei.",
	Cancel: "Abbrechen", Browse: "Ordner wählen...", SubTitle: "GOG OFFLINE-INSTALLER",
	Copying: "Installationsdateien werden kopiert...", InsertDisc: "Bitte Disc %d von %d einlegen",
	WaitHint: "Die Disc wird automatisch erkannt. Bei Ordnern oder eingebundenen Abbildern nutze \"Ordner wählen...\".",
	Running:  "Setup läuft...", RunningHint: "Lass dieses Fenster geöffnet, bis das Setup fertig ist. Danach kannst du die temporären Dateien löschen.",
	ErrNoINI:       "GOGAutoLauncher.ini wurde neben dem Launcher nicht gefunden.",
	ErrNoInstaller: "In GOGAutoLauncher.ini fehlt der Eintrag Installer= im Abschnitt [Game].",
	ErrNotFound:    "Die Datei wurde nicht gefunden:\r\n%s", ErrOpen: "Die Datei konnte nicht geöffnet werden:\r\n%s", ErrStart: "Der Installer konnte nicht gestartet werden:\r\n%s",
	ErrNoSpace: "Nicht genug freier Speicher für die temporären Installationsdateien.\r\n\r\nBenötigt: %s\r\nVerfügbar: %s\r\n\r\nEinen anderen Ordner auf einem Laufwerk mit genug Platz wählen?",
	ErrNotDisc: "Dieser Ordner enthält nicht Disc %d dieser Edition.", ErrStage: "Die Installationsdateien konnten nicht vorbereitet werden:\r\n%s",
	AskDelete:    "Das Setup wurde beendet.\r\n\r\nTemporäre Installationsdateien (%s) löschen?\r\n\r\nWähle Ja nur, wenn das Setup vollständig abgeschlossen ist.",
	KeepTemp:     "Die temporären Dateien bleiben hier erhalten:\r\n%s",
	CloseRunning: "Das Setup läuft möglicherweise noch. Wenn du den Launcher jetzt schließt, bleiben die temporären Dateien hier:\r\n%s\r\n\r\nTrotzdem schließen?",
	PickFolder:   "Wähle den Ordner mit dem Inhalt von Disc %d",
	ChooseMode:   "Wie möchtest du installieren?\r\n\r\nJA - Direkt von den Discs (empfohlen). Das Setup fragt nach der nächsten Disc, wenn es sie braucht.\r\n\r\nNEIN - Erst alles auf die Festplatte kopieren und danach ohne weitere Disc-Wechsel installieren. Benötigt ca. %s freien Platz (verfügbar: %s).\r\n\r\nABBRECHEN - Zurück.",
	NativeInfo:   "Diese Edition besteht aus %d Discs.\r\n\r\nDas Setup fragt gleich zu Beginn nach Disc 2. Lege sie ein, bestätige mit OK und folge den Anweisungen. Während der Installation kann das Setup die Discs erneut anfordern.\r\n\r\nFortfahren?",
	NoSpaceMsg:   "",
}
