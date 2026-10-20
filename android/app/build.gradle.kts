plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.android")
}

// CI builds stay unsigned unless all four private upload-key inputs are
// explicitly provided by the restricted manual signing workflow. A partial
// configuration must fail, never silently fall back to a debug certificate.
val playSigningEnv = listOf(
    "GHOSTFTP_PLAY_KEYSTORE_FILE",
    "GHOSTFTP_PLAY_STORE_PASSWORD",
    "GHOSTFTP_PLAY_KEY_ALIAS",
    "GHOSTFTP_PLAY_KEY_PASSWORD"
)
val suppliedPlaySigning = playSigningEnv.count { !System.getenv(it).isNullOrBlank() }
check(suppliedPlaySigning == 0 || suppliedPlaySigning == playSigningEnv.size) {
    "Incomplete Ghost FTP Play signing configuration; refusing release signing."
}
val playSigningEnabled = suppliedPlaySigning == playSigningEnv.size

android {
    namespace = "com.ghostftp.android"
    compileSdk = 36

    defaultConfig {
        applicationId = "com.ghostftp.android"
        minSdk = 26
        targetSdk = 36
        versionCode = 955000
        versionName = "0.95.50"
        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"
    }

    signingConfigs {
        if (playSigningEnabled) {
            create("playUpload") {
                val signingFile = file(System.getenv("GHOSTFTP_PLAY_KEYSTORE_FILE"))
                check(signingFile.isFile && signingFile.length() > 0L) {
                    "Play upload keystore is missing or empty."
                }
                storeFile = signingFile
                storePassword = System.getenv("GHOSTFTP_PLAY_STORE_PASSWORD")
                keyAlias = System.getenv("GHOSTFTP_PLAY_KEY_ALIAS")
                keyPassword = System.getenv("GHOSTFTP_PLAY_KEY_PASSWORD")
            }
        }
    }

    buildTypes {
        release {
            if (playSigningEnabled) {
                signingConfig = signingConfigs.getByName("playUpload")
            }
            isMinifyEnabled = true
            isShrinkResources = true
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro"
            )
        }

        create("preview") {
            initWith(getByName("release"))
            applicationIdSuffix = ".preview"
            versionNameSuffix = "-preview"
            signingConfig = signingConfigs.getByName("debug")
            isDebuggable = false
            matchingFallbacks += listOf("release")
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    kotlinOptions {
        jvmTarget = "17"
    }
}

dependencies {
    implementation("commons-net:commons-net:3.13.0")
    implementation("com.github.mwiede:jsch:2.28.7")
    implementation("com.squareup.okhttp3:okhttp:4.12.0")

    testImplementation("junit:junit:4.13.2")

    androidTestImplementation("androidx.test:core:1.7.0")
    androidTestImplementation("androidx.test:runner:1.7.0")
    androidTestImplementation("androidx.test.ext:junit:1.3.0")
    androidTestImplementation("androidx.test.espresso:espresso-core:3.7.0")
}
