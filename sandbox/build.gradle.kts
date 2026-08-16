plugins {
    kotlin("jvm") version "2.4.10"
    application
    id("com.google.protobuf") version "0.9.4"
}

group = "com.tsunagu"
version = "0.1.0"

repositories {
    mavenCentral()
    google()
    maven("https://jitpack.io")
    maven("https://github.com/Suwayomi/Suwayomi-Server/raw/android-jar/")
}

val grpcVersion = "1.65.0"
val protobufVersion = "3.25.3"

dependencies {
    implementation("io.grpc:grpc-netty-shaded:$grpcVersion")
    implementation("io.grpc:grpc-protobuf:$grpcVersion")
    implementation("io.grpc:grpc-stub:$grpcVersion")
    implementation("io.grpc:grpc-services:$grpcVersion")
    implementation("com.google.protobuf:protobuf-java:$protobufVersion")

    implementation(files("${System.getenv("ANDROID_HOME") ?: (System.getProperty("user.home") + "/android-sdk")}/platforms/android-34/android.jar"))

    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-core:1.8.1")
    implementation("io.github.oshai:kotlin-logging-jvm:7.0.3")
    implementation("org.slf4j:slf4j-simple:2.0.16")
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-json:1.11.0")
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-json-okio:1.11.0")
    implementation("com.squareup.okio:okio:3.9.0")

    compileOnly("javax.annotation:javax.annotation-api:1.3.2")

    implementation("com.squareup.okhttp3:okhttp:5.4.0")
    implementation("org.jsoup:jsoup:1.17.2")

    implementation("net.dongliu:apk-parser:2.6.10")
    implementation("com.android.tools.build:apksig:9.3.1")
    implementation("de.femtopedia.dex2jar:dex-translator:2.4.38")
    implementation("de.femtopedia.dex2jar:dex-tools:2.4.38")
    implementation("org.ow2.asm:asm:9.9.1")

    implementation("io.reactivex:rxjava:1.3.8")
    implementation("io.insert-koin:koin-core:4.2.2")
    implementation("com.github.null2264:injekt-koin:ee267b2e27")

    implementation("com.ibm.icu:icu4j:78.3")
    implementation("com.twelvemonkeys.imageio:imageio-webp:3.14.0")
    implementation("com.twelvemonkeys.imageio:imageio-jpeg:3.14.0")
    implementation("com.twelvemonkeys.imageio:imageio-core:3.14.0")

    testImplementation(kotlin("test"))
}

application {
    mainClass.set("tsunagu.MainKt")
    applicationDefaultJvmArgs = listOf("-noverify")
}

tasks.test {
    useJUnitPlatform()
}

kotlin {
    jvmToolchain(21)
}

protobuf {
    protoc {
        artifact = "com.google.protobuf:protoc:$protobufVersion"
    }
    plugins {
        create("grpc") {
            artifact = "io.grpc:protoc-gen-grpc-java:$grpcVersion"
        }
    }
    generateProtoTasks {
        all().forEach {
            it.plugins {
                create("grpc")
            }
        }
    }
}

sourceSets {
    main {
        proto {
            srcDir("../proto")
        }
    }
}