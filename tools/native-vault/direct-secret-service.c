/* Test-only candidate: never link into the desktop or use real credentials. */
#include <libsecret/secret.h>
#include <gio/gio.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

static const char *fixture_service = "com.ghostftp.native-vault.compat.fixture";
static const char *fixture_value = "ghostftp-test-only-credential";
static const char *legacy_account = "legacy";
static const char *direct_account = "direct";

static void require(gboolean ok, const char *stage) {
    if (!ok) {
        /* Do not print GError messages or credential values. */
        fprintf(stderr, "direct-vault failure: %s\n", stage);
        exit(1);
    }
}

static GHashTable *attributes(const char *account) {
    GHashTable *a = g_hash_table_new(g_str_hash, g_str_equal);
    g_hash_table_insert(a, (gpointer)"application", (gpointer)"ghostftp");
    g_hash_table_insert(a, (gpointer)"service", (gpointer)fixture_service);
    g_hash_table_insert(a, (gpointer)"account", (gpointer)account);
    return a;
}

static SecretService *open_service(void) {
    GError *error = NULL;
    SecretService *s = secret_service_open_sync(SECRET_TYPE_SERVICE, NULL,
        SECRET_SERVICE_OPEN_SESSION, NULL, &error);
    require(s != NULL && error == NULL, "open service");
    return s;
}

static SecretCollection *persistent_collection(SecretService *s) {
    GError *error = NULL;
    SecretCollection *c = secret_collection_for_alias_sync(s, "default",
        SECRET_COLLECTION_NONE, NULL, &error);
    require(c != NULL && error == NULL, "resolve default collection");
    SecretCollection *session = secret_collection_for_alias_sync(s, "session",
        SECRET_COLLECTION_NONE, NULL, &error);
    require(error == NULL, "resolve session alias");
    if (session != NULL) {
        require(strcmp(g_dbus_proxy_get_object_path(G_DBUS_PROXY(c)),
                       g_dbus_proxy_get_object_path(G_DBUS_PROXY(session))) != 0,
                "default must not be session collection");
        g_object_unref(session);
    }
    return c;
}

/* Return 0 for absence, 1 for a unique unlocked persistent item, -1 on error.
 * No automatic unlock, no session fallback, no retained item/secret cache. */
static int find_item(SecretService *s, SecretCollection *c,
                     const char *account, SecretItem **result) {
    *result = NULL;
    if (secret_collection_get_locked(c))
        return -1;
    GHashTable *a = attributes(account);
    GError *error = NULL;
    GList *items = secret_service_search_sync(s, NULL, a,
        SECRET_SEARCH_ALL, NULL, &error);
    g_hash_table_unref(a);
    int state = 0;
    if (error != NULL) {
        g_error_free(error);
        state = -1;
    } else if (items != NULL) {
        SecretItem *item = SECRET_ITEM(items->data);
        const char *collection_path = g_dbus_proxy_get_object_path(G_DBUS_PROXY(c));
        const char *item_path = g_dbus_proxy_get_object_path(G_DBUS_PROXY(item));
        size_t n = strlen(collection_path);
        if (items->next != NULL || secret_item_get_locked(item) ||
            !g_str_has_prefix(item_path, collection_path) || item_path[n] != '/') {
            state = -1;
        } else {
            *result = g_object_ref(item);
            state = 1;
        }
    }
    g_list_free_full(items, g_object_unref);
    return state;
}

static int verify(SecretService *s, SecretCollection *c, const char *account) {
    SecretItem *item = NULL;
    int state = find_item(s, c, account, &item);
    if (state != 1)
        return state;
    GError *error = NULL;
    if (!secret_item_load_secret_sync(item, NULL, &error) || error != NULL) {
        g_clear_error(&error);
        g_object_unref(item);
        return -1;
    }
    SecretValue *value = secret_item_get_secret(item);
    gsize length = 0;
    const gchar *bytes = value ? secret_value_get(value, &length) : NULL;
    gboolean match = bytes != NULL && length == strlen(fixture_value) &&
        memcmp(bytes, fixture_value, length) == 0;
    if (value != NULL)
        secret_value_unref(value);
    g_object_unref(item);
    return match ? 1 : -1;
}

static void store_fixture(SecretService *s, SecretCollection *c) {
    SecretItem *item = NULL;
    require(find_item(s, c, direct_account, &item) == 0, "new item absent");
    GHashTable *a = attributes(direct_account);
    SecretValue *value = secret_value_new(fixture_value, -1, "text/plain");
    GError *error = NULL;
    gboolean stored = secret_service_store_sync(s, NULL, a,
        g_dbus_proxy_get_object_path(G_DBUS_PROXY(c)), "Ghost FTP test fixture",
        value, NULL, &error);
    secret_value_unref(value);
    g_hash_table_unref(a);
    require(stored && error == NULL, "direct write");
    require(verify(s, c, direct_account) == 1, "direct write readback");
}

static void remove_fixture(SecretService *s, SecretCollection *c,
                           const char *account) {
    SecretItem *item = NULL;
    require(find_item(s, c, account, &item) == 1, "delete unique item");
    GError *error = NULL;
    gboolean removed = secret_item_delete_sync(item, NULL, &error);
    g_object_unref(item);
    require(removed && error == NULL, "delete");
    require(find_item(s, c, account, &item) == 0, "delete readback");
}

static void benchmark(SecretService *s, SecretCollection *c) {
    const char *order = "ABBA";
    for (int round = 0; round < 4; round++) {
        for (int sample = 0; sample < 100; sample++) {
            gint64 started = g_get_monotonic_time();
            if (order[round] == 'A') {
                gchar *argv[] = {(gchar *)"secret-tool", (gchar *)"lookup",
                    (gchar *)"--", (gchar *)"application", (gchar *)"ghostftp",
                    (gchar *)"service", (gchar *)fixture_service,
                    (gchar *)"account", (gchar *)legacy_account, NULL};
                gchar *out = NULL;
                gint status = 0;
                GError *error = NULL;
                gboolean ok = g_spawn_sync(NULL, argv, NULL,
                    G_SPAWN_SEARCH_PATH | G_SPAWN_STDERR_TO_DEV_NULL,
                    NULL, NULL, &out, NULL, &status, &error);
                require(ok && error == NULL && status == 0 && out != NULL &&
                        strcmp(out, fixture_value) == 0, "legacy lookup");
                g_free(out);
            } else {
                require(verify(s, c, legacy_account) == 1, "direct lookup");
            }
            printf("lookup_sample variant=%c round=%d sample=%d elapsed_us=%"
                   G_GINT64_FORMAT "\n", order[round], round, sample,
                   g_get_monotonic_time() - started);
        }
    }
}

int main(int argc, char **argv) {
    require(g_strcmp0(g_getenv("GHOSTFTP_VAULT_PROBE_OPT_IN"),
                       "disposable-fixtures") == 0, "explicit fixture opt-in");
    require(argc == 2, "mode");
    SecretService *s = open_service();
    SecretCollection *c = persistent_collection(s);
    if (strcmp(argv[1], "exercise") == 0) {
        require(verify(s, c, legacy_account) == 1, "legacy to direct");
        store_fixture(s, c);
        benchmark(s, c);
    } else if (strcmp(argv[1], "lock") == 0) {
        require(verify(s, c, legacy_account) == 1, "before lock");
        GList *objects = g_list_append(NULL, c);
        GList *locked = NULL;
        GError *error = NULL;
        gint count = secret_service_lock_sync(s, objects, NULL, &locked, &error);
        g_list_free(objects);
        g_list_free_full(locked, g_object_unref);
        require(count > 0 && error == NULL, "lock collection");
        g_object_unref(c);
        g_object_unref(s);
        s = open_service();
        c = persistent_collection(s);
        require(secret_collection_get_locked(c), "collection locked");
        require(verify(s, c, legacy_account) == -1, "locked must not mean absent");
    } else if (strcmp(argv[1], "reopen") == 0) {
        require(verify(s, c, legacy_account) == 1, "legacy persisted");
        require(verify(s, c, direct_account) == 1, "direct persisted");
        remove_fixture(s, c, direct_account);
        remove_fixture(s, c, legacy_account);
    } else {
        require(FALSE, "unknown mode");
    }
    g_object_unref(c);
    g_object_unref(s);
    puts("direct_vault_probe passed");
    return 0;
}
