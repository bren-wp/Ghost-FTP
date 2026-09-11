package app.ghostftp.client;

import android.annotation.SuppressLint;
import android.app.Activity;
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
import android.text.InputType;
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
import java.util.List;
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

    private final ExecutorService io = Executors.newSingleThreadExecutor();
    private final List<LocalEntry> localEntries = new ArrayList<>();
    private final List<RemoteEntry> remoteEntries = new ArrayList<>();
    private final Deque<String> localParents = new ArrayDeque<>();
    private final List<SiteProfile> profiles = new ArrayList<>();
    private final List<Button> navigationButtons = new ArrayList<>();

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
    private Button remoteSetStart;
    private Button remoteAddBookmark;
    private Button remoteRemoveBookmark;
    private Button remoteOpenBookmark;
    private Button transferCancel;
    private CheckBox rememberEndpointToggle;
    private CheckBox showFileSizesToggle;

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
    private boolean busy;
    private boolean rememberEndpoint = true;
    private boolean showFileSizes = true;
    private volatile boolean transferActive;
    private volatile boolean transferFinalizing;
    private volatile long transferGeneration;
    private volatile TransferCommitGate activeTransferGate;

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
        renderLocal();
        renderRemote();
        refreshButtons();
        showSection(Section.FILES);
    }

    @Override
    protected void onDestroy() {
        transferGeneration++;
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
        TextView platform = label("Android development client", 11, GhostTheme.MUTED);
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
        content.addView(surfaceHeading("Files", "Local SAF storage and the active FTP/FTPS server. Only this workspace owns file selection and transfer actions."));

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

        LinearLayout transferCard = card("TRANSFER", "Transfer only the selected file. Staged upload/download and cancellation safety remain authoritative.");
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
        LinearLayout card = card("LOCAL", "Android Storage Access Framework only; Ghost FTP never requests broad all-files access.");
        localPath = pathLabel("No folder selected");
        card.addView(localPath, matchWrapSpaced());
        LinearLayout actions = row();
        Button choose = button("Choose folder");
        Button up = button("Up");
        Button refresh = button("Refresh");
        actions.addView(choose, weightedSpaced());
        actions.addView(up, weightedSpaced());
        actions.addView(refresh, weightedSpaced());
        card.addView(actions, matchWrap());
        choose.setOnClickListener(v -> chooseFolder());
        up.setOnClickListener(v -> localUp());
        refresh.setOnClickListener(v -> refreshLocal());
        localList = new ListView(this);
        GhostTheme.styleList(localList);
        card.addView(localList, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, dp(280)));
        localList.setOnItemClickListener((parent, view, position, id) -> selectLocal(position));
        return card;
    }

    private LinearLayout buildRemoteFilesCard() {
        LinearLayout card = card("SERVER", "Fresh MLSD listings over the active FTP/FTPS session; FTPS keeps strict TLS and hostname verification.");
        remotePath = pathLabel(currentRemotePath);
        card.addView(remotePath, matchWrapSpaced());
        LinearLayout actions = row();
        Button up = button("Up");
        Button refresh = button("Refresh");
        actions.addView(up, weightedSpaced());
        actions.addView(refresh, weightedSpaced());
        card.addView(actions, matchWrap());
        up.setOnClickListener(v -> remoteUp());
        refresh.setOnClickListener(v -> refreshRemote(currentRemotePath));
        remoteList = new ListView(this);
        GhostTheme.styleList(remoteList);
        card.addView(remoteList, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, dp(280)));
        remoteList.setOnItemClickListener((parent, view, position, id) -> selectRemote(position));
        return card;
    }

    private View buildSitesSurface() {
        LinearLayout content = surfaceContent();
        content.addView(surfaceHeading("Sites", "Quick Connect stays transient. Saved sites contain non-secret connection and navigation metadata only."));

        LinearLayout connectionCard = card("QUICK CONNECT / CONNECTION", "Password is memory-only. Android currently exposes FTP and strict explicit FTPS; SFTP stays hidden until strict host-key verification exists.");
        protocol = new Spinner(this);
        GhostTheme.styleSpinner(protocol);
        protocol.setAdapter(GhostTheme.spinnerAdapter(this, java.util.Arrays.asList(new String[]{"FTPS", "FTP"})));
        connectionCard.addView(protocol, matchWrapSpaced());
        host = field("Server host", false);
        port = field("Port", false);
        port.setInputType(InputType.TYPE_CLASS_NUMBER);
        username = field("Username", false);
        password = field("Password (memory only)", true);
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

        LinearLayout savedCard = card("SAVED SITES", "Load, create, update or delete an explicit saved site. Quick Connect never creates hidden profiles.");
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
        content.addView(surfaceHeading("Bookmarks", "Navigation state is explicit and account-bound. Quick Connect does not create hidden bookmarks."));

        LinearLayout localCard = card("LOCAL SAF BOOKMARKS", "Local starts/bookmarks are SAF capability URIs and are freshly revalidated before navigation.");
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

        LinearLayout remoteCard = card("SERVER BOOKMARKS", "Remote paths are bound to protocol, canonical host, port and exact username, and are freshly listed before visible commit.");
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
        content.addView(surfaceHeading("Transfers", "This surface shows the real active transfer lifecycle. No decorative queue or fake history is displayed."));
        LinearLayout card = card("ACTIVE TRANSFER", "Progress comes from actual bytes read/written. Cancellation is available only before the irreversible final-name commit gate.");
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
        content.addView(surfaceHeading("Settings", "Only settings with a real Android runtime owner are interactive."));
        LinearLayout uiCard = card("UI / LOCAL PREFERENCES", "Ghost FTP Android uses the canonical dark brand palette. These options change actual local runtime behavior.");
        rememberEndpointToggle = checkBox("Remember Quick Connect host, username, protocol and port");
        rememberEndpointToggle.setOnClickListener(v -> {
            rememberEndpoint = rememberEndpointToggle.isChecked();
            savePreferences();
            setStatus(rememberEndpoint
                    ? "Quick Connect metadata will be remembered. Passwords remain memory-only."
                    : "Quick Connect metadata persistence disabled and stored endpoint metadata cleared.");
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

        LinearLayout securityCard = card("SECURITY", "Runtime security policy is informational here and cannot be weakened from the UI.");
        securityCard.addView(infoLine("FTPS", "Platform trust store + strict hostname verification"), matchWrapSpaced());
        securityCard.addView(infoLine("Passwords", "Memory-only; never stored in site JSON/preferences"), matchWrapSpaced());
        securityCard.addView(infoLine("Local storage", "Android SAF grants only; no all-files permission"), matchWrapSpaced());
        securityCard.addView(infoLine("SFTP", "Hidden until strict Android host-key identity verification exists"), matchWrapSpaced());
        securityCard.addView(infoLine("Privacy", "No telemetry, analytics, ads, fingerprinting or Ghost FTP cloud"), matchWrapSpaced());
        content.addView(securityCard, cardParams());
        return scrollSurface(content);
    }

    private View buildAboutSurface() {
        LinearLayout content = surfaceContent();
        content.addView(surfaceHeading("About", "Build identity and privacy/security status for this Android development client."));
        LinearLayout card = card("GHOST FTP", "Private file transfer client for direct connections to servers you control.");
        card.addView(infoLine("Version", BuildConfig.VERSION_NAME), matchWrapSpaced());
        card.addView(infoLine("Package", BuildConfig.APPLICATION_ID), matchWrapSpaced());
        card.addView(infoLine("Protocols", "FTP + strict explicit FTPS on Android source line"), matchWrapSpaced());
        card.addView(infoLine("Release status", "Repository build " + BuildConfig.VERSION_NAME + "; Android APK remains development-only and is not a public Android release"), matchWrapSpaced());
        card.addView(infoLine("Data collection", "None: no telemetry, analytics, ads or hidden backend"), matchWrapSpaced());
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
                return "Sites / Connections";
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
        rememberEndpointToggle.setChecked(rememberEndpoint);
        showFileSizesToggle.setChecked(showFileSizes);
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
            setStatus("Finalizing transfer. The final-name commit has started and cannot be cancelled safely.");
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
                ? "Cancellation accepted. Cleaning staged data before closing the session."
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
        return currentDocumentId.equals(rootDocumentId) ? "Selected SAF folder" : currentDocumentId;
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
        renderBookmarks();
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
                        setBusy(false, "Upload finalization lost its connection. Reconnect and refresh before retrying.");
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
            setStatus("Select a server file, not a directory, to download.");
            return;
        }
        Uri selectedTree = treeUri;
        String selectedDocumentId = currentDocumentId;
        Uri parent = DocumentsContract.buildDocumentUriUsingTree(selectedTree, selectedDocumentId);
        String remoteBase = currentRemotePath;
        TransferAttempt attempt = beginTransfer("Downloading " + entry.name + " to a staged local document…");
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
                if (staged == null) throw new IOException("Could not create staged local download document.");

                requireTransferCurrent(attempt);
                OutputStream destination = getContentResolver().openOutputStream(staged, "w");
                if (destination == null) throw new IOException("Could not open staged local download document.");
                try (OutputStream out = ProgressStreams.output(destination, progress::onTransferred)) {
                    current.download(FtpSession.joinRemote(remoteBase, entry.name), out, attempt.gate);
                }
                requireTransferCurrent(attempt);

                ensureNoLocalNameConflict(selectedTree, selectedDocumentId, entry.name,
                        "A local item with the destination name appeared during download; staged data was not committed.");
                requireTransferCurrent(attempt);
                beginFinalCommit(attempt, "Finalizing download name…");

                Uri committed = DocumentsContract.renameDocument(getContentResolver(), staged, entry.name);
                if (committed == null) throw new IOException("Storage provider rejected the final download name commit.");
                staged = committed;
                String committedName = queryDocumentDisplayName(committed);
                if (!entry.name.equals(committedName)) {
                    throw new IOException("Storage provider changed the requested final download name; commit was rejected.");
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
                        setBusy(false, "Download committed, but the connection was lost during finalization. Reconnect before another transfer.");
                        return;
                    }
                    setBusy(false, "Download completed and committed: " + entry.name);
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
            throw new IOException("Transfer cancelled before final-name commit.");
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
        if (localPath != null) localPath.setText(displayLocalPath());
        if (bookmarkLocalCurrent != null) bookmarkLocalCurrent.setText(displayLocalPath());
        List<String> labels = new ArrayList<>();
        for (int i = 0; i < localEntries.size(); i++) {
            LocalEntry e = localEntries.get(i);
            String marker = i == selectedLocal ? "●  " : "   ";
            String type = e.directory ? "DIR   " : "FILE  ";
            String size = !e.directory && showFileSizes ? "   " + TransferProgress.formatBytes(e.size) : "";
            labels.add(marker + type + e.name + size);
        }
        if (localList != null) localList.setAdapter(GhostTheme.listAdapter(this, labels));
        refreshButtons();
    }

    private void renderRemote() {
        if (remotePath != null) remotePath.setText(currentRemotePath);
        if (bookmarkRemoteCurrent != null) bookmarkRemoteCurrent.setText(currentRemotePath);
        List<String> labels = new ArrayList<>();
        for (int i = 0; i < remoteEntries.size(); i++) {
            RemoteEntry e = remoteEntries.get(i);
            String marker = i == selectedRemote ? "●  " : "   ";
            String type = e.directory ? "DIR   " : "FILE  ";
            String size = !e.directory && showFileSizes ? "   " + TransferProgress.formatBytes(e.size) : "";
            labels.add(marker + type + e.name + size);
        }
        if (remoteList != null) remoteList.setAdapter(GhostTheme.listAdapter(this, labels));
        refreshButtons();
    }

    private void setBusy(boolean value, String message) {
        busy = value;
        setStatus(message);
        refreshButtons();
    }

    private void refreshButtons() {
        boolean connected = session != null && session.isConnected();
        boolean selectedLocalFile = selectedLocal >= 0 && selectedLocal < localEntries.size() && !localEntries.get(selectedLocal).directory;
        boolean selectedRemoteFile = selectedRemote >= 0 && selectedRemote < remoteEntries.size() && !remoteEntries.get(selectedRemote).directory;
        SiteProfile profile = activeProfile();
        boolean activeSite = profile != null;
        boolean connectedSite = activeSite && connected && connectedIdentityKey != null && profile.identityKey().equals(connectedIdentityKey);

        connect.setEnabled(!busy && !connected);
        disconnect.setText(transferFinalizing ? "Finalizing…" : transferActive ? "Cancel transfer" : "Disconnect");
        disconnect.setEnabled((transferActive && !transferFinalizing) || (!busy && connected));
        upload.setEnabled(!busy && connected && selectedLocalFile);
        download.setEnabled(!busy && connected && selectedRemoteFile && treeUri != null);
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

        updateConnectionBadge(connected);
        updateTransferSurface();
        updateEnabledAlpha(connect, disconnect, upload, download, saveSite, deleteSite,
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
        updateTransferSurface();
    }

    private static String safeMessage(Exception e) {
        String value = e.getMessage();
        return value == null || value.trim().isEmpty() ? e.getClass().getSimpleName() : value.replace('\n', ' ').replace('\r', ' ');
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

        LocalEntry(String documentId, String name, boolean directory, long size) {
            this.documentId = documentId;
            this.name = name;
            this.directory = directory;
            this.size = size;
        }
    }
}
