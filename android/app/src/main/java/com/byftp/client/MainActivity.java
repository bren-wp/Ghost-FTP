package com.byftp.client;

import android.app.Activity;
import android.app.AlertDialog;
import android.content.Intent;
import android.database.Cursor;
import android.net.Uri;
import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;
import android.provider.OpenableColumns;
import android.text.InputType;
import android.view.Gravity;
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
import com.byftp.client.model.ConnectionConfig;
import com.byftp.client.model.RemoteEntry;
import com.byftp.client.model.RemotePaths;
import com.byftp.client.remote.RemoteClient;
import com.byftp.client.remote.RemoteClientFactory;
import java.io.InputStream;
import java.io.OutputStream;
import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

public final class MainActivity extends Activity {
    private static final int REQUEST_UPLOAD = 1001;
    private static final int REQUEST_DOWNLOAD = 1002;

    private final ExecutorService io = Executors.newSingleThreadExecutor();
    private final Handler main = new Handler(Looper.getMainLooper());
    private final List<RemoteEntry> entries = new ArrayList<>();

    private Spinner protocol;
    private EditText host;
    private EditText port;
    private EditText username;
    private EditText password;
    private EditText fingerprint;
    private Button connect;
    private Button up;
    private Button refresh;
    private Button upload;
    private Button newFolder;
    private TextView path;
    private TextView status;
    private ListView list;
    private ArrayAdapter<String> listAdapter;

    private RemoteClient client;
    private String currentPath = "/";
    private RemoteEntry selected;
    private String pendingDownloadPath;
    private boolean busy;

    @Override protected void onCreate(Bundle state) {
        super.onCreate(state);
        setContentView(buildUi());
        bindEvents();
        updateConnectionUi(false);
    }

    private View buildUi() {
        int pad = dp(16);
        LinearLayout root = new LinearLayout(this);
        root.setOrientation(LinearLayout.VERTICAL);
        root.setPadding(pad, pad, pad, pad);
        root.setBackgroundColor(0xFFF8FAFC);

        TextView title = new TextView(this);
        title.setText(getString(R.string.app_name));
        title.setTextSize(26);
        title.setTextColor(0xFF0F172A);
        title.setTypeface(android.graphics.Typeface.DEFAULT_BOLD);
        root.addView(title);

        TextView subtitle = new TextView(this);
        subtitle.setText(getString(R.string.app_subtitle));
        subtitle.setTextColor(0xFF475569);
        subtitle.setPadding(0, 0, 0, dp(12));
        root.addView(subtitle);

        ScrollView formScroll = new ScrollView(this);
        LinearLayout form = new LinearLayout(this);
        form.setOrientation(LinearLayout.VERTICAL);
        formScroll.addView(form);
        root.addView(formScroll, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT));

        protocol = new Spinner(this);
        protocol.setAdapter(new ArrayAdapter<>(this, android.R.layout.simple_spinner_dropdown_item, ConnectionConfig.Protocol.values()));
        form.addView(protocol, matchWrap());

        host = input(R.string.host, InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_VARIATION_URI);
        port = input(R.string.port, InputType.TYPE_CLASS_NUMBER);
        username = input(R.string.username, InputType.TYPE_CLASS_TEXT);
        password = input(R.string.password, InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_VARIATION_PASSWORD);
        fingerprint = input(R.string.fingerprint_hint, InputType.TYPE_CLASS_TEXT);
        form.addView(host); form.addView(port); form.addView(username); form.addView(password); form.addView(fingerprint);

        connect = button(R.string.connect);
        form.addView(connect, matchWrap());

        LinearLayout actions = new LinearLayout(this);
        actions.setOrientation(LinearLayout.HORIZONTAL);
        actions.setGravity(Gravity.CENTER_VERTICAL);
        up = button(R.string.up); refresh = button(R.string.refresh); upload = button(R.string.upload); newFolder = button(R.string.new_folder);
        actions.addView(up, weight()); actions.addView(refresh, weight()); actions.addView(upload, weight()); actions.addView(newFolder, weight());
        root.addView(actions, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT));

        path = new TextView(this);
        path.setText(R.string.root_path);
        path.setTextColor(0xFF334155);
        path.setPadding(0, dp(10), 0, dp(6));
        root.addView(path);

        list = new ListView(this);
        listAdapter = new ArrayAdapter<>(this, android.R.layout.simple_list_item_activated_1, new ArrayList<>());
        list.setAdapter(listAdapter);
        list.setChoiceMode(ListView.CHOICE_MODE_SINGLE);
        root.addView(list, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, 0, 1f));

        status = new TextView(this);
        status.setText(R.string.status_ready);
        status.setTextColor(0xFF475569);
        status.setPadding(0, dp(8), 0, 0);
        root.addView(status);
        return root;
    }

    private void bindEvents() {
        protocol.setOnItemSelectedListener(new android.widget.AdapterView.OnItemSelectedListener() {
            @Override public void onItemSelected(android.widget.AdapterView<?> parent, View view, int position, long id) {
                ConnectionConfig.Protocol value = (ConnectionConfig.Protocol) protocol.getSelectedItem();
                port.setHint(Integer.toString(value.defaultPort()));
                fingerprint.setVisibility(value == ConnectionConfig.Protocol.SFTP ? View.VISIBLE : View.GONE);
                if (value == ConnectionConfig.Protocol.FTP) status.setText(R.string.ftp_warning);
                else if (client == null) status.setText(R.string.status_ready);
            }
            @Override public void onNothingSelected(android.widget.AdapterView<?> parent) {}
        });
        connect.setOnClickListener(v -> { if (client == null) connect(); else disconnect(); });
        refresh.setOnClickListener(v -> refresh());
        up.setOnClickListener(v -> { if (!currentPath.equals("/")) openDirectory(RemotePaths.parent(currentPath)); });
        upload.setOnClickListener(v -> pickUpload());
        newFolder.setOnClickListener(v -> promptNewFolder());
        list.setOnItemClickListener((parent, view, position, id) -> {
            RemoteEntry entry = entries.get(position);
            selected = entry;
            if (entry.directory()) openDirectory(RemotePaths.child(currentPath, entry.name()));
            else status.setText(getString(R.string.selected_file, entry.name()));
        });
        list.setOnItemLongClickListener((parent, view, position, id) -> {
            selected = entries.get(position);
            showEntryActions(selected);
            return true;
        });
    }

    private void connect() {
        if (busy) return;
        ConnectionConfig config;
        try {
            config = ConnectionConfig.create((ConnectionConfig.Protocol) protocol.getSelectedItem(), text(host), text(port), text(username), password.getText().toString(), text(fingerprint));
        } catch (IllegalArgumentException error) {
            showError(error);
            return;
        }
        setBusy(true, R.string.status_connecting);
        io.execute(() -> {
            RemoteClient next = RemoteClientFactory.create(config);
            try {
                next.connect();
                List<RemoteEntry> initial = next.list("/");
                main.post(() -> {
                    closeClient();
                    client = next;
                    currentPath = "/";
                    replaceEntries(initial);
                    updateConnectionUi(true);
                    setBusy(false, R.string.status_connected);
                });
            } catch (Exception error) {
                next.close();
                main.post(() -> { setBusy(false, R.string.status_ready); showError(error); });
            }
        });
    }

    private void disconnect() {
        if (busy) return;
        RemoteClient current = client;
        client = null;
        setBusy(true, R.string.status_working);
        io.execute(() -> {
            if (current != null) current.close();
            main.post(() -> {
                entries.clear(); listAdapter.clear(); currentPath = "/"; path.setText(currentPath); selected = null;
                updateConnectionUi(false); setBusy(false, R.string.status_disconnected);
            });
        });
    }

    private void refresh() { openDirectory(currentPath); }

    private void openDirectory(String nextPath) {
        RemoteClient current = client;
        if (current == null || busy) return;
        String normalized = RemotePaths.normalizeDirectory(nextPath);
        setBusy(true, R.string.status_working);
        io.execute(() -> {
            try {
                List<RemoteEntry> next = current.list(normalized);
                main.post(() -> { currentPath = normalized; selected = null; replaceEntries(next); setBusy(false, R.string.status_connected); });
            } catch (Exception error) {
                main.post(() -> { setBusy(false, R.string.status_connected); showError(error); });
            }
        });
    }

    private void replaceEntries(List<RemoteEntry> next) {
        entries.clear(); entries.addAll(next);
        List<String> labels = new ArrayList<>();
        for (RemoteEntry entry : entries) labels.add(entry.displayLabel());
        listAdapter.clear(); listAdapter.addAll(labels); listAdapter.notifyDataSetChanged();
        path.setText(currentPath);
    }

    private void pickUpload() {
        if (client == null || busy) return;
        Intent intent = new Intent(Intent.ACTION_OPEN_DOCUMENT);
        intent.addCategory(Intent.CATEGORY_OPENABLE);
        intent.setType("*/*");
        startActivityForResult(intent, REQUEST_UPLOAD);
    }

    private void startDownload(RemoteEntry entry) {
        if (entry.directory()) return;
        pendingDownloadPath = RemotePaths.child(currentPath, entry.name());
        Intent intent = new Intent(Intent.ACTION_CREATE_DOCUMENT);
        intent.addCategory(Intent.CATEGORY_OPENABLE);
        intent.setType("application/octet-stream");
        intent.putExtra(Intent.EXTRA_TITLE, entry.name());
        startActivityForResult(intent, REQUEST_DOWNLOAD);
    }

    @Override protected void onActivityResult(int requestCode, int resultCode, Intent data) {
        super.onActivityResult(requestCode, resultCode, data);
        if (resultCode != RESULT_OK || data == null || data.getData() == null || client == null) return;
        Uri uri = data.getData();
        if (requestCode == REQUEST_UPLOAD) uploadUri(uri);
        else if (requestCode == REQUEST_DOWNLOAD && pendingDownloadPath != null) downloadUri(uri, pendingDownloadPath);
    }

    private void uploadUri(Uri uri) {
        String name = displayName(uri);
        try { RemotePaths.validateName(name); } catch (IllegalArgumentException error) { showError(error); return; }
        String remotePath = RemotePaths.child(currentPath, name);
        runIo(() -> {
            try (InputStream source = getContentResolver().openInputStream(uri)) {
                if (source == null) throw new IllegalStateException("Unable to open selected file.");
                requireClient().upload(remotePath, source);
            }
        }, true);
    }

    private void downloadUri(Uri uri, String remotePath) {
        pendingDownloadPath = null;
        runIo(() -> {
            try (OutputStream destination = getContentResolver().openOutputStream(uri, "w")) {
                if (destination == null) throw new IllegalStateException("Unable to open destination file.");
                requireClient().download(remotePath, destination);
            }
        }, false);
    }

    private void promptNewFolder() {
        if (client == null || busy) return;
        EditText name = input(R.string.folder_name, InputType.TYPE_CLASS_TEXT);
        new AlertDialog.Builder(this).setTitle(R.string.new_folder).setView(name)
            .setNegativeButton(R.string.cancel, null)
            .setPositiveButton(R.string.create, (dialog, which) -> mutateName(name.getText().toString(), null, true))
            .show();
    }

    private void promptRename(RemoteEntry entry) {
        EditText name = input(R.string.new_name, InputType.TYPE_CLASS_TEXT);
        name.setText(entry.name()); name.selectAll();
        new AlertDialog.Builder(this).setTitle(R.string.rename).setView(name)
            .setNegativeButton(R.string.cancel, null)
            .setPositiveButton(R.string.save, (dialog, which) -> mutateName(name.getText().toString(), entry, false))
            .show();
    }

    private void mutateName(String rawName, RemoteEntry existing, boolean createDirectory) {
        String name = rawName == null ? "" : rawName.trim();
        try { RemotePaths.validateName(name); } catch (IllegalArgumentException error) { showError(error); return; }
        runIo(() -> {
            if (createDirectory) requireClient().mkdir(RemotePaths.child(currentPath, name));
            else requireClient().rename(RemotePaths.child(currentPath, existing.name()), RemotePaths.child(currentPath, name));
        }, true);
    }

    private void showEntryActions(RemoteEntry entry) {
        String[] actions = entry.directory()
            ? new String[]{getString(R.string.rename), getString(R.string.delete)}
            : new String[]{getString(R.string.download), getString(R.string.rename), getString(R.string.delete)};
        new AlertDialog.Builder(this).setTitle(entry.name()).setItems(actions, (dialog, which) -> {
            String action = actions[which];
            if (action.equals(getString(R.string.download))) startDownload(entry);
            else if (action.equals(getString(R.string.rename))) promptRename(entry);
            else confirmDelete(entry);
        }).show();
    }

    private void confirmDelete(RemoteEntry entry) {
        new AlertDialog.Builder(this).setMessage(getString(R.string.delete_confirm, entry.name()))
            .setNegativeButton(R.string.cancel, null)
            .setPositiveButton(R.string.delete, (dialog, which) -> runIo(() -> requireClient().delete(RemotePaths.child(currentPath, entry.name()), entry.directory()), true))
            .show();
    }

    private void runIo(ThrowingAction action, boolean refreshAfter) {
        if (busy || client == null) return;
        setBusy(true, R.string.status_working);
        io.execute(() -> {
            try {
                action.run();
                List<RemoteEntry> next = refreshAfter ? requireClient().list(currentPath) : null;
                main.post(() -> { if (next != null) replaceEntries(next); setBusy(false, R.string.status_connected); });
            } catch (Exception error) {
                main.post(() -> { setBusy(false, R.string.status_connected); showError(error); });
            }
        });
    }

    private RemoteClient requireClient() {
        RemoteClient current = client;
        if (current == null) throw new IllegalStateException("Not connected.");
        return current;
    }

    private String displayName(Uri uri) {
        try (Cursor cursor = getContentResolver().query(uri, new String[]{OpenableColumns.DISPLAY_NAME}, null, null, null)) {
            if (cursor != null && cursor.moveToFirst()) {
                int index = cursor.getColumnIndex(OpenableColumns.DISPLAY_NAME);
                if (index >= 0) return cursor.getString(index);
            }
        }
        String last = uri.getLastPathSegment();
        return last == null || last.isBlank() ? "upload.bin" : last.substring(last.lastIndexOf('/') + 1);
    }

    private void setBusy(boolean value, int statusText) {
        busy = value; status.setText(statusText); updateEnabled();
    }

    private void updateConnectionUi(boolean connected) {
        connect.setText(connected ? R.string.disconnect : R.string.connect);
        protocol.setEnabled(!connected); host.setEnabled(!connected); port.setEnabled(!connected); username.setEnabled(!connected); password.setEnabled(!connected); fingerprint.setEnabled(!connected);
        updateEnabled();
    }

    private void updateEnabled() {
        boolean connected = client != null;
        connect.setEnabled(!busy);
        up.setEnabled(connected && !busy && !currentPath.equals("/"));
        refresh.setEnabled(connected && !busy); upload.setEnabled(connected && !busy); newFolder.setEnabled(connected && !busy); list.setEnabled(connected && !busy);
    }

    private void showError(Throwable error) {
        String message = error.getMessage();
        if (message == null || message.isBlank()) message = error.getClass().getSimpleName();
        message = message.replace('\r', ' ').replace('\n', ' ').trim();
        if (message.length() > 320) message = message.substring(0, 320);
        status.setText(getString(R.string.error_prefix, message));
    }

    private void closeClient() { if (client != null) { client.close(); client = null; } }
    private String text(EditText value) { return value.getText().toString(); }
    private EditText input(int hint, int inputType) { EditText e = new EditText(this); e.setHint(hint); e.setInputType(inputType); e.setSingleLine(true); e.setLayoutParams(matchWrap()); return e; }
    private Button button(int text) { Button b = new Button(this); b.setText(text); b.setAllCaps(false); return b; }
    private LinearLayout.LayoutParams matchWrap() { return new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT); }
    private LinearLayout.LayoutParams weight() { return new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f); }
    private int dp(int value) { return Math.round(value * getResources().getDisplayMetrics().density); }

    @Override protected void onDestroy() {
        RemoteClient current = client;
        client = null;
        io.execute(() -> { if (current != null) current.close(); });
        io.shutdown();
        super.onDestroy();
    }

    @FunctionalInterface private interface ThrowingAction { void run() throws Exception; }
}
