package com.brendigo.ghostftp;

import android.annotation.SuppressLint;
import android.app.Activity;
import android.content.ContentResolver;
import android.content.Intent;
import android.content.SharedPreferences;
import android.database.Cursor;
import android.graphics.Color;
import android.net.Uri;
import android.os.Bundle;
import android.provider.DocumentsContract;
import android.text.InputType;
import android.view.View;
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
import java.util.ArrayList;
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

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        buildUi();
        restoreNonSecretPreferences();
        refreshConnectionButtons();
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
        int pad = dp(14);
        LinearLayout root = new LinearLayout(this);
        root.setOrientation(LinearLayout.VERTICAL);
        root.setPadding(pad, pad, pad, pad);
        root.setBackgroundColor(Color.rgb(16, 19, 23));

        TextView title = label("GHOST FTP · ANDROID", 22, Color.WHITE);
        root.addView(title);
        TextView subtitle = label("Private FTP/FTPS client · no telemetry · no stored password", 13, Color.LTGRAY);
        root.addView(subtitle);

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

        LinearLayout connectionButtons = row();
        connect = button("Connect");
        disconnect = button("Disconnect");
        connectionButtons.addView(connect, weighted());
        connectionButtons.addView(disconnect, weighted());
        root.addView(connectionButtons, matchWrap());

        connect.setOnClickListener(v -> connect());
        disconnect.setOnClickListener(v -> disconnect());

        root.addView(section("LOCAL STORAGE"));
        localPath = label("No folder selected", 14, Color.LTGRAY);
        root.addView(localPath);
        LinearLayout localButtons = row();
        Button chooseLocal = button("Choose folder");
        Button localUp = button("Up");
        localButtons.addView(chooseLocal, weighted());
        localButtons.addView(localUp, weighted());
        root.addView(localButtons, matchWrap());
        chooseLocal.setOnClickListener(v -> chooseLocalFolder());
        localUp.setOnClickListener(v -> localUp());

        localList = new ListView(this);
        root.addView(localList, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, dp(220)));
        localList.setOnItemClickListener((parent, view, position, id) -> selectLocal(position));

        root.addView(section("SERVER"));
        remotePath = label(currentRemotePath, 14, Color.LTGRAY);
        root.addView(remotePath);
        LinearLayout remoteButtons = row();
        Button remoteRefresh = button("Refresh");
        Button remoteUp = button("Up");
        remoteButtons.addView(remoteRefresh, weighted());
        remoteButtons.addView(remoteUp, weighted());
        root.addView(remoteButtons, matchWrap());
        remoteRefresh.setOnClickListener(v -> refreshRemote());
        remoteUp.setOnClickListener(v -> remoteUp());

        remoteList = new ListView(this);
        root.addView(remoteList, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, dp(220)));
        remoteList.setOnItemClickListener((parent, view, position, id) -> selectRemote(position));

        LinearLayout transferButtons = row();
        upload = button("Upload →");
        download = button("← Download");
        transferButtons.addView(upload, weighted());
        transferButtons.addView(download, weighted());
        root.addView(transferButtons, matchWrap());
        upload.setOnClickListener(v -> uploadSelected());
        download.setOnClickListener(v -> downloadSelected());

        status = label("Ready.", 13, Color.LTGRAY);
        root.addView(status);

        ScrollView scroll = new ScrollView(this);
        scroll.addView(root);
        setContentView(scroll);
    }

    private void restoreNonSecretPreferences() {
        SharedPreferences prefs = getSharedPreferences(PREFS, MODE_PRIVATE);
        host.setText(prefs.getString("host", ""));
        username.setText(prefs.getString("username", ""));
        String savedProtocol = prefs.getString("protocol", "FTPS");
        protocol.setSelection("FTP".equals(savedProtocol) ? 1 : 0);
        port.setText(prefs.getString("port", "FTPS".equals(savedProtocol) ? "21" : "21"));
        String savedTree = prefs.getString("treeUri", "");
        if (!savedTree.isEmpty()) {
            treeUri = Uri.parse(savedTree);
            try {
                rootDocumentId = DocumentsContract.getTreeDocumentId(treeUri);
                currentDocumentId = rootDocumentId;
                refreshLocal();
            } catch (RuntimeException e) {
                treeUri = null;
                rootDocumentId = null;
                currentDocumentId = null;
            }
        }
    }

    private void saveNonSecretPreferences() {
        getSharedPreferences(PREFS, MODE_PRIVATE).edit()
                .putString("host", host.getText().toString().trim())
                .putString("username", username.getText().toString().trim())
                .putString("protocol", protocol.getSelectedItem().toString())
                .putString("port", port.getText().toString().trim())
                .putString("treeUri", treeUri == null ? "" : treeUri.toString())
                .apply();
    }

    private void connect() {
        final String hostValue = host.getText().toString().trim();
        final String userValue = username.getText().toString().trim();
        final String passwordValue = password.getText().toString();
        final boolean secure = "FTPS".equals(protocol.getSelectedItem().toString());
        final int portValue;
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
                FtpSession connectedSession = next;
                runOnUiThread(() -> {
                    FtpSession old = session;
                    session = connectedSession;
                    if (old != null) {
                        old.close();
                    }
                    currentRemotePath = start;
                    remoteEntries.clear();
                    remoteEntries.addAll(entries);
                    remotePath.setText(currentRemotePath);
                    renderRemoteList();
                    password.setText("");
                    saveNonSecretPreferences();
                    setBusy(false, secure ? "FTPS connected. Certificate and hostname verified." : "FTP connected. Warning: transport is unencrypted.");
                });
            } catch (Exception e) {
                if (next != null) {
                    next.close();
                }
                postError("Connection failed", e);
            }
        });
    }

    private void disconnect() {
        FtpSession current = session;
        session = null;
        if (current != null) {
            io.execute(current::close);
        }
        remoteEntries.clear();
        selectedRemote = -1;
        renderRemoteList();
        refreshConnectionButtons();
        setStatus("Disconnected.");
    }

    private void refreshRemote() {
        FtpSession current = session;
        if (current == null || !current.isConnected()) {
            setStatus("Connect first.");
            return;
        }
        setBusy(true, "Refreshing server directory…");
        String target = currentRemotePath;
        io.execute(() -> {
            try {
                List<RemoteEntry> entries = current.list(target);
                runOnUiThread(() -> {
                    remoteEntries.clear();
                    remoteEntries.addAll(entries);
                    selectedRemote = -1;
                    renderRemoteList();
                    setBusy(false, "Server directory refreshed.");
                });
            } catch (Exception e) {
                postError("Remote refresh failed", e);
            }
        });
    }

    private void selectRemote(int position) {
        if (position < 0 || position >= remoteEntries.size()) {
            return;
        }
        RemoteEntry entry = remoteEntries.get(position);
        if (entry.directory) {
            try {
                currentRemotePath = FtpSession.joinRemote(currentRemotePath, entry.name);
                remotePath.setText(currentRemotePath);
                refreshRemote();
            } catch (IOException e) {
                setStatus(e.getMessage());
            }
            return;
        }
        selectedRemote = position;
        renderRemoteList();
        refreshConnectionButtons();
    }

    private void remoteUp() {
        try {
            currentRemotePath = FtpSession.parentRemote(currentRemotePath);
            remotePath.setText(currentRemotePath);
            refreshRemote();
        } catch (IOException e) {
            setStatus(e.getMessage());
        }
    }

    private void chooseLocalFolder() {
        Intent intent = new Intent(Intent.ACTION_OPEN_DOCUMENT_TREE);
        intent.addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION | Intent.FLAG_GRANT_WRITE_URI_PERMISSION | Intent.FLAG_GRANT_PERSISTABLE_URI_PERMISSION);
        startActivityForResult(intent, REQUEST_TREE);
    }

    @Override
    protected void onActivityResult(int requestCode, int resultCode, Intent data) {
        super.onActivityResult(requestCode, resultCode, data);
        if (requestCode != REQUEST_TREE || resultCode != RESULT_OK || data == null || data.getData() == null) {
            return;
        }
        Uri selected = data.getData();
        boolean canRead = (data.getFlags() & Intent.FLAG_GRANT_READ_URI_PERMISSION) != 0;
        boolean canWrite = (data.getFlags() & Intent.FLAG_GRANT_WRITE_URI_PERMISSION) != 0;
        try {
            if (canRead && canWrite) {
                getContentResolver().takePersistableUriPermission(
                        selected,
                        Intent.FLAG_GRANT_READ_URI_PERMISSION | Intent.FLAG_GRANT_WRITE_URI_PERMISSION
                );
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
        saveNonSecretPreferences();
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
        renderLocalList();
        refreshConnectionButtons();
    }

    private List<LocalEntry> queryChildren(String documentId) {
        List<LocalEntry> result = new ArrayList<>();
        Uri children = DocumentsContract.buildChildDocumentsUriUsingTree(treeUri, documentId);
        String[] projection = {
                DocumentsContract.Document.COLUMN_DOCUMENT_ID,
                DocumentsContract.Document.COLUMN_DISPLAY_NAME,
                DocumentsContract.Document.COLUMN_MIME_TYPE,
                DocumentsContract.Document.COLUMN_SIZE
        };
        try (Cursor cursor = getContentResolver().query(children, projection, null, null, null)) {
            if (cursor == null) {
                return result;
            }
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
        if (position < 0 || position >= localEntries.size()) {
            return;
        }
        LocalEntry entry = localEntries.get(position);
        if (entry.directory) {
            currentDocumentId = entry.documentId;
            refreshLocal();
            return;
        }
        selectedLocal = position;
        renderLocalList();
        refreshConnectionButtons();
    }

    private void localUp() {
        if (treeUri == null || currentDocumentId == null || currentDocumentId.equals(rootDocumentId)) {
            return;
        }
        String parent = findParentDocumentId(currentDocumentId);
        currentDocumentId = parent == null ? rootDocumentId : parent;
        refreshLocal();
    }

    private String findParentDocumentId(String childId) {
        if (childId == null || childId.equals(rootDocumentId)) {
            return null;
        }
        if (childId.startsWith(rootDocumentId + "/")) {
            int slash = childId.lastIndexOf('/');
            return slash > rootDocumentId.length() ? childId.substring(0, slash) : rootDocumentId;
        }
        return rootDocumentId;
    }

    private void uploadSelected() {
        FtpSession current = session;
        if (current == null || selectedLocal < 0 || selectedLocal >= localEntries.size()) {
            return;
        }
        LocalEntry entry = localEntries.get(selectedLocal);
        Uri document = DocumentsContract.buildDocumentUriUsingTree(treeUri, entry.documentId);
        setBusy(true, "Uploading " + entry.name + "…");
        io.execute(() -> {
            try (InputStream in = getContentResolver().openInputStream(document)) {
                if (in == null) {
                    throw new IOException("Could not open local file.");
                }
                current.upload(FtpSession.joinRemote(currentRemotePath, entry.name), in);
                runOnUiThread(() -> {
                    setBusy(false, "Upload completed: " + entry.name);
                    refreshRemote();
                });
            } catch (Exception e) {
                postError("Upload failed", e);
            }
        });
    }

    private void downloadSelected() {
        FtpSession current = session;
        if (current == null || selectedRemote < 0 || selectedRemote >= remoteEntries.size() || treeUri == null) {
            return;
        }
        RemoteEntry entry = remoteEntries.get(selectedRemote);
        Uri parent = DocumentsContract.buildDocumentUriUsingTree(treeUri, currentDocumentId);
        setBusy(true, "Downloading " + entry.name + "…");
        io.execute(() -> {
            Uri created = null;
            try {
                created = DocumentsContract.createDocument(getContentResolver(), parent, "application/octet-stream", entry.name);
                if (created == null) {
                    throw new IOException("Could not create local destination file.");
                }
                try (OutputStream out = getContentResolver().openOutputStream(created, "w")) {
                    if (out == null) {
                        throw new IOException("Could not open local destination file.");
                    }
                    current.download(FtpSession.joinRemote(currentRemotePath, entry.name), out);
                }
                runOnUiThread(() -> {
                    setBusy(false, "Download completed: " + entry.name);
                    refreshLocal();
                });
            } catch (Exception e) {
                if (created != null) {
                    try {
                        DocumentsContract.deleteDocument(getContentResolver(), created);
                    } catch (Exception ignored) {
                        // Best-effort cleanup of incomplete local file.
                    }
                }
                postError("Download failed", e);
            }
        });
    }

    private void renderLocalList() {
        List<String> labels = new ArrayList<>();
        for (int i = 0; i < localEntries.size(); i++) {
            LocalEntry entry = localEntries.get(i);
            String prefix = entry.directory ? "DIR   " : "FILE  ";
            labels.add((i == selectedLocal ? "› " : "  ") + prefix + entry.name + (entry.directory ? "" : "  (" + entry.size + " B)"));
        }
        localList.setAdapter(new ArrayAdapter<>(this, android.R.layout.simple_list_item_1, labels));
    }

    private void renderRemoteList() {
        List<String> labels = new ArrayList<>();
        for (int i = 0; i < remoteEntries.size(); i++) {
            labels.add((i == selectedRemote ? "› " : "  ") + remoteEntries.get(i).toString());
        }
        remoteList.setAdapter(new ArrayAdapter<>(this, android.R.layout.simple_list_item_1, labels));
        refreshConnectionButtons();
    }

    private void setBusy(boolean busy, String message) {
        connect.setEnabled(!busy && session == null);
        disconnect.setEnabled(!busy && session != null);
        upload.setEnabled(!busy && session != null && selectedLocal >= 0);
        download.setEnabled(!busy && session != null && selectedRemote >= 0 && treeUri != null);
        setStatus(message);
    }

    private void refreshConnectionButtons() {
        boolean connected = session != null && session.isConnected();
        connect.setEnabled(!connected);
        disconnect.setEnabled(connected);
        upload.setEnabled(connected && selectedLocal >= 0);
        download.setEnabled(connected && selectedRemote >= 0 && treeUri != null);
    }

    private void postError(String prefix, Exception e) {
        runOnUiThread(() -> {
            setBusy(false, prefix + ": " + safeMessage(e));
            refreshConnectionButtons();
        });
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
        if (secret) {
            edit.setInputType(InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_VARIATION_PASSWORD);
        }
        return edit;
    }

    private Button button(String text) {
        Button button = new Button(this);
        button.setText(text);
        button.setAllCaps(false);
        return button;
    }

    private LinearLayout row() {
        LinearLayout row = new LinearLayout(this);
        row.setOrientation(LinearLayout.HORIZONTAL);
        return row;
    }

    private LinearLayout.LayoutParams weighted() {
        return new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f);
    }

    private LinearLayout.LayoutParams matchWrap() {
        return new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT);
    }

    private int dp(int value) {
        return Math.round(value * getResources().getDisplayMetrics().density);
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
