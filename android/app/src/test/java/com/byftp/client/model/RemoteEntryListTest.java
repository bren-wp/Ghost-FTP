package com.byftp.client.model;

import static org.junit.Assert.assertEquals;

import java.util.List;
import org.junit.Test;

public final class RemoteEntryListTest {
    @Test public void sortsDirectoriesFirstThenNamesCaseInsensitively() {
        List<RemoteEntry> result = RemoteEntryList.sorted(List.of(
            new RemoteEntry("z.txt", false, 1, 0),
            new RemoteEntry("beta", true, 0, 0),
            new RemoteEntry("Alpha.txt", false, 1, 0),
            new RemoteEntry("Alpha", true, 0, 0)
        ));

        assertEquals(List.of("Alpha", "beta", "Alpha.txt", "z.txt"), result.stream().map(RemoteEntry::name).toList());
    }

    @Test public void filtersWithoutMutatingSortedSource() {
        List<RemoteEntry> source = RemoteEntryList.sorted(List.of(
            new RemoteEntry("public_html", true, 0, 0),
            new RemoteEntry("index.html", false, 12, 0),
            new RemoteEntry("INDEX.php", false, 12, 0)
        ));
        List<RemoteEntry> result = RemoteEntryList.filtered(source, " index ");

        assertEquals(List.of("index.html", "INDEX.php"), result.stream().map(RemoteEntry::name).toList());
        assertEquals(3, source.size());
    }
}
