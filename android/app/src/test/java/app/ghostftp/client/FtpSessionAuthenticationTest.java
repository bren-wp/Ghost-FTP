package app.ghostftp.client;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertFalse;
import static org.junit.Assert.assertThrows;

import java.io.BufferedReader;
import java.io.BufferedWriter;
import java.io.IOException;
import java.io.InputStreamReader;
import java.io.OutputStreamWriter;
import java.net.ServerSocket;
import java.net.Socket;
import java.nio.charset.StandardCharsets;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.Future;
import java.util.concurrent.TimeUnit;

import org.junit.Test;

public final class FtpSessionAuthenticationTest {
    @Test
    public void rejectedPasswordReplyDoesNotLeakSecret() throws Exception {
        String secret = "ghostftp-regression-secret-7419";
        assertPasswordFailureRedacted(
                secret,
                "FTP authentication failed (server response code 530).",
                "530-Authentication rejected for " + secret,
                "530 Do not expose " + secret);
    }

    @Test
    public void malformedPasswordReplyDoesNotLeakSecret() throws Exception {
        String secret = "ghostftp-malformed-secret-8526";
        assertPasswordFailureRedacted(
                secret,
                "FTP authentication failed before a valid server response was received.",
                "not-a-valid-ftp-reply " + secret);
    }

    private static void assertPasswordFailureRedacted(
            String secret,
            String expectedMessage,
            String... passwordReplies) throws Exception {
        ExecutorService serverExecutor = Executors.newSingleThreadExecutor();

        try (ServerSocket server = new ServerSocket(0)) {
            Future<?> serverRun = serverExecutor.submit(() -> {
                try (Socket client = server.accept();
                     BufferedReader reader = new BufferedReader(new InputStreamReader(client.getInputStream(), StandardCharsets.US_ASCII));
                     BufferedWriter writer = new BufferedWriter(new OutputStreamWriter(client.getOutputStream(), StandardCharsets.US_ASCII))) {
                    send(writer, "220 Test FTP ready");
                    assertEquals("USER test-user", reader.readLine());
                    send(writer, "331 Password required");
                    assertEquals("PASS " + secret, reader.readLine());
                    for (String reply : passwordReplies) {
                        send(writer, reply);
                    }
                }
                return null;
            });

            FtpSession session = new FtpSession("127.0.0.1", server.getLocalPort(), false);
            try {
                IOException failure = assertThrows(IOException.class, () -> session.connect("test-user", secret));
                assertEquals(expectedMessage, failure.getMessage());
                assertFalse(failure.getMessage().contains(secret));
                serverRun.get(5, TimeUnit.SECONDS);
            } finally {
                session.abort();
            }
        } finally {
            serverExecutor.shutdownNow();
        }
    }

    private static void send(BufferedWriter writer, String line) throws IOException {
        writer.write(line);
        writer.write("\r\n");
        writer.flush();
    }
}
