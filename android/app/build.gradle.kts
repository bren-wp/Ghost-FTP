plugins {
    id("com.android.application")
}

val canonicalVersion = rootProject.file("../VERSION").readText().trim()
val versionParts = canonicalVersion.split('.').map { it.toInt() }
require(versionParts.size == 3) { "VERSION must be semantic major.minor.patch" }
require(versionParts.all { it in 0..999 }) { "VERSION components must be between 0 and 999" }

// Android requires an integer versionCode. Ghost FTP uses a deterministic encoding
// of the canonical semantic version because the GhostFTP application ID is new.
val canonicalVersionCode =
    versionParts[0] * 1_000_000 + versionParts[1] * 1_000 + versionParts[2]
require(canonicalVersionCode > 0) { "Ghost FTP Android versionCode must be positive" }

android {
    namespace = "com.ghostftp.client"
    compileSdk = 37

    defaultConfig {
        applicationId = "com.ghostftp.client"
        minSdk = 26
        targetSdk = 37
        versionCode = canonicalVersionCode
        versionName = canonicalVersion
    }

    buildTypes {
        debug {
            applicationIdSuffix = ".debug"
            versionNameSuffix = "-debug"
        }
        release {
            isMinifyEnabled = true
            isShrinkResources = true
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro",
            )
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    lint {
        abortOnError = true
        warningsAsErrors = true
        checkReleaseBuilds = true
        // Third-party JARs may contain compatibility implementations that trigger
        // these library-level checks. Ghost FTP source is audited separately and
        // uses the platform trust implementation.
        disable += setOf("OldTargetApi", "TrustAllX509TrustManager")
    }

    packaging {
        resources {
            excludes += setOf("META-INF/DEPENDENCIES", "META-INF/LICENSE*", "META-INF/NOTICE*")
        }
    }
}

dependencies {
    implementation("commons-net:commons-net:3.13.0")
    implementation("com.hierynomus:sshj:0.40.0")
    testImplementation("junit:junit:4.13.2")
}
