package io.github.sjincho.sqlglot

import java.lang.foreign.Arena
import java.lang.foreign.FunctionDescriptor
import java.lang.foreign.Linker
import java.lang.foreign.MemorySegment
import java.lang.foreign.SymbolLookup
import java.lang.foreign.ValueLayout
import java.lang.invoke.MethodHandle
import java.nio.file.Files
import java.nio.file.Path
import java.nio.file.StandardCopyOption

/**
 * In-process JVM binding to sqlglot-go's SQL column-lineage probe, via the Foreign Function &
 * Memory API (java.lang.foreign). The native library (built from `cmd/libsqlglot` by the
 * `buildNativeLib` Gradle task and bundled on the classpath) is loaded once for the JVM's lifetime.
 *
 * Thread-safe: the Go `ProbeJSON` is a pure function of its inputs, and each call here uses its own
 * confined [Arena]. Fail-closed: the Go side (`probe.ProbeJSONSafe`) never panics and always returns
 * a valid ProbeResult JSON, so a malformed query yields `{"resolved":false,...}` rather than an error.
 *
 * Requires JDK 22+ (FFM stable since 22). Run with `--enable-native-access=ALL-UNNAMED` to silence
 * the restricted-method warning.
 */
object Sqlglot {
    private val linker = Linker.nativeLinker()
    private val probeJsonHandle: MethodHandle
    private val freeHandle: MethodHandle

    init {
        val libPath = extractNativeLib()
        val lookup = SymbolLookup.libraryLookup(libPath, Arena.global())
        probeJsonHandle = linker.downcallHandle(
            lookup.find("ProbeJSON").orElseThrow { IllegalStateException("ProbeJSON not exported by native lib") },
            FunctionDescriptor.of(ValueLayout.ADDRESS, ValueLayout.ADDRESS, ValueLayout.ADDRESS, ValueLayout.ADDRESS),
        )
        freeHandle = linker.downcallHandle(
            lookup.find("FreeCString").orElseThrow { IllegalStateException("FreeCString not exported by native lib") },
            FunctionDescriptor.ofVoid(ValueLayout.ADDRESS),
        )
    }

    /**
     * Analyze one SQL statement and return the ProbeResult contract as a JSON string.
     *
     * @param sql        a single SQL statement
     * @param dialect    "mysql" or "postgres"
     * @param schemaJson the catalog as `{"table":{"column":"TYPE"}}`
     */
    fun probeJson(sql: String, dialect: String, schemaJson: String): String {
        Arena.ofConfined().use { arena ->
            val sqlSeg = arena.allocateFrom(sql)
            val dialectSeg = arena.allocateFrom(dialect)
            val schemaSeg = arena.allocateFrom(schemaJson)
            val resultPtr = probeJsonHandle.invoke(sqlSeg, dialectSeg, schemaSeg) as MemorySegment
            try {
                // The returned char* comes back as a zero-length native segment; reinterpret it so
                // the null-terminated UTF-8 string can be read out.
                return resultPtr.reinterpret(Long.MAX_VALUE).getString(0)
            } finally {
                freeHandle.invoke(resultPtr) // release the Go-malloc'd C string
            }
        }
    }

    private fun extractNativeLib(): Path {
        val (os, arch, ext) = platform()
        val resource = "/native/$os-$arch/libsqlglot.$ext"
        val stream = Sqlglot::class.java.getResourceAsStream(resource)
            ?: throw IllegalStateException(
                "native library not bundled: $resource — build it with the buildNativeLib Gradle task " +
                    "(needs the Go toolchain + a C compiler)",
            )
        val tmp = Files.createTempFile("libsqlglot", ".$ext")
        tmp.toFile().deleteOnExit()
        stream.use { Files.copy(it, tmp, StandardCopyOption.REPLACE_EXISTING) }
        return tmp
    }

    private data class Platform(val os: String, val arch: String, val ext: String)

    private fun platform(): Platform {
        val osName = System.getProperty("os.name").lowercase()
        val os = when {
            osName.contains("mac") || osName.contains("darwin") -> "darwin"
            osName.contains("linux") -> "linux"
            else -> throw IllegalStateException("unsupported OS for the sqlglot-go native binding: $osName")
        }
        val arch = when (val a = System.getProperty("os.arch").lowercase()) {
            "aarch64", "arm64" -> "arm64"
            "x86_64", "amd64" -> "amd64"
            else -> throw IllegalStateException("unsupported CPU arch for the sqlglot-go native binding: $a")
        }
        return Platform(os, arch, if (os == "darwin") "dylib" else "so")
    }
}
