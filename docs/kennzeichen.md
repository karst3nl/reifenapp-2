# Kennzeichen-Auswahlliste

Die lokal eingebettete Liste `internal/models/plate_districts.txt` enthält
731 Unterscheidungszeichen aus den alphabetischen Tabellen der
[Liste der Kfz-Kennzeichen in Deutschland](https://de.wikipedia.org/wiki/Liste_der_Kfz-Kennzeichen_in_Deutschland),
abgerufen am 10. November 2026. Übernommen wurden ausschließlich die kurzen
Kennungen, keine Beschreibungstexte. Kein Netzwerkzugriff zur Laufzeit.
Die Liste ist ein gepflegter Ausgangsstand, keine Garantie für tagesaktuelle
Zuteilbarkeit eines Kennzeichens; behördliche Sonderkennzeichen sind nicht Ziel
des Werkstattformulars. Änderungen erfolgen gemeinsam an der lokalen Liste.

Mainz und Umgebung stehen bewusst zuerst: MZ, BIN, AZ, WO, WI, GG, RÜD, KH,
MTK, DÜW, FT, LU, DA, HP, KL, KIB, SIM, EMS, F, OF. Danach alphabetisch.
Das ist eine regionale Priorisierung, keine Häufigkeitsstatistik.

Das Formular kombiniert Dropdown, 1–2 Buchstaben und 1–4 Ziffern, optional mit
E-/H-Suffix, zum bisherigen API-Feld `plate`, z. B. `MZ-AB 123`.
Die API prüft Format und Kürzel ebenfalls. Bereits gespeicherte Kennzeichen
(einschließlich fiktiver TEST-Beispiele) bleiben unverändert auffindbar.
