#!/usr/bin/env python3
"""Integrate the ByFTP setup-language picker and normalize remaining setup paths."""

from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def replace_required(text: str, old: str, new: str, label: str) -> str:
    if old not in text:
        raise SystemExit(f"missing installer integration marker: {label}")
    return text.replace(old, new, 1)


installer = ROOT / "cmd" / "installer" / "main.go"
text = installer.read_text(encoding="utf-8")

text = replace_required(text, "func register(appPath, uninstallPath, dir string) error {", "func register(appPath, uninstallPath, dir, language string) error {", "register signature")
text = replace_required(text, '\t\t{uninstallKey, "DisplayVersion", version},\n', '\t\t{uninstallKey, "DisplayVersion", version},\n\t\t{uninstallKey, "InstallLanguage", language},\n', "InstallLanguage registry entry")
text = replace_required(text, 'func installerTitle() string {\n\treturn brand.ProductName + " — Instalacija"\n}', 'func installerTitle() string {\n\treturn brand.ProductName + " Setup"\n}', "English installer title")

old_content = '''\tcontent := brand.Company + "\\n" + brand.Website + "\\n" + brand.Support + "\\n\\n" +\n\t\t"ByFTP će se instalirati za vaš Windows korisnički račun i bit će dostupan iz izbornika Start."\n\n\tif !platform.ConfirmDialog(\n\t\tinstallerTitle(),\n\t\t"Instalirati "+brand.ProductFull+" "+version+"?",\n\t\tcontent,\n\t) {\n\t\treturn 0\n\t}\n'''
new_content = '''\tinstallLanguage, ok := selectInstallerLanguage()\n\tif !ok {\n\t\treturn 0\n\t}\n\n\tcontent := brand.Website + "\\n" + brand.Support + "\\n\\n" +\n\t\t"ByFTP will be installed for your Windows user account and will be available from the Start menu."\n\n\tif !platform.ConfirmDialog(\n\t\tinstallerTitle(),\n\t\t"Install "+brand.ProductFull+" "+version+"?",\n\t\tcontent,\n\t) {\n\t\treturn 0\n\t}\n'''
text = replace_required(text, old_content, new_content, "language selection")
text = replace_required(text, "if err := register(appPath, uninstallPath, dir); err != nil {", "if err := register(appPath, uninstallPath, dir, installLanguage); err != nil {", "register call")
text = replace_required(text, '''\ttransactionCommitted = true\n\n\tshortcutWarning := ""\n''', '''\ttransactionCommitted = true\n\n\tlanguageWarning := ""\n\tif err := persistInstallerLanguage(installLanguage); err != nil {\n\t\tlanguageWarning = "\\n\\nThe selected language could not be saved. ByFTP will start in English; you can change the language in Settings."\n\t}\n\n\tshortcutWarning := ""\n''', "persist selected language")
text = replace_required(text, '"ByFTP je spreman za korištenje."+shortcutWarning+"\\n\\nPokrenuti ByFTP sada?"', '"ByFTP is ready to use."+languageWarning+shortcutWarning+"\\n\\nLaunch ByFTP now?"', "success message")

translations = {
    "komprimirani instalacijski payload nije dostupan": "compressed installer payload is unavailable",
    "instalacijski payload je prazan": "installer payload is empty",
    "instalacijski payload nije ispravan ZIP": "installer payload is not a valid ZIP archive",
    "instalacijski payload sadrži dupliciranu datoteku": "installer payload contains a duplicate file",
    "instalacijski payload sadrži duplicirani manifest": "installer payload contains a duplicate manifest",
    "instalacijski payload sadrži neočekivanu datoteku": "installer payload contains an unexpected file",
    "instalacijski payload ne sadrži sve obavezne datoteke": "installer payload is missing required files",
    "instalacijski payload ima neispravnu veličinu": "installer payload has an invalid size",
    "instalacijsku datoteku nije moguće otvoriti": "installer file could not be opened",
    "instalacijski payload je oštećen ili nepotpun": "installer payload is damaged or incomplete",
    "instalacijski manifest nije ispravan": "installer manifest is invalid",
    "instalacijski manifest sadrži višak podataka": "installer manifest contains trailing data",
    "instalacijski manifest nije podržan": "installer manifest is unsupported",
    "instalacijski manifest ne odgovara paketu": "installer manifest does not match the package",
    "instalacijski manifest sadrži neispravan SHA-256": "installer manifest contains an invalid SHA-256 digest",
    "provjera integriteta instalacijskog paketa nije uspjela": "installer package integrity verification failed",
    "instalacijski manifest nije potpun": "installer manifest is incomplete",
    "provjera veličine instalacijske datoteke nije uspjela": "installer file size verification failed",
    "provjera integriteta instalacijske datoteke nije uspjela": "installer file integrity verification failed",
    "instalacijska transakcija ne odgovara ciljnoj datoteci": "installer transaction does not match the target file",
    "Windows registracija nije uspjela": "Windows registration failed",
    "Radnja nije dovršena": "Setup did not finish",
    "Instalacija nije dovršena. Ponovno pokrenite računalo i pokušajte ponovno.": "Setup did not finish. Restart Windows and try again.",
    "Instalacijski paket nije ispravan": "Installer package is invalid",
    "Preuzmite novu kopiju instalacijskog paketa i pokušajte ponovno.": "Download a fresh copy of the installer and try again.",
    "Instalacija nije moguća": "Setup cannot continue",
    "Korisnička mapa sustava Windows nije dostupna.": "The Windows user folder is unavailable.",
    "Lokalna korisnička mapa sustava Windows nije dostupna.": "The Windows local application-data folder is unavailable.",
    "Instalacijska mapa nije sigurna": "The installation folder is not safe",
    "Uklonite preusmjeravanje ByFTP instalacijske mape i pokušajte ponovno.": "Remove any redirect from the ByFTP installation folder and try again.",
    "Instalacija nije pokrenuta": "Setup was not started",
    "Instalacijsku mapu nije moguće pripremiti. Provjerite slobodan prostor i dopuštenja pa pokušajte ponovno.": "The installation folder could not be prepared. Check free space and permissions, then try again.",
    "Nadogradnja nije pokrenuta": "Upgrade was not started",
    "Postojeću instalaciju nije moguće pripremiti za nadogradnju. Zatvorite ByFTP i pokušajte ponovno.": "The existing installation could not be prepared for upgrade. Close ByFTP and try again.",
    "Postojeće postavke instalacije nije moguće sigurno pripremiti. Pokušajte ponovno.": "Existing installation settings could not be prepared safely. Try again.",
    " Prethodnu instalaciju nije bilo moguće potpuno vratiti; ponovno pokrenite Windows prije novog pokušaja.": " The previous installation could not be fully restored; restart Windows before trying again.",
    "ByFTP nije moguće instalirati": "ByFTP could not be installed",
    "Zatvorite pokrenuti ByFTP i pokušajte ponovno.": "Close ByFTP if it is running and try again.",
    "Instalacija nije dovršena": "Setup did not finish",
    "Potrebne datoteke nije moguće spremiti. Pokušajte ponovno.": "Required files could not be saved. Try again.",
    "Windows nije uspio dovršiti postavljanje aplikacije. Pokušajte ponovno.": "Windows could not finish registering the application. Try again.",
    "Prečac nije moguće izraditi. ByFTP možete pokrenuti iz instalacijske mape.": "A shortcut could not be created. You can start ByFTP from its installation folder.",
    "Instalacija je uspješno dovršena": "Setup completed successfully",
    "ByFTP je instaliran": "ByFTP is installed",
    "Aplikaciju nije moguće automatski pokrenuti. Pokrenite ByFTP iz izbornika Start.": "ByFTP could not be launched automatically. Start it from the Windows Start menu.",
}
for old, new in translations.items():
    text = text.replace(old, new)
installer.write_text(text, encoding="utf-8", newline="\n")

snapshot = ROOT / "cmd" / "installer" / "registry_snapshot.go"
text = snapshot.read_text(encoding="utf-8")
text = replace_required(text, '\t{uninstallKey, "DisplayVersion"},\n', '\t{uninstallKey, "DisplayVersion"},\n\t{uninstallKey, "InstallLanguage"},\n', "registry snapshot language")
snapshot.write_text(text, encoding="utf-8", newline="\n")

engine = ROOT / "internal" / "api" / "engine.go"
text = engine.read_text(encoding="utf-8")
text = text.replace('errors.New("interna greška aplikacije")', 'errors.New("internal application error")')
text = text.replace('e.transfers.DisableAndCancel(ctx, "Otkazano prekidom veze")', 'e.transfers.DisableAndCancel(ctx, "Cancelled by disconnect")')
text = replace_required(text, 'return filepath.Join(base, brand.Company, brand.ProductName), nil', 'return filepath.Join(base, brand.ProductName), nil', "canonical data directory")
engine.write_text(text, encoding="utf-8", newline="\n")

uninstaller = ROOT / "cmd" / "uninstaller" / "main.go"
text = uninstaller.read_text(encoding="utf-8")
text = text.replace('return filepath.Join(base, brand.Company, brand.ProductName), nil', 'return filepath.Join(base, brand.ProductName), nil')
uninstaller_replacements = {
    "korisnička mapa nije dostupna": "user data folder is unavailable",
    "Radnja nije dovršena": "Action did not finish",
    "Uklanjanje nije dovršeno. Ponovno pokrenite računalo i pokušajte ponovno.": "Uninstall did not finish. Restart Windows and try again.",
    "Deinstalacija nije pokrenuta": "Uninstall was not started",
    "ByFTP trenutačno nije moguće ukloniti. Ponovno pokrenite računalo i pokušajte ponovno.": "ByFTP cannot be removed right now. Restart Windows and try again.",
    "Pokrenite uklanjanje ByFTP-a iz njegove instalirane lokacije ili iz Windows postavki aplikacija.": "Start the ByFTP uninstaller from its installed location or from Windows Installed apps.",
    "Ukloniti ": "Remove ",
    "Aplikacija će biti uklonjena. Nakon toga možete odlučiti želite li zadržati spremljene profile i postavke.": "The application will be removed. You can then choose whether to keep saved profiles and settings.",
    "Obrisati i spremljene profile i postavke?": "Delete saved profiles and settings too?",
    "Da = brišu se lokalni ByFTP profili i postavke s ovog računala.\\nNe = ostaju sačuvani za moguću ponovnu instalaciju.": "Yes = delete local ByFTP profiles and settings from this computer.\\nNo = keep them for a possible reinstall.",
    "Uklanjanje nije potpuno dovršeno": "Uninstall did not fully finish",
    "Neke datoteke ili Windows postavke nije bilo moguće ukloniti. Ponovno pokrenite računalo pa ponovno pokrenite uklanjanje iz Windows postavki aplikacija.": "Some files or Windows settings could not be removed. Restart Windows, then run uninstall again from Installed apps.",
    "Korisnički profili i postavke ostali su sačuvani na ovom računalu.": "User profiles and settings were kept on this computer.",
    "Lokalni ByFTP profili i postavke su obrisani.": "Local ByFTP profiles and settings were deleted.",
    " Završno čišćenje dovršit će se nakon ponovnog pokretanja sustava Windows.": " Final cleanup will finish after Windows restarts.",
    "ByFTP je uklonjen": "ByFTP was removed",
}
for old, new in uninstaller_replacements.items():
    text = text.replace(old, new)
uninstaller.write_text(text, encoding="utf-8", newline="\n")

print("INSTALLER_INTEGRATION=PASS")
