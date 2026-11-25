/* Isolated acceptance harness for the exact Rust candidate C bridge.
 * Only synthetic credentials; never run on a normal desktop keyring. */
#include <gio/gio.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

extern void *ghostftp_vault_session_new(void);
extern void ghostftp_vault_session_free(void *);
extern int ghostftp_vault_call(void *, int, const char *, const char *,
    const char *, uint8_t *, size_t, size_t *, uint32_t);

static const char *service = "com.ghostftp.native-vault.failure.fixture";
static const char *value = "disposable-credential";
static char *account;
static void require(gboolean ok, const char *stage) {
    if (!ok) {
        fprintf(stderr, "failure_probe failed: %s\n", stage);
        exit(1);
    }
}
static void gate(const char *root, const char *name) {
    char *ready = g_strdup_printf("%s/%s.ready", root, name);
    char *go = g_strdup_printf("%s/%s.go", root, name);
    require(g_file_set_contents(ready, "", 0, NULL), "ready");
    gint64 limit = g_get_monotonic_time() + 20000000;
    while (!g_file_test(go, G_FILE_TEST_EXISTS) && g_get_monotonic_time() < limit)
        g_usleep(10000);
    require(g_file_test(go, G_FILE_TEST_EXISTS), "gate timeout");
    g_free(ready);
    g_free(go);
}
static int invoke(void *s, int op, const char *input, unsigned ms, gboolean verify) {
    uint8_t output[8191] = {0};
    size_t length = 0;
    int result = ghostftp_vault_call(s, op, service, account, input,
        output, sizeof(output), &length, ms);
    if (verify)
        require(result == 1 && length == strlen(value) &&
            memcmp(output, value, length) == 0, "credential readback");
    if (result < 0)
        require(length == 0, "failed call must not publish length");
    memset(output, 0, sizeof(output));
    return result;
}
int main(int argc, char **argv) {
    require(g_strcmp0(g_getenv("GHOSTFTP_VAULT_PROBE_OPT_IN"),
                       "disposable-fixtures") == 0, "fixture opt-in");
    require(argc == 3, "mode and scratch directory");
    account = g_uuid_string_random();
    void *s = ghostftp_vault_session_new();
    require(s != NULL, "session");
    require(invoke(s, 0, "", 3000, FALSE) == 0, "unique absence");
    require(invoke(s, 1, value, 3000, FALSE) == 0, "store");
    invoke(s, 0, "", 3000, TRUE);
    if (strcmp(argv[1], "gnome-faults") == 0) {
        gate(argv[2], "block");
        gint64 start = g_get_monotonic_time();
        require(invoke(s, 0, "", 200, FALSE) < 0, "blocked lookup fails");
        gint64 elapsed = g_get_monotonic_time() - start;
        require(elapsed >= 100000 && elapsed < 2000000, "cancellation bound");
        printf("blocked_lookup elapsed_us=%" G_GINT64_FORMAT "\n", elapsed);
        fflush(stdout);
        gate(argv[2], "resume");
        invoke(s, 0, "", 3000, TRUE);
        gate(argv[2], "interrupt");
        require(invoke(s, 0, "", 5000, FALSE) < 0, "interrupted lookup fails");
        puts("interrupted_lookup failed closed");
        fflush(stdout);
        gate(argv[2], "restart");
        invoke(s, 0, "", 3000, TRUE);
        puts("new service session retained fixture");
    } else {
        require(strcmp(argv[1], "provider-crud") == 0, "unknown mode");
    }
    require(invoke(s, 2, "", 3000, FALSE) == 0, "delete");
    require(invoke(s, 0, "", 3000, FALSE) == 0, "delete verified");
    ghostftp_vault_session_free(s);
    g_free(account);
    puts("native_failure_probe passed");
    return 0;
}
