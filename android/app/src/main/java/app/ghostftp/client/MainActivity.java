package app.ghostftp.client;

import android.annotation.SuppressLint;
import android.app.Activity;
import android.app.AlertDialog;
import android.content.Intent;
import android.content.SharedPreferences;
import android.content.UriPermission;
import android.content.res.ColorStateList;
import android.database.Cursor;
import android.graphics.Color;
import android.graphics.Typeface;
import android.net.Uri;
import android.os.Bundle;
import android.provider.DocumentsContract;
import android.text.Editable;
import android.text.InputType;
import android.text.TextWatcher;
import android.view.Gravity;
import android.view.View;
import android.view.ViewGroup;
import android.view.accessibility.AccessibilityEvent;
import android.widget.Button;
import android.widget.CheckBox;
import android.widget.EditText;
import android.widget.FrameLayout;
import android.widget.ImageButton;
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
import java.util.HashSet;
import java.util.List;
import java.util.Set;
import java.util.UUID;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

@SuppressLint("SetTextI18n")
public final class MainActivity extends Activity {
    private static final int REQUEST_TREE = 1001;
    private static final String PREFS = "ghostftp_android";
    private static final int TABLET_SIDEBAR_MIN_DP = 700;

    private enum Section {
        FILES,
        SITES,
        BOOKMARKS,
        TRANSFERS,
        SETTINGS,
        ABOUT
    }

    private interface TextPromptAction {
        void accept(String value);
    }

    private final ExecutorService io = Executors.newSingleThreadExecutor();
    private final List<LocalEntry> localEntries = new ArrayList<>();
    private final List<RemoteEntry> remoteEntries = new ArrayList<>();
    private final Deque<String> localParents = new ArrayDeque<>();
    private final List<SiteProfile> profiles = new ArrayList<>();
    private final List<Button> navigationButtons = new ArrayList<>();
    private final List<WorkspaceOps.Item> localVisibleItems = new ArrayList<>();
    private final List<WorkspaceOps.Item> remoteVisibleItems = new ArrayList<>();

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
    private TextView bookmarkLocalCurrent;
    private TextView bookmarkRemoteCurrent;
    private TextView status;
    private TextView transferStatus;
    private TextView sectionTitle;
    private TextView connectionBadge;
    private TextView localEmptyState;
    private TextView remoteEmptyState;
    private ListView localList;
    private ListView remoteList;
    private Button connect;
    private Button disconnect;
    private Button upload;
    private Button download;
    private Button localCreateDirectory;
    private Button localRename;
    private Button localDelete;
    private Button remoteCreateDirectory;
    private Button remoteRename;
    private Button remoteDelete;
    private Button remoteChmod;
    private Button localFilter;
    private Button localSort;
    private Button localSearch;
    private Button remoteFilter;
    private Button remoteSort;
    private Button remoteSearch;
    private Button directoryCompare;
    private Button remoteEdit;
    private Button saveSite;
    private Button deleteSite;
    private Button localSetStart;
    private Button localAddBookmark;
    private Button localRemoveBookmark;
    private Button localOpenBookmark;
    private Button remoteSetStart;
    private Button remoteAddBookmark;
    private Button remoteRemoveBookmark;
    private Button remoteOpenBookmark;
    private Button transferCancel;
    private Button applyAppearance;
    private CheckBox rememberEndpointToggle;
    private CheckBox showFileSizesToggle;
    private Spinner appearanceSpinner;

    private FrameLayout contentHost;
    private LinearLayout navigationPanel;
    private View drawerScrim;
    private ImageButton menuToggle;
    private View filesSurface;
    private View sitesSurface;
    private View bookmarksSurface;
    private View transfersSurface;
    private View settingsSurface;
    private View aboutSurface;
    private boolean tabletLayout;
    private Section activeSection = Section.FILES;

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
    private volatile FtpSession connectingSession;
    private volatile boolean lifecycleDestroyed;
    private boolean busy;
    private boolean rememberEndpoint = true;
    private boolean showFileSizes = true;
    private String appearanceMode = GhostTheme.APPEARANCE_DARK;
    private String localFilterQuery = "";
    private String remoteFilterQuery = "";
    private WorkspaceOps.SortKey localSortKey = WorkspaceOps.SortKey.NAME;
    private WorkspaceOps.SortKey remoteSortKey = WorkspaceOps.SortKey.NAME;
    private boolean localSortAscending = true;
    private boolean remoteSortAscending = true;
    private volatile long advancedOperationGeneration;
    private RemoteEditorState remoteEditorState;
    private volatile boolean transferActive;
    private volatile boolean transferFinalizing;
    private volatile long transferGeneration;
    private volatile TransferCommitGate activeTransferGate;

    @Override
    protected void onCreate(Bundle state) {
        super.onCreate(state);
        SharedPreferences preferences = getSharedPreferences(PREFS, MODE_PRIVATE);
        appearanceMode = GhostTheme.normalizeAppearance(
                preferences.getString("appearance", GhostTheme.APPEARANCE_DARK));
        GhostTheme.apply(this, appearanceMode);
        GhostTheme.applySystemBars(this);
        profileStore = new SiteProfileStore(preferences);
        profiles.addAll(profileStore.load());
        buildUi();
        restorePreferences();
        renderSites();
        renderBookmarks();
        renderLocal();
        renderRemote();
        refreshButtons();
        showSection(Section.FILES);
    }

    @Override
    protected void onDestroy() {
        lifecycleDestroyed = true;
        FtpSession pending = connectingSession;
        connectingSession = null;
        if (pending != null) {
            pending.abort();
        }
        transferGeneration++;
        advancedOperationGeneration++;
        RemoteEditorState editorState = remoteEditorState;
        remoteEditorState = null;
        if (editorState != null && editorState.dialog != null) editorState.dialog.dismiss();
        transferActive = false;
        transferFinalizing = false;
        TransferCommitGate gate = activeTransferGate;
        activeTransferGate = null;
        FtpSession current = session;
        session = null;
        connectedIdentityKey = null;
        if (current != null) {
            if (gate == null) {
                current.cancelActiveTransfer();
            } else {
                TransferCommitGate.CancelDisposition disposition = current.cancelActiveTransfer(gate);
                if (disposition == TransferCommitGate.CancelDisposition.ALREADY_CANCELLED) {
                    current.cancelActiveTransfer();
                }
            }
        }
        io.shutdownNow();
        super.onDestroy();
    }

    @SuppressWarnings("deprecation")
    @Override
    public void onBackPressed() {
        if (!tabletLayout && navigationPanel != null && navigationPanel.getVisibility() == View.VISIBLE) {
            closeNavigationDrawer();
            return;
        }
        super.onBackPressed();
    }

    @SuppressWarnings("deprecation")
    private void buildUi() {
        tabletLayout = getResources().getConfiguration().screenWidthDp >= TABLET_SIDEBAR_MIN_DP;

        FrameLayout shell = new FrameLayout(this);
        shell.setBackgroundColor(GhostTheme.WINDOW);
        shell.setOnApplyWindowInsetsListener((view, insets) -> {
            view.setPadding(
                    insets.getSystemWindowInsetLeft(),
                    insets.getSystemWindowInsetTop(),
                    insets.getSystemWindowInsetRight(),
                    insets.getSystemWindowInsetBottom());
            return insets;
        });

        if (tabletLayout) {
            LinearLayout body = new LinearLayout(this);
            body.setOrientation(LinearLayout.HORIZONTAL);
            body.setBackgroundColor(GhostTheme.WINDOW);
            navigationPanel = buildNavigationPanel();
            body.addView(navigationPanel, new LinearLayout.LayoutParams(dp(236), ViewGroup.LayoutParams.MATCH_PARENT));
            body.addView(buildMainColumn(), new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.MATCH_PARENT, 1f));
            shell.addView(body, new FrameLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.MATCH_PARENT));
        } else {
            shell.addView(buildMainColumn(), new FrameLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.MATCH_PARENT));

            drawerScrim = new View(this);
            drawerScrim.setBackgroundColor(0x99000000);
            drawerScrim.setVisibility(View.GONE);
            drawerScrim.setOnClickListener(v -> closeNavigationDrawer());
            shell.addView(drawerScrim, new FrameLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.MATCH_PARENT));

            navigationPanel = buildNavigationPanel();
            navigationPanel.setVisibility(View.GONE);
            FrameLayout.LayoutParams drawerParams = new FrameLayout.LayoutParams(dp(286), ViewGroup.LayoutParams.MATCH_PARENT);
            drawerParams.gravity = Gravity.START;
            shell.addView(navigationPanel, drawerParams);
        }

        setContentView(shell);
        shell.requestApplyInsets();
    }

    private LinearLayout buildMainColumn() {
        LinearLayout main = new LinearLayout(this);
        main.setOrientation(LinearLayout.VERTICAL);
        main.setBackgroundColor(GhostTheme.WINDOW);

        LinearLayout appBar = new LinearLayout(this);
        appBar.setOrientation(LinearLayout.HORIZONTAL);
        appBar.setGravity(Gravity.CENTER_VERTICAL);
        appBar.setPadding(dp(12), dp(10), dp(12), dp(10));
        appBar.setBackgroundColor(GhostTheme.PANEL);

        menuToggle = new ImageButton(this);
        menuToggle.setImageResource(R.drawable.ic_menu);
        menuToggle.setImageTintList(ColorStateList.valueOf(GhostTheme.TEXT));
        menuToggle.setBackground(GhostTheme.rounded(this, GhostTheme.LIST, GhostTheme.BORDER, 10));
        menuToggle.setContentDescription("Open navigation");
        menuToggle.setPadding(dp(10), dp(10), dp(10), dp(10));
        menuToggle.setOnClickListener(v -> openNavigationDrawer());
        menuToggle.setVisibility(tabletLayout ? View.GONE : View.VISIBLE);
        appBar.addView(menuToggle, new LinearLayout.LayoutParams(dp(44), dp(44)));

        LinearLayout titleStack = new LinearLayout(this);
        titleStack.setOrientation(LinearLayout.VERTICAL);
        titleStack.setPadding(dp(12), 0, dp(8), 0);
        TextView brand = label("GHOST FTP", 16, GhostTheme.TEXT);
        brand.setTypeface(Typeface.DEFAULT, Typeface.BOLD);
        sectionTitle = label("Files", 12, GhostTheme.MUTED);
        sectionTitle.setVisibility(tabletLayout ? View.VISIBLE : View.GONE);
        titleStack.addView(brand, matchWrap());
        titleStack.addView(sectionTitle, matchWrap());
        appBar.addView(titleStack, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));

        connectionBadge = label("DISCONNECTED", 10, GhostTheme.MUTED);
        GhostTheme.styleBadge(connectionBadge, GhostTheme.MUTED);
        appBar.addView(connectionBadge, wrapWrap());
        main.addView(appBar, matchWrap());

        status = label("Ready.", 12, GhostTheme.MUTED);
        status.setPadding(dp(14), dp(9), dp(14), dp(9));
        status.setBackgroundColor(GhostTheme.WINDOW);
        main.addView(status, matchWrap());

        contentHost = new FrameLayout(this);
        contentHost.setBackgroundColor(GhostTheme.WINDOW);
        main.addView(contentHost, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, 0, 1f));

        filesSurface = buildFilesSurface();
        sitesSurface = buildSitesSurface();
        bookmarksSurface = buildBookmarksSurface();
        transfersSurface = buildTransfersSurface();
        settingsSurface = buildSettingsSurface();
        aboutSurface = buildAboutSurface();
        addSurface(filesSurface);
        addSurface(sitesSurface);
        addSurface(bookmarksSurface);
        addSurface(transfersSurface);
        addSurface(settingsSurface);
        addSurface(aboutSurface);
        return main;
    }

    private LinearLayout buildNavigationPanel() {
        LinearLayout navigation = new LinearLayout(this);
        navigation.setOrientation(LinearLayout.VERTICAL);
        navigation.setPadding(dp(12), dp(18), dp(12), dp(18));
        navigation.setBackgroundColor(GhostTheme.PANEL);
        navigation.setImportantForAccessibility(View.IMPORTANT_FOR_ACCESSIBILITY_YES);
        if (android.os.Build.VERSION.SDK_INT >= android.os.Build.VERSION_CODES.P) {
            navigation.setAccessibilityPaneTitle("Navigation");
        }

        TextView product = label("Ghost FTP", 20, GhostTheme.TEXT);
        product.setTypeface(Typeface.DEFAULT, Typeface.BOLD);
        navigation.addView(product, matchWrap());
        TextView platform = label("Android app", 11, GhostTheme.MUTED);
        platform.setPadding(0, dp(2), 0, dp(18));
        navigation.addView(platform, matchWrap());

        navigation.addView(navButton("Files", R.drawable.ic_files, Section.FILES), navParams());
        navigation.addView(navButton("Sites", R.drawable.ic_sites, Section.SITES), navParams());
        navigation.addView(navButton("Bookmarks", R.drawable.ic_bookmarks, Section.BOOKMARKS), navParams());
        navigation.addView(navButton("Transfers", R.drawable.ic_transfers, Section.TRANSFERS), navParams());
        navigation.addView(navButton("Settings", R.drawable.ic_settings, Section.SETTINGS), navParams());
        navigation.addView(navButton("About", R.drawable.ic_about, Section.ABOUT), navParams());

        TextView privacy = label("No telemetry · no ads · no Ghost FTP cloud", 10, GhostTheme.MUTED);
        privacy.setPadding(dp(4), dp(18), dp(4), 0);
        navigation.addView(privacy, matchWrap());
        return navigation;
    }

    private Button navButton(String text, int iconRes, Section section) {
        Button button = new Button(this);
        button.setText(text);
        button.setContentDescription("Navigate to " + text);
        button.setImportantForAccessibility(View.IMPORTANT_FOR_ACCESSIBILITY_YES);
        button.setAllCaps(false);
        button.setGravity(Gravity.START | Gravity.CENTER_VERTICAL);
        button.setCompoundDrawablesWithIntrinsicBounds(iconRes, 0, 0, 0);
        button.setCompoundDrawablePadding(dp(12));
        button.setCompoundDrawableTintList(ColorStateList.valueOf(GhostTheme.MUTED));
        button.setTag(section);
        button.setOnClickListener(v -> showSection((Section) v.getTag()));
        navigationButtons.add(button);
        styleNavigationButton(button, false);
        return button;
    }

    private View buildFilesSurface() {
        LinearLayout content = surfaceContent();
        content.addView(surfaceHeading("Files", "Browse local files and your connected server from one workspace."));

        LinearLayout panes = new LinearLayout(this);
        boolean wideFiles = getResources().getConfiguration().screenWidthDp >= 900;
        panes.setOrientation(wideFiles ? LinearLayout.HORIZONTAL : LinearLayout.VERTICAL);
        LinearLayout localPane = buildLocalFilesCard();
        LinearLayout remotePane = buildRemoteFilesCard();
        if (wideFiles) {
            panes.addView(localPane, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
            LinearLayout.LayoutParams remoteParams = new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f);
            remoteParams.setMargins(dp(10), 0, 0, 0);
            panes.addView(remotePane, remoteParams);
        } else {
            panes.addView(localPane, cardParams());
            panes.addView(remotePane, cardParams());
        }
        content.addView(panes, matchWrap());

        LinearLayout transferCard = card("TRANSFER", "Move the selected file safely and follow its progress while the transfer is active.");
        LinearLayout actions = row();
        upload = primaryButton("Upload →");
        download = primaryButton("← Download");
        actions.addView(upload, weightedSpaced());
        actions.addView(download, weightedSpaced());
        transferCard.addView(actions, matchWrap());
        upload.setOnClickListener(v -> uploadSelected());
        download.setOnClickListener(v -> downloadSelected());
        content.addView(transferCard, cardParams());
        return scrollSurface(content);
    }

    private LinearLayout buildLocalFilesCard() {
        LinearLayout card = card("LOCAL", "Browse and manage files only in folders you allow Ghost FTP to use. Long-press a folder to select it.");
        localPath = pathLabel("No folder selected");
        card.addView(localPath, matchWrapSpaced());
        LinearLayout navigationActions = row();
        Button choose = button("Choose folder");
        Button up = button("Up");
        Button refresh = button("Refresh");
        navigationActions.addView(choose, weightedSpaced());
        navigationActions.addView(up, weightedSpaced());
        navigationActions.addView(refresh, weightedSpaced());
        card.addView(navigationActions, matchWrap());
        choose.setOnClickListener(v -> chooseFolder());
        up.setOnClickListener(v -> localUp());
        refresh.setOnClickListener(v -> refreshLocal());

        LinearLayout viewActions = row();
        localFilter = button("Filter");
        localSort = button("Sort: Name ↑");
        localSearch = button("Search");
        viewActions.addView(localFilter, weightedSpaced());
        viewActions.addView(localSort, weightedSpaced());
        viewActions.addView(localSearch, weightedSpaced());
        card.addView(viewActions, matchWrap());
        localFilter.setOnClickListener(v -> editLocalFilter());
        localSort.setOnClickListener(v -> cycleLocalSort());
        localSort.setOnLongClickListener(v -> { toggleLocalSortDirection(); return true; });
        localSearch.setOnClickListener(v -> promptLocalRecursiveSearch());

        LinearLayout fileActions = row();
        localCreateDirectory = button("New folder");
        localRename = button("Rename");
        localDelete = dangerButton("Delete");
        fileActions.addView(localCreateDirectory, weightedSpaced());
        fileActions.addView(localRename, weightedSpaced());
        fileActions.addView(localDelete, weightedSpaced());
        card.addView(fileActions, matchWrap());
        localCreateDirectory.setOnClickListener(v -> createLocalDirectory());
        localRename.setOnClickListener(v -> renameLocalSelected());
        localDelete.setOnClickListener(v -> deleteLocalSelected());

        localEmptyState = workspaceEmptyState("Choose a folder to browse local files.");
        card.addView(localEmptyState, matchWrapSpaced());
        localList = new ListView(this);
        GhostTheme.styleList(localList);
        localList.setVisibility(View.GONE);
        card.addView(localList, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, dp(220)));
        localList.setOnItemClickListener((parent, view, position, id) -> selectLocal(position));
        localList.setOnItemLongClickListener((parent, view, position, id) -> {
            int sourceIndex = localSourceIndex(position);
            if (busy || sourceIndex < 0 || sourceIndex >= localEntries.size()) return true;
            selectedLocal = sourceIndex;
            renderLocal();
            setStatus("Local item selected for file management: " + localEntries.get(sourceIndex).name);
            return true;
        });
        return card;
    }

    private LinearLayout buildRemoteFilesCard() {
        LinearLayout card = card("SERVER", "Browse and manage files on the connected server. Long-press a folder to select it.");
        remotePath = pathLabel(currentRemotePath);
        card.addView(remotePath, matchWrapSpaced());
        LinearLayout navigationActions = row();
        Button up = button("Up");
        Button refresh = button("Refresh");
        navigationActions.addView(up, weightedSpaced());
        navigationActions.addView(refresh, weightedSpaced());
        card.addView(navigationActions, matchWrap());
        up.setOnClickListener(v -> remoteUp());
        refresh.setOnClickListener(v -> refreshRemote(currentRemotePath));

        LinearLayout viewActions = row();
        remoteFilter = button("Filter");
        remoteSort = button("Sort: Name ↑");
        remoteSearch = button("Search");
        viewActions.addView(remoteFilter, weightedSpaced());
        viewActions.addView(remoteSort, weightedSpaced());
        viewActions.addView(remoteSearch, weightedSpaced());
        card.addView(viewActions, matchWrap());
        remoteFilter.setOnClickListener(v -> editRemoteFilter());
        remoteSort.setOnClickListener(v -> cycleRemoteSort());
        remoteSort.setOnLongClickListener(v -> { toggleRemoteSortDirection(); return true; });
        remoteSearch.setOnClickListener(v -> promptRemoteRecursiveSearch());

        LinearLayout primaryActions = row();
        remoteCreateDirectory = button("New folder");
        remoteRename = button("Rename");
        remoteDelete = dangerButton("Delete");
        primaryActions.addView(remoteCreateDirectory, weightedSpaced());
        primaryActions.addView(remoteRename, weightedSpaced());
        primaryActions.addView(remoteDelete, weightedSpaced());
        card.addView(primaryActions, matchWrap());
        remoteCreateDirectory.setOnClickListener(v -> createRemoteDirectory());
        remoteRename.setOnClickListener(v -> renameRemoteSelected());
        remoteDelete.setOnClickListener(v -> deleteRemoteSelected());

        LinearLayout advancedActions = row();
        remoteChmod = button("Permissions");
        directoryCompare = button("Compare folders");
        remoteEdit = primaryButton("Remote Edit");
        advancedActions.addView(remoteChmod, weightedSpaced());
        advancedActions.addView(directoryCompare, weightedSpaced());
        advancedActions.addView(remoteEdit, weightedSpaced());
        card.addView(advancedActions, matchWrap());
        remoteChmod.setOnClickListener(v -> chmodRemoteSelected());
        directoryCompare.setOnClickListener(v -> showDirectoryComparison());
        remoteEdit.setOnClickListener(v -> openRemoteEditorSelected());

        remoteEmptyState = workspaceEmptyState("Connect from Sites to browse server files.");
        card.addView(remoteEmptyState, matchWrapSpaced());
        remoteList = new ListView(this);
        GhostTheme.styleList(remoteList);
        remoteList.setVisibility(View.GONE);
        card.addView(remoteList, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, dp(220)));
        remoteList.setOnItemClickListener((parent, view, position, id) -> selectRemote(position));
        remoteList.setOnItemLongClickListener((parent, view, position, id) -> {
            int sourceIndex = remoteSourceIndex(position);
            if (busy || sourceIndex < 0 || sourceIndex >= remoteEntries.size()) return true;
            selectedRemote = sourceIndex;
            renderRemote();
            setStatus("Server item selected for file management: " + remoteEntries.get(sourceIndex).name);
            return true;
        });
        return card;
    }

    private View buildSitesSurface() {
        LinearLayout content = surfaceContent();
        content.addView(surfaceHeading("Sites", "Connect quickly or save the server details you use often."));

        LinearLayout connectionCard = card("QUICK CONNECT", "Passwords are never saved. FTP and secure FTPS are available on Android.");
        protocol = new Spinner(this);
        GhostTheme.styleSpinner(protocol);
        protocol.setAdapter(GhostTheme.spinnerAdapter(this, java.util.Arrays.asList(new String[]{"FTPS", "FTP"})));
        connectionCard.addView(protocol, matchWrapSpaced());
        host = field("Server host", false);
        port = field("Port", false);
        port.setInputType(InputType.TYPE_CLASS_NUMBER);
        username = field("Username", false);
        password = field("Password (never saved)", true);
        connectionCard.addView(host, matchWrapSpaced());
        LinearLayout credentials = row();
        credentials.addView(port, fixedWidthSpaced(96));
        credentials.addView(username, weightedSpaced());
        connectionCard.addView(credentials, matchWrap());
        connectionCard.addView(password, matchWrapSpaced());
        LinearLayout connectionActions = row();
        connect = primaryButton("Connect");
        disconnect = dangerButton("Disconnect");
        connectionActions.addView(connect, weightedSpaced());
        connectionActions.addView(disconnect, weightedSpaced());
        connectionCard.addView(connectionActions, matchWrap());
        connect.setOnClickListener(v -> connect());
        disconnect.setOnClickListener(v -> disconnect());
        content.addView(connectionCard, cardParams());

        LinearLayout savedCard = card("SAVED SITES", "Save only the connection details you choose. Passwords are never stored.");
        siteSpinner = new Spinner(this);
        GhostTheme.styleSpinner(siteSpinner);
        savedCard.addView(siteSpinner, matchWrapSpaced());
        profileName = field("Site name", false);
        savedCard.addView(profileName, matchWrapSpaced());
        LinearLayout actions = row();
        Button load = button("Load");
        saveSite = primaryButton("Save / update");
        deleteSite = dangerButton("Delete");
        actions.addView(load, weightedSpaced());
        actions.addView(saveSite, weightedSpaced());
        actions.addView(deleteSite, weightedSpaced());
        savedCard.addView(actions, matchWrap());
        load.setOnClickListener(v -> loadSelectedSite());
        saveSite.setOnClickListener(v -> saveOrUpdateSite());
        deleteSite.setOnClickListener(v -> deleteActiveSite());
        content.addView(savedCard, cardParams());
        return scrollSurface(content);
    }

    private View buildBookmarksSurface() {
        LinearLayout content = surfaceContent();
        content.addView(surfaceHeading("Bookmarks", "Save frequently used local folders and remote paths for faster navigation."));

        LinearLayout localCard = card("LOCAL BOOKMARKS", "Saved local folders remain limited to locations you selected.");
        bookmarkLocalCurrent = pathLabel("No folder selected");
        localCard.addView(bookmarkLocalCurrent, matchWrapSpaced());
        LinearLayout localActions = row();
        localSetStart = button("Set start");
        localAddBookmark = primaryButton("Add");
        localRemoveBookmark = dangerButton("Remove");
        localActions.addView(localSetStart, weightedSpaced());
        localActions.addView(localAddBookmark, weightedSpaced());
        localActions.addView(localRemoveBookmark, weightedSpaced());
        localCard.addView(localActions, matchWrap());
        localSetStart.setOnClickListener(v -> setLocalStart());
        localAddBookmark.setOnClickListener(v -> addLocalBookmark());
        localRemoveBookmark.setOnClickListener(v -> removeLocalBookmark());
        localBookmarkSpinner = new Spinner(this);
        GhostTheme.styleSpinner(localBookmarkSpinner);
        localCard.addView(localBookmarkSpinner, matchWrapSpaced());
        localOpenBookmark = button("Open local bookmark");
        localCard.addView(localOpenBookmark, matchWrapSpaced());
        localOpenBookmark.setOnClickListener(v -> openLocalBookmark());
        content.addView(localCard, cardParams());

        LinearLayout remoteCard = card("SERVER BOOKMARKS", "Saved remote paths stay linked to the matching server and account.");
        bookmarkRemoteCurrent = pathLabel(currentRemotePath);
        remoteCard.addView(bookmarkRemoteCurrent, matchWrapSpaced());
        LinearLayout remoteActions = row();
        remoteSetStart = button("Set start");
        remoteAddBookmark = primaryButton("Add");
        remoteRemoveBookmark = dangerButton("Remove");
        remoteActions.addView(remoteSetStart, weightedSpaced());
        remoteActions.addView(remoteAddBookmark, weightedSpaced());
        remoteActions.addView(remoteRemoveBookmark, weightedSpaced());
        remoteCard.addView(remoteActions, matchWrap());
        remoteSetStart.setOnClickListener(v -> setRemoteStart());
        remoteAddBookmark.setOnClickListener(v -> addRemoteBookmark());
        remoteRemoveBookmark.setOnClickListener(v -> removeRemoteBookmark());
        remoteBookmarkSpinner = new Spinner(this);
        GhostTheme.styleSpinner(remoteBookmarkSpinner);
        remoteCard.addView(remoteBookmarkSpinner, matchWrapSpaced());
        remoteOpenBookmark = button("Open server bookmark");
        remoteCard.addView(remoteOpenBookmark, matchWrapSpaced());
        remoteOpenBookmark.setOnClickListener(v -> openRemoteBookmark());
        content.addView(remoteCard, cardParams());
        return scrollSurface(content);
    }

    private View buildTransfersSurface() {
        LinearLayout content = surfaceContent();
        content.addView(surfaceHeading("Transfers", "Follow the current transfer and cancel it while cancellation is still safe."));
        LinearLayout card = card("ACTIVE TRANSFER", "Progress reflects the current file transfer.");
        transferStatus = label("No active transfer.", 14, GhostTheme.MUTED);
        transferStatus.setPadding(dp(10), dp(12), dp(10), dp(12));
        transferStatus.setBackground(GhostTheme.rounded(this, GhostTheme.LIST, GhostTheme.BORDER, 10));
        card.addView(transferStatus, matchWrapSpaced());
        transferCancel = dangerButton("Cancel active transfer");
        transferCancel.setOnClickListener(v -> cancelTransfer());
        card.addView(transferCancel, matchWrapSpaced());
        Button openFiles = button("Open Files");
        openFiles.setOnClickListener(v -> showSection(Section.FILES));
        card.addView(openFiles, matchWrapSpaced());
        content.addView(card, cardParams());
        return scrollSurface(content);
    }

    private View buildSettingsSurface() {
        LinearLayout content = surfaceContent();
        content.addView(surfaceHeading("Settings", "Choose how Ghost FTP behaves on this device."));
        LinearLayout uiCard = card("APP PREFERENCES", "Adjust local preferences for browsing and quick connections.");

        TextView appearanceLabel = label("Appearance", 12, GhostTheme.MUTED);
        appearanceLabel.setPadding(dp(4), dp(4), dp(4), dp(5));
        uiCard.addView(appearanceLabel, matchWrap());
        LinearLayout appearanceRow = row();
        appearanceSpinner = new Spinner(this);
        List<String> appearanceOptions = new ArrayList<>();
        appearanceOptions.add("Dark");
        appearanceOptions.add("Light");
        appearanceSpinner.setAdapter(GhostTheme.spinnerAdapter(this, appearanceOptions));
        GhostTheme.styleSpinner(appearanceSpinner);
        appearanceSpinner.setSelection(GhostTheme.APPEARANCE_LIGHT.equals(appearanceMode) ? 1 : 0);
        appearanceRow.addView(appearanceSpinner, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        applyAppearance = primaryButton("Apply");
        LinearLayout.LayoutParams applyAppearanceParams = new LinearLayout.LayoutParams(dp(96), ViewGroup.LayoutParams.WRAP_CONTENT);
        applyAppearanceParams.setMargins(dp(8), 0, 0, 0);
        appearanceRow.addView(applyAppearance, applyAppearanceParams);
        uiCard.addView(appearanceRow, matchWrapSpaced());
        TextView appearanceHelp = label(
                "Dark is the Ghost FTP default. Light keeps a neutral gray secondary palette.",
                11,
                GhostTheme.MUTED);
        appearanceHelp.setPadding(dp(4), 0, dp(4), dp(8));
        uiCard.addView(appearanceHelp, matchWrap());
        applyAppearance.setOnClickListener(v -> applyAppearancePreference());

        rememberEndpointToggle = checkBox("Remember Quick Connect host, username, protocol and port");
        rememberEndpointToggle.setOnClickListener(v -> {
            rememberEndpoint = rememberEndpointToggle.isChecked();
            savePreferences();
            setStatus(rememberEndpoint
                    ? "Quick Connect details will be remembered. Passwords are never saved."
                    : "Saved Quick Connect details were cleared.");
        });
        uiCard.addView(rememberEndpointToggle, matchWrapSpaced());
        showFileSizesToggle = checkBox("Show file sizes in Files lists");
        showFileSizesToggle.setOnClickListener(v -> {
            showFileSizes = showFileSizesToggle.isChecked();
            savePreferences();
            renderLocal();
            renderRemote();
            setStatus(showFileSizes ? "File sizes are visible." : "File sizes are hidden from list rows.");
        });
        uiCard.addView(showFileSizesToggle, matchWrapSpaced());
        content.addView(uiCard, cardParams());

        LinearLayout securityCard = card("SECURITY", "Security protections stay enforced automatically.");
        securityCard.addView(infoLine("FTPS", "Secure certificate checks are enabled"), matchWrapSpaced());
        securityCard.addView(infoLine("Passwords", "Never saved"), matchWrapSpaced());
        securityCard.addView(infoLine("Local storage", "Access limited to folders you select"), matchWrapSpaced());
        securityCard.addView(infoLine("SFTP", "Not available in the Android app"), matchWrapSpaced());
        securityCard.addView(infoLine("Privacy", "No telemetry, analytics, ads or Ghost FTP cloud"), matchWrapSpaced());
        content.addView(securityCard, cardParams());
        return scrollSurface(content);
    }

    private View buildAboutSurface() {
        LinearLayout content = surfaceContent();
        content.addView(surfaceHeading("About", "Version, supported protocols and privacy information."));
        LinearLayout card = card("GHOST FTP", "Private file transfer client for direct connections to servers you control.");
        card.addView(infoLine("Version", BuildConfig.VERSION_NAME), matchWrapSpaced());
        card.addView(infoLine("Protocols", "FTP and explicit FTPS"), matchWrapSpaced());
        card.addView(infoLine("Data collection", "No telemetry, analytics or ads"), matchWrapSpaced());
        content.addView(card, cardParams());
        return scrollSurface(content);
    }

    private void addSurface(View surface) {
        surface.setVisibility(View.GONE);
        contentHost.addView(surface, new FrameLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.MATCH_PARENT));
    }

    private void showSection(Section section) {
        activeSection = section;
        filesSurface.setVisibility(section == Section.FILES ? View.VISIBLE : View.GONE);
        sitesSurface.setVisibility(section == Section.SITES ? View.VISIBLE : View.GONE);
        bookmarksSurface.setVisibility(section == Section.BOOKMARKS ? View.VISIBLE : View.GONE);
        transfersSurface.setVisibility(section == Section.TRANSFERS ? View.VISIBLE : View.GONE);
        settingsSurface.setVisibility(section == Section.SETTINGS ? View.VISIBLE : View.GONE);
        aboutSurface.setVisibility(section == Section.ABOUT ? View.VISIBLE : View.GONE);
        sectionTitle.setText(sectionTitle(section));
        refreshNavigationSelection();
        if (!tabletLayout) closeNavigationDrawer();
        refreshButtons();
    }

    private String sectionTitle(Section section) {
        switch (section) {
            case SITES:
                return "Sites";
            case BOOKMARKS:
                return "Bookmarks";
            case TRANSFERS:
                return "Transfers";
            case SETTINGS:
                return "Settings";
            case ABOUT:
                return "About";
            case FILES:
            default:
                return "Files";
        }
    }

    private void refreshNavigationSelection() {
        for (Button button : navigationButtons) {
            Object tag = button.getTag();
            styleNavigationButton(button, tag == activeSection);
        }
    }

    private void styleNavigationButton(Button button, boolean active) {
        button.setTextColor(active ? GhostTheme.TEXT : GhostTheme.MUTED);
        button.setCompoundDrawableTintList(ColorStateList.valueOf(active ? GhostTheme.ACCENT_STRONG : GhostTheme.MUTED));
        button.setBackground(GhostTheme.rounded(this, active ? GhostTheme.SELECTION : GhostTheme.PANEL,
                active ? GhostTheme.ACCENT : GhostTheme.PANEL, 10));
        button.setMinHeight(dp(48));
        button.setPadding(dp(12), 0, dp(12), 0);
    }

    private void openNavigationDrawer() {
        if (tabletLayout || navigationPanel == null) return;
        drawerScrim.setVisibility(View.VISIBLE);
        navigationPanel.setVisibility(View.VISIBLE);
        navigationPanel.post(() -> navigationPanel.sendAccessibilityEvent(AccessibilityEvent.TYPE_WINDOW_CONTENT_CHANGED));
    }

    private void closeNavigationDrawer() {
        if (tabletLayout || navigationPanel == null) return;
        navigationPanel.setVisibility(View.GONE);
        drawerScrim.setVisibility(View.GONE);
    }

    private void restorePreferences() {
        SharedPreferences prefs = getSharedPreferences(PREFS, MODE_PRIVATE);
        rememberEndpoint = prefs.getBoolean("rememberEndpoint", true);
        showFileSizes = prefs.getBoolean("showFileSizes", true);
        appearanceMode = GhostTheme.normalizeAppearance(
                prefs.getString("appearance", GhostTheme.APPEARANCE_DARK));
        rememberEndpointToggle.setChecked(rememberEndpoint);
        showFileSizesToggle.setChecked(showFileSizes);
        if (appearanceSpinner != null) {
            appearanceSpinner.setSelection(GhostTheme.APPEARANCE_LIGHT.equals(appearanceMode) ? 1 : 0);
        }
        if (rememberEndpoint) {
            host.setText(prefs.getString("host", ""));
            username.setText(prefs.getString("username", ""));
            String savedProtocol = prefs.getString("protocol", "FTPS");
            protocol.setSelection("FTP".equals(savedProtocol) ? 1 : 0);
            port.setText(prefs.getString("port", "21"));
        } else {
            protocol.setSelection(0);
            port.setText("21");
        }
        String savedTree = prefs.getString("treeUri", "");
        if (!savedTree.isEmpty()) {
            tryActivateLocalTree(Uri.parse(savedTree), "Saved local folder is no longer available.", true);
        }
    }

    private void savePreferences() {
        SharedPreferences.Editor editor = getSharedPreferences(PREFS, MODE_PRIVATE).edit()
                .putBoolean("rememberEndpoint", rememberEndpoint)
                .putBoolean("showFileSizes", showFileSizes)
                .putString("appearance", appearanceMode)
                .putString("treeUri", treeUri == null ? "" : treeUri.toString());
        if (rememberEndpoint) {
            editor.putString("host", host.getText().toString().trim())
                    .putString("username", username.getText().toString().trim())
                    .putString("protocol", protocol.getSelectedItem().toString())
                    .putString("port", port.getText().toString().trim());
        } else {
            editor.remove("host").remove("username").remove("protocol").remove("port");
        }
        editor.apply();
    }

    private void applyAppearancePreference() {
        if (appearanceSpinner == null || applyAppearance == null) return;
        if (busy || transferActive || transferFinalizing || remoteEditorState != null) {
            setStatus("Finish the active operation before changing appearance.");
            return;
        }

        String next = appearanceSpinner.getSelectedItemPosition() == 1
                ? GhostTheme.APPEARANCE_LIGHT
                : GhostTheme.APPEARANCE_DARK;
        next = GhostTheme.normalizeAppearance(next);
        if (next.equals(appearanceMode)) {
            setStatus(GhostTheme.APPEARANCE_LIGHT.equals(next)
                    ? "Light appearance is already active."
                    : "Dark appearance is already active.");
            return;
        }

        String profileNameValue = profileName == null ? "" : profileName.getText().toString();
        String hostValue = host == null ? "" : host.getText().toString();
        String portValue = port == null ? "21" : port.getText().toString();
        String usernameValue = username == null ? "" : username.getText().toString();
        String passwordValue = password == null ? "" : password.getText().toString();
        int protocolPosition = protocol == null ? 0 : protocol.getSelectedItemPosition();
        String statusValue = status == null ? "Ready." : status.getText().toString();
        Section previousSection = activeSection;

        appearanceMode = next;
        getSharedPreferences(PREFS, MODE_PRIVATE).edit()
                .putString("appearance", appearanceMode)
                .apply();
        GhostTheme.apply(this, appearanceMode);
        GhostTheme.applySystemBars(this);

        navigationButtons.clear();
        buildUi();

        rememberEndpointToggle.setChecked(rememberEndpoint);
        showFileSizesToggle.setChecked(showFileSizes);
        appearanceSpinner.setSelection(GhostTheme.APPEARANCE_LIGHT.equals(appearanceMode) ? 1 : 0);
        profileName.setText(profileNameValue);
        host.setText(hostValue);
        port.setText(portValue);
        username.setText(usernameValue);
        password.setText(passwordValue);
        protocol.setSelection(Math.max(0, Math.min(protocolPosition, protocol.getCount() - 1)));

        renderSites();
        renderBookmarks();
        renderLocal();
        renderRemote();
        refreshButtons();
        showSection(previousSection);
        setStatus(statusValue);
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
        if (bookmarkLocalCurrent != null) {
            bookmarkLocalCurrent.setText(treeUri == null ? "No folder selected" : displayLocalPath());
        }
        if (bookmarkRemoteCurrent != null) bookmarkRemoteCurrent.setText(currentRemotePath);
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
            setStatus("Quick Connect ready. Use Save / update if you want to keep these server details.");
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
            setStatus("Site loaded. Enter your password to connect.");
        }
    }

    private void saveOrUpdateSite() {
        if (busy || session != null) {
            setStatus("Disconnect before changing saved connection details.");
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
                setStatus("Site updated. Saved server paths and bookmarks were cleared because the connection details changed.");
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
        setStatus("Saved site deleted.");
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

    private boolean connectionAttemptCurrent(FtpSession candidate) {
        return !lifecycleDestroyed && connectingSession == candidate;
    }

    private void connect() {
        if (lifecycleDestroyed || busy || session != null || connectingSession != null) return;
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
            setStatus("Connection details changed. Save the site or switch to Quick Connect before connecting.");
            return;
        }
        String requestedStart = profile == null ? null : profile.remoteStartPath;
        final FtpSession next;
        try {
            next = new FtpSession(hostValue, portValue, secure);
        } catch (IllegalArgumentException e) {
            setStatus(e.getMessage());
            return;
        }
        connectingSession = next;
        setBusy(true, secure ? "Connecting securely…" : "Connecting with FTP…");
        io.execute(() -> {
            try {
                if (!connectionAttemptCurrent(next)) {
                    next.abort();
                    return;
                }
                next.connect(userValue, passwordValue);
                if (!connectionAttemptCurrent(next)) {
                    next.abort();
                    return;
                }
                String start = requestedStart == null ? next.pwd() : requestedStart;
                List<RemoteEntry> entries = next.list(start);
                if (!connectionAttemptCurrent(next)) {
                    next.abort();
                    return;
                }
                runOnUiThread(() -> {
                    if (!connectionAttemptCurrent(next)) {
                        next.abort();
                        return;
                    }
                    connectingSession = null;
                    session = next;
                    connectedIdentityKey = identity;
                    currentRemotePath = start;
                    remoteEntries.clear();
                    remoteEntries.addAll(entries);
                    selectedRemote = -1;
                    password.setText("");
                    savePreferences();
                    renderRemote();
                    setBusy(false, secure
                            ? "FTPS connected securely."
                            : "FTP connected. Warning: this connection is not encrypted.");
                });
            } catch (Exception e) {
                next.abort();
                runOnUiThread(() -> {
                    if (!connectionAttemptCurrent(next)) return;
                    connectingSession = null;
                    setBusy(false, (profile == null ? "Connection failed" : "Connection failed or the saved start folder is unavailable")
                            + ": " + safeMessage(e));
                });
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
        FtpSession current = session;
        TransferCommitGate gate = activeTransferGate;
        if (current == null || gate == null) {
            transferGeneration++;
            transferActive = false;
            transferFinalizing = false;
            activeTransferGate = null;
            if (current != null) current.cancelActiveTransfer();
            busy = false;
            setStatus("Transfer cancelled. Connection closed; reconnect before another transfer.");
            refreshButtons();
            return;
        }

        TransferCommitGate.CancelDisposition disposition = current.cancelActiveTransfer(gate);
        if (disposition == TransferCommitGate.CancelDisposition.TOO_LATE) {
            transferFinalizing = true;
            setStatus("Finalizing transfer. It can no longer be cancelled safely.");
            refreshButtons();
            return;
        }
        if (disposition == TransferCommitGate.CancelDisposition.ALREADY_CANCELLED) {
            return;
        }

        transferGeneration++;
        transferActive = false;
        transferFinalizing = false;
        activeTransferGate = null;
        session = null;
        connectedIdentityKey = null;
        remoteEntries.clear();
        selectedRemote = -1;
        currentRemotePath = "/";
        busy = false;
        renderRemote();
        setStatus(disposition == TransferCommitGate.CancelDisposition.CLEANUP_STAGING
                ? "Cancellation accepted. Cleaning temporary data before closing the connection."
                : "Transfer cancelled. Connection closed; reconnect before another transfer.");
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
        setBusy(true, "Refreshing server folder…");
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
                    setBusy(false, "Server folder refreshed.");
                });
            } catch (Exception e) {
                postError("Server folder is unavailable; current folder was not changed", e);
            }
        });
    }

    private void selectRemote(int position) {
        int sourceIndex = remoteSourceIndex(position);
        if (busy || sourceIndex < 0 || sourceIndex >= remoteEntries.size()) return;
        RemoteEntry entry = remoteEntries.get(sourceIndex);
        if (entry.directory) {
            try {
                refreshRemote(FtpSession.joinRemote(currentRemotePath, entry.name));
            } catch (IOException e) {
                setStatus(e.getMessage());
            }
        } else {
            selectedRemote = sourceIndex;
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

    private void createRemoteDirectory() {
        FtpSession current = session;
        if (busy || current == null || !current.isConnected()) return;
        promptText("Create server folder", "", "Folder name", false, name -> {
            final String child;
            try {
                child = validateItemName(name);
            } catch (IOException e) {
                setStatus(e.getMessage());
                return;
            }
            runRemoteMutation(current, "Creating server folder…", "Server folder created: " + child,
                    () -> current.createDirectory(FtpSession.joinRemote(currentRemotePath, child)));
        });
    }

    private void renameRemoteSelected() {
        FtpSession current = session;
        if (busy || current == null || selectedRemote < 0 || selectedRemote >= remoteEntries.size()) return;
        RemoteEntry entry = remoteEntries.get(selectedRemote);
        promptText("Rename server item", entry.name, "New name", false, name -> {
            final String child;
            try {
                child = validateItemName(name);
            } catch (IOException e) {
                setStatus(e.getMessage());
                return;
            }
            if (entry.name.equals(child)) {
                setStatus("Server item name is unchanged.");
                return;
            }
            try {
                String from = FtpSession.joinRemote(currentRemotePath, entry.name);
                String to = FtpSession.joinRemote(currentRemotePath, child);
                runRemoteMutation(current, "Renaming server item…", "Server item renamed to: " + child,
                        () -> current.rename(from, to));
            } catch (IOException e) {
                setStatus(e.getMessage());
            }
        });
    }

    private void deleteRemoteSelected() {
        FtpSession current = session;
        if (busy || current == null || selectedRemote < 0 || selectedRemote >= remoteEntries.size()) return;
        RemoteEntry entry = remoteEntries.get(selectedRemote);
        confirmDestructive("Delete server item?", "Delete “" + entry.name + "” from the server? This cannot be undone.", () -> {
            try {
                String path = FtpSession.joinRemote(currentRemotePath, entry.name);
                runRemoteMutation(current, "Deleting server item…", "Server item deleted: " + entry.name,
                        () -> current.delete(path, entry.directory));
            } catch (IOException e) {
                setStatus(e.getMessage());
            }
        });
    }

    private void chmodRemoteSelected() {
        FtpSession current = session;
        if (busy || current == null || selectedRemote < 0 || selectedRemote >= remoteEntries.size()) return;
        RemoteEntry entry = remoteEntries.get(selectedRemote);
        promptText("Server permissions", "755", "Octal mode (for example 755)", true, mode -> {
            try {
                String path = FtpSession.joinRemote(currentRemotePath, entry.name);
                runRemoteMutation(current, "Changing server permissions…", "Server permissions updated for: " + entry.name,
                        () -> current.chmod(path, mode));
            } catch (IOException e) {
                setStatus(e.getMessage());
            }
        });
    }

    private interface RemoteMutation {
        void run() throws IOException;
    }

    private void runRemoteMutation(FtpSession current, String progress, String success, RemoteMutation mutation) {
        if (busy || current == null || session != current || !current.isConnected()) return;
        final String directory = currentRemotePath;
        setBusy(true, progress);
        io.execute(() -> {
            try {
                mutation.run();
                List<RemoteEntry> fresh = current.list(directory);
                runOnUiThread(() -> {
                    if (session != current || !current.isConnected()) return;
                    remoteEntries.clear();
                    remoteEntries.addAll(fresh);
                    selectedRemote = -1;
                    renderRemote();
                    setBusy(false, success);
                });
            } catch (Exception e) {
                postError("Server file operation failed", e);
            }
        });
    }

    private void setRemoteStart() {
        SiteProfile profile = requireConnectedActiveProfile();
        if (profile == null) return;
        SiteProfile next = profile.withRemoteStartPath(currentRemotePath);
        replaceProfile(next);
        profileStore.save(profiles);
        setStatus("Server start folder saved: " + currentRemotePath);
    }

    private void addRemoteBookmark() {
        SiteProfile profile = requireConnectedActiveProfile();
        if (profile == null) return;
        SiteProfile next = profile.withRemoteBookmark(currentRemotePath);
        replaceProfile(next);
        profileStore.save(profiles);
        renderBookmarks();
        setStatus("Server bookmark added.");
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
        List<String> nextBookmarks = new ArrayList<>(profile.remoteBookmarks);
        nextBookmarks.remove(index);
        SiteProfile next = new SiteProfile(profile.id, profile.name, profile.protocol, profile.host, profile.port,
                profile.username, profile.localStartTreeUri, profile.remoteStartPath, profile.localBookmarks, nextBookmarks);
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
            setStatus("Load or save a site first.");
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
                    ? "Local folder selected."
                    : "Local folder opened for this session only.");
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

    private String displayLocalPath() {
        if (treeUri == null || currentDocumentId == null) return "No folder selected";
        try {
            Uri current = DocumentsContract.buildDocumentUriUsingTree(treeUri, currentDocumentId);
            String[] projection = {DocumentsContract.Document.COLUMN_DISPLAY_NAME};
            try (Cursor cursor = getContentResolver().query(current, projection, null, null, null)) {
                if (cursor != null && cursor.moveToFirst()) {
                    String name = cursor.getString(0);
                    if (name != null && !name.trim().isEmpty()) return name;
                }
            }
        } catch (RuntimeException ignored) {
            // Use a neutral label when the selected folder name cannot be read.
        }
        return "Selected folder";
    }

    private void refreshLocal() {
        if (treeUri == null || currentDocumentId == null) {
            renderLocal();
            return;
        }
        try {
            List<LocalEntry> next = queryChildren(treeUri, currentDocumentId);
            localEntries.clear();
            localEntries.addAll(next);
            selectedLocal = -1;
            renderLocal();
        } catch (IOException e) {
            clearLocalRoot();
            renderLocal();
            setStatus("Local folder is no longer available. Choose the folder again.");
        }
    }

    private List<LocalEntry> queryChildren(Uri rootTreeUri, String documentId) throws IOException {
        List<LocalEntry> result = new ArrayList<>();
        Uri children = DocumentsContract.buildChildDocumentsUriUsingTree(rootTreeUri, documentId);
        String[] projection = {DocumentsContract.Document.COLUMN_DOCUMENT_ID, DocumentsContract.Document.COLUMN_DISPLAY_NAME,
                DocumentsContract.Document.COLUMN_MIME_TYPE, DocumentsContract.Document.COLUMN_SIZE,
                DocumentsContract.Document.COLUMN_LAST_MODIFIED};
        try (Cursor cursor = getContentResolver().query(children, projection, null, null, null)) {
            if (cursor == null) throw new IOException("This folder could not be opened.");
            while (cursor.moveToNext()) {
                String id = cursor.getString(0);
                String name = cursor.getString(1);
                String mime = cursor.getString(2);
                long size = cursor.isNull(3) ? 0L : cursor.getLong(3);
                long modified = cursor.isNull(4) ? 0L : cursor.getLong(4);
                result.add(new LocalEntry(id, name, DocumentsContract.Document.MIME_TYPE_DIR.equals(mime), size, modified));
            }
        } catch (SecurityException e) {
            throw new IOException("Local folder access is no longer available.", e);
        }
        return result;
    }

    private void selectLocal(int position) {
        int sourceIndex = localSourceIndex(position);
        if (busy || sourceIndex < 0 || sourceIndex >= localEntries.size()) return;
        LocalEntry entry = localEntries.get(sourceIndex);
        if (entry.directory) {
            try {
                List<LocalEntry> next = queryChildren(treeUri, entry.documentId);
                localParents.push(currentDocumentId);
                currentDocumentId = entry.documentId;
                localEntries.clear();
                localEntries.addAll(next);
                selectedLocal = -1;
                renderLocal();
            } catch (IOException e) {
                setStatus("Local folder is unavailable; current folder was not changed: " + safeMessage(e));
            }
        } else {
            selectedLocal = sourceIndex;
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
            renderLocal();
        } catch (IOException e) {
            setStatus("Parent folder is unavailable; current folder was not changed: " + safeMessage(e));
        }
    }


    private void editLocalFilter() {
        promptText("Local filter", localFilterQuery, "Name contains… (blank clears)", false, value -> {
            localFilterQuery = value == null ? "" : value.trim();
            selectedLocal = -1;
            renderLocal();
            setStatus(localFilterQuery.isEmpty() ? "Local filter cleared." : "Local filter applied: " + localFilterQuery);
        });
    }

    private void editRemoteFilter() {
        promptText("Server filter", remoteFilterQuery, "Name contains… (blank clears)", false, value -> {
            remoteFilterQuery = value == null ? "" : value.trim();
            selectedRemote = -1;
            renderRemote();
            setStatus(remoteFilterQuery.isEmpty() ? "Server filter cleared." : "Server filter applied: " + remoteFilterQuery);
        });
    }

    private void cycleLocalSort() {
        if (busy) return;
        localSortKey = WorkspaceOps.nextLocalSortKey(localSortKey);
        renderLocal();
        setStatus("Local sort: " + WorkspaceOps.sortLabel(localSortKey, localSortAscending));
    }

    private void toggleLocalSortDirection() {
        if (busy) return;
        localSortAscending = !localSortAscending;
        renderLocal();
        setStatus("Local sort: " + WorkspaceOps.sortLabel(localSortKey, localSortAscending));
    }

    private void cycleRemoteSort() {
        if (busy) return;
        remoteSortKey = WorkspaceOps.nextRemoteSortKey(remoteSortKey);
        renderRemote();
        setStatus("Server sort: " + WorkspaceOps.sortLabel(remoteSortKey, remoteSortAscending));
    }

    private void toggleRemoteSortDirection() {
        if (busy) return;
        remoteSortAscending = !remoteSortAscending;
        renderRemote();
        setStatus("Server sort: " + WorkspaceOps.sortLabel(remoteSortKey, remoteSortAscending));
    }

    private int localSourceIndex(int visiblePosition) {
        if (visiblePosition < 0 || visiblePosition >= localVisibleItems.size()) return -1;
        return localVisibleItems.get(visiblePosition).sourceIndex;
    }

    private int remoteSourceIndex(int visiblePosition) {
        if (visiblePosition < 0 || visiblePosition >= remoteVisibleItems.size()) return -1;
        return remoteVisibleItems.get(visiblePosition).sourceIndex;
    }

    private List<WorkspaceOps.Item> localWorkspaceItems() {
        List<WorkspaceOps.Item> result = new ArrayList<>();
        for (int i = 0; i < localEntries.size(); i++) {
            LocalEntry entry = localEntries.get(i);
            result.add(new WorkspaceOps.Item(i, entry.name, entry.directory, entry.size, entry.modifiedEpochMillis, ""));
        }
        return result;
    }

    private List<WorkspaceOps.Item> remoteWorkspaceItems() {
        List<WorkspaceOps.Item> result = new ArrayList<>();
        for (int i = 0; i < remoteEntries.size(); i++) {
            RemoteEntry entry = remoteEntries.get(i);
            result.add(new WorkspaceOps.Item(i, entry.name, entry.directory, entry.size, entry.modifiedEpochMillis, entry.permissions));
        }
        return result;
    }

    private void promptLocalRecursiveSearch() {
        if (busy || treeUri == null || rootDocumentId == null) return;
        promptText("Search local folders", "", "Name contains…", false, value -> {
            String query = value == null ? "" : value.trim();
            if (query.isEmpty()) {
                setStatus("Search text is required.");
                return;
            }
            runLocalRecursiveSearch(query);
        });
    }

    private void runLocalRecursiveSearch(String query) {
        Uri searchTree = treeUri;
        String searchRoot = rootDocumentId;
        if (busy || searchTree == null || searchRoot == null) return;
        long generation = ++advancedOperationGeneration;
        setBusy(true, "Searching local folders…");
        io.execute(() -> {
            try {
                List<LocalSearchResult> results = new ArrayList<>();
                ArrayDeque<LocalSearchNode> queue = new ArrayDeque<>();
                queue.add(new LocalSearchNode(searchRoot, "", new ArrayList<>(), 0));
                int directories = 0;
                while (!queue.isEmpty()
                        && directories < WorkspaceOps.MAX_SEARCH_DIRECTORIES
                        && results.size() < WorkspaceOps.MAX_SEARCH_RESULTS) {
                    LocalSearchNode node = queue.removeFirst();
                    directories++;
                    List<LocalEntry> children = queryChildren(searchTree, node.documentId);
                    for (LocalEntry entry : children) {
                        String display = node.displayPath.isEmpty() ? entry.name : node.displayPath + "/" + entry.name;
                        if (WorkspaceOps.matchesSearch(entry.name, query)) {
                            results.add(new LocalSearchResult(node.documentId, node.ancestors, entry, display));
                            if (results.size() >= WorkspaceOps.MAX_SEARCH_RESULTS) break;
                        }
                        if (entry.directory && node.depth < WorkspaceOps.MAX_SEARCH_DEPTH) {
                            List<String> ancestors = new ArrayList<>(node.ancestors);
                            ancestors.add(node.documentId);
                            queue.addLast(new LocalSearchNode(entry.documentId, display, ancestors, node.depth + 1));
                        }
                    }
                }
                runOnUiThread(() -> {
                    if (lifecycleDestroyed || generation != advancedOperationGeneration || treeUri == null || !treeUri.equals(searchTree)) return;
                    busy = false;
                    refreshButtons();
                    showLocalSearchResults(query, results);
                });
            } catch (Exception e) {
                runOnUiThread(() -> {
                    if (generation != advancedOperationGeneration || lifecycleDestroyed) return;
                    setBusy(false, "Local search failed: " + safeMessage(e));
                });
            }
        });
    }

    private void showLocalSearchResults(String query, List<LocalSearchResult> results) {
        if (results.isEmpty()) {
            setStatus("No local search results for: " + query);
            return;
        }
        String[] labels = new String[results.size()];
        for (int i = 0; i < results.size(); i++) {
            LocalSearchResult result = results.get(i);
            labels[i] = (result.entry.directory ? "Folder · " : "File · ") + result.displayPath;
        }
        new AlertDialog.Builder(this)
                .setTitle("Local search · " + results.size() + " result(s)")
                .setItems(labels, (dialog, which) -> navigateLocalSearchResult(results.get(which)))
                .setNegativeButton("Close", null)
                .show();
    }

    private void navigateLocalSearchResult(LocalSearchResult result) {
        Uri targetTree = treeUri;
        if (busy || targetTree == null) return;
        long generation = ++advancedOperationGeneration;
        setBusy(true, "Opening local search result…");
        io.execute(() -> {
            try {
                List<LocalEntry> fresh = queryChildren(targetTree, result.parentDocumentId);
                runOnUiThread(() -> {
                    if (lifecycleDestroyed || generation != advancedOperationGeneration || treeUri == null || !treeUri.equals(targetTree)) return;
                    currentDocumentId = result.parentDocumentId;
                    localParents.clear();
                    for (String ancestor : result.ancestors) localParents.push(ancestor);
                    localEntries.clear();
                    localEntries.addAll(fresh);
                    selectedLocal = findLocalByDocumentId(result.entry.documentId);
                    localFilterQuery = "";
                    renderLocal();
                    setBusy(false, "Opened local search result: " + result.displayPath);
                });
            } catch (Exception e) {
                runOnUiThread(() -> {
                    if (generation != advancedOperationGeneration || lifecycleDestroyed) return;
                    setBusy(false, "Local search result is no longer available: " + safeMessage(e));
                });
            }
        });
    }

    private int findLocalByDocumentId(String documentId) {
        for (int i = 0; i < localEntries.size(); i++) {
            if (localEntries.get(i).documentId.equals(documentId)) return i;
        }
        return -1;
    }

    private void promptRemoteRecursiveSearch() {
        FtpSession current = session;
        if (busy || current == null || !current.isConnected()) return;
        promptText("Search server folders", "", "Name contains…", false, value -> {
            String query = value == null ? "" : value.trim();
            if (query.isEmpty()) {
                setStatus("Search text is required.");
                return;
            }
            runRemoteRecursiveSearch(current, query);
        });
    }

    private void runRemoteRecursiveSearch(FtpSession owner, String query) {
        if (busy || owner == null || session != owner || !owner.isConnected()) return;
        String searchRoot = currentRemotePath;
        long generation = ++advancedOperationGeneration;
        long deadlineNanos = System.nanoTime() + WorkspaceOps.MAX_REMOTE_SEARCH_MILLIS * 1_000_000L;
        setBusy(true, "Searching server folders…");
        AlertDialog searchDialog = new AlertDialog.Builder(this)
                .setTitle("Search server folders")
                .setMessage("Searching server folders. Cancelling the search will close the current connection.")
                .setNegativeButton("Cancel search", null)
                .setCancelable(false)
                .create();
        searchDialog.setOnShowListener(ignored -> searchDialog.getButton(AlertDialog.BUTTON_NEGATIVE).setOnClickListener(v -> {
            if (generation != advancedOperationGeneration || session != owner) return;
            advancedOperationGeneration++;
            owner.abort();
            session = null;
            connectedIdentityKey = null;
            remoteEntries.clear();
            selectedRemote = -1;
            currentRemotePath = "/";
            busy = false;
            renderRemote();
            setStatus("Search cancelled. Connection closed; reconnect before continuing.");
            refreshButtons();
            searchDialog.dismiss();
        }));
        searchDialog.show();
        io.execute(() -> {
            try {
                List<RemoteSearchResult> results = new ArrayList<>();
                ArrayDeque<RemoteSearchNode> queue = new ArrayDeque<>();
                Set<String> visited = new HashSet<>();
                queue.add(new RemoteSearchNode(searchRoot, 0));
                int directories = 0;
                while (!queue.isEmpty()
                        && directories < WorkspaceOps.MAX_SEARCH_DIRECTORIES
                        && results.size() < WorkspaceOps.MAX_SEARCH_RESULTS) {
                    if (generation != advancedOperationGeneration) throw new IOException("Search was cancelled.");
                    if (System.nanoTime() > deadlineNanos) throw new IOException("Search took too long and was stopped.");
                    if (session != owner || !owner.isConnected()) throw new IOException("The server connection changed during search.");
                    RemoteSearchNode node = queue.removeFirst();
                    if (!visited.add(node.path)) continue;
                    directories++;
                    List<RemoteEntry> children = owner.list(node.path);
                    for (RemoteEntry entry : children) {
                        String childPath = FtpSession.joinRemote(node.path, entry.name);
                        if (WorkspaceOps.matchesSearch(entry.name, query)) {
                            results.add(new RemoteSearchResult(node.path, entry, childPath));
                            if (results.size() >= WorkspaceOps.MAX_SEARCH_RESULTS) break;
                        }
                        if (entry.directory && node.depth < WorkspaceOps.MAX_SEARCH_DEPTH) {
                            queue.addLast(new RemoteSearchNode(childPath, node.depth + 1));
                        }
                    }
                }
                runOnUiThread(() -> {
                    if (lifecycleDestroyed || generation != advancedOperationGeneration || session != owner || !owner.isConnected()) return;
                    searchDialog.dismiss();
                    busy = false;
                    refreshButtons();
                    showRemoteSearchResults(owner, query, results);
                });
            } catch (Exception e) {
                runOnUiThread(() -> {
                    if (generation != advancedOperationGeneration || lifecycleDestroyed) return;
                    searchDialog.dismiss();
                    if (session == owner && !owner.isConnected()) {
                        session = null;
                        connectedIdentityKey = null;
                        remoteEntries.clear();
                        selectedRemote = -1;
                        currentRemotePath = "/";
                        renderRemote();
                    }
                    setBusy(false, "Server search failed: " + safeMessage(e));
                });
            }
        });
    }

    private void showRemoteSearchResults(FtpSession owner, String query, List<RemoteSearchResult> results) {
        if (results.isEmpty()) {
            setStatus("No server search results for: " + query);
            return;
        }
        String[] labels = new String[results.size()];
        for (int i = 0; i < results.size(); i++) {
            RemoteSearchResult result = results.get(i);
            labels[i] = (result.entry.directory ? "Folder · " : "File · ") + result.displayPath;
        }
        new AlertDialog.Builder(this)
                .setTitle("Server search · " + results.size() + " result(s)")
                .setItems(labels, (dialog, which) -> navigateRemoteSearchResult(owner, results.get(which)))
                .setNegativeButton("Close", null)
                .show();
    }

    private void navigateRemoteSearchResult(FtpSession owner, RemoteSearchResult result) {
        if (busy || session != owner || !owner.isConnected()) return;
        long generation = ++advancedOperationGeneration;
        setBusy(true, "Opening server search result…");
        io.execute(() -> {
            try {
                List<RemoteEntry> fresh = owner.list(result.parentPath);
                runOnUiThread(() -> {
                    if (lifecycleDestroyed || generation != advancedOperationGeneration || session != owner || !owner.isConnected()) return;
                    currentRemotePath = result.parentPath;
                    remoteEntries.clear();
                    remoteEntries.addAll(fresh);
                    selectedRemote = findRemoteByName(result.entry.name);
                    remoteFilterQuery = "";
                    renderRemote();
                    setBusy(false, "Opened server search result: " + result.displayPath);
                });
            } catch (Exception e) {
                runOnUiThread(() -> {
                    if (generation != advancedOperationGeneration || lifecycleDestroyed) return;
                    setBusy(false, "Server search result is no longer available: " + safeMessage(e));
                });
            }
        });
    }

    private int findRemoteByName(String name) {
        for (int i = 0; i < remoteEntries.size(); i++) {
            if (remoteEntries.get(i).name.equals(name)) return i;
        }
        return -1;
    }

    private void showDirectoryComparison() {
        if (busy || treeUri == null || currentDocumentId == null || session == null || !session.isConnected()) return;
        List<WorkspaceOps.Comparison> rows = WorkspaceOps.compareDirectories(localWorkspaceItems(), remoteWorkspaceItems());
        if (rows.isEmpty()) {
            setStatus("Both current folders are empty.");
            return;
        }
        String[] labels = new String[rows.size()];
        for (int i = 0; i < rows.size(); i++) {
            WorkspaceOps.Comparison row = rows.get(i);
            String marker;
            switch (row.difference) {
                case ONLY_LOCAL:
                    marker = "Only on this device";
                    break;
                case ONLY_REMOTE:
                    marker = "Only on server";
                    break;
                case DIFFERENT:
                    marker = "Different";
                    break;
                case SAME:
                default:
                    marker = "Same";
                    break;
            }
            labels[i] = marker + "   " + row.name + (row.canSynchronizeDirectoryNavigation() ? "   › open both" : "");
        }
        new AlertDialog.Builder(this)
                .setTitle("Compare folders")
                .setItems(labels, (dialog, which) -> {
                    WorkspaceOps.Comparison row = rows.get(which);
                    if (row.canSynchronizeDirectoryNavigation()) {
                        synchronizedNavigateInto(row.name);
                    } else {
                        setStatus("Comparison selected: " + row.name);
                    }
                })
                .setNegativeButton("Close", null)
                .show();
    }

    private void synchronizedNavigateInto(String name) {
        FtpSession owner = session;
        Uri ownerTree = treeUri;
        if (busy || owner == null || ownerTree == null || !owner.isConnected()) return;
        LocalEntry localDirectory = null;
        RemoteEntry remoteDirectory = null;
        for (LocalEntry entry : localEntries) if (entry.directory && entry.name.equals(name)) localDirectory = entry;
        for (RemoteEntry entry : remoteEntries) if (entry.directory && entry.name.equals(name)) remoteDirectory = entry;
        if (localDirectory == null || remoteDirectory == null) {
            setStatus("The matching folders changed. Refresh and try again.");
            return;
        }
        final LocalEntry localTarget = localDirectory;
        final String localParent = currentDocumentId;
        final String remoteParent = currentRemotePath;
        final String remoteTarget;
        try {
            remoteTarget = FtpSession.joinRemote(remoteParent, name);
        } catch (IOException e) {
            setStatus(e.getMessage());
            return;
        }
        long generation = ++advancedOperationGeneration;
        setBusy(true, "Opening matching folders…");
        io.execute(() -> {
            try {
                List<LocalEntry> localFresh = queryChildren(ownerTree, localTarget.documentId);
                List<RemoteEntry> remoteFresh = owner.list(remoteTarget);
                runOnUiThread(() -> {
                    if (lifecycleDestroyed || generation != advancedOperationGeneration || session != owner || treeUri == null || !treeUri.equals(ownerTree)) return;
                    if (!localParent.equals(currentDocumentId) || !remoteParent.equals(currentRemotePath)) return;
                    localParents.push(localParent);
                    currentDocumentId = localTarget.documentId;
                    currentRemotePath = remoteTarget;
                    localEntries.clear();
                    localEntries.addAll(localFresh);
                    remoteEntries.clear();
                    remoteEntries.addAll(remoteFresh);
                    selectedLocal = -1;
                    selectedRemote = -1;
                    localFilterQuery = "";
                    remoteFilterQuery = "";
                    renderLocal();
                    renderRemote();
                    setBusy(false, "Opened matching folders: " + name);
                });
            } catch (Exception e) {
                runOnUiThread(() -> {
                    if (generation != advancedOperationGeneration || lifecycleDestroyed) return;
                    setBusy(false, "Could not open the matching folders: " + safeMessage(e));
                });
            }
        });
    }

    private void openRemoteEditorSelected() {
        FtpSession owner = session;
        if (busy || owner == null || selectedRemote < 0 || selectedRemote >= remoteEntries.size() || !owner.isConnected()) return;
        RemoteEntry entry = remoteEntries.get(selectedRemote);
        if (!entry.regularFile) {
            setStatus("Select a regular text file to edit. Links and special entries cannot be edited.");
            return;
        }
        if (entry.size > WorkspaceOps.MAX_REMOTE_EDIT_BYTES) {
            setStatus("Remote Edit supports text files up to 1 MiB.");
            return;
        }
        final String path;
        try {
            path = FtpSession.joinRemote(currentRemotePath, entry.name);
        } catch (IOException e) {
            setStatus(e.getMessage());
            return;
        }
        long generation = ++advancedOperationGeneration;
        setBusy(true, "Opening file for editing…");
        io.execute(() -> {
            try {
                RemoteTextDocument.Snapshot snapshot = RemoteEditIo.open(owner, path);
                runOnUiThread(() -> {
                    if (lifecycleDestroyed || generation != advancedOperationGeneration || session != owner || !owner.isConnected()) return;
                    showRemoteEditor(owner, path, entry.name, entry.permissions, generation, snapshot);
                });
            } catch (Exception e) {
                runOnUiThread(() -> {
                    if (generation != advancedOperationGeneration || lifecycleDestroyed) return;
                    setBusy(false, "Remote Edit could not open file: " + safeMessage(e));
                });
            }
        });
    }

    private void showRemoteEditor(FtpSession owner, String path, String name, String originalMode, long generation, RemoteTextDocument.Snapshot snapshot) {
        EditText editor = field("File contents", false);
        editor.setSingleLine(false);
        editor.setGravity(Gravity.TOP | Gravity.START);
        editor.setMinLines(16);
        editor.setText(snapshot.text);

        FrameLayout holder = new FrameLayout(this);
        int pad = dp(16);
        holder.setPadding(pad, dp(8), pad, 0);
        holder.addView(editor, new FrameLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, dp(420)));

        RemoteEditorState state = new RemoteEditorState(owner, path, name, originalMode, generation, snapshot, editor);
        AlertDialog dialog = new AlertDialog.Builder(this)
                .setTitle("Remote Edit — " + name)
                .setView(holder)
                .setPositiveButton("Save", null)
                .setNeutralButton("Reload", null)
                .setNegativeButton("Close", null)
                .create();
        state.dialog = dialog;
        remoteEditorState = state;

        editor.addTextChangedListener(new TextWatcher() {
            @Override public void beforeTextChanged(CharSequence s, int start, int count, int after) { }
            @Override public void onTextChanged(CharSequence s, int start, int before, int count) { }
            @Override public void afterTextChanged(Editable s) {
                if (state.applying || remoteEditorState != state) return;
                state.dirty = true;
                state.dialog.setTitle("Remote Edit • modified — " + state.name);
            }
        });

        dialog.setOnShowListener(ignored -> {
            dialog.getButton(AlertDialog.BUTTON_POSITIVE).setOnClickListener(v -> saveRemoteEditor(state));
            dialog.getButton(AlertDialog.BUTTON_NEUTRAL).setOnClickListener(v -> reloadRemoteEditor(state));
            dialog.getButton(AlertDialog.BUTTON_NEGATIVE).setOnClickListener(v -> closeRemoteEditor(state));
            setStatus("Remote file opened for editing.");
            refreshButtons();
        });
        dialog.setOnDismissListener(ignored -> {
            if (remoteEditorState != state) return;
            remoteEditorState = null;
            advancedOperationGeneration++;
            busy = false;
            setStatus("Remote Edit closed.");
            refreshButtons();
        });
        dialog.show();
    }

    private void saveRemoteEditor(RemoteEditorState state) {
        if (!remoteEditorUsable(state) || state.running) return;
        state.running = true;
        setRemoteEditorButtonsEnabled(state, false);
        String text = state.editor.getText().toString();
        setStatus("Checking for changes and saving…");
        io.execute(() -> {
            try {
                RemoteTextDocument.Snapshot saved = RemoteEditIo.save(
                        state.owner, state.path, state.snapshot.sha256, state.snapshot.lineEnding, state.originalMode, text);
                runOnUiThread(() -> {
                    if (!remoteEditorUsable(state)) return;
                    applyRemoteEditorSnapshot(state, saved);
                    state.running = false;
                    setRemoteEditorButtonsEnabled(state, true);
                    setStatus("Remote file saved: " + state.name);
                    refreshRemoteAfterEditor(state.owner);
                });
            } catch (Exception e) {
                runOnUiThread(() -> {
                    if (remoteEditorState != state || lifecycleDestroyed) return;
                    state.running = false;
                    setRemoteEditorButtonsEnabled(state, true);
                    setStatus("Remote Edit save blocked: " + safeMessage(e));
                });
            }
        });
    }

    private void reloadRemoteEditor(RemoteEditorState state) {
        if (!remoteEditorUsable(state) || state.running) return;
        state.running = true;
        setRemoteEditorButtonsEnabled(state, false);
        setStatus("Reloading Remote Edit from server…");
        io.execute(() -> {
            try {
                RemoteTextDocument.Snapshot fresh = RemoteEditIo.reload(state.owner, state.path);
                runOnUiThread(() -> {
                    if (!remoteEditorUsable(state)) return;
                    applyRemoteEditorSnapshot(state, fresh);
                    state.running = false;
                    setRemoteEditorButtonsEnabled(state, true);
                    setStatus("Remote Edit reloaded from server: " + state.name);
                });
            } catch (Exception e) {
                runOnUiThread(() -> {
                    if (remoteEditorState != state || lifecycleDestroyed) return;
                    state.running = false;
                    setRemoteEditorButtonsEnabled(state, true);
                    setStatus("Remote Edit reload failed: " + safeMessage(e));
                });
            }
        });
    }

    private void closeRemoteEditor(RemoteEditorState state) {
        if (remoteEditorState != state || state.dialog == null || state.running) return;
        if (!state.dirty) {
            state.dialog.dismiss();
            return;
        }
        new AlertDialog.Builder(this)
                .setTitle("Discard Remote Edit changes?")
                .setMessage("Unsaved changes to “" + state.name + "” will be discarded.")
                .setNegativeButton("Keep editing", null)
                .setPositiveButton("Discard", (dialog, which) -> state.dialog.dismiss())
                .show();
    }

    private boolean remoteEditorUsable(RemoteEditorState state) {
        return !lifecycleDestroyed
                && remoteEditorState == state
                && state.generation == advancedOperationGeneration
                && session == state.owner
                && state.owner.isConnected();
    }

    private void applyRemoteEditorSnapshot(RemoteEditorState state, RemoteTextDocument.Snapshot snapshot) {
        state.applying = true;
        state.snapshot = snapshot;
        state.editor.setText(snapshot.text);
        state.editor.setSelection(state.editor.length());
        state.dirty = false;
        state.dialog.setTitle("Remote Edit — " + state.name);
        state.applying = false;
    }

    private void setRemoteEditorButtonsEnabled(RemoteEditorState state, boolean enabled) {
        if (state.dialog == null) return;
        if (state.dialog.getButton(AlertDialog.BUTTON_POSITIVE) != null) state.dialog.getButton(AlertDialog.BUTTON_POSITIVE).setEnabled(enabled);
        if (state.dialog.getButton(AlertDialog.BUTTON_NEUTRAL) != null) state.dialog.getButton(AlertDialog.BUTTON_NEUTRAL).setEnabled(enabled);
        if (state.dialog.getButton(AlertDialog.BUTTON_NEGATIVE) != null) state.dialog.getButton(AlertDialog.BUTTON_NEGATIVE).setEnabled(enabled);
    }

    private void refreshRemoteAfterEditor(FtpSession owner) {
        if (remoteEditorState == null || session != owner || !owner.isConnected()) return;
        String directory = currentRemotePath;
        io.execute(() -> {
            try {
                List<RemoteEntry> fresh = owner.list(directory);
                runOnUiThread(() -> {
                    if (session != owner || !owner.isConnected() || !directory.equals(currentRemotePath)) return;
                    String selectedName = selectedRemote >= 0 && selectedRemote < remoteEntries.size() ? remoteEntries.get(selectedRemote).name : "";
                    remoteEntries.clear();
                    remoteEntries.addAll(fresh);
                    selectedRemote = findRemoteByName(selectedName);
                    renderRemote();
                });
            } catch (Exception ignored) {
                // Save/read-back verification already succeeded; metadata refresh is best-effort.
            }
        });
    }

    private void createLocalDirectory() {
        if (busy || treeUri == null || currentDocumentId == null) return;
        promptText("Create local folder", "", "Folder name", false, name -> {
            final String child;
            try {
                child = validateItemName(name);
            } catch (IOException e) {
                setStatus(e.getMessage());
                return;
            }
            try {
                ensureNoLocalNameConflict(treeUri, currentDocumentId, child, "A local item with this exact name already exists.");
                Uri parent = DocumentsContract.buildDocumentUriUsingTree(treeUri, currentDocumentId);
                Uri created = DocumentsContract.createDocument(getContentResolver(), parent, DocumentsContract.Document.MIME_TYPE_DIR, child);
                if (created == null) throw new IOException("The folder could not be created.");
                String actual = queryDocumentDisplayName(created);
                if (!child.equals(actual)) {
                    try { DocumentsContract.deleteDocument(getContentResolver(), created); } catch (Exception ignored) { }
                    throw new IOException("The folder name could not be verified, so no folder was kept.");
                }
                refreshLocal();
                setStatus("Local folder created: " + child);
            } catch (Exception e) {
                setStatus("Local folder creation failed: " + safeMessage(e));
            }
        });
    }

    private void renameLocalSelected() {
        if (busy || treeUri == null || selectedLocal < 0 || selectedLocal >= localEntries.size()) return;
        LocalEntry entry = localEntries.get(selectedLocal);
        promptText("Rename local item", entry.name, "New name", false, name -> {
            final String child;
            try {
                child = validateItemName(name);
            } catch (IOException e) {
                setStatus(e.getMessage());
                return;
            }
            if (entry.name.equals(child)) {
                setStatus("Local item name is unchanged.");
                return;
            }
            try {
                ensureNoLocalNameConflict(treeUri, currentDocumentId, child, "A local item with this exact name already exists.");
                Uri document = DocumentsContract.buildDocumentUriUsingTree(treeUri, entry.documentId);
                Uri renamed = DocumentsContract.renameDocument(getContentResolver(), document, child);
                if (renamed == null) throw new IOException("The item could not be renamed.");
                String actual = queryDocumentDisplayName(renamed);
                if (!child.equals(actual)) {
                    throw new IOException("The new item name could not be verified.");
                }
                refreshLocal();
                setStatus("Local item renamed to: " + child);
            } catch (Exception e) {
                refreshLocal();
                setStatus("Local rename failed: " + safeMessage(e));
            }
        });
    }

    private void deleteLocalSelected() {
        if (busy || treeUri == null || selectedLocal < 0 || selectedLocal >= localEntries.size()) return;
        LocalEntry entry = localEntries.get(selectedLocal);
        confirmDestructive("Delete local item?", "Delete “" + entry.name + "” from the selected folder? This cannot be undone.", () -> {
            try {
                Uri document = DocumentsContract.buildDocumentUriUsingTree(treeUri, entry.documentId);
                if (!DocumentsContract.deleteDocument(getContentResolver(), document)) {
                    throw new IOException("The item could not be deleted.");
                }
                refreshLocal();
                setStatus("Local item deleted: " + entry.name);
            } catch (Exception e) {
                refreshLocal();
                setStatus("Local delete failed: " + safeMessage(e));
            }
        });
    }

    private void setLocalStart() {
        SiteProfile profile = activeProfile();
        if (profile == null) {
            setStatus("Save or load a site before setting a start folder.");
            return;
        }
        String uri = persistedCurrentTreeUri();
        if (uri.isEmpty()) {
            setStatus("Choose a folder that Ghost FTP can reopen before setting it as the start folder.");
            return;
        }
        SiteProfile next = profile.withLocalStartTreeUri(uri);
        replaceProfile(next);
        profileStore.save(profiles);
        renderBookmarks();
        setStatus("Local start folder saved.");
    }

    private void addLocalBookmark() {
        SiteProfile profile = activeProfile();
        if (profile == null) {
            setStatus("Save or load a site before adding a local bookmark.");
            return;
        }
        String uri = persistedCurrentTreeUri();
        if (uri.isEmpty()) {
            setStatus("Choose a folder that Ghost FTP can reopen before adding it as a bookmark.");
            return;
        }
        SiteProfile next = profile.withLocalBookmark(uri);
        replaceProfile(next);
        profileStore.save(profiles);
        renderBookmarks();
        setStatus("Local bookmark added.");
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
        List<String> nextBookmarks = new ArrayList<>(profile.localBookmarks);
        nextBookmarks.remove(index);
        SiteProfile next = new SiteProfile(profile.id, profile.name, profile.protocol, profile.host, profile.port,
                profile.username, profile.localStartTreeUri, profile.remoteStartPath, nextBookmarks, profile.remoteBookmarks);
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
                "Local bookmark is no longer available. Re-select the folder to restore access.",
                true)) {
            return;
        }
        savePreferences();
        setStatus("Local bookmark opened.");
    }

    private void uploadSelected() {
        FtpSession current = session;
        if (busy || current == null || selectedLocal < 0 || selectedLocal >= localEntries.size()) return;
        LocalEntry entry = localEntries.get(selectedLocal);
        if (entry.directory) {
            setStatus("Select a local file, not a folder, to upload.");
            return;
        }
        Uri document = DocumentsContract.buildDocumentUriUsingTree(treeUri, entry.documentId);
        String remoteBase = currentRemotePath;
        TransferAttempt attempt = beginTransfer("Uploading " + entry.name + "…");
        TransferProgress progress = transferProgress(entry.size, "Uploading", current, attempt);
        io.execute(() -> {
            try {
                requireTransferCurrent(attempt);
                InputStream source = getContentResolver().openInputStream(document);
                if (source == null) throw new IOException("Could not open local file.");
                try (InputStream in = ProgressStreams.input(source, progress::onTransferred)) {
                    requireTransferCurrent(attempt);
                    current.upload(FtpSession.joinRemote(remoteBase, entry.name), in, attempt.gate);
                }
                if (transferCancelled(attempt.token)) {
                    current.close();
                    return;
                }
                runOnUiThread(() -> {
                    if (!finishTransferState(attempt)) return;
                    if (session != current || !current.isConnected()) {
                        setBusy(false, "The connection closed while finishing the upload. Reconnect and refresh before retrying.");
                        return;
                    }
                    setBusy(false, "Upload completed: " + entry.name);
                    refreshRemote(currentRemotePath);
                });
            } catch (Exception e) {
                finishTransferFailure(attempt, current, "Upload failed", e, false);
            }
        });
    }

    private void downloadSelected() {
        FtpSession current = session;
        if (busy || current == null || selectedRemote < 0 || selectedRemote >= remoteEntries.size() || treeUri == null) return;
        RemoteEntry entry = remoteEntries.get(selectedRemote);
        if (entry.directory) {
            setStatus("Select a server file, not a folder, to download.");
            return;
        }
        Uri selectedTree = treeUri;
        String selectedDocumentId = currentDocumentId;
        Uri parent = DocumentsContract.buildDocumentUriUsingTree(selectedTree, selectedDocumentId);
        String remoteBase = currentRemotePath;
        TransferAttempt attempt = beginTransfer("Downloading " + entry.name + "…");
        TransferProgress progress = transferProgress(entry.size, "Downloading", current, attempt);
        io.execute(() -> {
            Uri staged = null;
            try {
                requireTransferCurrent(attempt);
                ensureNoLocalNameConflict(selectedTree, selectedDocumentId, entry.name,
                        "A local item with this exact name already exists. Remove or rename it before downloading.");
                requireTransferCurrent(attempt);

                String stagedName = ".ghostftp-download-" + UUID.randomUUID() + ".part";
                staged = DocumentsContract.createDocument(getContentResolver(), parent, "application/octet-stream", stagedName);
                if (staged == null) throw new IOException("Could not prepare the local download.");

                requireTransferCurrent(attempt);
                OutputStream destination = getContentResolver().openOutputStream(staged, "w");
                if (destination == null) throw new IOException("Could not open the local download destination.");
                try (OutputStream out = ProgressStreams.output(destination, progress::onTransferred)) {
                    current.download(FtpSession.joinRemote(remoteBase, entry.name), out, attempt.gate);
                }
                requireTransferCurrent(attempt);

                ensureNoLocalNameConflict(selectedTree, selectedDocumentId, entry.name,
                        "A file with that name appeared during download. No existing file was replaced.");
                requireTransferCurrent(attempt);
                beginFinalCommit(attempt, "Finalizing download…");

                Uri committed = DocumentsContract.renameDocument(getContentResolver(), staged, entry.name);
                if (committed == null) throw new IOException("The download could not be finalized.");
                staged = committed;
                String committedName = queryDocumentDisplayName(committed);
                if (!entry.name.equals(committedName)) {
                    throw new IOException("The downloaded file name could not be verified.");
                }
                attempt.gate.finish();
                staged = null;

                if (transferCancelled(attempt.token)) {
                    current.close();
                    return;
                }
                runOnUiThread(() -> {
                    if (!finishTransferState(attempt)) return;
                    if (session != current || !current.isConnected()) {
                        refreshLocal();
                        setBusy(false, "Download completed, but the connection closed. Reconnect before another transfer.");
                        return;
                    }
                    setBusy(false, "Download completed: " + entry.name);
                    refreshLocal();
                });
            } catch (Exception e) {
                if (staged != null) {
                    try { DocumentsContract.deleteDocument(getContentResolver(), staged); } catch (Exception ignored) { }
                }
                if (attempt.gate.isCancelled()) {
                    current.closeCancelledTransferSession();
                }
                finishTransferFailure(attempt, current, "Download failed", e, true);
            }
        });
    }

    private TransferProgress transferProgress(long totalBytes, String action, FtpSession current, TransferAttempt attempt) {
        return new TransferProgress(totalBytes, action, text -> runOnUiThread(() -> {
            if (!isTransferCurrent(attempt) || !current.isConnected()) return;
            setStatus(text);
        }));
    }

    private TransferAttempt beginTransfer(String message) {
        TransferCommitGate gate = new TransferCommitGate();
        long token = ++transferGeneration;
        activeTransferGate = gate;
        transferActive = true;
        transferFinalizing = false;
        setBusy(true, message);
        return new TransferAttempt(token, gate);
    }

    private boolean transferCancelled(long token) {
        return token != transferGeneration;
    }

    private boolean isTransferCurrent(TransferAttempt attempt) {
        return attempt != null
                && attempt.token == transferGeneration
                && activeTransferGate == attempt.gate;
    }

    private void requireTransferCurrent(TransferAttempt attempt) throws IOException {
        if (!isTransferCurrent(attempt) || attempt.gate.isCancelled()) {
            throw new IOException("Transfer cancelled.");
        }
    }

    private void beginFinalCommit(TransferAttempt attempt, String message) throws IOException {
        requireTransferCurrent(attempt);
        if (!attempt.gate.beginCommit()) {
            throw new IOException("Transfer cancelled.");
        }
        transferFinalizing = true;
        runOnUiThread(() -> {
            if (!isTransferCurrent(attempt)) return;
            setStatus(message);
            refreshButtons();
        });
    }

    private boolean finishTransferState(TransferAttempt attempt) {
        if (!isTransferCurrent(attempt)) return false;
        transferActive = false;
        transferFinalizing = false;
        activeTransferGate = null;
        return true;
    }

    private void finishTransferFailure(TransferAttempt attempt, FtpSession current, String prefix, Exception e, boolean refreshLocalAfter) {
        boolean cancelled = attempt.gate.isCancelled() || transferCancelled(attempt.token);
        runOnUiThread(() -> {
            if (!finishTransferState(attempt)) return;
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
                throw new IOException("The downloaded file name could not be verified.");
            }
            String name = cursor.getString(0);
            if (name == null || name.isEmpty()) {
                throw new IOException("The downloaded file name could not be verified.");
            }
            return name;
        } catch (SecurityException e) {
            throw new IOException("Storage access was lost while finishing the download.", e);
        }
    }

    private static String validateItemName(String value) throws IOException {
        String name = value == null ? "" : value.trim();
        if (name.isEmpty()) throw new IOException("Name is required.");
        if (".".equals(name) || "..".equals(name)) throw new IOException("Choose a different name.");
        if (name.indexOf('/') >= 0 || name.indexOf('\\') >= 0
                || name.indexOf('\0') >= 0 || name.indexOf('\r') >= 0 || name.indexOf('\n') >= 0) {
            throw new IOException("Names cannot contain slashes or control characters.");
        }
        return name;
    }

    private void promptText(String title, String initial, String hint, boolean numeric, TextPromptAction action) {
        if (busy || lifecycleDestroyed) return;
        EditText input = field(hint, false);
        input.setText(initial == null ? "" : initial);
        if (numeric) input.setInputType(InputType.TYPE_CLASS_NUMBER);
        int pad = dp(20);
        FrameLayout holder = new FrameLayout(this);
        holder.setPadding(pad, dp(8), pad, 0);
        holder.addView(input, new FrameLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT));
        AlertDialog dialog = new AlertDialog.Builder(this)
                .setTitle(title)
                .setView(holder)
                .setNegativeButton("Cancel", null)
                .setPositiveButton("Apply", null)
                .create();
        dialog.setOnShowListener(ignored -> dialog.getButton(AlertDialog.BUTTON_POSITIVE).setOnClickListener(v -> {
            String value = input.getText().toString();
            dialog.dismiss();
            action.accept(value);
        }));
        dialog.show();
    }

    private void confirmDestructive(String title, String message, Runnable action) {
        if (busy || lifecycleDestroyed) return;
        new AlertDialog.Builder(this)
                .setTitle(title)
                .setMessage(message)
                .setNegativeButton("Cancel", null)
                .setPositiveButton("Delete", (dialog, which) -> action.run())
                .show();
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
        if (localPath != null) localPath.setText(displayLocalPath());
        if (bookmarkLocalCurrent != null) bookmarkLocalCurrent.setText(displayLocalPath());
        localVisibleItems.clear();
        localVisibleItems.addAll(WorkspaceOps.filterAndSort(localWorkspaceItems(), localFilterQuery, localSortKey, localSortAscending));
        List<String> labels = new ArrayList<>();
        for (WorkspaceOps.Item visible : localVisibleItems) {
            LocalEntry e = localEntries.get(visible.sourceIndex);
            String marker = visible.sourceIndex == selectedLocal ? "●  " : "   ";
            String type = e.directory ? "Folder · " : "File · ";
            String size = !e.directory && showFileSizes ? "   " + TransferProgress.formatBytes(e.size) : "";
            labels.add(marker + type + e.name + size);
        }
        if (localList != null) localList.setAdapter(GhostTheme.listAdapter(this, labels));
        if (localList != null && localEmptyState != null) {
            boolean hasVisibleItems = !localVisibleItems.isEmpty();
            String emptyMessage;
            if (treeUri == null || currentDocumentId == null) {
                emptyMessage = "Choose a folder to browse local files.";
            } else if (!localFilterQuery.isEmpty() && !localEntries.isEmpty()) {
                emptyMessage = "No local files match this filter.";
            } else {
                emptyMessage = "This local folder is empty.";
            }
            updateWorkspaceListState(localList, localEmptyState, hasVisibleItems, emptyMessage);
        }
        if (localFilter != null) localFilter.setText(localFilterQuery.isEmpty() ? "Filter" : "Filter: " + localFilterQuery);
        if (localSort != null) localSort.setText("Sort: " + WorkspaceOps.sortLabel(localSortKey, localSortAscending));
        refreshButtons();
    }

    private void renderRemote() {
        if (remotePath != null) remotePath.setText(currentRemotePath);
        if (bookmarkRemoteCurrent != null) bookmarkRemoteCurrent.setText(currentRemotePath);
        remoteVisibleItems.clear();
        remoteVisibleItems.addAll(WorkspaceOps.filterAndSort(remoteWorkspaceItems(), remoteFilterQuery, remoteSortKey, remoteSortAscending));
        List<String> labels = new ArrayList<>();
        for (WorkspaceOps.Item visible : remoteVisibleItems) {
            RemoteEntry e = remoteEntries.get(visible.sourceIndex);
            String marker = visible.sourceIndex == selectedRemote ? "●  " : "   ";
            String type = e.directory ? "Folder · " : "File · ";
            String size = !e.directory && showFileSizes ? "   " + TransferProgress.formatBytes(e.size) : "";
            String permissions = e.permissions.isEmpty() ? "" : "   [" + e.permissions + "]";
            labels.add(marker + type + e.name + size + permissions);
        }
        if (remoteList != null) remoteList.setAdapter(GhostTheme.listAdapter(this, labels));
        if (remoteList != null && remoteEmptyState != null) {
            boolean connected = session != null && session.isConnected();
            boolean hasVisibleItems = connected && !remoteVisibleItems.isEmpty();
            String emptyMessage;
            if (!connected) {
                emptyMessage = "Connect from Sites to browse server files.";
            } else if (!remoteFilterQuery.isEmpty() && !remoteEntries.isEmpty()) {
                emptyMessage = "No server files match this filter.";
            } else {
                emptyMessage = "This server folder is empty.";
            }
            updateWorkspaceListState(remoteList, remoteEmptyState, hasVisibleItems, emptyMessage);
        }
        if (remoteFilter != null) remoteFilter.setText(remoteFilterQuery.isEmpty() ? "Filter" : "Filter: " + remoteFilterQuery);
        if (remoteSort != null) remoteSort.setText("Sort: " + WorkspaceOps.sortLabel(remoteSortKey, remoteSortAscending));
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

        connect.setEnabled(!busy && !connected);
        disconnect.setText(transferFinalizing ? "Finalizing…" : transferActive ? "Cancel transfer" : "Disconnect");
        disconnect.setEnabled((transferActive && !transferFinalizing) || (!busy && connected));
        upload.setEnabled(!busy && connected && selectedLocalFile);
        download.setEnabled(!busy && connected && selectedRemoteFile && treeUri != null);
        localCreateDirectory.setEnabled(!busy && treeUri != null && currentDocumentId != null);
        localRename.setEnabled(!busy && selectedLocalItem && treeUri != null);
        localDelete.setEnabled(!busy && selectedLocalItem && treeUri != null);
        remoteCreateDirectory.setEnabled(!busy && connected);
        remoteRename.setEnabled(!busy && connected && selectedRemoteItem);
        remoteDelete.setEnabled(!busy && connected && selectedRemoteItem);
        remoteChmod.setEnabled(!busy && connected && selectedRemoteItem);
        localFilter.setEnabled(!busy && treeUri != null);
        localSort.setEnabled(!busy && !localEntries.isEmpty());
        localSearch.setEnabled(!busy && treeUri != null && rootDocumentId != null);
        remoteFilter.setEnabled(!busy && connected);
        remoteSort.setEnabled(!busy && connected && !remoteEntries.isEmpty());
        remoteSearch.setEnabled(!busy && connected);
        directoryCompare.setEnabled(!busy && connected && treeUri != null && currentDocumentId != null);
        remoteEdit.setEnabled(!busy && connected && selectedRemoteFile
                && remoteEntries.get(selectedRemote).size <= WorkspaceOps.MAX_REMOTE_EDIT_BYTES);
        saveSite.setEnabled(!busy && !connected);
        deleteSite.setEnabled(!busy && !connected && activeSite);
        localSetStart.setEnabled(!busy && activeSite && !persistedCurrentTreeUri().isEmpty());
        localAddBookmark.setEnabled(!busy && activeSite && !persistedCurrentTreeUri().isEmpty());
        localRemoveBookmark.setEnabled(!busy && activeSite && !profile.localBookmarks.isEmpty());
        localOpenBookmark.setEnabled(!busy && activeSite && !profile.localBookmarks.isEmpty());
        remoteSetStart.setEnabled(!busy && connectedSite);
        remoteAddBookmark.setEnabled(!busy && connectedSite);
        remoteRemoveBookmark.setEnabled(!busy && activeSite && !profile.remoteBookmarks.isEmpty());
        remoteOpenBookmark.setEnabled(!busy && connectedSite && !profile.remoteBookmarks.isEmpty());
        transferCancel.setText(transferFinalizing ? "Finalizing…" : "Cancel active transfer");
        transferCancel.setEnabled(transferActive && !transferFinalizing);
        transferCancel.setAlpha(transferCancel.isEnabled() ? 1f : 0.45f);
        if (applyAppearance != null) {
            boolean appearanceReady = !busy && !transferActive && !transferFinalizing && remoteEditorState == null;
            applyAppearance.setEnabled(appearanceReady);
            applyAppearance.setAlpha(appearanceReady ? 1f : 0.45f);
        }

        updateConnectionBadge(connected);
        updateTransferSurface();
        updateEnabledAlpha(connect, disconnect, upload, download,
                localCreateDirectory, localRename, localDelete,
                remoteCreateDirectory, remoteRename, remoteDelete, remoteChmod,
                localFilter, localSort, localSearch, remoteFilter, remoteSort, remoteSearch,
                directoryCompare, remoteEdit, saveSite, deleteSite,
                localSetStart, localAddBookmark, localRemoveBookmark, localOpenBookmark,
                remoteSetStart, remoteAddBookmark, remoteRemoveBookmark, remoteOpenBookmark);
    }

    private void updateConnectionBadge(boolean connected) {
        if (connectionBadge == null) return;
        String text;
        int color;
        if (transferFinalizing) {
            text = "FINALIZING";
            color = GhostTheme.WARN;
        } else if (transferActive) {
            text = "TRANSFER ACTIVE";
            color = GhostTheme.WARN;
        } else if (connected) {
            boolean secure = protocol.getSelectedItem() != null && "FTPS".equals(protocol.getSelectedItem().toString());
            text = secure ? "FTPS CONNECTED" : "FTP CONNECTED";
            color = secure ? GhostTheme.SUCCESS : GhostTheme.WARN;
        } else {
            text = "DISCONNECTED";
            color = GhostTheme.MUTED;
        }
        connectionBadge.setText(text);
        GhostTheme.styleBadge(connectionBadge, color);
    }

    private void updateTransferSurface() {
        if (transferStatus == null) return;
        String value = status == null ? "Ready." : status.getText().toString();
        if (!transferActive && !transferFinalizing && (value.isEmpty() || "Ready.".equals(value))) {
            value = "No active transfer.";
        }
        transferStatus.setText(value);
        transferStatus.setTextColor(GhostTheme.statusColor(value));
    }

    private void updateEnabledAlpha(Button... buttons) {
        for (Button button : buttons) {
            if (button != null) button.setAlpha(button.isEnabled() ? 1f : 0.45f);
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
        if (lifecycleDestroyed) return;
        runOnUiThread(() -> {
            if (lifecycleDestroyed) return;
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
        updateTransferSurface();
    }

    private static String safeMessage(Exception e) {
        String value = e == null ? null : e.getMessage();
        if (value == null || value.trim().isEmpty()) return "The operation could not be completed.";
        String safe = value.replace('\n', ' ').replace('\r', ' ').trim();
        String lower = safe.toLowerCase(java.util.Locale.ROOT);
        if (safe.length() > 180
                || safe.contains("/home/")
                || safe.contains("/data/user/")
                || safe.contains("/data/data/")
                || safe.contains("java.")
                || safe.contains("javax.")
                || safe.contains("android.")
                || safe.contains("app.ghostftp.")
                || safe.contains(".java:")
                || safe.contains("Exception")
                || safe.contains("StackTrace")
                || lower.contains("stag" + "ing")
                || lower.contains("final " + "commit")
                || lower.contains("server response " + "code")
                || lower.contains("control " + "connection")
                || lower.contains("cancellation " + "lifecycle")
                || lower.contains("data " + "channel")) {
            return "The operation could not be completed. Check the connection and try again.";
        }
        return safe;
    }

    private LinearLayout surfaceContent() {
        LinearLayout content = new LinearLayout(this);
        content.setOrientation(LinearLayout.VERTICAL);
        content.setPadding(dp(12), dp(12), dp(12), dp(24));
        content.setBackgroundColor(GhostTheme.WINDOW);
        return content;
    }

    private View scrollSurface(LinearLayout content) {
        ScrollView scroll = new ScrollView(this);
        scroll.setFillViewport(true);
        scroll.setBackgroundColor(GhostTheme.WINDOW);
        scroll.addView(content, matchWrap());
        return scroll;
    }

    private LinearLayout surfaceHeading(String title, String description) {
        LinearLayout block = new LinearLayout(this);
        block.setOrientation(LinearLayout.VERTICAL);
        block.setPadding(dp(2), dp(2), dp(2), dp(12));
        TextView heading = label(title, 24, GhostTheme.TEXT);
        heading.setTypeface(Typeface.DEFAULT, Typeface.BOLD);
        TextView copy = label(description, 12, GhostTheme.MUTED);
        copy.setPadding(0, dp(4), 0, 0);
        block.addView(heading, matchWrap());
        block.addView(copy, matchWrap());
        return block;
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

    private TextView workspaceEmptyState(String value) {
        TextView view = label(value, 12, GhostTheme.MUTED);
        view.setGravity(Gravity.CENTER_VERTICAL);
        view.setMinHeight(dp(78));
        view.setPadding(dp(12), dp(12), dp(12), dp(12));
        view.setBackground(GhostTheme.rounded(this, GhostTheme.LIST, GhostTheme.BORDER, 10));
        return view;
    }

    private void updateWorkspaceListState(ListView list, TextView emptyState, boolean hasItems, String emptyMessage) {
        emptyState.setText(emptyMessage);
        emptyState.setVisibility(hasItems ? View.GONE : View.VISIBLE);
        list.setVisibility(hasItems ? View.VISIBLE : View.GONE);
    }

    private TextView pathLabel(String value) {
        TextView view = label(value, 13, GhostTheme.MUTED);
        view.setPadding(dp(10), dp(9), dp(10), dp(9));
        view.setBackground(GhostTheme.rounded(this, GhostTheme.LIST, GhostTheme.BORDER, 9));
        return view;
    }

    private TextView infoLine(String name, String value) {
        TextView view = label(name + "\n" + value, 12, GhostTheme.TEXT);
        view.setPadding(dp(10), dp(9), dp(10), dp(9));
        view.setBackground(GhostTheme.rounded(this, GhostTheme.LIST, GhostTheme.BORDER, 9));
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
        GhostTheme.styleField(edit);
        if (secret) edit.setInputType(InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_VARIATION_PASSWORD);
        return edit;
    }

    private CheckBox checkBox(String text) {
        CheckBox check = new CheckBox(this);
        check.setText(text);
        check.setTextColor(GhostTheme.TEXT);
        check.setTextSize(13f);
        check.setButtonTintList(new ColorStateList(
                new int[][]{new int[]{android.R.attr.state_checked}, new int[]{}},
                new int[]{GhostTheme.ACCENT, GhostTheme.MUTED}));
        check.setPadding(dp(4), dp(6), dp(4), dp(6));
        return check;
    }

    private Button button(String text) {
        Button button = new Button(this);
        button.setText(text);
        GhostTheme.styleSecondaryButton(button);
        return button;
    }

    private Button primaryButton(String text) {
        Button button = new Button(this);
        button.setText(text);
        GhostTheme.stylePrimaryButton(button);
        return button;
    }

    private Button dangerButton(String text) {
        Button button = new Button(this);
        button.setText(text);
        GhostTheme.styleDangerButton(button);
        return button;
    }

    private LinearLayout row() {
        LinearLayout row = new LinearLayout(this);
        row.setOrientation(LinearLayout.HORIZONTAL);
        return row;
    }

    private LinearLayout.LayoutParams navParams() {
        LinearLayout.LayoutParams params = matchWrap();
        params.setMargins(0, 0, 0, dp(5));
        return params;
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

    private static final class LocalSearchNode {
        final String documentId;
        final String displayPath;
        final List<String> ancestors;
        final int depth;

        LocalSearchNode(String documentId, String displayPath, List<String> ancestors, int depth) {
            this.documentId = documentId;
            this.displayPath = displayPath;
            this.ancestors = new ArrayList<>(ancestors);
            this.depth = depth;
        }
    }

    private static final class LocalSearchResult {
        final String parentDocumentId;
        final List<String> ancestors;
        final LocalEntry entry;
        final String displayPath;

        LocalSearchResult(String parentDocumentId, List<String> ancestors, LocalEntry entry, String displayPath) {
            this.parentDocumentId = parentDocumentId;
            this.ancestors = new ArrayList<>(ancestors);
            this.entry = entry;
            this.displayPath = displayPath;
        }
    }

    private static final class RemoteSearchNode {
        final String path;
        final int depth;

        RemoteSearchNode(String path, int depth) {
            this.path = path;
            this.depth = depth;
        }
    }

    private static final class RemoteSearchResult {
        final String parentPath;
        final RemoteEntry entry;
        final String displayPath;

        RemoteSearchResult(String parentPath, RemoteEntry entry, String displayPath) {
            this.parentPath = parentPath;
            this.entry = entry;
            this.displayPath = displayPath;
        }
    }

    private static final class RemoteEditorState {
        final FtpSession owner;
        final String path;
        final String name;
        final String originalMode;
        final long generation;
        final EditText editor;
        RemoteTextDocument.Snapshot snapshot;
        AlertDialog dialog;
        boolean applying;
        boolean dirty;
        boolean running;

        RemoteEditorState(FtpSession owner, String path, String name, String originalMode, long generation,
                          RemoteTextDocument.Snapshot snapshot, EditText editor) {
            this.owner = owner;
            this.path = path;
            this.name = name;
            this.originalMode = originalMode == null ? "" : originalMode.trim();
            this.generation = generation;
            this.snapshot = snapshot;
            this.editor = editor;
        }
    }

    private static final class TransferAttempt {
        final long token;
        final TransferCommitGate gate;

        TransferAttempt(long token, TransferCommitGate gate) {
            this.token = token;
            this.gate = gate;
        }
    }

    private static final class LocalEntry {
        final String documentId;
        final String name;
        final boolean directory;
        final long size;
        final long modifiedEpochMillis;

        LocalEntry(String documentId, String name, boolean directory, long size, long modifiedEpochMillis) {
            this.documentId = documentId;
            this.name = name;
            this.directory = directory;
            this.size = size;
            this.modifiedEpochMillis = Math.max(0L, modifiedEpochMillis);
        }
    }
}
