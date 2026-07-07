package expressions_test

import (
	"testing"

	exp "github.com/sjincho/sqlglot-go/expressions"
)

func TestRegisteredFunctions(t *testing.T) {
	// Remaining test_functions entries are deferred to slice 1c (long-tail functions and generator-dependent assertions).
	cases := []struct {
		sql  string
		kind exp.Kind
	}{
		{"ABS(a)", exp.KindAbs},
		{"AVG(a)", exp.KindAvg},
		{"SUM(a)", exp.KindSum},
		{"MIN(a)", exp.KindMin},
		{"MAX(a)", exp.KindMax},
		{"COUNT(a)", exp.KindCount},
		{"COUNT_IF(a > 0)", exp.KindCountIf},
		{"COALESCE(a, b)", exp.KindCoalesce},
		{"IFNULL(a, b)", exp.KindCoalesce},
		{"NVL(a, b)", exp.KindCoalesce},
		{"GREATEST(a, b)", exp.KindGreatest},
		{"LEAST(a, b)", exp.KindLeast},
		{"IF(a, b, c)", exp.KindIf},
		{"SQRT(a)", exp.KindSqrt},
		{"LN(a)", exp.KindLn},
		{"EXP(a)", exp.KindExp},
		{"POW(a, 2)", exp.KindPow},
		{"POWER(a, 2)", exp.KindPow},
		{"ROUND(a)", exp.KindRound},
		{"ROUND(a, 2)", exp.KindRound},
		{"STDDEV(a)", exp.KindStddev},
		{"STDDEV_POP(a)", exp.KindStddevPop},
		{"STDDEV_SAMP(a)", exp.KindStddevSamp},
		{"VARIANCE(a)", exp.KindVariance},
		{"VAR_POP(a)", exp.KindVariancePop},
		{"DAY(a)", exp.KindDay},
		{"MONTH(a)", exp.KindMonth},
		{"YEAR(a)", exp.KindYear},
		{"QUARTER(a)", exp.KindQuarter},
		{"APPROX_DISTINCT(a)", exp.KindApproxDistinct},
		{"APPROX_COUNT_DISTINCT(a)", exp.KindApproxDistinct},
		{"HLL(a)", exp.KindHll},
		{"LOG(b, n)", exp.KindLog},
		{"QUANTILE(a, 0.90)", exp.KindQuantile},
		{"CAST(a AS INT)", exp.KindCast},
		{"TRY_CAST(a AS INT)", exp.KindTryCast},
		{"SAFE_CAST(a AS INT)", exp.KindTryCast},
		{"CAST(a, 'INT')", exp.KindCastToStrType},
		{"EXTRACT(DAY FROM a)", exp.KindExtract},
		{"POSITION(a IN b)", exp.KindStrPosition},
		{"SUBSTRING(a FROM 1 FOR 2)", exp.KindSubstring},
		{"SUBSTR(a, 1, 2)", exp.KindSubstring},
		{"TRIM(a)", exp.KindTrim},
		{"CEIL(a)", exp.KindCeil},
		{"CEILING(a)", exp.KindCeil},
		{"FLOOR(a)", exp.KindFloor},
		{"STRING_AGG(a, ',')", exp.KindGroupConcat},
		{"GROUP_CONCAT(a, ',')", exp.KindGroupConcat},
		{"LISTAGG(a, ',')", exp.KindGroupConcat},
	}
	for _, tc := range cases {
		expression := parseOne(t, tc.sql)
		if expression.Kind() != tc.kind {
			t.Fatalf("ParseOne(%q).Kind() = %v, want %v", tc.sql, expression.Kind(), tc.kind)
		}
	}
}

func TestGreatestVarArgs(t *testing.T) {
	expression := parseOne(t, "GREATEST(a, b, c)")
	if expression.Kind() != exp.KindGreatest {
		t.Fatalf("kind = %v, want Greatest", expression.Kind())
	}
	if got := len(expression.Expressions()); got != 2 {
		t.Fatalf("GREATEST expressions count = %d, want 2", got)
	}
}

// TestHllVarArgs guards the generic FromArgList var-len path: Hll.is_var_len_args is
// True upstream (core.py:2009), so trailing args must collect into the "expressions"
// list rather than a single node (which would also drop args beyond the second).
func TestHllVarArgs(t *testing.T) {
	expression := parseOne(t, "HLL(a, b, c)")
	if expression.Kind() != exp.KindHll {
		t.Fatalf("kind = %v, want Hll", expression.Kind())
	}
	if got := len(expression.Expressions()); got != 2 {
		t.Fatalf("HLL expressions count = %d, want 2", got)
	}
}
