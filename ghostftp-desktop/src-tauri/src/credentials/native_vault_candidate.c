/* Disposable native Rust test adapter. No production runtime selection. */
#include <libsecret/secret.h>
#include <gio/gio.h>
#include <stdint.h>
#include <string.h>

typedef struct {
    SecretService *service;
    gchar *owner;
    /* Only the isolated failure harness installs this callback. */
    void (*mutation_gate)(int, void *);
    void *gate_data;
} VaultSession;

typedef struct {
    GMutex mutex;
    GCond cond;
    gboolean finished;
    gint64 deadline;
    GCancellable *cancel;
} Deadline;

static gpointer cancel_at_deadline(gpointer data) {
    Deadline *d = data;
    g_mutex_lock(&d->mutex);
    while (!d->finished && g_cond_wait_until(&d->cond, &d->mutex, d->deadline)) {}
    gboolean expired = !d->finished;
    g_mutex_unlock(&d->mutex);
    if (expired)
        g_cancellable_cancel(d->cancel);
    return NULL;
}

static void reset(VaultSession *s) {
    g_clear_object(&s->service);
    g_clear_pointer(&s->owner, g_free);
}

void *ghostftp_vault_session_new(void) {
    return g_new0(VaultSession, 1);
}

void ghostftp_vault_session_free(void *ptr) {
    VaultSession *s = ptr;
    reset(s);
    g_free(s);
}

void ghostftp_vault_set_mutation_gate(void *ptr, void (*gate)(int, void *), void *data) {
    VaultSession *s = ptr;
    s->mutation_gate = gate;
    s->gate_data = data;
}

static gchar *owner_now(SecretService *s, GCancellable *cancel, GError **error) {
    GDBusConnection *connection = g_dbus_proxy_get_connection(G_DBUS_PROXY(s));
    GVariant *reply = g_dbus_connection_call_sync(connection,
        "org.freedesktop.DBus", "/org/freedesktop/DBus",
        "org.freedesktop.DBus", "GetNameOwner",
        g_variant_new("(s)", "org.freedesktop.secrets"),
        G_VARIANT_TYPE("(s)"), G_DBUS_CALL_FLAGS_NONE, 30000, cancel, error);
    if (!reply)
        return NULL;
    gchar *owner = NULL;
    g_variant_get(reply, "(s)", &owner);
    g_variant_unref(reply);
    return owner;
}

static gboolean same_owner(VaultSession *s, GCancellable *cancel, GError **error) {
    gchar *current = owner_now(s->service, cancel, error);
    gboolean same = current && s->owner && strcmp(current, s->owner) == 0;
    g_free(current);
    return same;
}

static gboolean connect_session(VaultSession *s, GCancellable *cancel, GError **error) {
    if (s->service)
        return same_owner(s, cancel, error);
    s->service = secret_service_open_sync(SECRET_TYPE_SERVICE, NULL,
        SECRET_SERVICE_OPEN_SESSION, cancel, error);
    if (!s->service)
        return FALSE;
    s->owner = g_dbus_proxy_get_name_owner(G_DBUS_PROXY(s->service));
    return same_owner(s, cancel, error);
}

static GHashTable *attributes(const char *service, const char *account) {
    GHashTable *a = g_hash_table_new(g_str_hash, g_str_equal);
    g_hash_table_insert(a, (gpointer)"application", (gpointer)"ghostftp");
    g_hash_table_insert(a, (gpointer)"service", (gpointer)service);
    g_hash_table_insert(a, (gpointer)"account", (gpointer)account);
    return a;
}

static int execute(VaultSession *s, int operation, const char *service,
                   const char *account, const char *input, uint8_t *output,
                   size_t capacity, size_t *length, GCancellable *cancel,
                   gboolean *mutation_attempted) {
    GError *error = NULL;
    SecretCollection *collection = NULL, *session = NULL;
    GHashTable *a = NULL;
    GList *items = NULL;
    SecretValue *value = NULL;
    int result = -1;
    if (!connect_session(s, cancel, &error))
        goto done;
    collection = secret_collection_for_alias_sync(s->service, "default",
        SECRET_COLLECTION_NONE, cancel, &error);
    if (!collection || error || secret_collection_get_locked(collection))
        goto done;
    session = secret_collection_for_alias_sync(s->service, "session",
        SECRET_COLLECTION_NONE, cancel, &error);
    if (error)
        goto done;
    const char *path = g_dbus_proxy_get_object_path(G_DBUS_PROXY(collection));
    if (session && strcmp(path, g_dbus_proxy_get_object_path(G_DBUS_PROXY(session))) == 0)
        goto done;
    a = attributes(service, account);
    items = secret_service_search_sync(s->service, NULL, a,
        SECRET_SEARCH_ALL, cancel, &error);
    if (error || (items && items->next))
        goto done;
    SecretItem *item = items ? SECRET_ITEM(items->data) : NULL;
    if (item) {
        const char *item_path = g_dbus_proxy_get_object_path(G_DBUS_PROXY(item));
        if (secret_item_get_locked(item) || !g_str_has_prefix(item_path, path) ||
            item_path[strlen(path)] != '/')
            goto done;
    }
    if (operation == 0) {
        if (!item) {
            result = 0;
        } else if (secret_item_load_secret_sync(item, cancel, &error) && !error) {
            value = secret_item_get_secret(item);
            gsize n = 0;
            const gchar *bytes = value ? secret_value_get(value, &n) : NULL;
            if (bytes && n <= capacity && !memchr(bytes, '\0', n)) {
                memcpy(output, bytes, n);
                *length = n;
                result = 1;
            }
        }
    } else if (operation == 1) {
        value = secret_value_new(input, -1, "text/plain");
        if (s->mutation_gate)
            s->mutation_gate(0, s->gate_data);
        if (g_cancellable_is_cancelled(cancel))
            goto done;
        *mutation_attempted = TRUE;
        gboolean stored = item
            ? secret_item_set_secret_sync(item, value, cancel, &error)
            : secret_service_store_sync(s->service, NULL, a, path,
                "Ghost FTP", value, cancel, &error);
        if (stored && !error)
            result = 0;
    } else if (operation == 2) {
        if (!item) {
            result = 0;
        } else {
            if (s->mutation_gate)
                s->mutation_gate(0, s->gate_data);
            if (g_cancellable_is_cancelled(cancel))
                goto done;
            *mutation_attempted = TRUE;
            if (secret_item_delete_sync(item, cancel, &error) && !error)
                result = 0;
        }
    }
    if (result == 0 && *mutation_attempted && s->mutation_gate)
        s->mutation_gate(1, s->gate_data);
    if (result >= 0 && !same_owner(s, cancel, &error))
        result = -1;
done:
    if (error || g_cancellable_is_cancelled(cancel))
        result = -1;
    g_clear_error(&error);
    if (value)
        secret_value_unref(value);
    g_list_free_full(items, g_object_unref);
    if (a)
        g_hash_table_unref(a);
    g_clear_object(&session);
    g_clear_object(&collection);
    return result;
}

/* The caller owns the buffer; erase any partial output before a failed
 * call returns, including an owner change after a successful secret read.
 * Volatile writes are intentional: ordinary memset may be optimized away. */
static void wipe_output(uint8_t *output, size_t capacity) {
    volatile uint8_t *bytes = output;
    while (capacity > 0) {
        *bytes++ = 0;
        --capacity;
    }
}

int ghostftp_vault_call(void *ptr, int operation, const char *service,
                       const char *account, const char *input, uint8_t *output,
                       size_t capacity, size_t *length, uint32_t timeout_ms) {
    VaultSession *s = ptr;
    Deadline d;
    g_mutex_init(&d.mutex);
    g_cond_init(&d.cond);
    d.finished = FALSE;
    d.deadline = g_get_monotonic_time() + (gint64)timeout_ms * 1000;
    d.cancel = g_cancellable_new();
    *length = 0;
    GThread *timer = g_thread_new("vault-deadline", cancel_at_deadline, &d);
    gboolean mutation_attempted = FALSE;
    int result = execute(s, operation, service, account, input,
                         output, capacity, length, d.cancel, &mutation_attempted);
    g_mutex_lock(&d.mutex);
    d.finished = TRUE;
    g_cond_signal(&d.cond);
    g_mutex_unlock(&d.mutex);
    g_thread_join(timer);
    if (g_cancellable_is_cancelled(d.cancel) || g_get_monotonic_time() >= d.deadline)
        result = -1;
    g_object_unref(d.cancel);
    g_cond_clear(&d.cond);
    g_mutex_clear(&d.mutex);
    if (result < 0) {
        wipe_output(output, capacity);
        reset(s);
        *length = 0;
        /* Once dispatched, cancellation does not establish non-commit. */
        if (mutation_attempted)
            return -2;
    }
    return result;
}
