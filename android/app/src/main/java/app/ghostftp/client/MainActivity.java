package app.ghostftp.client;

import android.annotation.SuppressLint;
import android.app.Activity;
import android.app.AlertDialog;
import android.content.Intent;
import android.content.SharedPreferences;
import android.content.UriPermission;
import android.database.Cursor;
import android.graphics.Typeface;
import android.net.Uri;
import android.os.Bundle;
import android.provider.DocumentsContract;
import android.text.InputType;
import android.view.ViewGroup;
import android.widget.ArrayAdapter;
import android.widget.Button;
import android.widget.EditText;
import android.widget.LinearLayout;
import android.widget.ListView;
import android.widget.ScrollView;
import android.widget.Spinner;
import android.widget.TextView;

import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.util.ArrayDeque;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Deque;
import java.util.List;
import java.util.UUID;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

@SuppressLint("SetTextI18n")
public final class MainActivity extends Activity {
    private static final int REQUEST_TREE = 1001;
    private static final String PREFS = "ghostftp_android";

    private final ExecutorService io = Executors.newSingleThreadExecutor();
    private final List<LocalEntry> localEntries = new ArrayList<>();
    private final List<RemoteEntry> remoteEntries = new ArrayList<>();
    private final Deque<String> localParents = new ArrayDeque<>();
    private final List<SiteProfile> profiles = new ArrayList<>();

    private Spinner siteSpinner;
    private Spinner protocol;
    private Spinner localBookmarkSpinner;
    private Spinner remoteBookmarkSpinner;
    private EditText profileName;
    private EditText host;
    private EditText port;
    private EditText username;
    private EditText password;
    private TextView localPath;
    private TextView remotePath;
    private TextView status;
    private TextView connectionState;
    private ListView localList;
    private ListView remoteList;
    private Button connect;
    private Button disconnect;
    private Button upload;
    private Button download;
    private Button saveSite;
    private Button deleteSite;
    private Button localSetStart;
    private Button localAddBookmark;
    private Button localRemoveBookmark;
    private Button localOpenBookmark;
    private Button localNewFolder;
    private Button localRename;
    private Button localDelete;
    private Button remoteSetStart;
    private Button remoteAddBookmark;
    private Button remoteRemoveBookmark;
    private Button remoteOpenBookmark;
    private Button remoteNewFolder;
    private Button remoteRename;
    private Button remoteDelete;

    private SiteProfileStore profileStore;
    private String activeProfileId;
    private String connectedIdentityKey;
    private Uri treeUri;
    private String rootDocumentId;
    private String currentDocumentId;
    private String currentRemotePath = "/";
    private int selectedLocal = -1;
    private int selectedRemote = -1;
    private FtpSession session;
    private boolean busy;
    private volatile boolean transferActive;
    private volatile long transferGeneration;

    @Override
    protected void onCreate(Bundle state) {
        super.onCreate(state);
        SharedPreferences preferences = getSharedPreferences(PREFS, MODE_PRIVATE);
        profileStore = new SiteProfileStore(preferences);
        profiles.addAll(profileStore.load());
        buildUi();
        restorePreferences();
        renderSites();
        renderBookmarks();
        refreshButtons();
    }

    @Override
    protected void onDestroy() {
        transferGeneration++;
        transferActive = false;
        FtpSession current = session;
        session = null;
        connectedIdentityKey = null;
        if (current != null) current.cancelActiveTransfer();
        io.shutdownNow();
        super.onDestroy();
    }

    private void buildUi() {
        LinearLayout root = new LinearLayout(this);
        root.setOrientation(LinearLayout.VERTICAL);
        int pad = dp(12);
        root.setPadding(pad, pad, pad, dp(24));
        root.setBackgroundColor(GhostTheme.WINDOW);

        LinearLayout header = card("GHOST FTP", "ANDROID · PRIVATE FTP/FTPS CLIENT");
        TextView intro = label("No telemetry · passwords stay in memory · strict FTPS hostname verification", 12, GhostTheme.MUTED);
        intro.setPadding(0, dp(4), 0, dp(10));
        header.addView(intro, matchWrap());
        connectionState = label("DISCONNECTED", 11, GhostTheme.MUTED);
        GhostTheme.styleBadge(connectionState, GhostTheme.MUTED);
        header.addView(connectionState, wrapWrap());
        root.addView(header, cardParams());

        LinearLayout sites = card("SAVED SITES", "Quick Connect stays transient until you explicitly save a site.");
        siteSpinner = new Spinner(this);
        GhostTheme.styleSpinner(siteSpinner);
        sites.addView(siteSpinner, matchWrapSpaced());
        profileName = field("Site name", false);
        sites.addView(profileName, matchWrapSpaced());
        LinearLayout siteActions = row();
        Button loadSite = button("Load");
        saveSite = primaryButton("Save / update");
        deleteSite = dangerButton("Delete");
        siteActions.addView(loadSite, weightedSpaced());
        siteActions.addView(saveSite, weightedSpaced());
        siteActions.addView(deleteSite, weightedSpaced());
        sites.addView(siteActions, matchWrap());
        loadSite.setOnClickListener(v -> loadSelectedSite());
        saveSite.setOnClickListener(v -> saveOrUpdateSite());
        deleteSite.setOnClickListener(v -> deleteActiveSite());
        root.addView(sites, cardParams());

        LinearLayout connection = card("CONNECTION", "Use FTPS whenever the server supports it. Plain FTP is unencrypted.");
        protocol = new Spinner(this);
        GhostTheme.styleSpinner(protocol);
        protocol.setAdapter(GhostTheme.spinnerAdapter(this, Arrays.asList("FTPS", "FTP")));
        connection.addView(protocol, matchWrapSpaced());

        LinearLayout endpoint = row();
        host = field("Server host", false);
        port = field("Port", false);
        port.setInputType(InputType.TYPE_CLASS_NUMBER);
        endpoint.addView(host, weightedSpaced());
        endpoint.addView(port, fixedWidthSpaced(94));
        connection.addView(endpoint, matchWrap());

        LinearLayout credentials = row();
        username = field("Username", false);
        password = field("Password (memory only)", true);
        credentials.addView(username, weightedSpaced());
        credentials.addView(password, weightedSpaced());
        connection.addView(credentials, matchWrap());

        LinearLayout connectionActions = row();
        connect = primaryButton("Connect");
        disconnect = dangerButton("Disconnect");
        connectionActions.addView(connect, weightedSpaced());
        connectionActions.addView(disconnect, weightedSpaced());
        connection.addView(connectionActions, matchWrap());
        connect.setOnClickListener(v -> connect());
        disconnect.setOnClickListener(v -> disconnect());
        root.addView(connection, cardParams());

        LinearLayout paneHost = new LinearLayout(this);
        boolean wide = getResources().getConfiguration().screenWidthDp >= 700;
        paneHost.setOrientation(wide ? LinearLayout.HORIZONTAL : LinearLayout.VERTICAL);
        LinearLayout localPane = buildLocalPane();
        LinearLayout remotePane = buildRemotePane();
        if (wide) {
            paneHost.addView(localPane, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
            paneHost.addView(remotePane, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        } else {
            paneHost.addView(localPane, matchWrap());
            paneHost.addView(remotePane, matchWrap());
        }
        root.addView(paneHost, matchWrap());

        LinearLayout transfers = card("TRANSFERS", "Selected local files upload to the current server directory; selected server files download into the active SAF folder.");
        LinearLayout transferActions = row();
        upload = primaryButton("Upload →");
        download = primaryButton("← Download");
        transferActions.addView(upload, weightedSpaced());
        transferActions.addView(download, weightedSpaced());
        transfers.addView(transferActions, matchWrap());
        upload.setOnClickListener(v -> uploadSelected());
        download.setOnClickListener(v -> downloadSelected());
        status = label("Ready.", 13, GhostTheme.MUTED);
        status.setTypeface(Typeface.DEFAULT, Typeface.BOLD);
        status.setPadding(dp(12), dp(12), dp(12), dp(12));
        status.setBackground(GhostTheme.rounded(this, GhostTheme.LIST, GhostTheme.BORDER, 10));
        transfers.addView(status, matchWrapSpaced());
        root.addView(transfers, cardParams());

        TextView footer = label("Local access stays inside Android Storage Access Framework grants. SFTP remains hidden until Android has strict host-key identity verification equivalent to desktop Ghost FTP.", 11, GhostTheme.MUTED);
        footer.setPadding(dp(4), dp(4), dp(4), dp(8));
        root.addView(footer, matchWrap());

        ScrollView scroll = new ScrollView(this);
        scroll.setFillViewport(true);
        scroll.setBackgroundColor(GhostTheme.WINDOW);
        scroll.addView(root);
        setContentView(scroll);
    }

    private LinearLayout buildLocalPane() {
        LinearLayout pane = card("LOCAL", "Tap a folder to open it. Long-press any item to select it for Rename/Delete.");
        localPath = label("No folder selected", 13, GhostTheme.MUTED);
        localPath.setPadding(dp(8), dp(8), dp(8), dp(8));
        localPath.setBackground(GhostTheme.rounded(this, GhostTheme.LIST, GhostTheme.BORDER, 8));
        pane.addView(localPath, matchWrapSpaced());

        LinearLayout nav = row();
        Button choose = button("Choose folder");
        Button localUp = button("Up");
        Button localRefresh = button("Refresh");
        nav.addView(choose, weightedSpaced());
        nav.addView(localUp, weightedSpaced());
        nav.addView(localRefresh, weightedSpaced());
        pane.addView(nav, matchWrap());
        choose.setOnClickListener(v -> chooseFolder());
        localUp.setOnClickListener(v -> localUp());
        localRefresh.setOnClickListener(v -> refreshLocal());

        LinearLayout manage = row();
        localNewFolder = button("New folder");
        localRename = button("Rename");
        localDelete = dangerButton("Delete");
        manage.addView(localNewFolder, weightedSpaced());
        manage.addView(localRename, weightedSpaced());
        manage.addView(localDelete, weightedSpaced());
        pane.addView(manage, matchWrap());
        localNewFolder.setOnClickListener(v -> createLocalFolder());
        localRename.setOnClickListener(v -> renameLocalSelected());
        localDelete.setOnClickListener(v -> deleteLocalSelected());

        LinearLayout bookmarkActions = row();
        localSetStart = button("Set start");
        localAddBookmark = button("Add bookmark");
        localRemoveBookmark = dangerButton("Remove bookmark");
        bookmarkActions.addView(localSetStart, weightedSpaced());
        bookmarkActions.addView(localAddBookmark, weightedSpaced());
        bookmarkActions.addView(localRemoveBookmark, weightedSpaced());
        pane.addView(bookmarkActions, matchWrap());
        localSetStart.setOnClickListener(v -> setLocalStart());
        localAddBookmark.setOnClickListener(v -> addLocalBookmark());
        localRemoveBookmark.setOnClickListener(v -> removeLocalBookmark());

        localBookmarkSpinner = new Spinner(this);
        GhostTheme.styleSpinner(localBookmarkSpinner);
        pane.addView(localBookmarkSpinner, matchWrapSpaced());
        localOpenBookmark = button("Open local bookmark");
        pane.addView(localOpenBookmark, matchWrapSpaced());
        localOpenBookmark.setOnClickListener(v -> openLocalBookmark());

        localList = new ListView(this);
        GhostTheme.styleList(localList);
        pane.addView(localList, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, dp(270)));
        localList.setOnItemClickListener((parent, view, position, id) -> selectLocal(position));
        localList.setOnItemLongClickListener((parent, view, position, id) -> {
            if (busy || position < 0 || position >= localEntries.size()) return true;
            selectedLocal = position;
            renderLocal();
            setStatus("Selected local item for management: " + localEntries.get(position).name);
            return true;
        });
        return pane;
    }

    private LinearLayout buildRemotePane() {
        LinearLayout pane = card("SERVER", "Tap a folder to open it. Long-press any item to select it for Rename/Delete.");
        remotePath = label(currentRemotePath, 13, GhostTheme.MUTED);
        remotePath.setPadding(dp(8), dp(8), dp(8), dp(8));
        remotePath.setBackground(GhostTheme.rounded(this, GhostTheme.LIST, GhostTheme.BORDER, 8));
        pane.addView(remotePath, matchWrapSpaced());

        LinearLayout nav = row();
        Button remoteRefresh = button("Refresh");
        Button remoteUp = button("Up");
        remoteNewFolder = button("New folder");
        nav.addView(remoteRefresh, weightedSpaced());
        nav.addView(remoteUp, weightedSpaced());
        nav.addView(remoteNewFolder, weightedSpaced());
        pane.addView(nav, matchWrap());
        remoteRefresh.setOnClickListener(v -> refreshRemote(currentRemotePath));
        remoteUp.setOnClickListener(v -> remoteUp());
        remoteNewFolder.setOnClickListener(v -> createRemoteFolder());

        LinearLayout manage = row();
        remoteRename = button("Rename");
        remoteDelete = dangerButton("Delete");
        manage.addView(remoteRename, weightedSpaced());
        manage.addView(remoteDelete, weightedSpaced());
        pane.addView(manage, matchWrap());
        remoteRename.setOnClickListener(v -> renameRemoteSelected());
        remoteDelete.setOnClickListener(v -> deleteRemoteSelected());

        LinearLayout bookmarkActions = row();
        remoteSetStart = button("Set start");
        remoteAddBookmark = button("Add bookmark");
        remoteRemoveBookmark = dangerButton("Remove bookmark");
        bookmarkActions.addView(remoteSetStart, weightedSpaced());
        bookmarkActions.addView(remoteAddBookmark, weightedSpaced());
        bookmarkActions.addView(remoteRemoveBookmark, weightedSpaced());
        pane.addView(bookmarkActions, matchWrap());
        remoteSetStart.setOnClickListener(v -> setRemoteStart());
        remoteAddBookmark.setOnClickListener(v -> addRemoteBookmark());
        remoteRemoveBookmark.setOnClickListener(v -> removeRemoteBookmark());

        remoteBookmarkSpinner = new Spinner(this);
        GhostTheme.styleSpinner(remoteBookmarkSpinner);
        pane.addView(remoteBookmarkSpinner, matchWrapSpaced());
        remoteOpenBookmark = button("Open server bookmark");
        pane.addView(remoteOpenBookmark, matchWrapSpaced());
        remoteOpenBookmark.setOnClickListener(v -> openRemoteBookmark());

        remoteList = new ListView(this);
        GhostTheme.styleList(remoteList);
        pane.addView(remoteList, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, dp(270)));
        remoteList.setOnItemClickListener((parent, view, position, id) -> selectRemote(position));
        remoteList.setOnItemLongClickListener((parent, view, position, id) -> {
            if (busy || position < 0 || position >= remoteEntries.size()) return true;
            selectedRemote = position;
            renderRemote();
            setStatus("Selected server item for management: " + remoteEntries.get(position).name);
            return true;
        });
        return pane;
    }

    private void restorePreferences() {
        SharedPreferences prefs = getSharedPreferences(PREFS, MODE_PRIVATE);
        host.setText(prefs.getString("host", ""));
        username.setText(prefs.getString("username", ""));
        String savedProtocol = prefs.getString("protocol", "FTPS");
        protocol.setSelection("FTP".equals(savedProtocol) ? 1 : 0);
        port.setText(prefs.getString("port", "21"));
        String savedTree = prefs.getString("treeUri", "");
        if (!savedTree.isEmpty()) {
            tryActivateLocalTree(Uri.parse(savedTree), "Saved local folder is no longer available.", true);
        }
    }

    private void savePreferences() {
        getSharedPreferences(PREFS, MODE_PRIVATE).edit()
                .putString("host", host.getText().toString().trim())
                .putString("username", username.getText().toString().trim())
                .putString("protocol", protocol.getSelectedItem().toString())
                .putString("port", port.getText().toString().trim())
                .putString("treeUri", treeUri == null ? "" : treeUri.toString())
                .apply();
    }

    private void renderSites() {
        List<String> labels = new ArrayList<>();
        labels.add("Quick Connect (not saved)");
        int selected = 0;
        for (int i = 0; i < profiles.size(); i++) {
            SiteProfile profile = profiles.get(i);
            labels.add(profile.toString());
            if (profile.id.equals(activeProfileId)) selected = i + 1;
        }
        siteSpinner.setAdapter(GhostTheme.spinnerAdapter(this, labels));
        siteSpinner.setSelection(selected);
    }

    private void renderBookmarks() {
        SiteProfile profile = activeProfile();
        List<String> local = new ArrayList<>();
        List<String> remote = new ArrayList<>();
        if (profile != null) {
            local.addAll(profile.localBookmarks);
            remote.addAll(profile.remoteBookmarks);
        }
        localBookmarkSpinner.setAdapter(GhostTheme.spinnerAdapter(this, local));
        remoteBookmarkSpinner.setAdapter(GhostTheme.spinnerAdapter(this, remote));
        refreshButtons();
    }

    private void loadSelectedSite() {
        if (busy || session != null) {
            setStatus("Disconnect before switching sites.");
            return;
        }
        int index = siteSpinner.getSelectedItemPosition() - 1;
        if (index < 0) {
            activeProfileId = null;
            profileName.setText("");
            currentRemotePath = "/";
            remoteEntries.clear();
            selectedRemote = -1;
            renderRemote();
            renderBookmarks();
            setStatus("Quick Connect mode. Connection details are not a saved site until you press Save / update.");
            return;
        }
        if (index >= profiles.size()) return;
        SiteProfile profile = profiles.get(index);
        activeProfileId = profile.id;
        profileName.setText(profile.name);
        protocol.setSelection("FTP".equals(profile.protocol) ? 1 : 0);
        host.setText(profile.host);
        port.setText(Integer.toString(profile.port));
        username.setText(profile.username);
        password.setText("");
        currentRemotePath = "/";
        remoteEntries.clear();
        selectedRemote = -1;
        renderRemote();
        clearLocalRoot();
        renderLocal();
        boolean localStartUnavailable = false;
        if (!profile.localStartTreeUri.isEmpty()) {
            localStartUnavailable = !tryActivateLocalTree(
                    Uri.parse(profile.localStartTreeUri),
                    "This site's local start folder is unavailable. Choose it again and update the site.",
                    true);
        }
        renderSites();
        renderBookmarks();
        if (localStartUnavailable) {
            setStatus("Site loaded, but its local start folder is unavailable. Choose it again and update the site.");
        } else {
            setStatus("Site loaded. Password remains blank; connect to validate the saved server start directory.");
        }
    }

    private void saveOrUpdateSite() {
        if (busy || session != null) {
            setStatus("Disconnect before saving site identity changes.");
            return;
        }
        String name = profileName.getText().toString().trim();
        if (name.isEmpty()) {
            setStatus("Site name is required.");
            return;
        }
        final int portValue;
        try {
            portValue = parsePort();
        } catch (IllegalArgumentException e) {
            setStatus(e.getMessage());
            return;
        }
        String hostValue = host.getText().toString().trim();
        if (hostValue.isEmpty()) {
            setStatus("Server host is required.");
            return;
        }
        SiteProfile previous = activeProfile();
        String id = previous == null ? UUID.randomUUID().toString() : previous.id;
        String localStart = previous == null ? persistedCurrentTreeUri() : previous.localStartTreeUri;
        String remoteStart = previous == null ? "/" : previous.remoteStartPath;
        List<String> localBookmarks = previous == null ? new ArrayList<>() : previous.localBookmarks;
        List<String> remoteBookmarks = previous == null ? new ArrayList<>() : previous.remoteBookmarks;
        try {
            SiteProfile next = new SiteProfile(
                    id,
                    name,
                    protocol.getSelectedItem().toString(),
                    hostValue,
                    portValue,
                    username.getText().toString().trim(),
                    localStart,
                    remoteStart,
                    localBookmarks,
                    remoteBookmarks).withRemoteStateResetForIdentityChange(previous);
            replaceProfile(next);
            activeProfileId = next.id;
            profileStore.save(profiles);
            renderSites();
            renderBookmarks();
            if (previous != null && !next.sameServerIdentity(previous)) {
                setStatus("Site identity updated. Server start path and server bookmarks were cleared to prevent cross-server inheritance.");
            } else {
                setStatus(previous == null ? "Site saved. No password was stored." : "Site updated. No password was stored.");
            }
        } catch (IllegalArgumentException e) {
            setStatus(e.getMessage());
        }
    }

    private void deleteActiveSite() {
        if (busy || session != null) {
            setStatus("Disconnect before deleting a saved site.");
            return;
        }
        SiteProfile profile = activeProfile();
        if (profile == null) {
            setStatus("Load a saved site before deleting it.");
            return;
        }
        confirm("Delete saved site?", "Delete “" + profile.name + "”? Passwords are not stored, but its start directories and bookmarks will be removed.", () -> {
            profiles.remove(profile);
            profileStore.save(profiles);
            activeProfileId = null;
            profileName.setText("");
            renderSites();
            renderBookmarks();
            setStatus("Saved site deleted. Quick Connect settings were not converted into another profile.");
        });
    }

    private void replaceProfile(SiteProfile next) {
        for (int i = 0; i < profiles.size(); i++) {
            if (profiles.get(i).id.equals(next.id)) {
                profiles.set(i, next);
                return;
            }
        }
        profiles.add(next);
    }

    private SiteProfile activeProfile() {
        if (activeProfileId == null) return null;
        for (SiteProfile profile : profiles) if (profile.id.equals(activeProfileId)) return profile;
        return null;
    }

    private void connect() {
        if (busy || session != null) return;
        String hostValue = host.getText().toString().trim();
        String userValue = username.getText().toString().trim();
        String passwordValue = password.getText().toString();
        boolean secure = "FTPS".equals(protocol.getSelectedItem().toString());
        final int portValue;
        try {
            portValue = parsePort();
        } catch (IllegalArgumentException e) {
            setStatus(e.getMessage());
            return;
        }
        if (hostValue.isEmpty()) {
            setStatus("Server host is required.");
            return;
        }
        SiteProfile profile = activeProfile();
        String identity = identityKey(protocol.getSelectedItem().toString(), hostValue, portValue, userValue);
        if (profile != null && !profile.identityKey().equals(identity)) {
            setStatus("Loaded site identity was edited. Save/update it first or switch to Quick Connect; saved server paths will not be reused across identities.");
            return;
        }
        String requestedStart = profile == null ? null : profile.remoteStartPath;
        setBusy(true, secure ? "Connecting with strict FTPS TLS verification…" : "Connecting with unencrypted FTP…");
        io.execute(() -> {
            FtpSession next = null;
            try {
                next = new FtpSession(hostValue, portValue, secure);
                next.connect(userValue, passwordValue);
                String start = requestedStart == null ? next.pwd() : requestedStart;
                List<RemoteEntry> entries = next.list(start);
                FtpSession ready = next;
                runOnUiThread(() -> {
                    session = ready;
                    connectedIdentityKey = identity;
                    currentRemotePath = start;
                    remoteEntries.clear();
                    remoteEntries.addAll(entries);
                    selectedRemote = -1;
                    password.setText("");
                    savePreferences();
                    renderRemote();
                    setBusy(false, secure
                            ? "FTPS connected. Certificate/hostname verified and server start directory freshly listed."
                            : "FTP connected. Warning: transport is unencrypted; server start directory freshly listed.");
                });
            } catch (Exception e) {
                if (next != null) next.close();
                postError(profile == null ? "Connection failed" : "Connection or saved server start directory failed", e);
            }
        });
    }

    private void disconnect() {
        if (transferActive) {
            cancelTransfer();
            return;
        }
        if (busy) return;
        FtpSession current = session;
        session = null;
        connectedIdentityKey = null;
        remoteEntries.clear();
        selectedRemote = -1;
        currentRemotePath = "/";
        renderRemote();
        if (current != null) io.execute(current::close);
        setStatus("Disconnected.");
        refreshButtons();
    }

    private void cancelTransfer() {
        if (!transferActive) return;
        transferGeneration++;
        transferActive = false;
        FtpSession current = session;
        session = null;
        connectedIdentityKey = null;
        remoteEntries.clear();
        selectedRemote = -1;
        currentRemotePath = "/";
        if (current != null) current.cancelActiveTransfer();
        renderRemote();
        setStatus("Transfer cancelled. Connection closed; reconnect before another transfer.");
        refreshButtons();
    }

    private void refreshRemote(String target) {
        FtpSession current = session;
        if (busy || current == null || !current.isConnected()) return;
        final String requested;
        try {
            requested = FtpSession.normalizeRemotePath(target);
        } catch (IOException e) {
            setStatus(e.getMessage());
            return;
        }
        setBusy(true, "Refreshing server directory…");
        io.execute(() -> {
            try {
                List<RemoteEntry> entries = current.list(requested);
                runOnUiThread(() -> {
                    if (session != current) return;
                    currentRemotePath = requested;
                    remoteEntries.clear();
                    remoteEntries.addAll(entries);
                    selectedRemote = -1;
                    renderRemote();
                    setBusy(false, "Server directory freshly listed.");
                });
            } catch (Exception e) {
                postError("Remote directory is unavailable; current path was not changed", e);
            }
        });
    }

    private void selectRemote(int position) {
        if (busy || position < 0 || position >= remoteEntries.size()) return;
        RemoteEntry entry = remoteEntries.get(position);
        if (entry.directory) {
            try {
                refreshRemote(FtpSession.joinRemote(currentRemotePath, entry.name));
            } catch (IOException e) {
                setStatus(e.getMessage());
            }
        } else {
            selectedRemote = position;
            renderRemote();
        }
    }

    private void remoteUp() {
        try {
            refreshRemote(FtpSession.parentRemote(currentRemotePath));
        } catch (IOException e) {
            setStatus(e.getMessage());
        }
    }

    private void createRemoteFolder() {
        FtpSession current = session;
        if (busy || current == null || !current.isConnected()) return;
        promptName("New server folder", "", "Create", name -> {
            String safe;
            try {
                safe = requireLeafName(name);
            } catch (IllegalArgumentException e) {
                setStatus(e.getMessage());
                return;
            }
            String base = currentRemotePath;
            setBusy(true, "Creating server folder…");
            io.execute(() -> {
                try {
                    current.makeDirectory(base, safe);
                    List<RemoteEntry> fresh = current.list(base);
                    runOnUiThread(() -> commitRemoteMutation(current, base, fresh, "Server folder created: " + safe));
                } catch (Exception e) {
                    postError("Create server folder failed", e);
                }
            });
        });
    }

    private void renameRemoteSelected() {
        FtpSession current = session;
        if (busy || current == null || !current.isConnected() || selectedRemote < 0 || selectedRemote >= remoteEntries.size()) return;
        RemoteEntry entry = remoteEntries.get(selectedRemote);
        String base = currentRemotePath;
        final String source;
        try {
            source = FtpSession.joinRemote(base, entry.name);
        } catch (IOException e) {
            setStatus(e.getMessage());
            return;
        }
        promptName("Rename server item", entry.name, "Rename", name -> {
            String safe;
            try {
                safe = requireLeafName(name);
            } catch (IllegalArgumentException e) {
                setStatus(e.getMessage());
                return;
            }
            if (safe.equals(entry.name)) return;
            setBusy(true, "Renaming server item…");
            io.execute(() -> {
                try {
                    current.renameRemote(source, safe);
                    List<RemoteEntry> fresh = current.list(base);
                    runOnUiThread(() -> commitRemoteMutation(current, base, fresh, "Server item renamed to: " + safe));
                } catch (Exception e) {
                    postError("Rename server item failed", e);
                }
            });
        });
    }

    private void deleteRemoteSelected() {
        FtpSession current = session;
        if (busy || current == null || !current.isConnected() || selectedRemote < 0 || selectedRemote >= remoteEntries.size()) return;
        RemoteEntry entry = remoteEntries.get(selectedRemote);
        String base = currentRemotePath;
        final String target;
        try {
            target = FtpSession.joinRemote(base, entry.name);
        } catch (IOException e) {
            setStatus(e.getMessage());
            return;
        }
        confirm("Delete server item?", "Delete “" + entry.name + "”? Remote directory deletion is non-recursive and succeeds only if the server accepts RMD.", () -> {
            setBusy(true, "Deleting server item…");
            io.execute(() -> {
                try {
                    current.deleteRemote(target, entry.directory);
                    List<RemoteEntry> fresh = current.list(base);
                    runOnUiThread(() -> commitRemoteMutation(current, base, fresh, "Server item deleted: " + entry.name));
                } catch (Exception e) {
                    postError("Delete server item failed", e);
                }
            });
        });
    }

    private void commitRemoteMutation(FtpSession current, String base, List<RemoteEntry> fresh, String message) {
        if (session != current || !current.isConnected() || !currentRemotePath.equals(base)) {
            setBusy(false, "Server operation completed, but the visible session/path changed. Refresh before continuing.");
            return;
        }
        remoteEntries.clear();
        remoteEntries.addAll(fresh);
        selectedRemote = -1;
        renderRemote();
        setBusy(false, message);
    }

    private void setRemoteStart() {
        SiteProfile profile = requireConnectedActiveProfile();
        if (profile == null) return;
        SiteProfile next = profile.withRemoteStartPath(currentRemotePath);
        replaceProfile(next);
        profileStore.save(profiles);
        setStatus("Saved server start directory updated after a successful listing: " + currentRemotePath);
    }

    private void addRemoteBookmark() {
        SiteProfile profile = requireConnectedActiveProfile();
        if (profile == null) return;
        SiteProfile next = profile.withRemoteBookmark(currentRemotePath);
        replaceProfile(next);
        profileStore.save(profiles);
        renderBookmarks();
        setStatus("Server bookmark added for this site identity only.");
    }

    private void removeRemoteBookmark() {
        SiteProfile profile = activeProfile();
        if (profile == null) {
            setStatus("Load a saved site before removing a server bookmark.");
            return;
        }
        int index = remoteBookmarkSpinner.getSelectedItemPosition();
        if (index < 0 || index >= profile.remoteBookmarks.size()) {
            setStatus("No server bookmark selected.");
            return;
        }
        SiteProfile next = profile.withoutRemoteBookmark(index);
        replaceProfile(next);
        profileStore.save(profiles);
        renderBookmarks();
        setStatus("Server bookmark removed.");
    }

    private void openRemoteBookmark() {
        SiteProfile profile = requireConnectedActiveProfile();
        if (profile == null) return;
        int index = remoteBookmarkSpinner.getSelectedItemPosition();
        if (index < 0 || index >= profile.remoteBookmarks.size()) {
            setStatus("No server bookmark selected.");
            return;
        }
        refreshRemote(profile.remoteBookmarks.get(index));
    }

    private SiteProfile requireConnectedActiveProfile() {
        SiteProfile profile = activeProfile();
        if (profile == null) {
            setStatus("Load or save a site first. Quick Connect does not create hidden profiles or bookmarks.");
            return null;
        }
        if (session == null || connectedIdentityKey == null || !profile.identityKey().equals(connectedIdentityKey)) {
            setStatus("Connect using this saved site before changing or opening its server paths.");
            return null;
        }
        return profile;
    }

    private void chooseFolder() {
        if (busy) return;
        Intent intent = new Intent(Intent.ACTION_OPEN_DOCUMENT_TREE);
        intent.addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION | Intent.FLAG_GRANT_WRITE_URI_PERMISSION | Intent.FLAG_GRANT_PERSISTABLE_URI_PERMISSION);
        startActivityForResult(intent, REQUEST_TREE);
    }

    @Override
    protected void onActivityResult(int requestCode, int resultCode, Intent data) {
        super.onActivityResult(requestCode, resultCode, data);
        if (requestCode != REQUEST_TREE || resultCode != RESULT_OK || data == null || data.getData() == null) return;
        Uri selected = data.getData();
        boolean canRead = (data.getFlags() & Intent.FLAG_GRANT_READ_URI_PERMISSION) != 0;
        boolean canWrite = (data.getFlags() & Intent.FLAG_GRANT_WRITE_URI_PERMISSION) != 0;
        try {
            if (canRead && canWrite) {
                getContentResolver().takePersistableUriPermission(selected, Intent.FLAG_GRANT_READ_URI_PERMISSION | Intent.FLAG_GRANT_WRITE_URI_PERMISSION);
            } else if (canWrite) {
                getContentResolver().takePersistableUriPermission(selected, Intent.FLAG_GRANT_WRITE_URI_PERMISSION);
            } else if (canRead) {
                getContentResolver().takePersistableUriPermission(selected, Intent.FLAG_GRANT_READ_URI_PERMISSION);
            }
        } catch (SecurityException ignored) {
            // A transient grant may still be usable for this Activity session.
        }
        boolean persisted = hasPersistedReadPermission(selected);
        if (tryActivateLocalTree(selected, "Selected folder could not be opened.", false)) {
            savePreferences();
            setStatus(persisted
                    ? "Local folder selected with persistent SAF permission."
                    : "Local folder opened for this session only; persistent permission was not granted, so it cannot become a saved site start/bookmark.");
        }
    }

    private boolean tryActivateLocalTree(Uri selected, String failureMessage, boolean requirePersisted) {
        if (selected == null || (requirePersisted && !hasPersistedReadPermission(selected))) {
            clearLocalRoot();
            renderLocal();
            setStatus(failureMessage);
            return false;
        }
        try {
            String documentId = DocumentsContract.getTreeDocumentId(selected);
            List<LocalEntry> next = queryChildren(selected, documentId);
            treeUri = selected;
            rootDocumentId = documentId;
            currentDocumentId = documentId;
            localParents.clear();
            localEntries.clear();
            localEntries.addAll(next);
            selectedLocal = -1;
            renderLocal();
            localPath.setText("Selected folder");
            return true;
        } catch (RuntimeException | IOException e) {
            clearLocalRoot();
            renderLocal();
            setStatus(failureMessage);
            return false;
        }
    }

    private boolean hasPersistedReadPermission(Uri uri) {
        for (UriPermission permission : getContentResolver().getPersistedUriPermissions()) {
            if (permission.isReadPermission() && permission.getUri().equals(uri)) return true;
        }
        return false;
    }

    private String persistedCurrentTreeUri() {
        return treeUri != null && hasPersistedReadPermission(treeUri) ? treeUri.toString() : "";
    }

    private void refreshLocal() {
        if (treeUri == null || currentDocumentId == null) {
            localPath.setText("No folder selected");
            renderLocal();
            return;
        }
        try {
            List<LocalEntry> next = queryChildren(treeUri, currentDocumentId);
            localEntries.clear();
            localEntries.addAll(next);
            selectedLocal = -1;
            localPath.setText(currentDocumentId.equals(rootDocumentId) ? "Selected folder" : currentDocumentId);
            renderLocal();
        } catch (IOException e) {
            clearLocalRoot();
            renderLocal();
            localPath.setText("No folder selected");
            setStatus("Local folder permission or provider is no longer available. Choose the folder again.");
        }
    }

    private List<LocalEntry> queryChildren(Uri rootTreeUri, String documentId) throws IOException {
        List<LocalEntry> result = new ArrayList<>();
        Uri children = DocumentsContract.buildChildDocumentsUriUsingTree(rootTreeUri, documentId);
        String[] projection = {DocumentsContract.Document.COLUMN_DOCUMENT_ID, DocumentsContract.Document.COLUMN_DISPLAY_NAME,
                DocumentsContract.Document.COLUMN_MIME_TYPE, DocumentsContract.Document.COLUMN_SIZE};
        try (Cursor cursor = getContentResolver().query(children, projection, null, null, null)) {
            if (cursor == null) throw new IOException("Folder provider returned no directory listing.");
            while (cursor.moveToNext()) {
                String id = cursor.getString(0);
                String name = cursor.getString(1);
                String mime = cursor.getString(2);
                long size = cursor.isNull(3) ? 0L : cursor.getLong(3);
                result.add(new LocalEntry(id, name, DocumentsContract.Document.MIME_TYPE_DIR.equals(mime), size));
            }
        } catch (SecurityException e) {
            throw new IOException("Local folder permission is no longer available.", e);
        }
        return result;
    }

    private void selectLocal(int position) {
        if (busy || position < 0 || position >= localEntries.size()) return;
        LocalEntry entry = localEntries.get(position);
        if (entry.directory) {
            try {
                List<LocalEntry> next = queryChildren(treeUri, entry.documentId);
                localParents.push(currentDocumentId);
                currentDocumentId = entry.documentId;
                localEntries.clear();
                localEntries.addAll(next);
                selectedLocal = -1;
                localPath.setText(currentDocumentId);
                renderLocal();
            } catch (IOException e) {
                setStatus("Local directory is unavailable; current path was not changed: " + safeMessage(e));
            }
        } else {
            selectedLocal = position;
            renderLocal();
        }
    }

    private void localUp() {
        if (busy || localParents.isEmpty()) return;
        String target = localParents.peek();
        try {
            List<LocalEntry> next = queryChildren(treeUri, target);
            localParents.pop();
            currentDocumentId = target;
            localEntries.clear();
            localEntries.addAll(next);
            selectedLocal = -1;
            localPath.setText(currentDocumentId.equals(rootDocumentId) ? "Selected folder" : currentDocumentId);
            renderLocal();
        } catch (IOException e) {
            setStatus("Parent folder is unavailable; current path was not changed: " + safeMessage(e));
        }
    }

    private void createLocalFolder() {
        if (busy || treeUri == null || currentDocumentId == null) return;
        promptName("New local folder", "", "Create", name -> {
            String safe;
            try {
                safe = requireLeafName(name);
            } catch (IllegalArgumentException e) {
                setStatus(e.getMessage());
                return;
            }
            Uri selectedTree = treeUri;
            String selectedParent = currentDocumentId;
            Uri parent = DocumentsContract.buildDocumentUriUsingTree(selectedTree, selectedParent);
            setBusy(true, "Creating local folder…");
            io.execute(() -> {
                Uri created = null;
                try {
                    ensureNoLocalNameConflict(selectedTree, selectedParent, safe, "A local item with this name already exists.");
                    created = DocumentsContract.createDocument(getContentResolver(), parent, DocumentsContract.Document.MIME_TYPE_DIR, safe);
                    if (created == null) throw new IOException("Storage provider rejected local folder creation.");
                    String actual = queryDocumentDisplayName(created);
                    if (!safe.equals(actual)) {
                        try { DocumentsContract.deleteDocument(getContentResolver(), created); } catch (Exception ignored) { }
                        throw new IOException("Storage provider changed the requested folder name; creation was rejected.");
                    }
                    List<LocalEntry> fresh = queryChildren(selectedTree, selectedParent);
                    runOnUiThread(() -> commitLocalMutation(selectedTree, selectedParent, fresh, "Local folder created: " + safe));
                } catch (Exception e) {
                    postError("Create local folder failed", e);
                }
            });
        });
    }

    private void renameLocalSelected() {
        if (busy || treeUri == null || currentDocumentId == null || selectedLocal < 0 || selectedLocal >= localEntries.size()) return;
        LocalEntry entry = localEntries.get(selectedLocal);
        promptName("Rename local item", entry.name, "Rename", name -> {
            String safe;
            try {
                safe = requireLeafName(name);
            } catch (IllegalArgumentException e) {
                setStatus(e.getMessage());
                return;
            }
            if (safe.equals(entry.name)) return;
            Uri selectedTree = treeUri;
            String selectedParent = currentDocumentId;
            Uri document = DocumentsContract.buildDocumentUriUsingTree(selectedTree, entry.documentId);
            setBusy(true, "Renaming local item…");
            io.execute(() -> {
                Uri renamed = null;
                try {
                    ensureNoLocalNameConflict(selectedTree, selectedParent, safe, "A local item with this name already exists.");
                    renamed = DocumentsContract.renameDocument(getContentResolver(), document, safe);
                    if (renamed == null) throw new IOException("Storage provider rejected local rename.");
                    String actual = queryDocumentDisplayName(renamed);
                    if (!safe.equals(actual)) throw new IOException("Storage provider changed the requested rename result.");
                    List<LocalEntry> fresh = queryChildren(selectedTree, selectedParent);
                    runOnUiThread(() -> commitLocalMutation(selectedTree, selectedParent, fresh, "Local item renamed to: " + safe));
                } catch (Exception e) {
                    postError("Rename local item failed", e);
                }
            });
        });
    }

    private void deleteLocalSelected() {
        if (busy || treeUri == null || currentDocumentId == null || selectedLocal < 0 || selectedLocal >= localEntries.size()) return;
        LocalEntry entry = localEntries.get(selectedLocal);
        confirm("Delete local item?", "Delete “" + entry.name + "” from the selected Android storage provider?", () -> {
            Uri selectedTree = treeUri;
            String selectedParent = currentDocumentId;
            Uri document = DocumentsContract.buildDocumentUriUsingTree(selectedTree, entry.documentId);
            setBusy(true, "Deleting local item…");
            io.execute(() -> {
                try {
                    boolean deleted = DocumentsContract.deleteDocument(getContentResolver(), document);
                    if (!deleted) throw new IOException("Storage provider rejected local delete.");
                    List<LocalEntry> fresh = queryChildren(selectedTree, selectedParent);
                    runOnUiThread(() -> commitLocalMutation(selectedTree, selectedParent, fresh, "Local item deleted: " + entry.name));
                } catch (Exception e) {
                    postError("Delete local item failed", e);
                }
            });
        });
    }

    private void commitLocalMutation(Uri selectedTree, String selectedParent, List<LocalEntry> fresh, String message) {
        if (treeUri == null || !treeUri.equals(selectedTree) || !selectedParent.equals(currentDocumentId)) {
            setBusy(false, "Local operation completed, but the visible folder changed. Refresh before continuing.");
            return;
        }
        localEntries.clear();
        localEntries.addAll(fresh);
        selectedLocal = -1;
        renderLocal();
        setBusy(false, message);
    }

    private void setLocalStart() {
        SiteProfile profile = activeProfile();
        if (profile == null) {
            setStatus("Load or save a site first. Quick Connect does not create hidden site state.");
            return;
        }
        String uri = persistedCurrentTreeUri();
        if (uri.isEmpty()) {
            setStatus("Choose a folder with persistent Android permission before setting the site start folder.");
            return;
        }
        SiteProfile next = profile.withLocalStartTreeUri(uri);
        replaceProfile(next);
        profileStore.save(profiles);
        setStatus("Local site start folder saved as a SAF capability URI; no filesystem-wide permission was added.");
    }

    private void addLocalBookmark() {
        SiteProfile profile = activeProfile();
        if (profile == null) {
            setStatus("Load or save a site first. Quick Connect does not create hidden bookmarks.");
            return;
        }
        String uri = persistedCurrentTreeUri();
        if (uri.isEmpty()) {
            setStatus("Choose a folder with persistent Android permission before bookmarking it.");
            return;
        }
        SiteProfile next = profile.withLocalBookmark(uri);
        replaceProfile(next);
        profileStore.save(profiles);
        renderBookmarks();
        setStatus("Local SAF bookmark added. It contains no credentials.");
    }

    private void removeLocalBookmark() {
        SiteProfile profile = activeProfile();
        if (profile == null) {
            setStatus("Load a saved site before removing a local bookmark.");
            return;
        }
        int index = localBookmarkSpinner.getSelectedItemPosition();
        if (index < 0 || index >= profile.localBookmarks.size()) {
            setStatus("No local bookmark selected.");
            return;
        }
        SiteProfile next = profile.withoutLocalBookmark(index);
        replaceProfile(next);
        profileStore.save(profiles);
        renderBookmarks();
        setStatus("Local bookmark removed.");
    }

    private void openLocalBookmark() {
        SiteProfile profile = activeProfile();
        if (profile == null) {
            setStatus("Load a saved site first.");
            return;
        }
        int index = localBookmarkSpinner.getSelectedItemPosition();
        if (index < 0 || index >= profile.localBookmarks.size()) {
            setStatus("No local bookmark selected.");
            return;
        }
        Uri uri = Uri.parse(profile.localBookmarks.get(index));
        if (!tryActivateLocalTree(
                uri,
                "Local bookmark is stale or its persisted permission is unavailable. Re-select the folder to restore access.",
                true)) {
            return;
        }
        savePreferences();
        setStatus("Local bookmark opened after persisted SAF permission and directory listing were revalidated.");
    }

    private void uploadSelected() {
        FtpSession current = session;
        if (busy || current == null || selectedLocal < 0 || selectedLocal >= localEntries.size()) return;
        LocalEntry entry = localEntries.get(selectedLocal);
        if (entry.directory) {
            setStatus("Select a local file, not a directory, to upload.");
            return;
        }
        Uri document = DocumentsContract.buildDocumentUriUsingTree(treeUri, entry.documentId);
        String remoteBase = currentRemotePath;
        long transferToken = beginTransfer("Uploading " + entry.name + "…");
        TransferProgress progress = transferProgress(entry.size, "Uploading", current, transferToken);
        io.execute(() -> {
            try {
                requireTransferCurrent(transferToken);
                InputStream source = getContentResolver().openInputStream(document);
                if (source == null) throw new IOException("Could not open local file.");
                try (InputStream in = ProgressStreams.input(source, progress::onTransferred)) {
                    requireTransferCurrent(transferToken);
                    current.upload(FtpSession.joinRemote(remoteBase, entry.name), in);
                }
                transferActive = false;
                runOnUiThread(() -> {
                    if (transferCancelled(transferToken) || session != current || !current.isConnected()) {
                        setBusy(false, "Upload commit raced with cancellation or connection loss. Reconnect and refresh the server before retrying.");
                        return;
                    }
                    setBusy(false, "Upload completed: " + entry.name);
                    refreshRemote(currentRemotePath);
                });
            } catch (Exception e) {
                finishTransferFailure(transferToken, current, "Upload failed", e, false);
            }
        });
    }

    private void downloadSelected() {
        FtpSession current = session;
        if (busy || current == null || selectedRemote < 0 || selectedRemote >= remoteEntries.size() || treeUri == null) return;
        RemoteEntry entry = remoteEntries.get(selectedRemote);
        if (entry.directory) {
            setStatus("Select a server file, not a directory, to download.");
            return;
        }
        Uri selectedTree = treeUri;
        String selectedDocumentId = currentDocumentId;
        Uri parent = DocumentsContract.buildDocumentUriUsingTree(selectedTree, selectedDocumentId);
        String remoteBase = currentRemotePath;
        long transferToken = beginTransfer("Downloading " + entry.name + " to a staged local document…");
        TransferProgress progress = transferProgress(entry.size, "Downloading", current, transferToken);
        io.execute(() -> {
            Uri staged = null;
            try {
                requireTransferCurrent(transferToken);
                ensureNoLocalNameConflict(selectedTree, selectedDocumentId, entry.name,
                        "A local item with this exact name already exists. Remove or rename it before downloading.");
                requireTransferCurrent(transferToken);

                String stagedName = ".ghostftp-download-" + UUID.randomUUID() + ".part";
                staged = DocumentsContract.createDocument(getContentResolver(), parent, "application/octet-stream", stagedName);
                if (staged == null) throw new IOException("Could not create staged local download document.");

                requireTransferCurrent(transferToken);
                OutputStream destination = getContentResolver().openOutputStream(staged, "w");
                if (destination == null) throw new IOException("Could not open staged local download document.");
                try (OutputStream out = ProgressStreams.output(destination, progress::onTransferred)) {
                    current.download(FtpSession.joinRemote(remoteBase, entry.name), out);
                }
                requireTransferCurrent(transferToken);

                ensureNoLocalNameConflict(selectedTree, selectedDocumentId, entry.name,
                        "A local item with the destination name appeared during download; staged data was not committed.");
                requireTransferCurrent(transferToken);

                Uri committed = DocumentsContract.renameDocument(getContentResolver(), staged, entry.name);
                if (committed == null) throw new IOException("Storage provider rejected the final download name commit.");
                staged = committed;
                String committedName = queryDocumentDisplayName(committed);
                if (!entry.name.equals(committedName)) {
                    throw new IOException("Storage provider changed the requested final download name; commit was rejected.");
                }
                staged = null;
                transferActive = false;

                runOnUiThread(() -> {
                    if (transferCancelled(transferToken) || session != current || !current.isConnected()) {
                        refreshLocal();
                        setBusy(false, "Download committed before cancellation or connection loss took effect. Connection closed; reconnect before another transfer.");
                        return;
                    }
                    setBusy(false, "Download completed and committed: " + entry.name);
                    refreshLocal();
                });
            } catch (Exception e) {
                if (staged != null) {
                    try { DocumentsContract.deleteDocument(getContentResolver(), staged); } catch (Exception ignored) { }
                }
                finishTransferFailure(transferToken, current, "Download failed", e, true);
            }
        });
    }

    private TransferProgress transferProgress(long totalBytes, String action, FtpSession current, long token) {
        return new TransferProgress(totalBytes, action, text -> runOnUiThread(() -> {
            if (!transferActive || transferCancelled(token) || session != current || !current.isConnected()) return;
            setStatus(text);
        }));
    }

    private long beginTransfer(String message) {
        transferActive = true;
        long token = ++transferGeneration;
        setBusy(true, message);
        return token;
    }

    private boolean transferCancelled(long token) {
        return token != transferGeneration;
    }

    private void requireTransferCurrent(long token) throws IOException {
        if (transferCancelled(token)) {
            throw new IOException("Transfer cancelled.");
        }
    }

    private void finishTransferFailure(long token, FtpSession current, String prefix, Exception e, boolean refreshLocalAfter) {
        boolean cancelled = transferCancelled(token);
        transferActive = false;
        runOnUiThread(() -> {
            if (session == current && !current.isConnected()) {
                session = null;
                connectedIdentityKey = null;
                remoteEntries.clear();
                selectedRemote = -1;
                currentRemotePath = "/";
                renderRemote();
            }
            if (refreshLocalAfter) refreshLocal();
            setBusy(false, cancelled
                    ? "Transfer cancelled. Connection closed; reconnect before another transfer."
                    : prefix + ": " + safeMessage(e));
        });
    }

    private void ensureNoLocalNameConflict(Uri rootTreeUri, String documentId, String name, String message) throws IOException {
        List<LocalEntry> fresh = queryChildren(rootTreeUri, documentId);
        for (LocalEntry local : fresh) {
            if (name.equals(local.name)) throw new IOException(message);
        }
    }

    private String queryDocumentDisplayName(Uri document) throws IOException {
        String[] projection = {DocumentsContract.Document.COLUMN_DISPLAY_NAME};
        try (Cursor cursor = getContentResolver().query(document, projection, null, null, null)) {
            if (cursor == null || !cursor.moveToFirst()) {
                throw new IOException("Storage provider could not verify the committed item name.");
            }
            String name = cursor.getString(0);
            if (name == null || name.isEmpty()) {
                throw new IOException("Storage provider returned an empty committed item name.");
            }
            return name;
        } catch (SecurityException e) {
            throw new IOException("Storage permission was lost while verifying the committed item.", e);
        }
    }

    private void clearLocalRoot() {
        treeUri = null;
        rootDocumentId = null;
        currentDocumentId = null;
        localParents.clear();
        localEntries.clear();
        selectedLocal = -1;
    }

    private void renderLocal() {
        List<String> labels = new ArrayList<>();
        for (int i = 0; i < localEntries.size(); i++) {
            LocalEntry e = localEntries.get(i);
            labels.add((i == selectedLocal ? "●  " : "   ") + (e.directory ? "▸  " : "   ") + e.name + (e.directory ? "" : "   " + TransferProgress.formatBytes(e.size)));
        }
        localList.setAdapter(GhostTheme.listAdapter(this, labels));
        refreshButtons();
    }

    private void renderRemote() {
        remotePath.setText(currentRemotePath);
        List<String> labels = new ArrayList<>();
        for (int i = 0; i < remoteEntries.size(); i++) {
            RemoteEntry e = remoteEntries.get(i);
            labels.add((i == selectedRemote ? "●  " : "   ") + (e.directory ? "▸  " : "   ") + e.name + (e.directory ? "" : "   " + TransferProgress.formatBytes(e.size)));
        }
        remoteList.setAdapter(GhostTheme.listAdapter(this, labels));
        refreshButtons();
    }

    private void setBusy(boolean value, String message) {
        busy = value;
        setStatus(message);
        refreshButtons();
    }

    private void refreshButtons() {
        boolean connected = session != null && session.isConnected();
        boolean selectedLocalItem = selectedLocal >= 0 && selectedLocal < localEntries.size();
        boolean selectedRemoteItem = selectedRemote >= 0 && selectedRemote < remoteEntries.size();
        boolean selectedLocalFile = selectedLocalItem && !localEntries.get(selectedLocal).directory;
        boolean selectedRemoteFile = selectedRemoteItem && !remoteEntries.get(selectedRemote).directory;
        SiteProfile profile = activeProfile();
        boolean activeSite = profile != null;
        boolean connectedSite = activeSite && connected && connectedIdentityKey != null && profile.identityKey().equals(connectedIdentityKey);

        setEnabled(connect, !busy && !connected);
        disconnect.setText(transferActive ? "Cancel transfer" : "Disconnect");
        setEnabled(disconnect, transferActive || (!busy && connected));
        setEnabled(upload, !busy && connected && selectedLocalFile);
        setEnabled(download, !busy && connected && selectedRemoteFile && treeUri != null);
        setEnabled(saveSite, !busy && !connected);
        setEnabled(deleteSite, !busy && !connected && activeSite);
        setEnabled(localNewFolder, !busy && treeUri != null && currentDocumentId != null);
        setEnabled(localRename, !busy && selectedLocalItem);
        setEnabled(localDelete, !busy && selectedLocalItem);
        setEnabled(localSetStart, !busy && activeSite && !persistedCurrentTreeUri().isEmpty());
        setEnabled(localAddBookmark, !busy && activeSite && !persistedCurrentTreeUri().isEmpty());
        setEnabled(localRemoveBookmark, !busy && activeSite && !profile.localBookmarks.isEmpty());
        setEnabled(localOpenBookmark, !busy && activeSite && !profile.localBookmarks.isEmpty());
        setEnabled(remoteNewFolder, !busy && connected);
        setEnabled(remoteRename, !busy && connected && selectedRemoteItem);
        setEnabled(remoteDelete, !busy && connected && selectedRemoteItem);
        setEnabled(remoteSetStart, !busy && connectedSite);
        setEnabled(remoteAddBookmark, !busy && connectedSite);
        setEnabled(remoteRemoveBookmark, !busy && activeSite && !profile.remoteBookmarks.isEmpty());
        setEnabled(remoteOpenBookmark, !busy && connectedSite && !profile.remoteBookmarks.isEmpty());

        if (transferActive) {
            connectionState.setText("TRANSFER ACTIVE");
            GhostTheme.styleBadge(connectionState, GhostTheme.WARN);
        } else if (connected) {
            boolean secure = "FTPS".equals(protocol.getSelectedItem().toString());
            connectionState.setText(secure ? "FTPS CONNECTED" : "FTP CONNECTED");
            GhostTheme.styleBadge(connectionState, secure ? GhostTheme.SUCCESS : GhostTheme.WARN);
        } else {
            connectionState.setText("DISCONNECTED");
            GhostTheme.styleBadge(connectionState, GhostTheme.MUTED);
        }
    }

    private int parsePort() {
        try {
            int value = Integer.parseInt(port.getText().toString().trim());
            if (value < 1 || value > 65535) throw new NumberFormatException();
            return value;
        } catch (NumberFormatException e) {
            throw new IllegalArgumentException("Port must be between 1 and 65535.");
        }
    }

    private static String identityKey(String protocol, String host, int port, String username) {
        return protocol.trim().toUpperCase(java.util.Locale.ROOT) + "\n"
                + host.trim().toLowerCase(java.util.Locale.ROOT) + "\n"
                + port + "\n" + (username == null ? "" : username.trim());
    }

    private void postError(String prefix, Exception e) {
        runOnUiThread(() -> {
            FtpSession current = session;
            if (current != null && !current.isConnected()) {
                session = null;
                connectedIdentityKey = null;
                remoteEntries.clear();
                selectedRemote = -1;
                currentRemotePath = "/";
                renderRemote();
            }
            setBusy(false, prefix + ": " + safeMessage(e));
        });
    }

    private void setStatus(String value) {
        String safe = value == null ? "" : value.replace('\n', ' ').replace('\r', ' ');
        status.setText(safe);
        status.setTextColor(GhostTheme.statusColor(safe));
    }

    private static String safeMessage(Exception e) {
        String value = e.getMessage();
        return value == null || value.trim().isEmpty() ? e.getClass().getSimpleName() : value.replace('\n', ' ').replace('\r', ' ');
    }

    private LinearLayout card(String title, String hint) {
        LinearLayout card = new LinearLayout(this);
        card.setOrientation(LinearLayout.VERTICAL);
        card.setPadding(dp(14), dp(14), dp(14), dp(14));
        card.setBackground(GhostTheme.rounded(this, GhostTheme.PANEL, GhostTheme.BORDER, 14));
        TextView heading = label(title, 15, GhostTheme.TEXT);
        heading.setTypeface(Typeface.DEFAULT, Typeface.BOLD);
        card.addView(heading, matchWrap());
        if (hint != null && !hint.isEmpty()) {
            TextView help = label(hint, 11, GhostTheme.MUTED);
            help.setPadding(0, dp(4), 0, dp(10));
            card.addView(help, matchWrap());
        }
        return card;
    }

    private TextView label(String text, int sp, int color) {
        TextView view = new TextView(this);
        view.setText(text);
        view.setTextSize(sp);
        view.setTextColor(color);
        return view;
    }

    private EditText field(String hint, boolean secret) {
        EditText edit = new EditText(this);
        edit.setHint(hint);
        GhostTheme.styleField(edit);
        if (secret) edit.setInputType(InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_VARIATION_PASSWORD);
        return edit;
    }

    private Button button(String text) {
        Button b = new Button(this);
        b.setText(text);
        GhostTheme.styleSecondaryButton(b);
        return b;
    }

    private Button primaryButton(String text) {
        Button b = new Button(this);
        b.setText(text);
        GhostTheme.stylePrimaryButton(b);
        return b;
    }

    private Button dangerButton(String text) {
        Button b = new Button(this);
        b.setText(text);
        GhostTheme.styleDangerButton(b);
        return b;
    }

    private LinearLayout row() {
        LinearLayout r = new LinearLayout(this);
        r.setOrientation(LinearLayout.HORIZONTAL);
        return r;
    }

    private void promptName(String title, String initial, String positive, NameAction action) {
        EditText input = field("Name", false);
        input.setText(initial == null ? "" : initial);
        input.setSelectAllOnFocus(true);
        LinearLayout holder = new LinearLayout(this);
        holder.setPadding(dp(18), dp(4), dp(18), 0);
        holder.addView(input, matchWrap());
        new AlertDialog.Builder(this)
                .setTitle(title)
                .setView(holder)
                .setNegativeButton("Cancel", null)
                .setPositiveButton(positive, (dialog, which) -> action.run(input.getText().toString()))
                .show();
    }

    private void confirm(String title, String message, Runnable action) {
        new AlertDialog.Builder(this)
                .setTitle(title)
                .setMessage(message)
                .setNegativeButton("Cancel", null)
                .setPositiveButton("Delete", (dialog, which) -> action.run())
                .show();
    }

    private static String requireLeafName(String value) {
        String name = value == null ? "" : value.trim();
        if (name.isEmpty() || ".".equals(name) || "..".equals(name)
                || name.indexOf('/') >= 0 || name.indexOf('\\') >= 0
                || name.indexOf('\0') >= 0 || name.indexOf('\r') >= 0 || name.indexOf('\n') >= 0) {
            throw new IllegalArgumentException("Name must be one safe path segment without / or \\ characters.");
        }
        return name;
    }

    private void setEnabled(Button button, boolean enabled) {
        if (button == null) return;
        button.setEnabled(enabled);
        button.setAlpha(enabled ? 1f : 0.45f);
    }

    private LinearLayout.LayoutParams weightedSpaced() {
        LinearLayout.LayoutParams params = new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f);
        params.setMargins(dp(3), dp(3), dp(3), dp(3));
        return params;
    }

    private LinearLayout.LayoutParams fixedWidthSpaced(int widthDp) {
        LinearLayout.LayoutParams params = new LinearLayout.LayoutParams(dp(widthDp), ViewGroup.LayoutParams.WRAP_CONTENT);
        params.setMargins(dp(3), dp(3), dp(3), dp(3));
        return params;
    }

    private LinearLayout.LayoutParams matchWrapSpaced() {
        LinearLayout.LayoutParams params = matchWrap();
        params.setMargins(0, dp(4), 0, dp(4));
        return params;
    }

    private LinearLayout.LayoutParams cardParams() {
        LinearLayout.LayoutParams params = matchWrap();
        params.setMargins(0, 0, 0, dp(10));
        return params;
    }

    private LinearLayout.LayoutParams matchWrap() {
        return new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT);
    }

    private LinearLayout.LayoutParams wrapWrap() {
        return new LinearLayout.LayoutParams(ViewGroup.LayoutParams.WRAP_CONTENT, ViewGroup.LayoutParams.WRAP_CONTENT);
    }

    private int dp(int value) {
        return GhostTheme.dp(this, value);
    }

    private interface NameAction {
        void run(String name);
    }

    private static final class LocalEntry {
        final String documentId;
        final String name;
        final boolean directory;
        final long size;

        LocalEntry(String documentId, String name, boolean directory, long size) {
            this.documentId = documentId;
            this.name = name;
            this.directory = directory;
            this.size = size;
        }
    }
}
