# SSHJ discovers cryptographic implementations from its dependency graph.
-dontwarn org.bouncycastle.**
-dontwarn net.i2p.crypto.eddsa.**

# Keep the small protocol boundary readable in crash reports.
-keep class com.GhostFTP.client.remote.** { *; }
