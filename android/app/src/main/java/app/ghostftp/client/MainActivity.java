package app.ghostftp.client;

import android.annotation.SuppressLint;
import android.app.Activity;
import android.content.Intent;
import android.content.SharedPreferences;
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

    private Spinner protocol;
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

    private Uri treeUri;
    private String rootDocumentId;
    private String currentDocumentId;
    private String currentRemotePath = "/";
    private int selectedLocal = -1;
    private int selectedRemote = -1;
    private FtpSession session;
    private boolean busy;

    @Override
    protected void onCreate(Bundle state) {
        super.onCreate(state);
        buildUi();
        restorePreferences();
        refreshButtons();
    }

    @Override
    protected void onDestroy() {
        FtpSession current = session;
        session = null;
        if (current != null) {
            current.close();
        }
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
        root.addView(label("Private FTP/FTPS client · no telemetry · password stays in memory", 13, Color.LTGRAY));

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
            try {
                treeUri = Uri.parse(savedTree);
                rootDocumentId = DocumentsContract.getTreeDocumentId(treeUri);
                currentDocumentId = rootDocumentId;
                refreshLocal();
            } catch (RuntimeException e) {
                clearLocalRoot();
            }
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

    private void connect() {
        if (busy || session != null) return;
        String hostValue = host.getText().toString().trim();
        String userValue = username.getText().toString().trim();
        String passwordValue = password.getText().toString();
        boolean secure = "FTPS".equals(protocol.getSelectedItem().toString());
        int portValue;
        try {
            portValue = Integer.parseInt(port.getText().toString().trim());
        } catch (NumberFormatException e) {
            setStatus("Invalid port.");
            return;
        }
        if (hostValue.isEmpty()) {
            setStatus("Server host is required.");
            return;
        }
        setBusy(true, secure ? "Connecting with strict FTPS TLS verification…" : "Connecting with unencrypted FTP…");
        io.execute(() -> {
            FtpSession next = null;
            try {
                next = new FtpSession(hostValue, portValue, secure);
                next.connect(userValue, passwordValue);
                String start = next.pwd();
                List<RemoteEntry> entries = next.list(start);
                FtpSession ready = next;
                runOnUiThread(() -> {
                    session = ready;
                    currentRemotePath = start;
                    remoteEntries.clear();
                    remoteEntries.addAll(entries);
                    selectedRemote = -1;
                    password.setText("");
                    savePreferences();
                    renderRemote();
                    setBusy(false, secure ? "FTPS connected. Certificate and hostname verified." : "FTP connected. Warning: transport is unencrypted.");
                });
            } catch (Exception e) {
                if (next != null) next.close();
                postError("Connection failed", e);
            }
        });
    }

    private void disconnect() {
        if (busy) return;
        FtpSession current = session;
        session = null;
        remoteEntries.clear();
        selectedRemote = -1;
        renderRemote();
        if (current != null) io.execute(current::close);
        setStatus("Disconnected.");
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
                    setBusy(false, "Server directory refreshed.");
                });
            } catch (Exception e) {
                postError("Remote refresh failed", e);
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
            setStatus("Folder selected for this session; persistent permission was not granted.");
        }
        treeUri = selected;
        rootDocumentId = DocumentsContract.getTreeDocumentId(selected);
        currentDocumentId = rootDocumentId;
        localParents.clear();
        savePreferences();
        refreshLocal();
    }

    private void refreshLocal() {
        if (treeUri == null || currentDocumentId == null) {
            localPath.setText("No folder selected");
            return;
        }
        List<LocalEntry> next = queryChildren(currentDocumentId);
        localEntries.clear();
        localEntries.addAll(next);
        selectedLocal = -1;
        localPath.setText(currentDocumentId.equals(rootDocumentId) ? "Selected folder" : currentDocumentId);
        renderLocal();
    }

    private List<LocalEntry> queryChildren(String documentId) {
        List<LocalEntry> result = new ArrayList<>();
        Uri children = DocumentsContract.buildChildDocumentsUriUsingTree(treeUri, documentId);
        String[] projection = {DocumentsContract.Document.COLUMN_DOCUMENT_ID, DocumentsContract.Document.COLUMN_DISPLAY_NAME,
                DocumentsContract.Document.COLUMN_MIME_TYPE, DocumentsContract.Document.COLUMN_SIZE};
        try (Cursor cursor = getContentResolver().query(children, projection, null, null, null)) {
            if (cursor == null) return result;
            while (cursor.moveToNext()) {
                String id = cursor.getString(0);
                String name = cursor.getString(1);
                String mime = cursor.getString(2);
                long size = cursor.isNull(3) ? 0L : cursor.getLong(3);
                result.add(new LocalEntry(id, name, DocumentsContract.Document.MIME_TYPE_DIR.equals(mime), size));
            }
        } catch (SecurityException e) {
            setStatus("Local folder permission is no longer available.");
        }
        return result;
    }

    private void selectLocal(int position) {
        if (busy || position < 0 || position >= localEntries.size()) return;
        LocalEntry entry = localEntries.get(position);
        if (entry.directory) {
            localParents.push(currentDocumentId);
            currentDocumentId = entry.documentId;
            refreshLocal();
        } else {
            selectedLocal = position;
            renderLocal();
        }
    }

    private void localUp() {
        if (busy || localParents.isEmpty()) return;
        currentDocumentId = localParents.pop();
        refreshLocal();
    }

    private void uploadSelected() {
        FtpSession current = session;
        if (busy || current == null || selectedLocal < 0 || selectedLocal >= localEntries.size()) return;
        LocalEntry entry = localEntries.get(selectedLocal);
        Uri document = DocumentsContract.buildDocumentUriUsingTree(treeUri, entry.documentId);
        String remoteBase = currentRemotePath;
        setBusy(true, "Uploading " + entry.name + "…");
        io.execute(() -> {
            try (InputStream in = getContentResolver().openInputStream(document)) {
                if (in == null) throw new IOException("Could not open local file.");
                current.upload(FtpSession.joinRemote(remoteBase, entry.name), in);
                runOnUiThread(() -> {
                    if (session != current) return;
                    setBusy(false, "Upload completed: " + entry.name);
                    refreshRemote(currentRemotePath);
                });
            } catch (Exception e) {
                postError("Upload failed", e);
            }
        });
    }

    private void downloadSelected() {
        FtpSession current = session;
        if (busy || current == null || selectedRemote < 0 || selectedRemote >= remoteEntries.size() || treeUri == null) return;
        RemoteEntry entry = remoteEntries.get(selectedRemote);
        Uri parent = DocumentsContract.buildDocumentUriUsingTree(treeUri, currentDocumentId);
        String remoteBase = currentRemotePath;
        setBusy(true, "Downloading " + entry.name + "…");
        io.execute(() -> {
            Uri created = null;
            try {
                created = DocumentsContract.createDocument(getContentResolver(), parent, "application/octet-stream", entry.name);
                if (created == null) throw new IOException("Could not create local destination file.");
                try (OutputStream out = getContentResolver().openOutputStream(created, "w")) {
                    if (out == null) throw new IOException("Could not open local destination file.");
                    current.download(FtpSession.joinRemote(remoteBase, entry.name), out);
                }
                runOnUiThread(() -> {
                    if (session != current) return;
                    setBusy(false, "Download completed: " + entry.name);
                    refreshLocal();
                });
            } catch (Exception e) {
                if (created != null) {
                    try { DocumentsContract.deleteDocument(getContentResolver(), created); } catch (Exception ignored) { }
                }
                postError("Download failed", e);
            }
        });
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
        connect.setEnabled(!busy && !connected);
        disconnect.setEnabled(!busy && connected);
        upload.setEnabled(!busy && connected && selectedLocal >= 0);
        download.setEnabled(!busy && connected && selectedRemote >= 0 && treeUri != null);
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
