//go:build windows

package main

type Strings struct {
	Install, Extras, NoExtras, Cancel, Browse, SubTitle                 string
	Copying, InsertDisc, WaitHint, Running, RunningHint                 string
	ErrNoINI, ErrNoInstaller, ErrNotFound, ErrOpen, ErrStart            string
	ErrNoSpace, ErrNotDisc, ErrStage, AskDelete, KeepTemp               string
	CloseRunning, PickFolder, ChooseMode, NativeInfo, NoCover, DiscsFmt string
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
	NoCover:      "NO COVER ART", DiscsFmt: "%d discs",
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
	NoCover:      "KEIN COVER", DiscsFmt: "%d Discs",
}

var pl = Strings{
	Install:        "ZAINSTALUJ GRĘ",
	Extras:         "DODATKI",
	NoExtras:       "Ta edycja nie zawiera dodatków.",
	Cancel:         "Anuluj",
	Browse:         "Wybierz folder...",
	SubTitle:       "INSTALATOR OFFLINE GOG",
	Copying:        "Kopiowanie plików instalacyjnych...",
	InsertDisc:     "Włóż płytę %d z %d",
	WaitHint:       "Płyta zostanie wykryta automatycznie. Jeśli pracujesz z folderami lub zamontowanymi obrazami, użyj „Wybierz folder...”.",
	Running:        "Instalator jest uruchomiony...",
	RunningHint:    "Nie zamykaj tego okna, dopóki instalacja się nie zakończy. Potem możesz usunąć pliki tymczasowe.",
	ErrNoINI:       "Nie znaleziono pliku GOGAutoLauncher.ini obok programu uruchamiającego.",
	ErrNoInstaller: "W pliku GOGAutoLauncher.ini brakuje wpisu Installer= w sekcji [Game].",
	ErrNotFound:    "Nie znaleziono pliku:\r\n%s",
	ErrOpen:        "Nie można otworzyć pliku:\r\n%s",
	ErrStart:       "Nie można uruchomić instalatora:\r\n%s",
	ErrNoSpace:     "Za mało wolnego miejsca na tymczasowe pliki instalacyjne.\r\n\r\nPotrzeba: %s\r\nDostępne: %s\r\n\r\nWybrać inny folder na dysku z wystarczającą ilością miejsca?",
	ErrNotDisc:     "Ten folder nie zawiera płyty %d tej edycji.",
	ErrStage:       "Nie udało się przygotować plików instalacyjnych:\r\n%s",
	AskDelete:      "Instalator został zakończony.\r\n\r\nUsunąć tymczasowe pliki instalacyjne (%s)?\r\n\r\nWybierz Tak tylko wtedy, gdy instalacja zakończyła się całkowicie.",
	KeepTemp:       "Pliki tymczasowe pozostają w:\r\n%s",
	CloseRunning:   "Instalator może nadal działać. Jeśli zamkniesz program teraz, pliki tymczasowe pozostaną w:\r\n%s\r\n\r\nZamknąć mimo to?",
	PickFolder:     "Wybierz folder z zawartością płyty %d",
	ChooseMode:     "Jak chcesz zainstalować?\r\n\r\nTAK - Bezpośrednio z płyt (zalecane). Instalator poprosi o następną płytę, gdy będzie jej potrzebować.\r\n\r\nNIE - Najpierw skopiuj wszystko na dysk twardy, a potem instaluj bez zmiany płyt. Wymaga ok. %s wolnego miejsca (dostępne: %s).\r\n\r\nANULUJ - Wróć.",
	NativeInfo:     "Ta edycja składa się z %d płyt.\r\n\r\nInstalator poprosi o płytę 2 zaraz na początku. Włóż ją, potwierdź przyciskiem OK i postępuj zgodnie z instrukcjami. Podczas instalacji instalator może ponownie poprosić o płyty.\r\n\r\nKontynuować?",
	NoCover:        "BRAK OKŁADKI",
	DiscsFmt:       "Płyty: %d",
}

var fr = Strings{
	Install:        "INSTALLER LE JEU",
	Extras:         "BONUS",
	NoExtras:       "Aucun bonus n'est inclus dans cette édition.",
	Cancel:         "Annuler",
	Browse:         "Choisir un dossier...",
	SubTitle:       "INSTALLATEUR HORS LIGNE GOG",
	Copying:        "Copie des fichiers d'installation...",
	InsertDisc:     "Veuillez insérer le disque %d sur %d",
	WaitHint:       "Le disque est détecté automatiquement. Si vous utilisez des dossiers ou des images montées, utilisez « Choisir un dossier... ».",
	Running:        "L'installation est en cours...",
	RunningHint:    "Gardez cette fenêtre ouverte jusqu'à la fin de l'installation. Vous pourrez ensuite supprimer les fichiers temporaires.",
	ErrNoINI:       "GOGAutoLauncher.ini est introuvable à côté du lanceur.",
	ErrNoInstaller: "Aucun installateur n'est défini dans GOGAutoLauncher.ini (entrée Installer= dans la section [Game]).",
	ErrNotFound:    "Fichier introuvable :\r\n%s",
	ErrOpen:        "Impossible d'ouvrir le fichier :\r\n%s",
	ErrStart:       "Impossible de démarrer l'installateur :\r\n%s",
	ErrNoSpace:     "Espace libre insuffisant pour les fichiers d'installation temporaires.\r\n\r\nNécessaire : %s\r\nDisponible : %s\r\n\r\nChoisir un autre dossier sur un lecteur disposant de suffisamment d'espace ?",
	ErrNotDisc:     "Ce dossier ne contient pas le disque %d de cette édition.",
	ErrStage:       "Les fichiers d'installation n'ont pas pu être préparés :\r\n%s",
	AskDelete:      "L'installation est terminée.\r\n\r\nSupprimer les fichiers d'installation temporaires (%s) ?\r\n\r\nChoisissez Oui uniquement si l'installation est entièrement terminée.",
	KeepTemp:       "Les fichiers temporaires restent dans :\r\n%s",
	CloseRunning:   "L'installation est peut-être encore en cours. Si vous fermez le lanceur maintenant, les fichiers temporaires restent dans :\r\n%s\r\n\r\nFermer quand même ?",
	PickFolder:     "Choisissez le dossier contenant le disque %d",
	ChooseMode:     "Comment souhaitez-vous installer ?\r\n\r\nOUI - Directement depuis les disques (recommandé). L'installateur demande le disque suivant quand il en a besoin.\r\n\r\nNON - Tout copier d'abord sur le disque dur, puis installer sans changer de disque. Nécessite environ %s d'espace libre (disponible : %s).\r\n\r\nANNULER - Retour.",
	NativeInfo:     "Cette édition comprend %d disques.\r\n\r\nL'installateur demandera le disque 2 dès le début. Insérez-le, confirmez avec OK et suivez les instructions. Pendant l'installation, il peut redemander les disques.\r\n\r\nContinuer ?",
	NoCover:        "PAS DE JAQUETTE",
	DiscsFmt:       "%d disques",
}

var es = Strings{
	Install:        "INSTALAR JUEGO",
	Extras:         "EXTRAS",
	NoExtras:       "Esta edición no incluye extras.",
	Cancel:         "Cancelar",
	Browse:         "Elegir carpeta...",
	SubTitle:       "INSTALADOR SIN CONEXIÓN DE GOG",
	Copying:        "Copiando archivos de instalación...",
	InsertDisc:     "Inserta el disco %d de %d",
	WaitHint:       "El disco se detecta automáticamente. Si trabajas con carpetas o imágenes montadas, usa «Elegir carpeta...».",
	Running:        "La instalación está en curso...",
	RunningHint:    "Mantén esta ventana abierta hasta que termine la instalación. Después podrás eliminar los archivos temporales.",
	ErrNoINI:       "No se encontró GOGAutoLauncher.ini junto al lanzador.",
	ErrNoInstaller: "No hay ningún instalador definido en GOGAutoLauncher.ini (entrada Installer= en la sección [Game]).",
	ErrNotFound:    "Archivo no encontrado:\r\n%s",
	ErrOpen:        "No se pudo abrir el archivo:\r\n%s",
	ErrStart:       "No se pudo iniciar el instalador:\r\n%s",
	ErrNoSpace:     "No hay suficiente espacio libre para los archivos de instalación temporales.\r\n\r\nNecesario: %s\r\nDisponible: %s\r\n\r\n¿Elegir otra carpeta en una unidad con suficiente espacio?",
	ErrNotDisc:     "Esta carpeta no contiene el disco %d de esta edición.",
	ErrStage:       "No se pudieron preparar los archivos de instalación:\r\n%s",
	AskDelete:      "La instalación ha terminado.\r\n\r\n¿Eliminar los archivos de instalación temporales (%s)?\r\n\r\nElige Sí solo si la instalación ha finalizado por completo.",
	KeepTemp:       "Los archivos temporales permanecen en:\r\n%s",
	CloseRunning:   "Es posible que la instalación siga en curso. Si cierras el lanzador ahora, los archivos temporales permanecen en:\r\n%s\r\n\r\n¿Cerrar de todos modos?",
	PickFolder:     "Elige la carpeta con el contenido del disco %d",
	ChooseMode:     "¿Cómo quieres instalar?\r\n\r\nSÍ - Directamente desde los discos (recomendado). El instalador pide el siguiente disco cuando lo necesita.\r\n\r\nNO - Copiar primero todo al disco duro e instalar después sin cambiar de disco. Necesita unos %s de espacio libre (disponible: %s).\r\n\r\nCANCELAR - Volver.",
	NativeInfo:     "Esta edición consta de %d discos.\r\n\r\nEl instalador pedirá el disco 2 nada más empezar. Insértalo, confirma con Aceptar y sigue las instrucciones. Durante la instalación puede volver a pedir los discos.\r\n\r\n¿Continuar?",
	NoCover:        "SIN PORTADA",
	DiscsFmt:       "%d discos",
}

var it = Strings{
	Install:        "INSTALLA IL GIOCO",
	Extras:         "EXTRA",
	NoExtras:       "Questa edizione non include extra.",
	Cancel:         "Annulla",
	Browse:         "Scegli cartella...",
	SubTitle:       "INSTALLER OFFLINE GOG",
	Copying:        "Copia dei file di installazione...",
	InsertDisc:     "Inserisci il disco %d di %d",
	WaitHint:       "Il disco viene rilevato automaticamente. Se usi cartelle o immagini montate, usa «Scegli cartella...».",
	Running:        "Installazione in corso...",
	RunningHint:    "Tieni aperta questa finestra fino al termine dell'installazione. Poi potrai eliminare i file temporanei.",
	ErrNoINI:       "GOGAutoLauncher.ini non è stato trovato accanto al launcher.",
	ErrNoInstaller: "In GOGAutoLauncher.ini non è definito alcun installer (voce Installer= nella sezione [Game]).",
	ErrNotFound:    "File non trovato:\r\n%s",
	ErrOpen:        "Impossibile aprire il file:\r\n%s",
	ErrStart:       "Impossibile avviare l'installer:\r\n%s",
	ErrNoSpace:     "Spazio libero insufficiente per i file di installazione temporanei.\r\n\r\nNecessario: %s\r\nDisponibile: %s\r\n\r\nScegliere un'altra cartella su un'unità con spazio sufficiente?",
	ErrNotDisc:     "Questa cartella non contiene il disco %d di questa edizione.",
	ErrStage:       "Impossibile preparare i file di installazione:\r\n%s",
	AskDelete:      "L'installazione è terminata.\r\n\r\nEliminare i file di installazione temporanei (%s)?\r\n\r\nScegli Sì solo se l'installazione è stata completata.",
	KeepTemp:       "I file temporanei restano in:\r\n%s",
	CloseRunning:   "L'installazione potrebbe essere ancora in corso. Se chiudi il launcher ora, i file temporanei restano in:\r\n%s\r\n\r\nChiudere comunque?",
	PickFolder:     "Scegli la cartella con il contenuto del disco %d",
	ChooseMode:     "Come vuoi installare?\r\n\r\nSÌ - Direttamente dai dischi (consigliato). L'installer chiede il disco successivo quando serve.\r\n\r\nNO - Copia prima tutto sul disco rigido, poi installa senza cambiare disco. Richiede circa %s di spazio libero (disponibile: %s).\r\n\r\nANNULLA - Indietro.",
	NativeInfo:     "Questa edizione è composta da %d dischi.\r\n\r\nL'installer chiederà il disco 2 subito all'inizio. Inseriscilo, conferma con OK e segui le istruzioni. Durante l'installazione potrebbe chiedere di nuovo i dischi.\r\n\r\nContinuare?",
	NoCover:        "NESSUNA COPERTINA",
	DiscsFmt:       "%d dischi",
}

// pickLanguage follows the Windows display language (primary language id); an INI override is optional.
func pickLanguage(override string, langID uint16) *Strings {
	code := override
	if code == "" {
		switch langID & 0x3FF {
		case 0x07:
			code = "de"
		case 0x15:
			code = "pl"
		case 0x0C:
			code = "fr"
		case 0x0A:
			code = "es"
		case 0x10:
			code = "it"
		}
	}
	switch code {
	case "de":
		return &de
	case "pl":
		return &pl
	case "fr":
		return &fr
	case "es":
		return &es
	case "it":
		return &it
	}
	return &en
}
