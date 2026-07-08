# Using sqlglot-go from a JVM project (Kotlin/Java)

sqlglot-go ships an in-process **JVM binding** (in [`jvm/`](../jvm)) that calls the Go SQL
column-lineage probe through the Java **Foreign Function & Memory API** (`java.lang.foreign`). One
call, strings in / JSON out:

```kotlin
val json: String = io.github.sjincho.sqlglot.Sqlglot.probeJson(sql, dialect, schemaJson)
```

- `sql` — a single statement
- `dialect` — `"mysql"` or `"postgres"`
- `schemaJson` — the catalog as `{"table":{"column":"TYPE"}}`
- returns the **ProbeResult** contract as JSON: `{resolved, failedStage, origins, references, isWrite,
  outputColumns, tracedColumns, rewrittenSql, detail}`. Malformed/unknown input returns a valid
  **fail-closed** result (`"resolved":false`), never an exception — the safe direction for a probe.

## Requirements

- **JDK 22+** to consume (FFM is stable since 22). Run the app with
  `--enable-native-access=ALL-UNNAMED` to silence the restricted-method warning.
- **To build the native library** (see below), the build machine needs the **Go toolchain (1.23+)**
  and a **C compiler** (cgo). The Gradle build runs `go build -buildmode=c-shared` for you; you do
  not write or run Go yourself.

## Recommended: vendor via `git subtree` + Gradle composite build

This pulls sqlglot-go (Go library + JVM binding) into your repo and builds the binding from source —
no Maven publishing, no committed binaries.

**1. Add sqlglot-go as a subtree** (run in your consumer repo root):

```bash
git subtree add --prefix third_party/sqlglot-go \
  https://github.com/sjincho/sqlglot-go.git main --squash
```

To update later:

```bash
git subtree pull --prefix third_party/sqlglot-go \
  https://github.com/sjincho/sqlglot-go.git main --squash
```

**2. Include the binding's Gradle build** in your `settings.gradle.kts`:

```kotlin
includeBuild("third_party/sqlglot-go/jvm")
```

**3. Depend on it** in the module that needs the probe (`build.gradle.kts`):

```kotlin
dependencies {
    implementation("io.github.sjincho:sqlglot-go-jvm")
}
```

Gradle resolves that coordinate to the included build, runs `buildNativeLib` (compiles the Go
c-shared lib for your host platform and bundles it into the jar), and puts the binding on your
classpath. The first build compiles the native lib; subsequent builds are cached.

**4. Enable native access** for your run/test tasks:

```kotlin
tasks.withType<JavaExec>().configureEach { jvmArgs("--enable-native-access=ALL-UNNAMED") }
tasks.withType<Test>().configureEach { jvmArgs("--enable-native-access=ALL-UNNAMED") }
```

## Example (Kotlin)

```kotlin
import io.github.sjincho.sqlglot.Sqlglot

val schema = """{"users":{"id":"BIGINT","rrn":"VARCHAR"}}"""
val json = Sqlglot.probeJson("SELECT id, rrn FROM users WHERE rrn = 'x'", "postgres", schema)
// {"resolved":true,"isWrite":false,"origins":[{"column":"id","origins":["users.id"]},
//  {"column":"rrn","origins":["users.rrn"]}],"references":{"PREDICATE":["users.rrn"]},...}
```

Deserialize `json` with whatever you already use (`kotlinx.serialization`, Jackson, …). The contract
is identical to the Python `probe.py` output, verified at 94/94 parity — so an existing consumer that
already parses that JSON works unchanged; only the call site changes.

## Notes & alternatives

- **Thread-safety:** `Sqlglot.probeJson` is safe to call concurrently (the native side is a pure
  function; each call uses its own confined `Arena`). No pool needed. Verified under a 16-thread /
  4000-call stress test.
- **Platforms:** `buildNativeLib` builds for the **host** platform. If your build and deploy
  platforms differ (e.g. build on macOS, deploy on Linux), build on the target platform, or extend
  the binding to bundle multiple prebuilt libs (`native/<os>-<arch>/`) via a CI matrix and publish a
  fat jar to a Maven repo — then depend on that coordinate instead of the composite build.
- **Publishing to Maven** (Central / internal Nexus) instead of subtree: the `jvm/` project already
  applies `maven-publish`; set real `group`/`version`, run a CI matrix to bundle all target
  `native/<os>-<arch>/` libs, and `./gradlew publish`. Consumers then just add the coordinate.
- **Fail-closed contract:** any internal error/panic on the Go side becomes `{"resolved":false,
  "failedStage":"LINEAGE",...}`. Treat a non-resolved result as DENY in a security context.
