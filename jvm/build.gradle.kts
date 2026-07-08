import org.gradle.api.tasks.Exec
import org.gradle.language.jvm.tasks.ProcessResources

plugins {
    kotlin("jvm") version "2.2.0"
    `java-library`
    `maven-publish`
}

// The JVM binding for sqlglot-go. Group/version are placeholders — a consumer using git subtree +
// includeBuild() ignores them; set real coordinates before publishing to a Maven repository.
group = "io.github.sjincho"
version = "0.1.0-SNAPSHOT"

repositories { mavenCentral() }

// FFM (java.lang.foreign) is stable since JDK 22; proxy-monster runs JDK 24.
kotlin { jvmToolchain(24) }

dependencies {
    testImplementation(kotlin("test"))
    testImplementation("org.junit.jupiter:junit-jupiter:5.10.2")
    testRuntimeOnly("org.junit.platform:junit-platform-launcher")
}

// ---- Native library: build cmd/libsqlglot (Go, c-shared) and bundle it as a resource ----
// The Go module root is the parent of this Gradle project (jvm/ is its own build).
private val goModuleDir = rootDir.parentFile

private fun nativeTriple(): Triple<String, String, String> {
    val osName = System.getProperty("os.name").lowercase()
    val os = when {
        osName.contains("mac") || osName.contains("darwin") -> "darwin"
        osName.contains("linux") -> "linux"
        else -> error("unsupported OS: $osName")
    }
    val arch = when (val a = System.getProperty("os.arch").lowercase()) {
        "aarch64", "arm64" -> "arm64"
        "x86_64", "amd64" -> "amd64"
        else -> error("unsupported arch: $a")
    }
    return Triple(os, arch, if (os == "darwin") "dylib" else "so")
}

private val nativeResourcesDir = layout.buildDirectory.dir("native-resources")

val buildNativeLib by tasks.registering(Exec::class) {
    group = "build"
    description = "Builds cmd/libsqlglot as a c-shared library for the host platform and stages it as a resource."
    val (os, arch, ext) = nativeTriple()
    val outFile = nativeResourcesDir.get().dir("native/$os-$arch").file("libsqlglot.$ext").asFile
    workingDir = goModuleDir
    commandLine("go", "build", "-buildmode=c-shared", "-o", outFile.absolutePath, "./cmd/libsqlglot")
    doFirst { outFile.parentFile.mkdirs() }
    // Rebuild when any Go source or go.mod changes. Restrict to *.go (+ go.mod/sum) and exclude the
    // Gradle build dir / reference / VCS so this task's input never overlaps another task's output.
    inputs.files(
        fileTree(goModuleDir) {
            include("**/*.go", "go.mod", "go.sum")
            exclude("jvm/**", ".reference/**", ".git/**")
        },
    ).withPathSensitivity(PathSensitivity.RELATIVE)
    outputs.file(outFile)
}

sourceSets.named("main") { resources.srcDir(nativeResourcesDir) }
tasks.named<ProcessResources>("processResources") {
    dependsOn(buildNativeLib)
    exclude("**/*.h") // the cgo header is emitted next to the lib; not needed at runtime
}

tasks.test {
    useJUnitPlatform()
    // FFM restricted methods (libraryLookup / reinterpret) — grant native access.
    jvmArgs("--enable-native-access=ALL-UNNAMED")
}

publishing {
    publications { create<MavenPublication>("maven") { from(components["java"]) } }
}
