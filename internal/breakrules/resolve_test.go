package breakrules

import (
	"strings"
	"testing"
)

func TestResolveSimple(t *testing.T) {
	rs, errs := Parse([]byte(`$A = [\p{Lu}]; $A;`))
	if len(errs) > 0 {
		t.Fatalf("parse: %v", errs)
	}
	if rerrs := Resolve(rs); rerrs != nil {
		t.Fatalf("resolve: %v", rerrs)
	}
	r := rs.Rules[0]
	if r.Expr.Kind != NodeCharClass {
		t.Fatalf("expected NodeCharClass, got %d", r.Expr.Kind)
	}
	if r.Expr.Name != `[\p{Lu}]` {
		t.Fatalf("expected [\\p{Lu}], got %q", r.Expr.Name)
	}
}

func TestResolveTransitive(t *testing.T) {
	src := `$A = [\p{Lu}]; $B = $A; $B;`
	rs, errs := Parse([]byte(src))
	if len(errs) > 0 {
		t.Fatalf("parse: %v", errs)
	}
	if rerrs := Resolve(rs); rerrs != nil {
		t.Fatalf("resolve: %v", rerrs)
	}
	r := rs.Rules[0]
	if r.Expr.Kind != NodeCharClass {
		t.Fatalf("expected NodeCharClass, got %d", r.Expr.Kind)
	}
}

func TestResolveCycle(t *testing.T) {
	src := `$A = $B; $B = $A; $A;`
	rs, errs := Parse([]byte(src))
	if len(errs) > 0 {
		t.Fatalf("parse: %v", errs)
	}
	rerrs := Resolve(rs)
	if rerrs == nil {
		t.Fatal("expected cycle error")
	}
	found := false
	for _, e := range rerrs {
		if strings.Contains(e.Error(), "cycle") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected cycle error, got: %v", rerrs)
	}
}

func TestResolveUndefined(t *testing.T) {
	src := `$X;`
	rs, errs := Parse([]byte(src))
	if len(errs) > 0 {
		t.Fatalf("parse: %v", errs)
	}
	rerrs := Resolve(rs)
	if rerrs == nil {
		t.Fatal("expected undefined variable error")
	}
	found := false
	for _, e := range rerrs {
		if strings.Contains(e.Error(), "undefined") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected undefined error, got: %v", rerrs)
	}
}

func TestResolveInExpr(t *testing.T) {
	src := `$A = [\p{Lu}]; $B = [\p{Ll}]; ($A | $B)*;`
	rs, errs := Parse([]byte(src))
	if len(errs) > 0 {
		t.Fatalf("parse: %v", errs)
	}
	if rerrs := Resolve(rs); rerrs != nil {
		t.Fatalf("resolve: %v", rerrs)
	}
	r := rs.Rules[0]
	if r.Expr.Kind != NodeStar {
		t.Fatalf("expected NodeStar, got %d", r.Expr.Kind)
	}
	alt := r.Expr.Child
	if alt.Kind != NodeAlt {
		t.Fatalf("expected NodeAlt, got %d", alt.Kind)
	}
	for i, c := range alt.Children {
		if c.Kind != NodeCharClass {
			t.Fatalf("child %d: expected NodeCharClass, got %d", i, c.Kind)
		}
	}
}

func TestResolveMultipleUsesAreIndependent(t *testing.T) {
	src := `$A = [\p{Lu}]; $A $A;`
	rs, errs := Parse([]byte(src))
	if len(errs) > 0 {
		t.Fatalf("parse: %v", errs)
	}
	if rerrs := Resolve(rs); rerrs != nil {
		t.Fatalf("resolve: %v", rerrs)
	}
	r := rs.Rules[0]
	if r.Expr.Kind != NodeConcat {
		t.Fatalf("expected NodeConcat, got %d", r.Expr.Kind)
	}
	c0 := r.Expr.Children[0]
	c1 := r.Expr.Children[1]
	if c0 == c1 {
		t.Fatal("expected different Node pointers for two uses of same variable")
	}
}
