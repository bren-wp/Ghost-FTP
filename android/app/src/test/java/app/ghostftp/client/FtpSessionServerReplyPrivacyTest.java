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

public final class FtpSessionServerReplyPrivacyTest {
    @Test
    public void rejectedPostAuthenticationReplyDoesNotExposeServerText() throws Exception {
        String secret = "ghostftp-post-auth-secret-1947";
        assertPwdFailureRedacted(
                secret,
                "FTP server rejected operation (server response code 550).",
                "550 Server echoed " + secret);
    }

    @Test
    public void malformedPostAuthenticationReplyDoesNotExposeServerText() throws Exception {
        String secret = "ghostftp-malformed-post-auth-secret-2858";
        assertPwdFailureRedacted(
                secret,
                "Invalid FTP response.",
                "not-a-valid-reply " + secret);
    }

    private static void assertPwdFailureRedacted(
            String secret,
            String expectedMessage,
            String pwdReply) throws Exception {
        ExecutorService serverExecutor = Executors.newSingleThreadExecutor();

        try (ServerSocket server = new ServerSocket(0)) {
            Future<?> serverRun = serverExecutor.submit(() -> {
                try (Socket client = server.accept();
                     BufferedReader reader = new BufferedReader(new InputStreamReader(client.getInputStream(), StandardCharsets.US_ASCII));
                     BufferedWriter writer = new BufferedWriter(new OutputStreamWriter(client.getOutputStream(), StandardCharsets.US_ASCII))) {
                    send(writer, "220 Test FTP ready");
                    assertEquals("USER test-user", reader.readLine());
                    send(writer, "331 Password required");
                    assertEquals("PASS synthetic-password", reader.readLine());
                    send(writer, "230 Logged in");
                    assertEquals("TYPE I", reader.readLine());
                    send(writer, "200 Binary mode enabled");
                    assertEquals("PWD", reader.readLine());
                    send(writer, pwdReply);
                }
                return null;
            });

            FtpSession session = new FtpSession("127.0.0.1", server.getLocalPort(), false);
            try {
                session.connect("test-user", "synthetic-password");
                IOException failure = assertThrows(IOException.class, session::pwd);
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
