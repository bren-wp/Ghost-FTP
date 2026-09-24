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

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    kotlinOptions {
        jvmTarget = "17"
    }
}

dependencies {
    implementation("commons-net:commons-net:3.11.1")
    implementation("com.jcraft:jsch:0.1.55")
}
