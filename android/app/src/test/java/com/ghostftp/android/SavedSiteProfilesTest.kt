package com.ghostftp.android

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class SavedSiteProfilesTest {
    private fun site(name: String, port: Int = 21) = SavedSiteDefinition(
        name = name,
        protocol = ConnectionProtocol.FTP,
        host = "server.test",
        port = port,
        username = "account",
        fingerprint = "",
        remotePath = "/",
        keepAliveSeconds = 15
    )

    @Test
    fun editRenamesOnlyOneSiteAndPreservesUnrelatedMetadata() {
        val existing = listOf(site("Alpha"), site("Beta", 2121), site("Gamma", 2222))
        val replacement = site("New Beta", 990)
        val changed = SavedSiteProfiles.updated(existing, "bETA", replacement)
        assertEquals(listOf(existing[0], replacement, existing[2]), changed)
        assertEquals("Beta", existing[1].name)
        assertEquals(2121, existing[1].port)
    }

    @Test
    fun updateRejectsMissingProfileAndCaseInsensitiveNameCollisions() {
        val existing = listOf(site("Alpha"), site("Beta"))
        assertNull(SavedSiteProfiles.updated(existing, "Missing", site("New")))
        assertNull(SavedSiteProfiles.updated(existing, "Beta", site("ALPHA")))
        assertEquals(listOf("Alpha", "Beta"), existing.map { it.name })
    }

    @Test
    fun deletionOnlyRemovesSelectedSite() {
        val existing = listOf(site("Alpha"), site("Beta", 2121), site("Gamma", 2222))
        assertEquals(listOf(existing[0], existing[2]), SavedSiteProfiles.without(existing, "bEtA"))
        assertNull(SavedSiteProfiles.without(existing, "Missing"))
        assertEquals(3, existing.size)
    }

    @Test
    fun deterministicAlphabeticSortingDoesNotMutateStoredOrder() {
        val existing = listOf(site("zeta"), site("Alpha"), site("beta"))
        assertEquals(listOf("Alpha", "beta", "zeta"), SavedSiteProfiles.ordered(existing).map { it.name })
        assertEquals("zeta", existing.first().name)
    }

    @Test
    fun invalidEditsAreRejectedWithoutChangingOtherSites() {
        val existing = listOf(site("Alpha"), site("Beta"))
        assertNull(SavedSiteProfiles.updated(existing, "Beta", site("Bad\nName")))
        assertEquals(listOf("Alpha", "Beta"), existing.map { it.name })
    }
}
