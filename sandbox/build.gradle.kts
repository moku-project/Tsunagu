plugins {
    kotlin("jvm") version "2.4.10"
    kotlin("plugin.serialization") version "2.4.10"
    application
    id("com.google.protobuf") version "0.9.4"
    id("com.gradleup.shadow") version "8.3.5"
}

group = "com.tsunagu"
version = "0.1.0"

repositories {
    mavenCentral()
    google()
    maven("https://jitpack.io")
}

val grpcVersion = "1.65.0"
val protobufVersion = "3.25.3"

dependencies {
    implementation("io.reactivex:rxjava:1.3.8")

    // gRPC
    implementation("io.grpc:grpc-netty-shaded:$grpcVersion")
    implementation("io.grpc:grpc-protobuf:$grpcVersion")
    implementation("io.grpc:grpc-stub:$grpcVersion")
    implementation("io.grpc:grpc-services:$grpcVersion")
    implementation("com.google.protobuf:protobuf-java:$protobufVersion")

    // Kotlin
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-core:1.9.0")
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-json:1.11.0")
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-protobuf:1.11.0")
    implementation("com.squareup.okio:okio:3.9.0")
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-json-okio:1.11.0")
    implementation("com.ibm.icu:icu4j:75.1")

    // Logging
    implementation("io.github.oshai:kotlin-logging-jvm:7.0.3")
    implementation("org.slf4j:slf4j-simple:2.0.16")

    // Networking
    implementation("com.squareup.okhttp3:okhttp:5.5.0")
    implementation("com.squareup.okhttp3:okhttp-brotli:5.5.0")
    implementation("com.squareup.okhttp3:okhttp-zstd:5.5.0")
    implementation("org.brotli:dec:0.1.2")
    implementation("com.squareup.okhttp3:logging-interceptor:5.4.0")
    implementation("org.jsoup:jsoup:1.17.2")

    // APK parsing and dex conversion
    implementation("net.dongliu:apk-parser:2.6.10")
    implementation("com.android.tools.build:apksig:9.3.1")
    implementation("de.femtopedia.dex2jar:dex-translator:2.4.38")
    implementation("de.femtopedia.dex2jar:dex-tools:2.4.38")
    implementation("org.ow2.asm:asm:9.9.1")

    // DI
    implementation("io.insert-koin:koin-core:4.2.2")
    implementation("com.github.null2264:injekt-koin:ee267b2e27")

    // GraalVM JS (novel plugin runtime)
    implementation("org.graalvm.polyglot:polyglot:24.1.1")
    implementation("org.graalvm.polyglot:js-community:24.1.1")

    compileOnly("javax.annotation:javax.annotation-api:1.3.2")

    testImplementation(kotlin("test"))
}

application {
    mainClass.set("tsunagu.MainKt")
    applicationDefaultJvmArgs = listOf(
        "-Dpolyglot.engine.WarnInterpreterOnly=false",
    )
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
tasks.register<Exec>("jlinkRuntime") {
    group = "distribution"
    description = "Builds a trimmed custom JRE via jlink using the shadow jar's module deps"
    dependsOn("shadowJar")

    val outputDir = layout.buildDirectory.dir("runtime").get().asFile
    val modulesFile = file("build-config/jlink-modules.txt")

    doFirst {
        if (outputDir.exists()) {
            outputDir.deleteRecursively()
        }
    }

    val modules = modulesFile.readText().trim()
    val javaHome = System.getProperty("java.home")

    commandLine(
        "$javaHome/bin/jlink",
        "--module-path", "$javaHome/jmods",
        "--add-modules", modules,
        "--output", outputDir.absolutePath,
        "--strip-debug",
        "--no-header-files",
        "--no-man-pages",
        "--compress=2"
    )
}
