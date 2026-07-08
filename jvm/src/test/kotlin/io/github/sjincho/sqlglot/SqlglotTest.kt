package io.github.sjincho.sqlglot

import java.util.concurrent.Executors
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicInteger
import kotlin.test.Test
import kotlin.test.assertTrue

class SqlglotTest {
    private val schema = """{"users":{"id":"BIGINT","name":"VARCHAR","rrn":"VARCHAR"},"orders":{"id":"BIGINT","user_id":"BIGINT"}}"""

    @Test
    fun resolvesSimpleLineage() {
        for (dialect in listOf("mysql", "postgres")) {
            val json = Sqlglot.probeJson("SELECT id, rrn FROM users", dialect, schema)
            assertTrue(json.contains("\"resolved\":true"), "[$dialect] expected resolved=true, got: $json")
            assertTrue(json.contains("users.id"), "[$dialect] expected origin users.id, got: $json")
            assertTrue(json.contains("users.rrn"), "[$dialect] expected origin users.rrn, got: $json")
        }
    }

    @Test
    fun classifiesWrite() {
        val json = Sqlglot.probeJson("INSERT INTO users (id, name) VALUES (1, 'x')", "postgres", schema)
        assertTrue(json.contains("\"isWrite\":true"), "expected isWrite=true, got: $json")
    }

    @Test
    fun failsClosedOnGarbage() {
        // Malformed / unknown input must return a valid fail-closed ProbeResult, never an exception.
        for (sql in listOf("this is not sql", "SELECT * FROM nonexistent_table", "SELECT ;; nonsense")) {
            val json = Sqlglot.probeJson(sql, "postgres", schema)
            assertTrue(json.contains("\"resolved\":false"), "expected resolved=false for $sql, got: $json")
        }
    }

    /**
     * Phase-0 de-risk: hammer the native boundary from many threads with a mix of valid and
     * malformed SQL. Proves the Go-in-JVM binding has no signal/crash conflict and every call
     * returns valid JSON under concurrency (the main risk of the in-process FFM approach).
     */
    @Test
    fun concurrentStress() {
        val queries = listOf(
            "SELECT id, rrn FROM users",
            "SELECT u.rrn, o.id FROM users u JOIN orders o ON u.id = o.user_id",
            "WITH c AS (SELECT rrn FROM users) SELECT rrn FROM c",
            "SELECT rrn FROM users UNION ALL SELECT name FROM users",
            "INSERT INTO users (id) VALUES (1)",
            "UPDATE users SET name = 'x' WHERE id = 1",
            "garbage not sql",
            "SELECT * FROM missing",
        )
        val pool = Executors.newFixedThreadPool(16)
        val ok = AtomicInteger()
        val bad = AtomicInteger()
        val total = 4000
        repeat(total) { i ->
            pool.submit {
                val q = queries[i % queries.size]
                val dialect = if (i % 2 == 0) "mysql" else "postgres"
                val json = Sqlglot.probeJson(q, dialect, schema)
                if (json.contains("\"resolved\":true") || json.contains("\"resolved\":false")) ok.incrementAndGet() else bad.incrementAndGet()
            }
        }
        pool.shutdown()
        assertTrue(pool.awaitTermination(60, TimeUnit.SECONDS), "concurrent probes did not finish in time")
        assertTrue(bad.get() == 0, "${bad.get()} calls returned invalid JSON")
        assertTrue(ok.get() == total, "expected $total valid results, got ${ok.get()}")
    }
}
