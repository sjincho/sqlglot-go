package parser_test

import (
	"testing"

	exp "github.com/sjincho/sqlglot-go/expressions"
)

func TestParsePivotLateralUnnestValues(t *testing.T) {
	root := parseOne(t, "SELECT * FROM t PIVOT (SUM(x) FOR y IN (1, 2))")
	if pivots := root.FindAll(exp.KindPivot); len(pivots) != 1 {
		t.Fatalf("pivot count = %d, want 1:\n%s", len(pivots), root.ToS())
	}

	root = parseOne(t, "SELECT * FROM t, LATERAL (SELECT 1)")
	lateral := root.Find(exp.KindLateral)
	if lateral == nil {
		t.Fatalf("missing lateral:\n%s", root.ToS())
	}
	inner := lateral.Find(exp.KindSelect)
	if inner == nil || inner.FindAncestor(exp.KindLateral) == nil {
		t.Fatalf("nested lateral ancestor mismatch:\n%s", root.ToS())
	}

	root = parseOne(t, "SELECT * FROM UNNEST(x)")
	if from := exprArg(t, root, "from_"); from.This() == nil || from.This().Kind() != exp.KindUnnest {
		t.Fatalf("unnest FROM mismatch:\n%s", root.ToS())
	}

	root = parseOne(t, "SELECT * FROM (VALUES (1), (2)) AS t(x)")
	if from := exprArg(t, root, "from_"); from.This() == nil || from.This().Kind() != exp.KindValues {
		t.Fatalf("VALUES FROM mismatch:\n%s", root.ToS())
	}
}

// CROSS/OUTER APPLY (SQL Server) route through parseJoin into the ported parseLateral
// branch, producing a Join whose `this` is a Lateral (LOCAL fix).
func TestParseApplyJoin(t *testing.T) {
	cases := []struct {
		sql        string
		crossApply bool
	}{
		{"SELECT * FROM t CROSS APPLY foo(x)", true},
		{"SELECT * FROM t OUTER APPLY (SELECT 1)", false},
	}
	for _, tc := range cases {
		root := parseOne(t, tc.sql)
		joins := expressionsForArg(root, "joins")
		if len(joins) != 1 {
			t.Fatalf("%s: join count = %d, want 1:\n%s", tc.sql, len(joins), root.ToS())
		}
		lateral := joins[0].This()
		if lateral == nil || lateral.Kind() != exp.KindLateral {
			t.Fatalf("%s: join.this should be a Lateral:\n%s", tc.sql, root.ToS())
		}
		if lateral.Arg("cross_apply") != tc.crossApply {
			t.Fatalf("%s: cross_apply = %v, want %v:\n%s", tc.sql, lateral.Arg("cross_apply"), tc.crossApply, root.ToS())
		}
	}
}

// A CROSS/OUTER APPLY over a table-function with a trailing bare alias must keep the
// alias on the Lateral. Eager firstExpression evaluation in parseLateral used to let
// parseIdVar swallow the alias after parseFunction consumed the function source (LOCAL fix).
func TestParseApplyFunctionAlias(t *testing.T) {
	for _, sql := range []string{
		"SELECT * FROM t CROSS APPLY foo(x) a",
		"SELECT * FROM t CROSS APPLY foo(x) AS a",
	} {
		root := parseOne(t, sql)
		joins := expressionsForArg(root, "joins")
		if len(joins) != 1 {
			t.Fatalf("%s: join count = %d, want 1:\n%s", sql, len(joins), root.ToS())
		}
		lateral := joins[0].This()
		if lateral == nil || lateral.Kind() != exp.KindLateral {
			t.Fatalf("%s: join.this should be a Lateral:\n%s", sql, root.ToS())
		}
		alias, ok := lateral.Arg("alias").(exp.Expression)
		if !ok || alias == nil || alias.Kind() != exp.KindTableAlias {
			t.Fatalf("%s: lateral alias should be a TableAlias, got %v:\n%s", sql, lateral.Arg("alias"), root.ToS())
		}
		if alias.Name() != "a" {
			t.Fatalf("%s: alias name = %q, want %q:\n%s", sql, alias.Name(), "a", root.ToS())
		}
	}
}
