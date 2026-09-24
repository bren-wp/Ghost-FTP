plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.android")
}

android {
    namespace = "com.ghostftp.android"
    compileSdk = 35

    defaultConfig {
        applicationId = "com.ghostftp.android"
        minSdk = 26
        targetSdk = 35
        versionCode = 211021
        versionName = "2.1.1-rc.21"
    }

    buildTypes {
        release {
            isMinifyEnabled = true
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro"
            )
        }
    }

    kotlinOptions {
        jvmTarget = "17"
    }
}
