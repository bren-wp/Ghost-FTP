plugins {
    id("com.android.application")
}

val canonicalVersion = rootProject.file("../VERSION").readText().trim()
val versionParts = canonicalVersion.split('.').map { it.toInt() }
require(versionParts.size == 3) { "VERSION must be semantic major.minor.patch" }
val canonicalVersionCode = versionParts[0] * 10000 + versionParts[1] * 100 + versionParts[2]

android {
    namespace = "com.byftp.client"
    compileSdk = 37

    defaultConfig {
        applicationId = "com.byftp.client"
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
        disable += setOf("OldTargetApi")
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
