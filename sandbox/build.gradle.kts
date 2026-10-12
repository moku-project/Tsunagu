plugins {
    kotlin("jvm") version "2.4.21"
    kotlin("plugin.serialization") version "2.4.21"
    application
    id("com.google.protobuf") version "0.10.0"
    // Shadow 9 only merges service files if duplicates are allowed, which then also
    // duplicates the ASM classes d2j-external bundles; 8.x keeps the first copy.
    id("com.gradleup.shadow") version "8.3.11"
}

group = "com.tsunagu"
version = "0.5.0"

repositories {
    mavenCentral()
    google()
    maven("https://jitpack.io")
}

val grpcVersion = "1.84.2"
val protobufVersion = "4.36.2"

dependencies {
    implementation("io.reactivex:rxjava:1.3.8")

    implementation("io.grpc:grpc-netty-shaded:$grpcVersion")
    implementation("io.grpc:grpc-protobuf:$grpcVersion")
    implementation("io.grpc:grpc-stub:$grpcVersion")
    implementation("io.grpc:grpc-services:$grpcVersion")
    implementation("com.google.protobuf:protobuf-java:$protobufVersion")

    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-core:1.11.0")
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-json:1.11.0")
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-protobuf:1.11.0")
    implementation("com.squareup.okio:okio:3.18.2")
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-json-okio:1.11.0")
    implementation("com.ibm.icu:icu4j:78.3")

    implementation("io.github.oshai:kotlin-logging-jvm:8.0.4")
    implementation("org.slf4j:slf4j-simple:2.0.20")

    implementation("com.squareup.okhttp3:okhttp:5.5.0")
    implementation("com.squareup.okhttp3:okhttp-brotli:5.5.0")
    implementation("com.squareup.okhttp3:okhttp-zstd:5.5.0")
    implementation("com.squareup.okhttp3:okhttp-dnsoverhttps:5.5.0")

    implementation("org.conscrypt:conscrypt-openjdk-uber:2.7.0")
    implementation("org.brotli:dec:0.1.2")
    implementation("com.squareup.okhttp3:logging-interceptor:5.5.0")
    implementation("org.jsoup:jsoup:1.23.2")
    implementation("org.json:json:20260814")

    implementation("net.dongliu:apk-parser:2.6.10")
    implementation("com.android.tools.build:apksig:9.4.1")
    implementation("de.femtopedia.dex2jar:dex-translator:2.4.38")
    implementation("de.femtopedia.dex2jar:dex-tools:2.4.38")
    implementation("org.ow2.asm:asm:9.11")

    implementation("io.insert-koin:koin-core:4.2.2")
    implementation("com.github.null2264:injekt-koin:ee267b2e27")

    implementation("org.graalvm.polyglot:polyglot:25.4.4.1.1")
    implementation("org.graalvm.polyglot:js-community:25.4.4.1.1")

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

tasks.shadowJar {
    mergeServiceFiles()
    // Truffle (GraalVM 25+) refuses to start from an uber jar without this.
    manifest { attributes["Multi-Release"] = "true" }
    // GraalVM publishes POM-only aggregator modules (js-community, js, ...); Gradle
    // resolves the .pom itself as an artifact, which Shadow would try to unzip.
    // Their real jars are separate transitive dependencies and still get bundled.
    dependencies {
        exclude { dep -> dep.moduleArtifacts.isNotEmpty() && dep.moduleArtifacts.all { it.extension == "pom" } }
    }
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
        "--no-man-pages"
    )
}
