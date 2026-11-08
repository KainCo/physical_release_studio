// SPDX-License-Identifier: GPL-3.0-or-later
package main

import "fmt"

// Oberflächensprache des Launchers: ausschließlich nach der Windows-Anzeigesprache.
// de/en sind geprüft; pl/fr/es/it wurden maschinell übersetzt und nicht von Muttersprachlern geprüft.

var lang = "en"

var tr = map[string]map[string]string{
	"en": {
		"install": "Install game", "play": "Play game", "reinstall": "Reinstall / repair", "uninstall": "Uninstall", "exit": "Exit",
		"installed": "Installed in: %s", "notInstalled": "The game is not installed yet.",
		"installTo": "Install into this folder:", "browse": "Browse...", "pick": "Choose the installation folder",
		"scDesktop": "Create desktop shortcut", "scStart": "Create Start menu shortcuts", "redist": "Also install required components (Visual C++, DirectX, .NET)",
		"space": "Required: %s   Free: %s", "noSpace": "There is not enough free space in the selected location.",
		"doInstall": "Install", "back": "Back", "cancel": "Cancel", "close": "Close",
		"copying": "Copying: %s", "discPrompt": "Please insert disc %d of %d and click OK.\n\nIf the disc is in a folder or on a USB drive, make sure it is connected.",
		"installing": "Installing...", "done": "The installation is complete.", "cancelled": "The installation was cancelled.",
		"errGeneric": "An error occurred:\n%s", "retry": "Problem with %s:\n%s\n\nTry again?",
		"corrupt": "file damaged or incomplete", "folderNotEmpty": "The selected folder is not empty. Install here anyway?",
		"confirmUn": "Do you really want to uninstall %s?", "extra": "The folder contains additional files (for example save games).\n\nDelete them as well?",
		"unDone": "%s was uninstalled.", "redistRun": "Installing component: %s", "redistFail": "Some components could not be installed. The game may ask you to install them manually.",
		"cfgErr": "The file launcher.ini is missing or damaged:\n%s", "badPath": "Please enter a valid folder.",
		"unNoLog":   "The uninstall information was not found. Nothing was removed.",
		"steamHint": "This game needs Steam. The files are copied into a Steam library. You need the Steam client and an account that owns the game. On the first start Steam may ask for the folder or verify the files (choose the drive of the installation).", "launchFail": "The game could not be started.",
		"comps": "Install required components (Visual C++, DirectX ...)", "compConfirm": "The included components (for example Visual C++ or DirectX) will be installed now. Windows may ask for administrator rights. Continue?",
		"compRunning": "Installing components...", "compDone": "The components were installed (or were already present).", "unSteam": "This is a Steam game. Steam may still list it as installed. Please also uninstall it in Steam.",
	},
	"de": {
		"install": "Spiel installieren", "play": "Spiel starten", "reinstall": "Neu installieren / reparieren", "uninstall": "Deinstallieren", "exit": "Beenden",
		"installed": "Installiert in: %s", "notInstalled": "Das Spiel ist noch nicht installiert.",
		"installTo": "In diesen Ordner installieren:", "browse": "Durchsuchen...", "pick": "Installationsordner wählen",
		"scDesktop": "Desktop-Verknüpfung erstellen", "scStart": "Startmenü-Verknüpfungen erstellen", "redist": "Benötigte Komponenten mitinstallieren (Visual C++, DirectX, .NET)",
		"space": "Benötigt: %s   Frei: %s", "noSpace": "Am gewählten Ort ist nicht genug Speicherplatz frei.",
		"doInstall": "Installieren", "back": "Zurück", "cancel": "Abbrechen", "close": "Schließen",
		"copying": "Kopiere: %s", "discPrompt": "Bitte legen Sie Disc %d von %d ein und klicken Sie auf OK.\n\nLiegt die Disc in einem Ordner oder auf einem USB-Stick, stellen Sie sicher, dass er angeschlossen ist.",
		"installing": "Installation läuft...", "done": "Die Installation ist abgeschlossen.", "cancelled": "Die Installation wurde abgebrochen.",
		"errGeneric": "Es ist ein Fehler aufgetreten:\n%s", "retry": "Problem bei %s:\n%s\n\nErneut versuchen?",
		"corrupt": "Datei beschädigt oder unvollständig", "folderNotEmpty": "Der gewählte Ordner ist nicht leer. Trotzdem hier installieren?",
		"confirmUn": "Möchten Sie %s wirklich deinstallieren?", "extra": "Der Ordner enthält weitere Dateien (z. B. Spielstände).\n\nAuch diese löschen?",
		"unDone": "%s wurde deinstalliert.", "redistRun": "Installiere Komponente: %s", "redistFail": "Einige Komponenten konnten nicht installiert werden. Das Spiel fordert sie ggf. manuell an.",
		"cfgErr": "Die Datei launcher.ini fehlt oder ist fehlerhaft:\n%s", "badPath": "Bitte geben Sie einen gültigen Ordner an.",
		"unNoLog":   "Die Deinstallationsdaten wurden nicht gefunden. Es wurde nichts entfernt.",
		"steamHint": "Dieses Spiel benötigt Steam. Die Dateien werden in eine Steam-Bibliothek kopiert. Sie benötigen den Steam-Client und ein Konto, dem das Spiel gehört. Beim ersten Start fragt Steam ggf. nach dem Ordner oder prüft die Dateien (wählen Sie das Laufwerk der Installation).", "launchFail": "Das Spiel konnte nicht gestartet werden.",
		"comps": "Benötigte Komponenten installieren (Visual C++, DirectX ...)", "compConfirm": "Die mitgelieferten Komponenten (z. B. Visual C++ oder DirectX) werden jetzt installiert. Windows fragt dabei ggf. nach Administratorrechten. Fortfahren?",
		"compRunning": "Installiere Komponenten...", "compDone": "Die Komponenten wurden installiert (oder waren schon vorhanden).", "unSteam": "Dies ist ein Steam-Spiel. Steam führt es möglicherweise weiter als installiert. Bitte deinstallieren Sie es zusätzlich in Steam.",
	},
	"pl": {
		"install": "Zainstaluj grę", "play": "Uruchom grę", "reinstall": "Zainstaluj ponownie / napraw", "uninstall": "Odinstaluj", "exit": "Zakończ",
		"installed": "Zainstalowano w: %s", "notInstalled": "Gra nie jest jeszcze zainstalowana.",
		"installTo": "Zainstaluj w tym folderze:", "browse": "Przeglądaj...", "pick": "Wybierz folder instalacji",
		"scDesktop": "Utwórz skrót na pulpicie", "scStart": "Utwórz skróty w menu Start", "redist": "Zainstaluj także wymagane składniki (Visual C++, DirectX, .NET)",
		"space": "Wymagane: %s   Wolne: %s", "noSpace": "W wybranej lokalizacji brakuje wolnego miejsca.",
		"doInstall": "Zainstaluj", "back": "Wstecz", "cancel": "Anuluj", "close": "Zamknij",
		"copying": "Kopiowanie: %s", "discPrompt": "Włóż dysk %d z %d i kliknij OK.\n\nJeśli dysk znajduje się w folderze lub na nośniku USB, upewnij się, że jest podłączony.",
		"installing": "Trwa instalacja...", "done": "Instalacja zakończona.", "cancelled": "Instalacja została anulowana.",
		"errGeneric": "Wystąpił błąd:\n%s", "retry": "Problem z %s:\n%s\n\nSpróbować ponownie?",
		"corrupt": "plik jest uszkodzony lub niekompletny", "folderNotEmpty": "Wybrany folder nie jest pusty. Zainstalować mimo to?",
		"confirmUn": "Czy na pewno odinstalować %s?", "extra": "Folder zawiera dodatkowe pliki (np. zapisy gry).\n\nUsunąć je również?",
		"unDone": "Odinstalowano %s.", "redistRun": "Instalowanie składnika: %s", "redistFail": "Nie udało się zainstalować niektórych składników. Gra może poprosić o ich ręczną instalację.",
		"cfgErr": "Plik launcher.ini jest niedostępny lub uszkodzony:\n%s", "badPath": "Podaj prawidłowy folder.",
		"unNoLog":   "Nie znaleziono danych dezinstalacji. Nic nie usunięto.",
		"steamHint": "Ta gra wymaga Steam. Pliki zostaną skopiowane do biblioteki Steam. Potrzebny jest klient Steam i konto, które posiada tę grę. Przy pierwszym uruchomieniu Steam może zapytać o folder lub sprawdzić pliki (wybierz dysk instalacji).", "launchFail": "Nie udało się uruchomić gry.",
		"comps": "Zainstaluj wymagane składniki (Visual C++, DirectX ...)", "compConfirm": "Dołączone składniki (np. Visual C++ lub DirectX) zostaną teraz zainstalowane. System Windows może poprosić o uprawnienia administratora. Kontynuować?",
		"compRunning": "Instalowanie składników...", "compDone": "Składniki zostały zainstalowane (lub były już obecne).", "unSteam": "To gra Steam. Steam może nadal wyświetlać ją jako zainstalowaną. Odinstaluj ją dodatkowo w Steam.",
	},
	"fr": {
		"install": "Installer le jeu", "play": "Lancer le jeu", "reinstall": "Réinstaller / réparer", "uninstall": "Désinstaller", "exit": "Quitter",
		"installed": "Installé dans : %s", "notInstalled": "Le jeu n'est pas encore installé.",
		"installTo": "Installer dans ce dossier :", "browse": "Parcourir...", "pick": "Choisir le dossier d'installation",
		"scDesktop": "Créer un raccourci sur le bureau", "scStart": "Créer des raccourcis dans le menu Démarrer", "redist": "Installer aussi les composants requis (Visual C++, DirectX, .NET)",
		"space": "Requis : %s   Libre : %s", "noSpace": "L'espace libre est insuffisant à l'emplacement choisi.",
		"doInstall": "Installer", "back": "Retour", "cancel": "Annuler", "close": "Fermer",
		"copying": "Copie : %s", "discPrompt": "Veuillez insérer le disque %d sur %d, puis cliquez sur OK.\n\nSi le disque se trouve dans un dossier ou sur une clé USB, vérifiez qu'il est connecté.",
		"installing": "Installation en cours...", "done": "L'installation est terminée.", "cancelled": "L'installation a été annulée.",
		"errGeneric": "Une erreur s'est produite :\n%s", "retry": "Problème avec %s :\n%s\n\nRéessayer ?",
		"corrupt": "fichier endommagé ou incomplet", "folderNotEmpty": "Le dossier choisi n'est pas vide. Installer quand même ici ?",
		"confirmUn": "Voulez-vous vraiment désinstaller %s ?", "extra": "Le dossier contient d'autres fichiers (par exemple des sauvegardes).\n\nLes supprimer aussi ?",
		"unDone": "%s a été désinstallé.", "redistRun": "Installation du composant : %s", "redistFail": "Certains composants n'ont pas pu être installés. Le jeu peut vous demander de les installer manuellement.",
		"cfgErr": "Le fichier launcher.ini est introuvable ou endommagé :\n%s", "badPath": "Veuillez indiquer un dossier valide.",
		"unNoLog":   "Les données de désinstallation sont introuvables. Rien n'a été supprimé.",
		"steamHint": "Ce jeu nécessite Steam. Les fichiers sont copiés dans une bibliothèque Steam. Il faut le client Steam et un compte qui possède le jeu. Au premier lancement, Steam peut demander le dossier ou vérifier les fichiers (choisissez le lecteur de l'installation).", "launchFail": "Le jeu n'a pas pu être lancé.",
		"comps": "Installer les composants requis (Visual C++, DirectX ...)", "compConfirm": "Les composants fournis (par exemple Visual C++ ou DirectX) vont être installés. Windows peut demander des droits d'administrateur. Continuer ?",
		"compRunning": "Installation des composants...", "compDone": "Les composants ont été installés (ou étaient déjà présents).", "unSteam": "Ceci est un jeu Steam. Steam peut encore l'afficher comme installé. Veuillez aussi le désinstaller dans Steam.",
	},
	"es": {
		"install": "Instalar el juego", "play": "Jugar", "reinstall": "Reinstalar / reparar", "uninstall": "Desinstalar", "exit": "Salir",
		"installed": "Instalado en: %s", "notInstalled": "El juego aún no está instalado.",
		"installTo": "Instalar en esta carpeta:", "browse": "Examinar...", "pick": "Elegir la carpeta de instalación",
		"scDesktop": "Crear acceso directo en el escritorio", "scStart": "Crear accesos directos en el menú Inicio", "redist": "Instalar también los componentes necesarios (Visual C++, DirectX, .NET)",
		"space": "Necesario: %s   Libre: %s", "noSpace": "No hay espacio libre suficiente en la ubicación elegida.",
		"doInstall": "Instalar", "back": "Atrás", "cancel": "Cancelar", "close": "Cerrar",
		"copying": "Copiando: %s", "discPrompt": "Inserte el disco %d de %d y haga clic en Aceptar.\n\nSi el disco está en una carpeta o en una unidad USB, asegúrese de que esté conectado.",
		"installing": "Instalando...", "done": "La instalación ha finalizado.", "cancelled": "La instalación se canceló.",
		"errGeneric": "Se ha producido un error:\n%s", "retry": "Problema con %s:\n%s\n\n¿Reintentar?",
		"corrupt": "archivo dañado o incompleto", "folderNotEmpty": "La carpeta elegida no está vacía. ¿Instalar aquí de todos modos?",
		"confirmUn": "¿Seguro que desea desinstalar %s?", "extra": "La carpeta contiene archivos adicionales (por ejemplo, partidas guardadas).\n\n¿Eliminarlos también?",
		"unDone": "%s se ha desinstalado.", "redistRun": "Instalando componente: %s", "redistFail": "No se pudieron instalar algunos componentes. Es posible que el juego le pida instalarlos manualmente.",
		"cfgErr": "El archivo launcher.ini falta o está dañado:\n%s", "badPath": "Indique una carpeta válida.",
		"unNoLog":   "No se encontraron los datos de desinstalación. No se eliminó nada.",
		"steamHint": "Este juego requiere Steam. Los archivos se copian a una biblioteca de Steam. Necesita el cliente de Steam y una cuenta que posea el juego. En el primer inicio, Steam puede pedir la carpeta o verificar los archivos (elija la unidad de la instalación).", "launchFail": "No se pudo iniciar el juego.",
		"comps": "Instalar los componentes necesarios (Visual C++, DirectX ...)", "compConfirm": "Se instalarán ahora los componentes incluidos (por ejemplo Visual C++ o DirectX). Windows puede solicitar derechos de administrador. ¿Continuar?",
		"compRunning": "Instalando componentes...", "compDone": "Los componentes se instalaron (o ya estaban presentes).", "unSteam": "Este es un juego de Steam. Steam puede seguir mostrándolo como instalado. Desinstálelo también en Steam.",
	},
	"it": {
		"install": "Installa il gioco", "play": "Avvia il gioco", "reinstall": "Reinstalla / ripara", "uninstall": "Disinstalla", "exit": "Esci",
		"installed": "Installato in: %s", "notInstalled": "Il gioco non è ancora installato.",
		"installTo": "Installa in questa cartella:", "browse": "Sfoglia...", "pick": "Scegli la cartella di installazione",
		"scDesktop": "Crea collegamento sul desktop", "scStart": "Crea collegamenti nel menu Start", "redist": "Installa anche i componenti necessari (Visual C++, DirectX, .NET)",
		"space": "Necessario: %s   Libero: %s", "noSpace": "Nella posizione scelta non c'è spazio libero sufficiente.",
		"doInstall": "Installa", "back": "Indietro", "cancel": "Annulla", "close": "Chiudi",
		"copying": "Copia in corso: %s", "discPrompt": "Inserire il disco %d di %d e fare clic su OK.\n\nSe il disco si trova in una cartella o su una chiavetta USB, assicurarsi che sia collegato.",
		"installing": "Installazione in corso...", "done": "L'installazione è completata.", "cancelled": "L'installazione è stata annullata.",
		"errGeneric": "Si è verificato un errore:\n%s", "retry": "Problema con %s:\n%s\n\nRiprovare?",
		"corrupt": "file danneggiato o incompleto", "folderNotEmpty": "La cartella scelta non è vuota. Installare comunque qui?",
		"confirmUn": "Disinstallare davvero %s?", "extra": "La cartella contiene altri file (ad esempio i salvataggi).\n\nEliminarli?",
		"unDone": "%s è stato disinstallato.", "redistRun": "Installazione del componente: %s", "redistFail": "Non è stato possibile installare alcuni componenti. Il gioco potrebbe chiedere di installarli manualmente.",
		"cfgErr": "Il file launcher.ini manca o è danneggiato:\n%s", "badPath": "Indicare una cartella valida.",
		"unNoLog":   "I dati di disinstallazione non sono stati trovati. Non è stato rimosso nulla.",
		"steamHint": "Questo gioco richiede Steam. I file vengono copiati in una libreria di Steam. Servono il client Steam e un account che possieda il gioco. Al primo avvio Steam può chiedere la cartella o verificare i file (scegliere l'unità dell'installazione).", "launchFail": "Impossibile avviare il gioco.",
		"comps": "Installa i componenti necessari (Visual C++, DirectX ...)", "compConfirm": "I componenti inclusi (ad es. Visual C++ o DirectX) verranno installati ora. Windows potrebbe richiedere i diritti di amministratore. Continuare?",
		"compRunning": "Installazione dei componenti...", "compDone": "I componenti sono stati installati (o erano già presenti).", "unSteam": "Questo è un gioco Steam. Steam potrebbe ancora mostrarlo come installato. Disinstallarlo anche in Steam.",
	},
}

// setLang wählt die Sprache anhand eines Codes ("de", "pl", ...); unbekannt = Englisch.
func setLang(code string) {
	if _, ok := tr[code]; ok {
		lang = code
	} else {
		lang = "en"
	}
}

// T liefert den Text zu einem Schlüssel (mit Rückfall auf Englisch).
func T(key string, args ...any) string {
	s, ok := tr[lang][key]
	if !ok {
		s = tr["en"][key]
	}
	if len(args) == 0 {
		return s
	}
	return fmt.Sprintf(s, args...)
}

// humanSize formatiert Bytes.
func humanSize(n int64) string {
	if n < 0 {
		return "?"
	}
	const u = 1024
	if n < u {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(u), 0
	for m := n / u; m >= u; m /= u {
		div *= u
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTP"[exp])
}
