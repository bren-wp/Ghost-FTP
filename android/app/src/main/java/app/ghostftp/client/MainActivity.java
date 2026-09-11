package app.ghostftp.client;

import android.annotation.SuppressLint;
import android.app.Activity;
import android.content.Intent;
import android.content.SharedPreferences;
import android.content.UriPermission;
import android.database.Cursor;
import android.graphics.Color;
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
    private Button localOpenBookmark;
    private Button remoteSetStart;
    private Button remoteAddBookmark;
    private Button remoteOpenBookmark;

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
    private boolean transferActive;
    private long transferGeneration;

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
        FtpSession current = session;
        transferGeneration++;
        transferActive = false;
        session = null;
        connectedIdentityKey = null;
        if (current != null) current.cancelActiveTransfer();
        io.shutdownNow();
        super.onDestroy();
    }

    private void buildUi() {
        LinearLayout root = new LinearLayout(this);
        root.setOrientation(LinearLayout.VERTICAL);
        int pad = dp(14);
        root.setPadding(pad, pad, pad, pad);
        root.setBackgroundColor(Color.rgb(16, 19, 23));
        root.addView(label("GHOST FTP · ANDROID", 22, Color.WHITE));
        root.addView(label("Private FTP/FTPS client · no telemetry · passwords are never saved", 13, Color.LTGRAY));

        root.addView(section("SAVED SITES"));
        siteSpinner = new Spinner(this);
        root.addView(siteSpinner, matchWrap());
        profileName = field("Site name", false);
        root.addView(profileName, matchWrap());
        LinearLayout siteActions = row();
        Button loadSite = button("Load");
        saveSite = button("Save / update");
        deleteSite = button("Delete");
        siteActions.addView(loadSite, weighted());
        siteActions.addView(saveSite, weighted());
        siteActions.addView(deleteSite, weighted());
        root.addView(siteActions, matchWrap());
        loadSite.setOnClickListener(v -> loadSelectedSite());
        saveSite.setOnClickListener(v -> saveOrUpdateSite());
        deleteSite.setOnClickListener(v -> deleteActiveSite());

        root.addView(section("QUICK CONNECT / SITE CONNECTION"));
        protocol = new Spinner(this);
        protocol.setAdapter(new ArrayAdapter<>(this, android.R.layout.simple_spinner_dropdown_item, new String[]{"FTPS", "FTP"}));
        root.addView(protocol, matchWrap());
        host = field("Server host", false);
        port = field("Port", false);
        port.setInputType(InputType.TYPE_CLASS_NUMBER);
        username = field("Username", false);
        password = field("Password (memory only)", true);
        root.addView(host, matchWrap());
        root.addView(port, matchWrap());
        root.addView(username, matchWrap());
        root.addView(password, matchWrap());

        LinearLayout connection = row();
        connect = button("Connect");
        disconnect = button("Disconnect");
        connection.addView(connect, weighted());
        connection.addView(disconnect, weighted());
        root.addView(connection, matchWrap());
        connect.setOnClickListener(v -> connect());
        disconnect.setOnClickListener(v -> disconnect());

        root.addView(section("LOCAL STORAGE"));
        localPath = label("No folder selected", 14, Color.LTGRAY);
        root.addView(localPath);
        LinearLayout localActions = row();
        Button choose = button("Choose folder");
        Button localUp = button("Up");
        localActions.addView(choose, weighted());
        localActions.addView(localUp, weighted());
        root.addView(localActions, matchWrap());
        choose.setOnClickListener(v -> chooseFolder());
        localUp.setOnClickListener(v -> localUp());

        LinearLayout localProfileActions = row();
        localSetStart = button("Set site start");
        localAddBookmark = button("Add bookmark");
        localProfileActions.addView(localSetStart, weighted());
        localProfileActions.addView(localAddBookmark, weighted());
        root.addView(localProfileActions, matchWrap());
        localSetStart.setOnClickListener(v -> setLocalStart());
        localAddBookmark.setOnClickListener(v -> addLocalBookmark());
        localBookmarkSpinner = new Spinner(this);
        root.addView(localBookmarkSpinner, matchWrap());
        localOpenBookmark = button("Open local bookmark");
        root.addView(localOpenBookmark, matchWrap());
        localOpenBookmark.setOnClickListener(v -> openLocalBookmark());

        localList = new ListView(this);
        root.addView(localList, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, dp(220)));
        localList.setOnItemClickListener((parent, view, position, id) -> selectLocal(position));

        root.addView(section("SERVER"));
        remotePath = label(currentRemotePath, 14, Color.LTGRAY);
        root.addView(remotePath);
        LinearLayout remoteActions = row();
        Button remoteRefresh = button("Refresh");
        Button remoteUp = button("Up");
        remoteActions.addView(remoteRefresh, weighted());
        remoteActions.addView(remoteUp, weighted());
        root.addView(remoteActions, matchWrap());
        remoteRefresh.setOnClickListener(v -> refreshRemote(currentRemotePath));
        remoteUp.setOnClickListener(v -> remoteUp());

        LinearLayout remoteProfileActions = row();
        remoteSetStart = button("Set site start");
        remoteAddBookmark = button("Add bookmark");
        remoteProfileActions.addView(remoteSetStart, weighted());
        remoteProfileActions.addView(remoteAddBookmark, weighted());
        root.addView(remoteProfileActions, matchWrap());
        remoteSetStart.setOnClickListener(v -> setRemoteStart());
        remoteAddBookmark.setOnClickListener(v -> addRemoteBookmark());
        remoteBookmarkSpinner = new Spinner(this);
        root.addView(remoteBookmarkSpinner, matchWrap());
        remoteOpenBookmark = button("Open server bookmark");
        root.addView(remoteOpenBookmark, matchWrap());
        remoteOpenBookmark.setOnClickListener(v -> openRemoteBookmark());

        remoteList = new ListView(this);
        root.addView(remoteList, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, dp(220)));
        remoteList.setOnItemClickListener((parent, view, position, id) -> selectRemote(position));

        LinearLayout transfers = row();
        upload = button("Upload →");
        download = button("← Download");
        transfers.addView(upload, weighted());
        transfers.addView(download, weighted());
        root.addView(transfers, matchWrap());
        upload.setOnClickListener(v -> uploadSelected());
        download.setOnClickListener(v -> downloadSelected());

        status = label("Ready.", 13, Color.LTGRAY);
        root.addView(status);
        ScrollView scroll = new ScrollView(this);
        scroll.addView(root);
        setContentView(scroll);
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
        siteSpinner.setAdapter(new ArrayAdapter<>(this, android.R.layout.simple_spinner_dropdown_item, labels));
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
        localBookmarkSpinner.setAdapter(new ArrayAdapter<>(this, android.R.layout.simple_spinner_dropdown_item, local));
        remoteBookmarkSpinner.setAdapter(new ArrayAdapter<>(this, android.R.layout.simple_spinner_dropdown_item, remote));
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
        profiles.remove(profile);
        profileStore.save(profiles);
        activeProfileId = null;
        profileName.setText("");
        renderSites();
        renderBookmarks();
        setStatus("Saved site deleted. Quick Connect settings were not converted into another profile.");
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
            cancelActiveTransfer();
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
        if (current != null) current.close();
        setStatus("Disconnected.");
        refreshButtons();
    }

    private void cancelActiveTransfer() {
        FtpSession current = session;
        if (!transferActive || current == null) return;
        transferGeneration++;
        transferActive = false;
        session = null;
        connectedIdentityKey = null;
        remoteEntries.clear();
        selectedRemote = -1;
        currentRemotePath = "/";
        current.cancelActiveTransfer();
        renderRemote();
        setBusy(false, "Transfer cancelled. Connection closed; reconnect before continuing.");
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
        Uri document = DocumentsContract.buildDocumentUriUsingTree(treeUri, entry.documentId);
        String remoteBase = currentRemotePath;
        long generation = beginTransfer("Uploading " + entry.name + "…");
        io.execute(() -> {
            try (InputStream in = getContentResolver().openInputStream(document)) {
                if (in == null) throw new IOException("Could not open local file.");
                current.upload(FtpSession.joinRemote(remoteBase, entry.name), in);
                runOnUiThread(() -> {
                    if (!transferIsCurrent(current, generation)) return;
                    transferActive = false;
                    setBusy(false, "Upload completed: " + entry.name);
                    refreshRemote(currentRemotePath);
                });
            } catch (Exception e) {
                postTransferError("Upload failed", current, generation, e);
            }
        });
    }

    private void downloadSelected() {
        FtpSession current = session;
        if (busy || current == null || selectedRemote < 0 || selectedRemote >= remoteEntries.size() || treeUri == null) return;
        RemoteEntry entry = remoteEntries.get(selectedRemote);
        Uri selectedTree = treeUri;
        String selectedDocumentId = currentDocumentId;
        Uri parent = DocumentsContract.buildDocumentUriUsingTree(selectedTree, selectedDocumentId);
        String remoteBase = currentRemotePath;
        long generation = beginTransfer("Downloading " + entry.name + " to a staged local document…");
        io.execute(() -> {
            Uri staged = null;
            try {
                ensureNoLocalNameConflict(selectedTree, selectedDocumentId, entry.name,
                        "A local item with this exact name already exists. Remove or rename it before downloading.");

                String stagedName = ".ghostftp-download-" + UUID.randomUUID() + ".part";
                staged = DocumentsContract.createDocument(getContentResolver(), parent, "application/octet-stream", stagedName);
                if (staged == null) throw new IOException("Could not create staged local download document.");

                try (OutputStream out = getContentResolver().openOutputStream(staged, "w")) {
                    if (out == null) throw new IOException("Could not open staged local download document.");
                    current.download(FtpSession.joinRemote(remoteBase, entry.name), out);
                }

                ensureNoLocalNameConflict(selectedTree, selectedDocumentId, entry.name,
                        "A local item with the destination name appeared during download; staged data was not committed.");

                Uri committed = DocumentsContract.renameDocument(getContentResolver(), staged, entry.name);
                if (committed == null) throw new IOException("Storage provider rejected the final download name commit.");
                staged = committed;
                String committedName = queryDocumentDisplayName(committed);
                if (!entry.name.equals(committedName)) {
                    throw new IOException("Storage provider changed the requested final download name; commit was rejected.");
                }
                staged = null;

                runOnUiThread(() -> {
                    if (!transferIsCurrent(current, generation)) return;
                    transferActive = false;
                    setBusy(false, "Download completed and committed: " + entry.name);
                    refreshLocal();
                });
            } catch (Exception e) {
                if (staged != null) {
                    try { DocumentsContract.deleteDocument(getContentResolver(), staged); } catch (Exception ignored) { }
                }
                postTransferError("Download failed", current, generation, e);
            }
        });
    }

    private long beginTransfer(String message) {
        transferGeneration++;
        transferActive = true;
        setBusy(true, message);
        return transferGeneration;
    }

    private boolean transferIsCurrent(FtpSession current, long generation) {
        return transferActive && transferGeneration == generation && session == current;
    }

    private void postTransferError(String prefix, FtpSession current, long generation, Exception e) {
        runOnUiThread(() -> {
            if (transferGeneration != generation) return;
            transferActive = false;
            if (session == current && !current.isConnected()) {
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
                throw new IOException("Storage provider could not verify the committed download name.");
            }
            String name = cursor.getString(0);
            if (name == null || name.isEmpty()) {
                throw new IOException("Storage provider returned an empty committed download name.");
            }
            return name;
        } catch (SecurityException e) {
            throw new IOException("Storage permission was lost while verifying the committed download.", e);
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
            labels.add((i == selectedLocal ? "› " : "  ") + (e.directory ? "DIR   " : "FILE  ") + e.name + (e.directory ? "" : "  (" + e.size + " B)"));
        }
        localList.setAdapter(new ArrayAdapter<>(this, android.R.layout.simple_list_item_1, labels));
        refreshButtons();
    }

    private void renderRemote() {
        remotePath.setText(currentRemotePath);
        List<String> labels = new ArrayList<>();
        for (int i = 0; i < remoteEntries.size(); i++) labels.add((i == selectedRemote ? "› " : "  ") + remoteEntries.get(i));
        remoteList.setAdapter(new ArrayAdapter<>(this, android.R.layout.simple_list_item_1, labels));
        refreshButtons();
    }

    private void setBusy(boolean value, String message) {
        busy = value;
        setStatus(message);
        refreshButtons();
    }

    private void refreshButtons() {
        boolean connected = session != null && session.isConnected();
        boolean canCancelTransfer = transferActive && session != null;
        SiteProfile profile = activeProfile();
        boolean activeSite = profile != null;
        boolean connectedSite = activeSite && connected && connectedIdentityKey != null && profile.identityKey().equals(connectedIdentityKey);
        connect.setEnabled(!busy && !connected);
        disconnect.setText(canCancelTransfer ? "Cancel transfer" : "Disconnect");
        disconnect.setEnabled(canCancelTransfer || (!busy && connected));
        upload.setEnabled(!busy && connected && selectedLocal >= 0);
        download.setEnabled(!busy && connected && selectedRemote >= 0 && treeUri != null);
        saveSite.setEnabled(!busy && !connected);
        deleteSite.setEnabled(!busy && !connected && activeSite);
        localSetStart.setEnabled(!busy && activeSite && persistedCurrentTreeUri().length() > 0);
        localAddBookmark.setEnabled(!busy && activeSite && persistedCurrentTreeUri().length() > 0);
        localOpenBookmark.setEnabled(!busy && activeSite && !profile.localBookmarks.isEmpty());
        remoteSetStart.setEnabled(!busy && connectedSite);
        remoteAddBookmark.setEnabled(!busy && connectedSite);
        remoteOpenBookmark.setEnabled(!busy && connectedSite && !profile.remoteBookmarks.isEmpty());
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
        runOnUiThread(() -> setBusy(false, prefix + ": " + safeMessage(e)));
    }

    private void setStatus(String value) {
        status.setText(value == null ? "" : value.replace('\n', ' ').replace('\r', ' '));
    }

    private static String safeMessage(Exception e) {
        String value = e.getMessage();
        return value == null || value.trim().isEmpty() ? e.getClass().getSimpleName() : value.replace('\n', ' ').replace('\r', ' ');
    }

    private TextView section(String text) {
        TextView view = label(text, 15, Color.rgb(94, 214, 200));
        view.setPadding(0, dp(18), 0, dp(6));
        return view;
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
        edit.setTextColor(Color.WHITE);
        edit.setHintTextColor(Color.GRAY);
        edit.setSingleLine(true);
        if (secret) edit.setInputType(InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_VARIATION_PASSWORD);
        return edit;
    }

    private Button button(String text) {
        Button b = new Button(this);
        b.setText(text);
        b.setAllCaps(false);
        return b;
    }

    private LinearLayout row() {
        LinearLayout r = new LinearLayout(this);
        r.setOrientation(LinearLayout.HORIZONTAL);
        return r;
    }

    private LinearLayout.LayoutParams weighted() { return new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f); }
    private LinearLayout.LayoutParams matchWrap() { return new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT); }
    private int dp(int value) { return Math.round(value * getResources().getDisplayMetrics().density); }

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
