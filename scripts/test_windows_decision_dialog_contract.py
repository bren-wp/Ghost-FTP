#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


class WindowsDecisionDialogContractTests(unittest.TestCase):
    def read(self, relative: str) -> str:
        return (ROOT / relative).read_text(encoding="utf-8")

    def test_confirm_info_and_error_prefer_application_owned_decision_card(self) -> None:
        source = self.read("internal/platform/windows.go")
        self.assertIn(
            "decisionCardDialog(title, instruction, content, decisionCardKindConfirm)", source
        )
        self.assertIn(
            "decisionCardDialog(title, instruction, content, decisionCardKindInfo)", source
        )
        self.assertIn(
            "decisionCardDialog(title, instruction, content, decisionCardKindError)", source
        )

        confirm = source[source.index("func ConfirmDialog"):source.index("func InfoDialog")]
        info = source[source.index("func InfoDialog"):source.index("func ErrorDialog")]
        error = source[source.index("func ErrorDialog"):source.index("func MessageBox")]
        for block in (confirm, info, error):
            self.assertIn("decisionCardDialog", block)
            self.assertIn("taskDialogCall", block)
            self.assertIn("MessageBox(title", block)
            self.assertLess(block.index("decisionCardDialog"), block.index("taskDialogCall"))

    def test_decision_card_reuses_shared_theme_dpi_owner_and_keyboard_shell(self) -> None:
        card = self.read("internal/platform/decision_card_windows.go")
        self.assertIn("premiumDialogTheme()", card)
        self.assertIn("theme.Danger", card)
        self.assertIn("theme.AccentStrong", card)
        self.assertIn("premiumDialogDPI(owner)", card)
        self.assertIn("premiumDialogOuterSize", card)
        self.assertIn("premiumModalOwner(owner)", card)
        self.assertIn("premiumRunDialogLoop(hwnd", card)
        self.assertIn("decisionIDYes", card)
        self.assertIn("decisionIDNo", card)
        self.assertIn("promptIDOK", card)
        self.assertNotIn("PostQuitMessage", card)

    def test_runtime_labels_are_resolved_from_the_active_desktop_locale(self) -> None:
        provider = self.read("internal/platform/dialog_label_provider_windows.go")
        desktop = self.read("internal/desktop/dialog_localization_windows.go")
        card = self.read("internal/platform/decision_card_windows.go")

        self.assertIn("SetDialogLabelProvider", provider)
        self.assertIn("resolvedDialogLabels", provider)
        self.assertIn("platform.SetDialogLabelProvider", desktop)
        self.assertIn("apps.Range", desktop)
        self.assertIn("a.languageCode()", desktop)
        self.assertIn('a.tr("common.cancel")', desktop)
        self.assertIn("[24]string", desktop)
        self.assertIn("resolvedDialogLabels()", card)

    def test_profile_privacy_flow_has_no_old_hardcoded_english_copy(self) -> None:
        flow = self.read("internal/desktop/connection_profiles_windows.go")
        localized = self.read("internal/desktop/profile_dialog_text_windows.go")

        for old_copy in (
            "Profile changes are unavailable during a connection",
            "Profile name is required",
            "Retain stored credentials?",
            "This profile already contains stored credentials",
            "Old credentials will not be transferred",
            "Credentials that no longer belong to the changed server",
        ):
            self.assertNotIn(old_copy, flow)

        for helper in (
            "profileBusyHeading(language)",
            "profileBusyBody(language)",
            "profileNameLabel(language)",
            "profileNameRequiredHeading(language)",
            "profileNameRequiredBody(language)",
            "profileRetainBody(language)",
            "profileAutoRemovedPrefix(language)",
            "profileRetainHeading(language)",
            "profileOldCredentialsHeading(language)",
            "profileOldCredentialsBody(language)",
        ):
            self.assertIn(helper, flow)

        self.assertGreaterEqual(localized.count("localizedPrompt(language, [24]string{"), 11)
        self.assertIn("yesLabel(language)", localized)
        self.assertIn("noLabel(language)", localized)

    def test_profile_credential_clear_semantics_remain_explicit(self) -> None:
        flow = self.read("internal/desktop/connection_profiles_windows.go")
        for marker in (
            "profilebinding.AccountMatches(",
            "profilebinding.PrivateKeyMatches(",
            "clearPassword = existing.HasPassword && !sameAccount",
            "clearPassphrase = existing.HasPassphrase && !samePrivateKey",
            "ClearPassword:   clearPassword",
            "ClearPassphrase: clearPassphrase",
        ):
            self.assertIn(marker, flow)

    def test_decision_card_contract_applies_to_current_public_line(self) -> None:
        version = self.read("VERSION").strip()
        self.assertRegex(version, r"^\d+\.\d+\.\d+$")
        parts = tuple(int(part) for part in version.split("."))
        self.assertGreaterEqual(parts, (0, 0, 1))


if __name__ == "__main__":
    unittest.main()
