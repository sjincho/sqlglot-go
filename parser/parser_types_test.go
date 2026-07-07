package parser_test

import (
	"testing"

	exp "github.com/sjincho/sqlglot-go/expressions"
)

func TestParseCastAndTypes(t *testing.T) {
	cast := parseOne(t, "CAST(x AS INT)")
	if cast.Kind() != exp.KindCast || !exp.IsType(exprArg(t, cast, "to"), exp.DTypeInt) {
		t.Fatalf("CAST type mismatch:\n%s", cast.ToS())
	}

	cast = parseOne(t, "CAST(x AS DECIMAL(10, 2))")
	to := exprArg(t, cast, "to")
	if !exp.IsType(to, exp.DTypeDecimal) || len(to.Expressions()) != 2 || to.Expressions()[0].Kind() != exp.KindDataTypeParam {
		t.Fatalf("DECIMAL params mismatch:\n%s", cast.ToS())
	}

	cast = parseOne(t, "x::int")
	if cast.Kind() != exp.KindCast || exprArg(t, cast, "this").Kind() != exp.KindColumn || !exp.IsType(exprArg(t, cast, "to"), exp.DTypeInt) {
		t.Fatalf("dcolon cast mismatch:\n%s", cast.ToS())
	}

	for _, sql := range []string{"TRY_CAST(x AS INT)", "SAFE_CAST(x AS INT)"} {
		tryCast := parseOne(t, sql)
		if tryCast.Kind() != exp.KindTryCast || tryCast.Arg("safe") != true {
			t.Fatalf("%s mismatch:\n%s", sql, tryCast.ToS())
		}
	}
}

func TestParseBracketAndArray(t *testing.T) {
	proj := parseOne(t, "SELECT a[0] FROM t").Expressions()[0]
	if proj.Kind() != exp.KindBracket || exprArg(t, proj, "this").Kind() != exp.KindColumn {
		t.Fatalf("bracket mismatch:\n%s", proj.ToS())
	}

	proj = parseOne(t, "SELECT a[0].b FROM t").Expressions()[0]
	if proj.Kind() != exp.KindDot || exprArg(t, proj, "this").Kind() != exp.KindBracket {
		t.Fatalf("bracket dot mismatch:\n%s", proj.ToS())
	}

	proj = parseOne(t, "SELECT [1, 2]").Expressions()[0]
	if proj.Kind() != exp.KindArray || len(proj.Expressions()) != 2 {
		t.Fatalf("array literal mismatch:\n%s", proj.ToS())
	}
}

func TestParseSpecialFunctions(t *testing.T) {
	cases := []struct {
		sql  string
		kind exp.Kind
	}{
		{"EXTRACT(DAY FROM x)", exp.KindExtract},
		{"SUBSTRING(x FROM 1 FOR 2)", exp.KindSubstring},
		{"TRIM(BOTH ' ' FROM x)", exp.KindTrim},
		{"POSITION(a IN b)", exp.KindStrPosition},
		{"CEIL(x)", exp.KindCeil},
		{"FLOOR(x, 2)", exp.KindFloor},
		{"STRING_AGG(x, ',')", exp.KindGroupConcat},
	}
	for _, tc := range cases {
		expression := parseOne(t, tc.sql)
		if expression.Kind() != tc.kind {
			t.Fatalf("%s kind = %v, want %v:\n%s", tc.sql, expression.Kind(), tc.kind, expression.ToS())
		}
	}
	trim := parseOne(t, "TRIM(BOTH ' ' FROM x)")
	if trim.Arg("position") != "BOTH" {
		t.Fatalf("trim position mismatch:\n%s", trim.ToS())
	}

	// EXTRACT's first arg alternatives must short-circuit (mirror Python `or`): when
	// the function/literal alternative matches, the FROM keyword must survive so the
	// FROM branch is taken (LOCAL fix for eager firstExpression / parseVarOrString).
	for _, sql := range []string{"EXTRACT(foo() FROM x)", "EXTRACT('lit' FROM x)", "EXTRACT(DAY FROM x)"} {
		extract := parseOne(t, sql)
		if extract.Kind() != exp.KindExtract {
			t.Fatalf("%s kind mismatch:\n%s", sql, extract.ToS())
		}
		if expr := exprArg(t, extract, "expression"); expr.Kind() != exp.KindColumn || expr.Name() != "x" {
			t.Fatalf("%s FROM operand mismatch:\n%s", sql, extract.ToS())
		}
	}
}
