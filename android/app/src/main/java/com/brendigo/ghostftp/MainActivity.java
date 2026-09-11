package com.brendigo.ghostftp;

import android.app.Activity;
import android.app.AlertDialog;
import android.content.Intent;
import android.database.Cursor;
import android.graphics.Typeface;
import android.net.Uri;
import android.os.Bundle;
import android.os.DocumentsContract;
import android.os.Looper;
import android.provider.OpenableColumns;
import android.text.Editable;
import android.text.InputType;
import android.view.Gravity;
import android.view.View;
import android.view.ViewGroup;
import android.widget.AdapterView;
import android.widget.ArrayAdapter;
import android.widget.Button;
import android.widget.EditText;
import android.widget.HorizontalScrollView;
import android.widget.LinearLayout;
import android.widget.ListView;
import android.widget.Spinner;
import android.widget.TextView;
import com.brendigo.ghostftp.protocol.ConnectionSpec;
import com.brendigo.ghostftp.protocol.FtpFtpsClient;
import com.brendigo.ghostftp.protocol.RemoteClient;
import com.brendigo.ghostftp.protocol.RemoteEntry;
import com.brendigo.ghostftp.protocol.SftpClient;
import com.brendigo.ghostftp.protocol.TrustPrompt;
import java.io.FileNotFoundException;
import java.io.InputStream;
import java.io.OutputStream;
import java.util.ArrayList;
import java.util.List;
import java.util.Locale;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicBoolean;

public final class MainActivity extends Activity implements TrustPrompt {
    private static final int REQUEST_UPLOAD = 2101;
    private static final int REQUEST_DOWNLOAD = 2102;

    private final ExecutorService io = Executors.newSingleThreadExecutor();
    private final List<RemoteEntry> entries = new ArrayList<>();
    private final List<String> entryLabels = new ArrayList<>();

    private Spinner protocolSpinner;
    private EditText hostField;
    private EditText portField;
    private EditText usernameField;
    private EditText passwordField;
    private EditText pathField;
    private Button connectButton;
    private Button disconnectButton;
    private Button openButton;
    private Button upButton;
    private Button refreshButton;
    private Button uploadButton;
    private Button downloadButton;
    private Button newFolderButton;
    private Button deleteButton;
    private ListView listView;
    private TextView statusView;
    private ArrayAdapter<String> listAdapter;

    private RemoteClient client;
    private int selectedIndex = -1;
    private boolean busy;
    private volatile boolean destroyed;
    private RemoteEntry pendingDownload;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(buildUi());
        configureProtocolSpinner();
        configureActions();
        setStatus(getString(R.string.status_ready));
        updateControls();
    }

    private View buildUi() {
        int gap = dp(12);
        LinearLayout root = new LinearLayout(this);
        root.setOrientation(LinearLayout.VERTICAL);
        root.setPadding(gap, gap, gap, gap);
        root.setBackgroundColor(getColor(R.color.ghost_background));

        TextView title = textView(getString(R.string.app_name), 24, true);
        root.addView(title, fullWrap());
        TextView subtitle = textView(getString(R.string.app_subtitle), 13, false);
        subtitle.setTextColor(getColor(R.color.ghost_muted));
        root.addView(subtitle, fullWrap());

        LinearLayout protocolRow = horizontalRow();
        protocolSpinner = new Spinner(this);
        protocolRow.addView(protocolSpinner, weightedWrap(1f));
        portField = editText(getString(R.string.port), InputType.TYPE_CLASS_NUMBER);
        portField.setText("21");
        protocolRow.addView(portField, new LinearLayout.LayoutParams(dp(110), ViewGroup.LayoutParams.WRAP_CONTENT));
        root.addView(protocolRow, marginTop(fullWrap(), gap));

        hostField = editText(getString(R.string.host), InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_VARIATION_URI);
        root.addView(hostField, marginTop(fullWrap(), dp(8)));
        usernameField = editText(getString(R.string.username), InputType.TYPE_CLASS_TEXT);
        root.addView(usernameField, marginTop(fullWrap(), dp(8)));
        passwordField = editText(getString(R.string.password), InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_VARIATION_PASSWORD);
        root.addView(passwordField, marginTop(fullWrap(), dp(8)));

        LinearLayout connectionRow = horizontalRow();
        connectButton = button(getString(R.string.connect));
        disconnectButton = button(getString(R.string.disconnect));
        connectionRow.addView(connectButton, weightedWrap(1f));
        connectionRow.addView(disconnectButton, weightedWrap(1f));
        root.addView(connectionRow, marginTop(fullWrap(), dp(8)));

        LinearLayout pathRow = horizontalRow();
        pathField = editText(getString(R.string.remote_path), InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_VARIATION_URI);
        pathField.setText("/");
        upButton = button(getString(R.string.up));
        refreshButton = button(getString(R.string.refresh));
        pathRow.addView(pathField, weightedWrap(1f));
        pathRow.addView(upButton, new LinearLayout.LayoutParams(dp(72), ViewGroup.LayoutParams.WRAP_CONTENT));
        pathRow.addView(refreshButton, new LinearLayout.LayoutParams(dp(96), ViewGroup.LayoutParams.WRAP_CONTENT));
        root.addView(pathRow, marginTop(fullWrap(), dp(12)));

        HorizontalScrollView actionScroll = new HorizontalScrollView(this);
        actionScroll.setHorizontalScrollBarEnabled(false);
        LinearLayout actionRow = horizontalRow();
        openButton = button(getString(R.string.open));
        uploadButton = button(getString(R.string.upload));
        downloadButton = button(getString(R.string.download));
        newFolderButton = button(getString(R.string.new_folder));
        deleteButton = button(getString(R.string.delete));
        for (Button action : new Button[]{openButton, uploadButton, downloadButton, newFolderButton, deleteButton}) {
            actionRow.addView(action, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.WRAP_CONTENT, ViewGroup.LayoutParams.WRAP_CONTENT));
        }
        actionScroll.addView(actionRow);
        root.addView(actionScroll, marginTop(fullWrap(), dp(8)));

        listView = new ListView(this);
        listView.setChoiceMode(ListView.CHOICE_MODE_SINGLE);
        listView.setDividerHeight(1);
        listAdapter = new ArrayAdapter<String>(this, android.R.layout.simple_list_item_activated_1, entryLabels) {
            @Override
            public View getView(int position, View convertView, ViewGroup parent) {
                TextView view = (TextView) super.getView(position, convertView, parent);
                view.setTextColor(getColor(R.color.ghost_text));
                view.setTextSize(15f);
                return view;
            }
        };
        listView.setAdapter(listAdapter);
        LinearLayout.LayoutParams listParams = new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, 0, 1f);
        listParams.topMargin = dp(8);
        root.addView(listView, listParams);

        statusView = textView("", 12, false);
        statusView.setTextColor(getColor(R.color.ghost_muted));
        statusView.setPadding(0, dp(8), 0, 0);
        root.addView(statusView, fullWrap());
        return root;
    }

    private void configureProtocolSpinner() {
        String[] protocols = {"FTPS", "SFTP", "FTP"};
        ArrayAdapter<String> adapter = new ArrayAdapter<>(this, android.R.layout.simple_spinner_item, protocols);
        adapter.setDropDownViewResource(android.R.layout.simple_spinner_dropdown_item);
        protocolSpinner.setAdapter(adapter);
        protocolSpinner.setSelection(0);
        protocolSpinner.setOnItemSelectedListener(new AdapterView.OnItemSelectedListener() {
            @Override
            public void onItemSelected(AdapterView<?> parent, View view, int position, long id) {
                String protocol = protocols[position];
                String port = portField.getText().toString().trim();
                String path = pathField.getText().toString().trim();
                if (protocol.equals("SFTP")) {
                    if (port.isEmpty() || port.equals("21")) {
                        portField.setText("22");
                    }
                    if (path.equals("/")) {
                        pathField.setText(".");
                    }
                } else {
                    if (port.isEmpty() || port.equals("22")) {
                        portField.setText("21");
                    }
                    if (path.equals(".")) {
                        pathField.setText("/");
                    }
                }
            }

            @Override
            public void onNothingSelected(AdapterView<?> parent) {}
        });
    }

    private void configureActions() {
        connectButton.setOnClickListener(view -> connectRequested());
        disconnectButton.setOnClickListener(view -> disconnectRequested());
        refreshButton.setOnClickListener(view -> loadPath(pathField.getText().toString()));
        upButton.setOnClickListener(view -> loadPath(".."));
        openButton.setOnClickListener(view -> openSelected());
        uploadButton.setOnClickListener(view -> chooseUpload());
        downloadButton.setOnClickListener(view -> chooseDownload());
        newFolderButton.setOnClickListener(view -> promptNewFolder());
        deleteButton.setOnClickListener(view -> confirmDelete());
        listView.setOnItemClickListener((parent, view, position, id) -> {
            selectedIndex = position;
            listView.setItemChecked(position, true);
            setStatus(getString(R.string.status_selected, entries.get(position).name()));
            updateControls();
        });
        listView.setOnItemLongClickListener((parent, view, position, id) -> {
            selectedIndex = position;
            listView.setItemChecked(position, true);
            RemoteEntry entry = entries.get(position);
            if (entry.isDirectory() && !entry.isSymlink()) {
                loadPath(entry.name());
            } else {
                setStatus(getString(R.string.status_selected, entry.name()));
                updateControls();
            }
            return true;
        });
    }

    private void connectRequested() {
        if (busy || client != null) {
            return;
        }
        ConnectionSpec spec;
        try {
            String protocolLabel = String.valueOf(protocolSpinner.getSelectedItem());
            int port = Integer.parseInt(portField.getText().toString().trim());
            Editable editable = passwordField.getText();
            char[] password = new char[editable.length()];
            editable.getChars(0, editable.length(), password, 0);
            passwordField.setText("");
            try {
                spec = new ConnectionSpec(
                        ConnectionSpec.Protocol.fromLabel(protocolLabel),
                        hostField.getText().toString(),
                        port,
                        usernameField.getText().toString(),
                        password);
            } finally {
                java.util.Arrays.fill(password, '\0');
            }
        } catch (Exception error) {
            showError(getString(R.string.validation_error));
            return;
        }

        if (spec.protocol() == ConnectionSpec.Protocol.FTP) {
            new AlertDialog.Builder(this)
                    .setTitle(R.string.ftp_warning_title)
                    .setMessage(R.string.ftp_warning_message)
                    .setNegativeButton(R.string.cancel, (dialog, which) -> spec.clearPassword())
                    .setPositiveButton(R.string.continue_action, (dialog, which) -> startConnect(spec))
                    .show();
            return;
        }
        startConnect(spec);
    }

    private void startConnect(ConnectionSpec spec) {
        String requestedPath = pathField.getText().toString();
        runTask(getString(R.string.status_connecting, spec.host()), () -> {
            RemoteClient next = spec.protocol() == ConnectionSpec.Protocol.SFTP
                    ? new SftpClient(getApplicationContext())
                    : new FtpFtpsClient();
            try {
                next.connect(spec, this);
                List<RemoteEntry> items = next.list(requestedPath);
                return new ConnectedState(next, next.currentPath(), items);
            } catch (Exception error) {
                next.close();
                throw error;
            } finally {
                spec.clearPassword();
            }
        }, state -> {
            client = state.client;
            applyListing(state.path, state.items);
            setStatus(getString(R.string.status_connected, hostField.getText().toString().trim()));
        });
    }

    private void disconnectRequested() {
        RemoteClient active = client;
        if (active == null || busy) {
            return;
        }
        runTask(getString(R.string.status_disconnected), () -> {
            active.close();
            return active;
        }, closed -> {
            if (client == closed) {
                client = null;
            }
            entries.clear();
            entryLabels.clear();
            listAdapter.notifyDataSetChanged();
            selectedIndex = -1;
            setStatus(getString(R.string.status_disconnected));
        });
    }

    private void loadPath(String requestedPath) {
        RemoteClient active = client;
        if (active == null || busy) {
            return;
        }
        runTask(getString(R.string.status_loading, requestedPath), () -> {
            List<RemoteEntry> items = active.list(requestedPath);
            return new ListingState(active.currentPath(), items);
        }, state -> {
            if (client == active) {
                applyListing(state.path, state.items);
                setStatus(getString(R.string.status_items, state.items.size()));
            }
        });
    }

    private void openSelected() {
        RemoteEntry entry = selectedEntry();
        if (entry == null) {
            showError(getString(R.string.select_entry));
            return;
        }
        if (entry.isSymlink()) {
            showError(getString(R.string.symlink_blocked));
            return;
        }
        if (!entry.isDirectory()) {
            showError(getString(R.string.select_directory));
            return;
        }
        loadPath(entry.name());
    }

    private void chooseUpload() {
        if (client == null || busy) {
            return;
        }
        Intent intent = new Intent(Intent.ACTION_OPEN_DOCUMENT);
        intent.addCategory(Intent.CATEGORY_OPENABLE);
        intent.setType("application/octet-stream");
        intent.addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION);
        startActivityForResult(intent, REQUEST_UPLOAD);
    }

    private void chooseDownload() {
        RemoteEntry entry = selectedEntry();
        if (entry == null || entry.isDirectory()) {
            showError(getString(R.string.select_file));
            return;
        }
        if (entry.isSymlink()) {
            showError(getString(R.string.symlink_blocked));
            return;
        }
        pendingDownload = entry;
        Intent intent = new Intent(Intent.ACTION_CREATE_DOCUMENT);
        intent.addCategory(Intent.CATEGORY_OPENABLE);
        intent.setType("application/octet-stream");
        intent.putExtra(Intent.EXTRA_TITLE, entry.name());
        intent.addFlags(Intent.FLAG_GRANT_WRITE_URI_PERMISSION);
        startActivityForResult(intent, REQUEST_DOWNLOAD);
    }

    @Override
    protected void onActivityResult(int requestCode, int resultCode, Intent data) {
        super.onActivityResult(requestCode, resultCode, data);
        if (resultCode != RESULT_OK || data == null || data.getData() == null) {
            if (requestCode == REQUEST_DOWNLOAD) {
                pendingDownload = null;
            }
            return;
        }
        Uri uri = data.getData();
        if (requestCode == REQUEST_UPLOAD) {
            uploadUri(uri);
        } else if (requestCode == REQUEST_DOWNLOAD) {
            RemoteEntry entry = pendingDownload;
            pendingDownload = null;
            if (entry != null) {
                downloadUri(entry, uri);
            }
        }
    }

    private void uploadUri(Uri uri) {
        RemoteClient active = client;
        if (active == null || busy) {
            return;
        }
        String name = displayName(uri);
        if (name == null || name.trim().isEmpty()) {
            showError(getString(R.string.validation_error));
            return;
        }
        for (RemoteEntry entry : entries) {
            if (entry.name().equals(name)) {
                showError(getString(R.string.upload_conflict, name));
                return;
            }
        }
        String finalName = name;
        runTask(getString(R.string.upload), () -> {
            try (InputStream input = getContentResolver().openInputStream(uri)) {
                if (input == null) {
                    throw new FileNotFoundException("Unable to open the selected Android document.");
                }
                active.upload(input, finalName);
            }
            List<RemoteEntry> items = active.list(".");
            return new ListingState(active.currentPath(), items);
        }, state -> {
            if (client == active) {
                applyListing(state.path, state.items);
                setStatus(getString(R.string.status_uploaded, finalName));
            }
        });
    }

    private void downloadUri(RemoteEntry entry, Uri uri) {
        RemoteClient active = client;
        if (active == null || busy) {
            return;
        }
        runTask(getString(R.string.download), () -> {
            try (OutputStream output = getContentResolver().openOutputStream(uri, "w")) {
                if (output == null) {
                    throw new FileNotFoundException("Unable to create the selected Android document.");
                }
                active.download(entry, output);
            } catch (Exception error) {
                deleteDocumentQuietly(uri);
                throw error;
            }
            return entry.name();
        }, name -> setStatus(getString(R.string.status_downloaded, name)));
    }

    private void promptNewFolder() {
        if (client == null || busy) {
            return;
        }
        EditText input = editText(getString(R.string.folder_name), InputType.TYPE_CLASS_TEXT);
        int pad = dp(20);
        LinearLayout holder = new LinearLayout(this);
        holder.setPadding(pad, 0, pad, 0);
        holder.addView(input, fullWrap());
        new AlertDialog.Builder(this)
                .setTitle(R.string.new_folder)
                .setView(holder)
                .setNegativeButton(R.string.cancel, null)
                .setPositiveButton(R.string.create, (dialog, which) -> createFolder(input.getText().toString()))
                .show();
    }

    private void createFolder(String name) {
        RemoteClient active = client;
        if (active == null || busy) {
            return;
        }
        runTask(getString(R.string.new_folder), () -> {
            active.mkdir(name);
            List<RemoteEntry> items = active.list(".");
            return new ListingState(active.currentPath(), items);
        }, state -> {
            if (client == active) {
                applyListing(state.path, state.items);
                setStatus(getString(R.string.status_created, name.trim()));
            }
        });
    }

    private void confirmDelete() {
        RemoteEntry entry = selectedEntry();
        if (entry == null) {
            showError(getString(R.string.select_entry));
            return;
        }
        if (entry.isSymlink()) {
            showError(getString(R.string.symlink_blocked));
            return;
        }
        new AlertDialog.Builder(this)
                .setTitle(R.string.delete_confirm_title)
                .setMessage(getString(R.string.delete_confirm_message, entry.name()))
                .setNegativeButton(R.string.cancel, null)
                .setPositiveButton(R.string.delete, (dialog, which) -> deleteEntry(entry))
                .show();
    }

    private void deleteEntry(RemoteEntry entry) {
        RemoteClient active = client;
        if (active == null || busy) {
            return;
        }
        runTask(getString(R.string.delete), () -> {
            active.delete(entry);
            List<RemoteEntry> items = active.list(".");
            return new ListingState(active.currentPath(), items);
        }, state -> {
            if (client == active) {
                applyListing(state.path, state.items);
                setStatus(getString(R.string.status_deleted, entry.name()));
            }
        });
    }

    private RemoteEntry selectedEntry() {
        if (selectedIndex < 0 || selectedIndex >= entries.size()) {
            return null;
        }
        return entries.get(selectedIndex);
    }

    private void applyListing(String path, List<RemoteEntry> items) {
        entries.clear();
        entries.addAll(items);
        entryLabels.clear();
        for (RemoteEntry entry : items) {
            String prefix = entry.isSymlink()
                    ? getString(R.string.link_prefix)
                    : entry.isDirectory() ? getString(R.string.directory_prefix) : getString(R.string.file_prefix);
            String size = entry.isDirectory() ? "" : "  " + humanSize(entry.size());
            entryLabels.add(String.format(Locale.ROOT, "%s  %s%s", prefix, entry.name(), size));
        }
        if (entryLabels.isEmpty()) {
            entryLabels.add(getString(R.string.no_items));
        }
        selectedIndex = -1;
        listView.clearChoices();
        listAdapter.notifyDataSetChanged();
        pathField.setText(path);
        updateControls();
    }

    private void updateControls() {
        boolean connected = client != null && client.isConnected();
        protocolSpinner.setEnabled(!connected && !busy);
        hostField.setEnabled(!connected && !busy);
        portField.setEnabled(!connected && !busy);
        usernameField.setEnabled(!connected && !busy);
        passwordField.setEnabled(!connected && !busy);
        connectButton.setEnabled(!connected && !busy);
        disconnectButton.setEnabled(connected && !busy);
        pathField.setEnabled(!busy);
        upButton.setEnabled(connected && !busy);
        refreshButton.setEnabled(connected && !busy);
        uploadButton.setEnabled(connected && !busy);
        newFolderButton.setEnabled(connected && !busy);
        RemoteEntry selected = selectedEntry();
        openButton.setEnabled(connected && !busy && selected != null && selected.isDirectory() && !selected.isSymlink());
        downloadButton.setEnabled(connected && !busy && selected != null && !selected.isDirectory() && !selected.isSymlink());
        deleteButton.setEnabled(connected && !busy && selected != null && !selected.isSymlink());
    }

    private <T> void runTask(String progress, CheckedTask<T> task, ResultHandler<T> success) {
        if (busy || destroyed) {
            return;
        }
        busy = true;
        setStatus(progress);
        updateControls();
        io.execute(() -> {
            try {
                T value = task.run();
                runOnUiThread(() -> {
                    if (destroyed) {
                        return;
                    }
                    busy = false;
                    success.handle(value);
                    updateControls();
                });
            } catch (Exception error) {
                runOnUiThread(() -> {
                    if (destroyed) {
                        return;
                    }
                    busy = false;
                    showError(userSafeMessage(error));
                    updateControls();
                });
            }
        });
    }

    private String displayName(Uri uri) {
        try (Cursor cursor = getContentResolver().query(uri, new String[]{OpenableColumns.DISPLAY_NAME}, null, null, null)) {
            if (cursor != null && cursor.moveToFirst()) {
                int index = cursor.getColumnIndex(OpenableColumns.DISPLAY_NAME);
                if (index >= 0) {
                    return cursor.getString(index);
                }
            }
        } catch (RuntimeException ignored) {
            // Provider metadata is optional; caller will reject a missing name.
        }
        return null;
    }

    private void deleteDocumentQuietly(Uri uri) {
        try {
            DocumentsContract.deleteDocument(getContentResolver(), uri);
        } catch (Exception ignored) {
            // The provider may not support deletion. The operation is still reported failed.
        }
    }

    private String userSafeMessage(Throwable error) {
        String message = error == null ? "" : error.getMessage();
        if (message == null || message.trim().isEmpty()) {
            message = getString(R.string.error_title);
        }
        message = message.replace('\r', ' ').replace('\n', ' ').trim();
        if (message.length() > 300) {
            message = message.substring(0, 297) + "...";
        }
        return message;
    }

    private void showError(String message) {
        setStatus(message);
        new AlertDialog.Builder(this)
                .setTitle(R.string.error_title)
                .setMessage(message)
                .setPositiveButton(R.string.ok, null)
                .show();
    }

    private void setStatus(String message) {
        if (statusView != null) {
            statusView.setText(message == null ? "" : message.replace('\r', ' ').replace('\n', ' '));
        }
    }

    @Override
    public boolean confirmNewHostKey(String host, String algorithm, String fingerprint) {
        String message = getString(R.string.sftp_new_key_message, host, algorithm, fingerprint);
        return blockingConfirmation(getString(R.string.sftp_new_key_title), message);
    }

    @Override
    public void reportChangedHostKey(String host, String fingerprint) {
        String message = getString(R.string.sftp_changed_key_message, host, fingerprint);
        blockingInformation(getString(R.string.sftp_changed_key_title), message);
    }

    private boolean blockingConfirmation(String title, String message) {
        if (Looper.myLooper() == Looper.getMainLooper() || destroyed) {
            return false;
        }
        CountDownLatch latch = new CountDownLatch(1);
        AtomicBoolean accepted = new AtomicBoolean(false);
        runOnUiThread(() -> {
            if (destroyed || isFinishing()) {
                latch.countDown();
                return;
            }
            AlertDialog dialog = new AlertDialog.Builder(this)
                    .setTitle(title)
                    .setMessage(message)
                    .setCancelable(false)
                    .setNegativeButton(R.string.cancel, (d, which) -> latch.countDown())
                    .setPositiveButton(R.string.trust, (d, which) -> {
                        accepted.set(true);
                        latch.countDown();
                    })
                    .create();
            dialog.show();
        });
        try {
            if (!latch.await(2, TimeUnit.MINUTES)) {
                return false;
            }
        } catch (InterruptedException error) {
            Thread.currentThread().interrupt();
            return false;
        }
        return accepted.get();
    }

    private void blockingInformation(String title, String message) {
        if (Looper.myLooper() == Looper.getMainLooper() || destroyed) {
            return;
        }
        CountDownLatch latch = new CountDownLatch(1);
        runOnUiThread(() -> {
            if (destroyed || isFinishing()) {
                latch.countDown();
                return;
            }
            new AlertDialog.Builder(this)
                    .setTitle(title)
                    .setMessage(message)
                    .setCancelable(false)
                    .setPositiveButton(R.string.ok, (dialog, which) -> latch.countDown())
                    .show();
        });
        try {
            latch.await(2, TimeUnit.MINUTES);
        } catch (InterruptedException error) {
            Thread.currentThread().interrupt();
        }
    }

    private TextView textView(String text, int sp, boolean bold) {
        TextView view = new TextView(this);
        view.setText(text);
        view.setTextSize(sp);
        view.setTextColor(getColor(R.color.ghost_text));
        if (bold) {
            view.setTypeface(Typeface.DEFAULT, Typeface.BOLD);
        }
        return view;
    }

    private EditText editText(String hint, int inputType) {
        EditText view = new EditText(this);
        view.setHint(hint);
        view.setHintTextColor(getColor(R.color.ghost_muted));
        view.setTextColor(getColor(R.color.ghost_text));
        view.setSingleLine(true);
        view.setInputType(inputType);
        return view;
    }

    private Button button(String text) {
        Button button = new Button(this);
        button.setText(text);
        button.setAllCaps(false);
        button.setGravity(Gravity.CENTER);
        return button;
    }

    private LinearLayout horizontalRow() {
        LinearLayout row = new LinearLayout(this);
        row.setOrientation(LinearLayout.HORIZONTAL);
        row.setGravity(Gravity.CENTER_VERTICAL);
        return row;
    }

    private LinearLayout.LayoutParams fullWrap() {
        return new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT);
    }

    private LinearLayout.LayoutParams weightedWrap(float weight) {
        LinearLayout.LayoutParams params = new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, weight);
        params.setMarginEnd(dp(8));
        return params;
    }

    private LinearLayout.LayoutParams marginTop(LinearLayout.LayoutParams params, int margin) {
        params.topMargin = margin;
        return params;
    }

    private int dp(int value) {
        return Math.round(value * getResources().getDisplayMetrics().density);
    }

    private static String humanSize(long bytes) {
        if (bytes < 1024L) {
            return bytes + " B";
        }
        if (bytes < 1024L * 1024L) {
            return String.format(Locale.ROOT, "%.1f KiB", bytes / 1024.0);
        }
        if (bytes < 1024L * 1024L * 1024L) {
            return String.format(Locale.ROOT, "%.1f MiB", bytes / (1024.0 * 1024.0));
        }
        return String.format(Locale.ROOT, "%.1f GiB", bytes / (1024.0 * 1024.0 * 1024.0));
    }

    @Override
    protected void onDestroy() {
        destroyed = true;
        RemoteClient active = client;
        client = null;
        if (active != null) {
            active.close();
        }
        io.shutdownNow();
        super.onDestroy();
    }

    private interface CheckedTask<T> {
        T run() throws Exception;
    }

    private interface ResultHandler<T> {
        void handle(T value);
    }

    private static final class ListingState {
        final String path;
        final List<RemoteEntry> items;

        ListingState(String path, List<RemoteEntry> items) {
            this.path = path;
            this.items = items;
        }
    }

    private static final class ConnectedState {
        final RemoteClient client;
        final String path;
        final List<RemoteEntry> items;

        ConnectedState(RemoteClient client, String path, List<RemoteEntry> items) {
            this.client = client;
            this.path = path;
            this.items = items;
        }
    }
}
