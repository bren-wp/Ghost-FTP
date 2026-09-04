plugins {
    id("com.android.application")
}

val canonicalVersion = rootProject.file("../VERSION").readText().trim()
val versionParts = canonicalVersion.split('.').map { it.toInt() }
require(versionParts.size == 3) { "VERSION must be semantic major.minor.patch" }

// Ghost FTP keeps the existing com.GhostFTP.client application identity so installed
// users can upgrade in place. Store version codes therefore need a monotonic epoch
// that remains above the highest published GhostFTP code even though the public product
// version restarts at 1.0.0.
val semanticVersionCode = versionParts[0] * 10000 + versionParts[1] * 100 + versionParts[2]
val ghostFtpVersionCodeEpoch = 1_000_000
val legacyGhostFTPVersionCodeFloor = 10_902
val canonicalVersionCode = ghostFtpVersionCodeEpoch + semanticVersionCode
require(canonicalVersionCode > legacyGhostFTPVersionCodeFloor) {
    "Ghost FTP Android versionCode must stay above the published GhostFTP versionCode floor"
}

android {
    namespace = "com.GhostFTP.client"
    compileSdk = 37

    defaultConfig {
        applicationId = "com.GhostFTP.client"
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
            proguardFiles(getDefaultProguardFile("proguard-android-optimize.txt"), "proguard-rules.pro")
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
        // These implementations live inside third-party JARs. Ghost FTP source is separately
        // audited to reject permissive TrustManager code and explicitly uses platform trust.
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
