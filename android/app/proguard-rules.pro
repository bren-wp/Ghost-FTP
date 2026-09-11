# JSch is invoked through its public API; retain names used by algorithm lookup.
-keep class com.jcraft.jsch.** { *; }
-dontwarn org.bouncycastle.**
