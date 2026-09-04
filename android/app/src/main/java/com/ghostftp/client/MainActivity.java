package com.GhostFTP.client;

import android.app.Activity;
import android.app.AlertDialog;
import android.content.ClipData;
import android.content.Intent;
import android.database.Cursor;
import android.net.Uri;
import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;
import android.provider.OpenableColumns;
import android.text.Editable;
import android.text.InputType;
import android.text.TextWatcher;
import android.view.Gravity;
import android.view.View;
import android.view.ViewGroup;
import android.widget.ArrayAdapter;
import android.widget.Button;
import android.widget.EditText;
import android.widget.ImageView;
import android.widget.LinearLayout;
import android.widget.ListView;
import android.widget.ProgressBar;
import android.widget.ScrollView;
import android.widget.Spinner;
import android.widget.TextView;
import com.GhostFTP.client.model.ConnectionConfig;
import com.GhostFTP.client.model.DocumentName;
import com.GhostFTP.client.model.RemoteEntry;
import com.GhostFTP.client.model.RemoteEntryList;
import com.GhostFTP.client.model.RemotePaths;
import com.GhostFTP.client.model.SharedHostingDiagnostics;
import com.GhostFTP.client.remote.RemoteClient;
import com.GhostFTP.client.remote.RemoteClientFactory;
import com.GhostFTP.client.remote.TransferStreams;
import java.io.InputStream;
import java.io.OutputStream;
import java.util.ArrayList;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Locale;
import java.util.Set;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

public final class MainActivity extends Activity {
    private static final int REQUEST_UPLOAD = 1001;
    private static final int REQUEST_DOWNLOAD = 1002;

    private final ExecutorService io = Executors.newSingleThreadExecutor();
    private final Handler main = new Handler(Looper.getMainLooper());
    private final List<RemoteEntry> entries = new ArrayList<>();
    private final List<RemoteEntry> visibleEntries = new ArrayList<>();

    private Spinner protocol;
    private EditText host;
    private EditText port;
    private EditText username;
    private EditText password;
    private EditText fingerprint;
    private EditText filter;
    private Button connect;
    private Button up;
    private Button refresh;
    private Button menu;
    private Button stopAfterCurrent;
    private ProgressBar transferProgress;
    private ScrollView formScroll;
    private TextView connectionSummary;
    private TextView path;
    private TextView status;
    private ListView list;
    private ArrayAdapter<String> listAdapter;
    private ConnectionPresetStore presetStore;

    private volatile RemoteClient client;
    private volatile RemoteClient connectingClient;
    private volatile boolean destroyed;
    private volatile boolean stopAfterCurrentRequested;
    private String currentPath = "/";
    private String pendingDownloadPath;
    private long pendingDownloadSize = -1L;
    private boolean busy;
    private boolean transferActive;

    @Override protected void onCreate(Bundle state) {
        super.onCreate(state);
        presetStore = new ConnectionPresetStore(this);
        setContentView(buildUi());
        restoreSavedConnection();
        bindEvents();
        updateConnectionUi(false);
    }

    private View buildUi() {
        int pad = dp(14);
        LinearLayout root = new LinearLayout(this);
        root.setOrientation(LinearLayout.VERTICAL);
        root.setPadding(pad, pad, pad, pad);
        root.setBackgroundColor(0xFFF8FAFC);

        LinearLayout header = new LinearLayout(this);
        header.setOrientation(LinearLayout.HORIZONTAL);
        header.setGravity(Gravity.CENTER_VERTICAL);
        header.setPadding(0, 0, 0, dp(10));

        ImageView logo = new ImageView(this);
        logo.setImageResource(R.drawable.ic_GhostFTP);
        logo.setContentDescription(getString(R.string.app_name));
        header.addView(logo, new LinearLayout.LayoutParams(dp(48), dp(48)));

        LinearLayout heading = new LinearLayout(this);
        heading.setOrientation(LinearLayout.VERTICAL);
        heading.setPadding(dp(10), 0, 0, 0);
        TextView title = new TextView(this);
        title.setText(getString(R.string.app_name));
        title.setTextSize(25);
        title.setTextColor(0xFF0F172A);
        title.setTypeface(android.graphics.Typeface.DEFAULT_BOLD);
        heading.addView(title);
        TextView subtitle = new TextView(this);
        subtitle.setText(getString(R.string.app_subtitle));
        subtitle.setTextColor(0xFF475569);
        heading.addView(subtitle);
        header.addView(heading, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        root.addView(header);

        formScroll = new ScrollView(this);
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
        form.addView(host);
        form.addView(port);
        form.addView(username);
        form.addView(password);
        form.addView(fingerprint);

        connect = button(R.string.connect);
        form.addView(connect, matchWrap());

        connectionSummary = new TextView(this);
        connectionSummary.setTextColor(0xFF334155);
        connectionSummary.setTextSize(14);
        connectionSummary.setPadding(dp(12), dp(10), dp(12), dp(10));
        connectionSummary.setBackgroundColor(0xFFEFF6FF);
        connectionSummary.setVisibility(View.GONE);
        root.addView(connectionSummary, matchWrap());

        LinearLayout actions = new LinearLayout(this);
        actions.setOrientation(LinearLayout.HORIZONTAL);
        actions.setGravity(Gravity.CENTER_VERTICAL);
        up = button(R.string.up);
        refresh = button(R.string.refresh);
        menu = button(R.string.menu);
        actions.addView(up, weight());
        actions.addView(refresh, weight());
        actions.addView(menu, weight());
        root.addView(actions, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT));

        path = new TextView(this);
        path.setText(R.string.root_path);
        path.setTextColor(0xFF334155);
        path.setTypeface(android.graphics.Typeface.MONOSPACE);
        path.setSingleLine(true);
        path.setEllipsize(android.text.TextUtils.TruncateAt.MIDDLE);
        path.setPadding(0, dp(8), 0, dp(4));
        root.addView(path);

        filter = input(R.string.filter_files, InputType.TYPE_CLASS_TEXT);
        filter.setVisibility(View.GONE);
        root.addView(filter);

        list = new ListView(this);
        listAdapter = new ArrayAdapter<>(this, android.R.layout.simple_list_item_activated_1, new ArrayList<>());
        list.setAdapter(listAdapter);
        list.setChoiceMode(ListView.CHOICE_MODE_SINGLE);
        root.addView(list, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, 0, 1f));

        transferProgress = new ProgressBar(this, null, android.R.attr.progressBarStyleHorizontal);
        transferProgress.setMax(1000);
        transferProgress.setVisibility(View.GONE);
        root.addView(transferProgress, matchWrap());

        stopAfterCurrent = button(R.string.stop_after_current);
        stopAfterCurrent.setVisibility(View.GONE);
        root.addView(stopAfterCurrent, matchWrap());

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
        connect.setOnClickListener(v -> connect());
        refresh.setOnClickListener(v -> refresh());
        up.setOnClickListener(v -> { if (!currentPath.equals("/")) openDirectory(RemotePaths.parent(currentPath)); });
        menu.setOnClickListener(v -> showMainMenu());
        stopAfterCurrent.setOnClickListener(v -> {
            if (!transferActive || !busy) return;
            stopAfterCurrentRequested = true;
            stopAfterCurrent.setEnabled(false);
            status.setText(R.string.status_stopping_after_current);
        });
        filter.addTextChangedListener(new TextWatcher() {
            @Override public void beforeTextChanged(CharSequence s, int start, int count, int after) {}
            @Override public void onTextChanged(CharSequence s, int start, int before, int count) { updateVisibleEntries(); }
            @Override public void afterTextChanged(Editable s) {}
        });
        list.setOnItemClickListener((parent, view, position, id) -> {
            if (position < 0 || position >= visibleEntries.size()) return;
            RemoteEntry entry = visibleEntries.get(position);
            if (entry.directory()) openDirectory(RemotePaths.child(currentPath, entry.name()));
            else status.setText(getString(R.string.selected_file, entry.name()));
        });
        list.setOnItemLongClickListener((parent, view, position, id) -> {
            if (position < 0 || position >= visibleEntries.size()) return true;
            showEntryActions(visibleEntries.get(position));
            return true;
        });
    }

    private void restoreSavedConnection() {
        ConnectionPresetStore.Preset saved = presetStore.load();
        if (saved == null) return;
        protocol.setSelection(saved.protocol().ordinal());
        host.setText(saved.host());
        port.setText(String.format(Locale.ROOT, "%d", saved.port()));
        username.setText(saved.username());
        fingerprint.setText(saved.fingerprint());
        password.setText("");
    }

    private void connect() {
        if (busy || client != null) return;
        ConnectionConfig config;
        try {
            config = ConnectionConfig.create(
                (ConnectionConfig.Protocol) protocol.getSelectedItem(),
                text(host),
                text(port),
                text(username),
                password.getText().toString(),
                text(fingerprint)
            );
        } catch (IllegalArgumentException error) {
            password.setText("");
            showError(error);
            return;
        }

        ConnectionConfig.Protocol safeProtocol = config.protocol();
        String safeHost = config.host();
        int safePort = config.port();
        String safeUsername = config.username();
        String safeFingerprint = config.fingerprint();
        password.setText("");

        setBusy(true, R.string.status_connecting);
        io.execute(() -> {
            RemoteClient next = RemoteClientFactory.create(config);
            connectingClient = next;
            try {
                next.connect();
                List<RemoteEntry> initial = next.list("/");
                SharedHostingDiagnostics diagnostics = SharedHostingDiagnostics.analyze(safeProtocol, initial);
                postToMain(() -> {
                    connectingClient = null;
                    client = next;
                    currentPath = "/";
                    presetStore.save(safeProtocol, safeHost, safePort, safeUsername, safeFingerprint);
                    String transportDiagnostic = getString(
                        diagnostics.secure() ? R.string.diagnostic_secure : R.string.diagnostic_plain_ftp
                    );
                    String rootDiagnostic;
                    if (diagnostics.webRootDetected()) {
                        rootDiagnostic = getString(R.string.diagnostic_web_root, diagnostics.webRoot());
                    } else if (diagnostics.rootMode().equals("home")) {
                        rootDiagnostic = getString(R.string.diagnostic_sftp_home);
                    } else {
                        rootDiagnostic = getString(R.string.diagnostic_account_root);
                    }
                    connectionSummary.setText(getString(
                        R.string.connected_with_diagnostics,
                        safeProtocol.toString(), safeHost, safePort, safeUsername,
                        transportDiagnostic, rootDiagnostic
                    ));
                    replaceEntries(initial);
                    updateConnectionUi(true);
                    setBusy(false, R.string.status_connected);
                });
            } catch (Exception error) {
                connectingClient = null;
                next.close();
                postToMain(() -> {
                    password.setText("");
                    setBusy(false, R.string.status_ready);
                    showError(error);
                });
            }
        });
    }

    private void disconnect() {
        if (busy || client == null) return;
        RemoteClient current = client;
        client = null;
        pendingDownloadPath = null;
        pendingDownloadSize = -1L;
        stopAfterCurrentRequested = false;
        setBusy(true, R.string.status_working);
        io.execute(() -> {
            current.close();
            postToMain(() -> {
                entries.clear();
                visibleEntries.clear();
                listAdapter.clear();
                currentPath = "/";
                path.setText(currentPath);
                filter.setText("");
                connectionSummary.setText("");
                finishTransferUi();
                updateConnectionUi(false);
                setBusy(false, R.string.status_disconnected);
            });
        });
    }

    private void refresh() { openDirectory(currentPath); }

    private void openDirectory(String nextPath) {
        RemoteClient current = client;
        if (current == null || busy) return;
        final String normalized;
        try {
            normalized = RemotePaths.normalizeDirectory(nextPath);
        } catch (IllegalArgumentException error) {
            showError(error);
            return;
        }
        setBusy(true, R.string.status_working);
        io.execute(() -> {
            try {
                List<RemoteEntry> next = current.list(normalized);
                postToMain(() -> {
                    currentPath = normalized;
                    filter.setText("");
                    replaceEntries(next);
                    setBusy(false, R.string.status_connected);
                });
            } catch (Exception error) {
                postToMain(() -> {
                    setBusy(false, R.string.status_connected);
                    showError(error);
                });
            }
        });
    }

    private void replaceEntries(List<RemoteEntry> next) {
        entries.clear();
        entries.addAll(RemoteEntryList.sorted(next));
        updateVisibleEntries();
        path.setText(currentPath);
    }

    private void updateVisibleEntries() {
        if (listAdapter == null || filter == null) return;
        visibleEntries.clear();
        visibleEntries.addAll(RemoteEntryList.filtered(entries, text(filter)));
        List<String> labels = new ArrayList<>(visibleEntries.size());
        for (RemoteEntry entry : visibleEntries) labels.add(entry.displayLabel());
        listAdapter.clear();
        listAdapter.addAll(labels);
        listAdapter.notifyDataSetChanged();
        if (client != null && !busy) {
            if (visibleEntries.isEmpty()) {
                status.setText(text(filter).trim().isEmpty() ? R.string.empty_directory : R.string.no_filter_results);
            } else {
                status.setText(R.string.status_connected);
            }
        }
    }

    private void showMainMenu() {
        if (busy || client == null) return;
        String[] actions = {
            getString(R.string.upload_files),
            getString(R.string.new_folder),
            getString(R.string.go_to_path),
            getString(R.string.forget_connection),
            getString(R.string.disconnect)
        };
        new AlertDialog.Builder(this)
            .setTitle(R.string.menu)
            .setItems(actions, (dialog, which) -> {
                switch (which) {
                    case 0 -> pickUpload();
                    case 1 -> promptNewFolder();
                    case 2 -> promptGoToPath();
                    case 3 -> {
                        presetStore.clear();
                        status.setText(R.string.saved_connection_forgotten);
                    }
                    case 4 -> disconnect();
                    default -> { }
                }
            })
            .show();
    }

    private void promptGoToPath() {
        EditText target = input(R.string.remote_path, InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_VARIATION_URI);
        target.setText(currentPath);
        target.selectAll();
        new AlertDialog.Builder(this)
            .setTitle(R.string.go_to_path)
            .setView(target)
            .setNegativeButton(R.string.cancel, null)
            .setPositiveButton(R.string.open, (dialog, which) -> openDirectory(target.getText().toString()))
            .show();
    }

    private void pickUpload() {
        if (client == null || busy) return;
        Intent intent = new Intent(Intent.ACTION_OPEN_DOCUMENT);
        intent.addCategory(Intent.CATEGORY_OPENABLE);
        intent.setType("*/*");
        intent.putExtra(Intent.EXTRA_ALLOW_MULTIPLE, true);
        startActivityForResult(intent, REQUEST_UPLOAD);
    }

    private void startDownload(RemoteEntry entry) {
        if (entry.directory()) return;
        pendingDownloadPath = RemotePaths.child(currentPath, entry.name());
        pendingDownloadSize = Math.max(0L, entry.size());
        Intent intent = new Intent(Intent.ACTION_CREATE_DOCUMENT);
        intent.addCategory(Intent.CATEGORY_OPENABLE);
        intent.setType("application/octet-stream");
        intent.putExtra(Intent.EXTRA_TITLE, entry.name());
        startActivityForResult(intent, REQUEST_DOWNLOAD);
    }

    @Override protected void onActivityResult(int requestCode, int resultCode, Intent data) {
        super.onActivityResult(requestCode, resultCode, data);

        if (requestCode == REQUEST_DOWNLOAD) {
            String remotePath = pendingDownloadPath;
            long remoteSize = pendingDownloadSize;
            pendingDownloadPath = null;
            pendingDownloadSize = -1L;
            if (resultCode != RESULT_OK || data == null || data.getData() == null || client == null || remotePath == null) return;
            downloadUri(data.getData(), remotePath, remoteSize);
            return;
        }

        if (requestCode == REQUEST_UPLOAD) {
            if (resultCode != RESULT_OK || data == null || client == null) return;
            Set<Uri> selected = new LinkedHashSet<>();
            ClipData clip = data.getClipData();
            if (clip != null) {
                for (int i = 0; i < clip.getItemCount(); i++) selected.add(clip.getItemAt(i).getUri());
            }
            if (data.getData() != null) selected.add(data.getData());
            if (!selected.isEmpty()) uploadUris(new ArrayList<>(selected));
        }
    }

    private void uploadUris(List<Uri> uris) {
        if (busy || client == null || uris.isEmpty()) return;
        List<UploadItem> items = new ArrayList<>(uris.size());
        Set<String> remoteNames = new LinkedHashSet<>();
        try {
            for (Uri uri : uris) {
                String name = displayName(uri);
                RemotePaths.validateName(name);
                if (!remoteNames.add(name)) {
                    throw new IllegalArgumentException(getString(R.string.duplicate_upload_name, name));
                }
                items.add(new UploadItem(uri, name, displaySize(uri)));
            }
        } catch (IllegalArgumentException error) {
            showError(error);
            return;
        }

        setBusy(true, R.string.status_uploading_files);
        beginTransferUi(items.size() > 1);
        io.execute(() -> {
            boolean stopped = false;
            try {
                for (int i = 0; i < items.size(); i++) {
                    UploadItem item = items.get(i);
                    String remotePath = RemotePaths.child(currentPath, item.name());
                    TransferReporter reporter = new TransferReporter(true, i + 1, items.size(), item.name(), item.size());
                    try (InputStream raw = getContentResolver().openInputStream(item.uri())) {
                        if (raw == null) throw new IllegalStateException("Unable to open selected file.");
                        InputStream source = TransferStreams.monitor(raw, reporter::report);
                        requireClient().upload(remotePath, source);
                    }
                    reporter.complete();
                    if (stopAfterCurrentRequested && i + 1 < items.size()) {
                        stopped = true;
                        break;
                    }
                }
                List<RemoteEntry> next = requireClient().list(currentPath);
                boolean finalStopped = stopped;
                postToMain(() -> {
                    replaceEntries(next);
                    finishTransferUi();
                    setBusy(false, finalStopped ? R.string.status_upload_stopped : R.string.status_connected);
                });
            } catch (Exception error) {
                postToMain(() -> {
                    finishTransferUi();
                    setBusy(false, R.string.status_connected);
                    showError(error);
                });
            }
        });
    }

    private void downloadUri(Uri uri, String remotePath, long remoteSize) {
        if (busy || client == null) return;
        setBusy(true, R.string.status_working);
        beginTransferUi(false);
        io.execute(() -> {
            try (OutputStream raw = getContentResolver().openOutputStream(uri, "w")) {
                if (raw == null) throw new IllegalStateException("Unable to open destination file.");
                TransferReporter reporter = new TransferReporter(false, 1, 1, "", remoteSize);
                OutputStream destination = TransferStreams.monitor(raw, reporter::report);
                requireClient().download(remotePath, destination);
                destination.flush();
                reporter.complete();
                postToMain(() -> {
                    finishTransferUi();
                    setBusy(false, R.string.status_connected);
                });
            } catch (Exception error) {
                postToMain(() -> {
                    finishTransferUi();
                    setBusy(false, R.string.status_connected);
                    showError(error);
                });
            }
        });
    }

    private void promptNewFolder() {
        if (client == null || busy) return;
        EditText name = input(R.string.folder_name, InputType.TYPE_CLASS_TEXT);
        new AlertDialog.Builder(this)
            .setTitle(R.string.new_folder)
            .setView(name)
            .setNegativeButton(R.string.cancel, null)
            .setPositiveButton(R.string.create, (dialog, which) -> mutateName(name.getText().toString(), null, true))
            .show();
    }

    private void promptRename(RemoteEntry entry) {
        EditText name = input(R.string.new_name, InputType.TYPE_CLASS_TEXT);
        name.setText(entry.name());
        name.selectAll();
        new AlertDialog.Builder(this)
            .setTitle(R.string.rename)
            .setView(name)
            .setNegativeButton(R.string.cancel, null)
            .setPositiveButton(R.string.save, (dialog, which) -> mutateName(name.getText().toString(), entry, false))
            .show();
    }

    private void mutateName(String rawName, RemoteEntry existing, boolean createDirectory) {
        String name = rawName == null ? "" : rawName;
        try {
            RemotePaths.validateName(name);
        } catch (IllegalArgumentException error) {
            showError(error);
            return;
        }
        runIo(() -> {
            if (createDirectory) requireClient().mkdir(RemotePaths.child(currentPath, name));
            else requireClient().rename(RemotePaths.child(currentPath, existing.name()), RemotePaths.child(currentPath, name));
        }, true);
    }

    private void showEntryActions(RemoteEntry entry) {
        if (entry.directory()) {
            String[] actions = {getString(R.string.rename), getString(R.string.delete)};
            new AlertDialog.Builder(this).setTitle(entry.name()).setItems(actions, (dialog, which) -> {
                if (which == 0) promptRename(entry);
                else confirmDelete(entry);
            }).show();
            return;
        }

        String[] actions = {getString(R.string.download), getString(R.string.rename), getString(R.string.delete)};
        new AlertDialog.Builder(this).setTitle(entry.name()).setItems(actions, (dialog, which) -> {
            if (which == 0) startDownload(entry);
            else if (which == 1) promptRename(entry);
            else confirmDelete(entry);
        }).show();
    }

    private void confirmDelete(RemoteEntry entry) {
        new AlertDialog.Builder(this)
            .setMessage(getString(R.string.delete_confirm, entry.name()))
            .setNegativeButton(R.string.cancel, null)
            .setPositiveButton(R.string.delete, (dialog, which) -> runIo(
                () -> requireClient().delete(RemotePaths.child(currentPath, entry.name()), entry.directory()),
                true
            ))
            .show();
    }

    private void runIo(ThrowingAction action, boolean refreshAfter) {
        runIo(R.string.status_working, action, refreshAfter);
    }

    private void runIo(int statusText, ThrowingAction action, boolean refreshAfter) {
        if (busy || client == null) return;
        setBusy(true, statusText);
        io.execute(() -> {
            try {
                action.run();
                List<RemoteEntry> next = refreshAfter ? requireClient().list(currentPath) : null;
                postToMain(() -> {
                    if (next != null) replaceEntries(next);
                    setBusy(false, R.string.status_connected);
                });
            } catch (Exception error) {
                postToMain(() -> {
                    setBusy(false, R.string.status_connected);
                    showError(error);
                });
            }
        });
    }

    private RemoteClient requireClient() {
        RemoteClient current = client;
        if (current == null) throw new IllegalStateException("Not connected.");
        return current;
    }

    private String displayName(Uri uri) {
        String providerName = null;
        try (Cursor cursor = getContentResolver().query(uri, new String[]{OpenableColumns.DISPLAY_NAME}, null, null, null)) {
            if (cursor != null && cursor.moveToFirst()) {
                int index = cursor.getColumnIndex(OpenableColumns.DISPLAY_NAME);
                if (index >= 0 && !cursor.isNull(index)) providerName = cursor.getString(index);
            }
        }
        return DocumentName.resolve(providerName, uri.getLastPathSegment());
    }

    private long displaySize(Uri uri) {
        try (Cursor cursor = getContentResolver().query(uri, new String[]{OpenableColumns.SIZE}, null, null, null)) {
            if (cursor != null && cursor.moveToFirst()) {
                int index = cursor.getColumnIndex(OpenableColumns.SIZE);
                if (index >= 0 && !cursor.isNull(index)) return Math.max(0L, cursor.getLong(index));
            }
        } catch (RuntimeException ignored) {
            // Some document providers do not expose a stable size. Progress falls back to transferred bytes.
        }
        return -1L;
    }

    private void beginTransferUi(boolean canStopAfterCurrent) {
        transferActive = true;
        stopAfterCurrentRequested = false;
        transferProgress.setIndeterminate(true);
        transferProgress.setProgress(0);
        transferProgress.setVisibility(View.VISIBLE);
        stopAfterCurrent.setVisibility(canStopAfterCurrent ? View.VISIBLE : View.GONE);
        stopAfterCurrent.setEnabled(canStopAfterCurrent);
    }

    private void finishTransferUi() {
        transferActive = false;
        stopAfterCurrentRequested = false;
        transferProgress.setProgress(0);
        transferProgress.setIndeterminate(false);
        transferProgress.setVisibility(View.GONE);
        stopAfterCurrent.setEnabled(false);
        stopAfterCurrent.setVisibility(View.GONE);
    }

    private void postToMain(Runnable action) {
        main.post(() -> { if (!destroyed) action.run(); });
    }

    private void setBusy(boolean value, int statusText) {
        busy = value;
        status.setText(statusText);
        updateEnabled();
    }

    private void updateConnectionUi(boolean connected) {
        formScroll.setVisibility(connected ? View.GONE : View.VISIBLE);
        connectionSummary.setVisibility(connected ? View.VISIBLE : View.GONE);
        filter.setVisibility(connected ? View.VISIBLE : View.GONE);
        protocol.setEnabled(!connected);
        host.setEnabled(!connected);
        port.setEnabled(!connected);
        username.setEnabled(!connected);
        password.setEnabled(!connected);
        fingerprint.setEnabled(!connected);
        updateEnabled();
    }

    private void updateEnabled() {
        boolean connected = client != null;
        connect.setEnabled(!busy && !connected);
        up.setEnabled(connected && !busy && !currentPath.equals("/"));
        refresh.setEnabled(connected && !busy);
        menu.setEnabled(connected && !busy);
        filter.setEnabled(connected && !busy);
        list.setEnabled(connected && !busy);
        if (stopAfterCurrent != null && stopAfterCurrent.getVisibility() == View.VISIBLE) {
            stopAfterCurrent.setEnabled(transferActive && busy && !stopAfterCurrentRequested);
        }
    }

    private void showError(Throwable error) {
        String message = error.getMessage();
        if (message == null || message.isBlank()) message = error.getClass().getSimpleName();
        message = message.replace('\r', ' ').replace('\n', ' ').trim();
        if (message.length() > 320) message = message.substring(0, 320);
        status.setText(getString(R.string.error_prefix, message));
    }

    private String text(EditText value) { return value.getText().toString(); }

    private EditText input(int hint, int inputType) {
        EditText field = new EditText(this);
        field.setHint(hint);
        field.setInputType(inputType);
        field.setSingleLine(true);
        field.setMinHeight(dp(48));
        field.setLayoutParams(matchWrap());
        return field;
    }

    private Button button(int text) {
        Button value = new Button(this);
        value.setText(text);
        value.setAllCaps(false);
        value.setMinHeight(dp(48));
        return value;
    }

    private LinearLayout.LayoutParams matchWrap() {
        return new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT);
    }

    private LinearLayout.LayoutParams weight() {
        return new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f);
    }

    private int dp(int value) { return Math.round(value * getResources().getDisplayMetrics().density); }

    private String humanBytes(long bytes) {
        if (bytes < 1024L) return bytes + " B";
        double value = bytes;
        String[] units = {"KB", "MB", "GB", "TB"};
        for (String unit : units) {
            value /= 1024.0;
            if (value < 1024.0) return String.format(Locale.ROOT, "%.1f %s", value, unit);
        }
        return bytes + " B";
    }

    @Override protected void onDestroy() {
        destroyed = true;
        stopAfterCurrentRequested = true;
        pendingDownloadPath = null;
        pendingDownloadSize = -1L;
        password.setText("");
        main.removeCallbacksAndMessages(null);
        RemoteClient current = client;
        RemoteClient pending = connectingClient;
        client = null;
        connectingClient = null;
        io.shutdownNow();
        if (current != null || pending != null) {
            Thread closeThread = new Thread(() -> {
                if (current != null) current.close();
                if (pending != null && pending != current) pending.close();
            }, "GhostFTP-close");
            closeThread.start();
        }
        super.onDestroy();
    }

    private record UploadItem(Uri uri, String name, long size) {}

    private final class TransferReporter {
        private static final long POST_INTERVAL_NANOS = 100_000_000L;

        private final boolean upload;
        private final int fileIndex;
        private final int fileCount;
        private final String name;
        private final long totalBytes;
        private long transferred;
        private long lastPostNanos;

        private TransferReporter(boolean upload, int fileIndex, int fileCount, String name, long totalBytes) {
            this.upload = upload;
            this.fileIndex = fileIndex;
            this.fileCount = fileCount;
            this.name = name;
            this.totalBytes = totalBytes;
        }

        private void report(long bytes) {
            transferred = Math.max(transferred, bytes);
            publish(false);
        }

        private void complete() {
            if (totalBytes >= 0L) transferred = Math.max(transferred, totalBytes);
            publish(true);
        }

        private void publish(boolean force) {
            long now = System.nanoTime();
            if (!force && lastPostNanos != 0L && now - lastPostNanos < POST_INTERVAL_NANOS) return;
            lastPostNanos = now;
            long current = transferred;
            postToMain(() -> renderTransferProgress(upload, fileIndex, fileCount, name, current, totalBytes));
        }
    }

    private void renderTransferProgress(boolean upload, int fileIndex, int fileCount, String name, long transferred, long totalBytes) {
        if (!transferActive) return;
        if (totalBytes >= 0L) {
            int percent = totalBytes == 0L ? 100 : (int) Math.min(100L, (transferred * 100L) / Math.max(1L, totalBytes));
            transferProgress.setIndeterminate(false);
            transferProgress.setProgress(percent * 10);
            if (stopAfterCurrentRequested && upload) {
                status.setText(R.string.status_stopping_after_current);
            } else if (upload) {
                status.setText(getString(R.string.upload_progress_known, fileIndex, fileCount, percent, name));
            } else {
                status.setText(getString(R.string.download_progress_known, percent, humanBytes(transferred)));
            }
        } else {
            transferProgress.setIndeterminate(true);
            if (stopAfterCurrentRequested && upload) {
                status.setText(R.string.status_stopping_after_current);
            } else if (upload) {
                status.setText(getString(R.string.upload_progress_unknown, fileIndex, fileCount, humanBytes(transferred), name));
            } else {
                status.setText(getString(R.string.download_progress_unknown, humanBytes(transferred)));
            }
        }
    }

    @FunctionalInterface private interface ThrowingAction { void run() throws Exception; }
}
