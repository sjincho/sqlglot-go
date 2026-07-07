package parser_test

import (
	"testing"

	exp "github.com/sjincho/sqlglot-go/expressions"
)

func TestParseCreate(t *testing.T) {
	create := parseOne(t, "CREATE TABLE t (a INT, b VARCHAR(10))")
	if create.Kind() != exp.KindCreate || create.Arg("kind") != "TABLE" {
		t.Fatalf("create table mismatch:\n%s", create.ToS())
	}
	schema := exprArg(t, create, "this")
	cols := schema.Expressions()
	if schema.Kind() != exp.KindSchema || len(cols) != 2 {
		t.Fatalf("create schema mismatch:\n%s", create.ToS())
	}
	if cols[0].Kind() != exp.KindColumnDef || !exp.IsType(exprArg(t, cols[0], "kind"), exp.DTypeInt) {
		t.Fatalf("first column type mismatch:\n%s", create.ToS())
	}
	if cols[1].Kind() != exp.KindColumnDef || !exp.IsType(exprArg(t, cols[1], "kind"), exp.DTypeVarchar) {
		t.Fatalf("second column type mismatch:\n%s", create.ToS())
	}

	create = parseOne(t, "CREATE TABLE t AS SELECT 1")
	if exprArg(t, create, "expression").Kind() != exp.KindSelect {
		t.Fatalf("CTAS expression mismatch:\n%s", create.ToS())
	}

	create = parseOne(t, "CREATE OR REPLACE VIEW v AS SELECT a FROM t")
	if create.Kind() != exp.KindCreate || create.Arg("kind") != "VIEW" || create.Arg("replace") != true {
		t.Fatalf("create or replace view mismatch:\n%s", create.ToS())
	}

	create = parseOne(t, "CREATE TABLE IF NOT EXISTS t (a INT)")
	if create.Arg("exists") != true {
		t.Fatalf("IF NOT EXISTS mismatch:\n%s", create.ToS())
	}

	command := parseOne(t, "CREATE TABLE t (a INT) ENGINE=InnoDB")
	if command.Kind() != exp.KindCommand {
		t.Fatalf("property-bearing CREATE should degrade to Command:\n%s", command.ToS())
	}

	// Column constraints are deferred to 1c; a CREATE that carries them must degrade
	// cleanly to a Command rather than fail on a stale "Expecting )" (LOCAL fix).
	for _, sql := range []string{
		"CREATE TABLE t (a INT NOT NULL)",
		"CREATE TABLE t (id INT PRIMARY KEY, name VARCHAR(50) NOT NULL)",
		"CREATE TABLE t (a INT DEFAULT 0)",
	} {
		degraded := parseOne(t, sql)
		if degraded.Kind() != exp.KindCommand || degraded.Arg("this") != "CREATE" {
			t.Fatalf("constrained CREATE should degrade to Command: %q ->\n%s", sql, degraded.ToS())
		}
	}
}
