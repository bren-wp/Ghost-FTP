# Ghost FTP Android currently uses only platform APIs.
# Keep runtime-visible activity and transport classes explicit for release builds.
-keep class com.brendigo.ghostftp.MainActivity { *; }
-keep class com.brendigo.ghostftp.FtpSession { *; }
