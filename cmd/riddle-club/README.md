# Riddle Club
## Mosaik Riddle
Ein Mosaik-Spiel, basierend auf einem Schachbrett als Untergrund. Darauf liegen Fliesen. Diese Fliesen müssen kombiniert werden. Manche Fliesen gleichen sich in ein oder mehreren Punkten (=Übereinstimmungen). Eine passende Fliese muss durch Klicken ausgewählt werden, die gleichen "Punkte" (=Muster) verschwinden anschließend vom Spielfeld. Die Logik ist so gestaltet, dass am Ende nichts mehr übrig bleibt. Es dürfen während des Spielverlaufs nur 3 Fehler passieren. Fehler: Man klickt auf ein Motiv das keine Übereinstimmungen mit dem aktuellen Motiv hat. Wenn ein Kästchen leer ist, gilt es als "Joker" und man darf sich das nächste Feld selbst aussuchen.
Im besten Fall kann man am Beginn des Spiels das gewünschte "Theme" aussuchen. 
Memo: Katzentheme erstellen
Das Schachbrett ist eine in CSS-modifizierte Tabelle. (ca 6x6?)
Die Motive bestehen aus SVG-Grafiken, die übereinandergelegt werden.
Das Spielfeld wird im Backend generiert. 
Die Spiellogik wird im Frontend gemanaged. 
evtl. Alpine benutzen? kein React!