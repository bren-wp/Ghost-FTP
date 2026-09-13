package app.ghostftp.client;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertFalse;
import static org.junit.Assert.assertTrue;

import java.util.Arrays;
import java.util.List;

import org.junit.Test;

public final class WorkspaceOpsTest {
    @Test
    public void filterAndSortKeepsDirectoriesFirstAndMapsSourceIndices() {
        List<WorkspaceOps.Item> source = Arrays.asList(
                item(0, "zeta.txt", false, 90, 30, "644"),
                item(1, "alpha", true, 0, 20, "755"),
                item(2, "beta.txt", false, 10, 10, "600"));

        List<WorkspaceOps.Item> visible = WorkspaceOps.filterAndSort(
                source, "", WorkspaceOps.SortKey.SIZE, true);

        assertEquals(3, visible.size());
        assertEquals(1, visible.get(0).sourceIndex);
        assertEquals(2, visible.get(1).sourceIndex);
        assertEquals(0, visible.get(2).sourceIndex);
    }

    @Test
    public void filterIsCaseInsensitiveAndDoesNotMutateAuthoritativeSnapshot() {
        List<WorkspaceOps.Item> source = Arrays.asList(
                item(0, "ReadMe.TXT", false, 20, 0, "644"),
                item(1, "src", true, 0, 0, "755"));

        List<WorkspaceOps.Item> visible = WorkspaceOps.filterAndSort(
                source, "readme", WorkspaceOps.SortKey.NAME, true);

        assertEquals(1, visible.size());
        assertEquals("ReadMe.TXT", visible.get(0).name);
        assertEquals(2, source.size());
    }

    @Test
    public void remoteSortCyclesThroughPermissions() {
        WorkspaceOps.SortKey key = WorkspaceOps.SortKey.NAME;
        key = WorkspaceOps.nextRemoteSortKey(key);
        assertEquals(WorkspaceOps.SortKey.TYPE, key);
        key = WorkspaceOps.nextRemoteSortKey(key);
        assertEquals(WorkspaceOps.SortKey.SIZE, key);
        key = WorkspaceOps.nextRemoteSortKey(key);
        assertEquals(WorkspaceOps.SortKey.MODIFIED, key);
        key = WorkspaceOps.nextRemoteSortKey(key);
        assertEquals(WorkspaceOps.SortKey.PERMISSIONS, key);
        key = WorkspaceOps.nextRemoteSortKey(key);
        assertEquals(WorkspaceOps.SortKey.NAME, key);
    }

    @Test
    public void directoryComparisonIsConservativeAndSyncNavigationRequiresBothDirectories() {
        List<WorkspaceOps.Item> local = Arrays.asList(
                item(0, "docs", true, 0, 0, ""),
                item(1, "same.bin", false, 100, 0, ""),
                item(2, "different.bin", false, 10, 0, ""),
                item(3, "local-only.txt", false, 1, 0, ""));
        List<WorkspaceOps.Item> remote = Arrays.asList(
                item(0, "docs", true, 0, 0, "755"),
                item(1, "same.bin", false, 100, 0, "644"),
                item(2, "different.bin", false, 11, 0, "644"),
                item(3, "remote-only.txt", false, 1, 0, "644"));

        List<WorkspaceOps.Comparison> rows = WorkspaceOps.compareDirectories(local, remote);

        WorkspaceOps.Comparison docs = find(rows, "docs");
        assertEquals(WorkspaceOps.Difference.SAME, docs.difference);
        assertTrue(docs.canSynchronizeDirectoryNavigation());

        WorkspaceOps.Comparison same = find(rows, "same.bin");
        assertEquals(WorkspaceOps.Difference.SAME, same.difference);
        assertFalse(same.canSynchronizeDirectoryNavigation());

        assertEquals(WorkspaceOps.Difference.DIFFERENT, find(rows, "different.bin").difference);
        assertEquals(WorkspaceOps.Difference.ONLY_LOCAL, find(rows, "local-only.txt").difference);
        assertEquals(WorkspaceOps.Difference.ONLY_REMOTE, find(rows, "remote-only.txt").difference);
    }

    @Test
    public void searchRequiresNonEmptyQueryAndUsesCaseInsensitiveContains() {
        assertFalse(WorkspaceOps.matchesSearch("alpha.txt", ""));
        assertTrue(WorkspaceOps.matchesSearch("Alpha.TXT", "pha.t"));
        assertFalse(WorkspaceOps.matchesSearch("beta.txt", "alpha"));
    }

    private static WorkspaceOps.Item item(int index, String name, boolean directory, long size, long modified, String permissions) {
        return new WorkspaceOps.Item(index, name, directory, size, modified, permissions);
    }

    private static WorkspaceOps.Comparison find(List<WorkspaceOps.Comparison> rows, String name) {
        for (WorkspaceOps.Comparison row : rows) {
            if (name.equals(row.name)) {
                return row;
            }
        }
        throw new AssertionError("Missing comparison row: " + name);
    }
}
