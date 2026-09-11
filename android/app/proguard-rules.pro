# Ghost FTP Android currently uses only platform APIs.
# Keep runtime-visible activity and transport classes explicit for release builds.
-keep class app.ghostftp.client.MainActivity { *; }
-keep class app.ghostftp.client.FtpSession { *; }
