package dice

import (
	"strings"
	"testing"
)

func TestRoll_Simple(t *testing.T) {
	r := Roll("[1d20]")
	if r == nil {
		t.Fatal("expected result, got nil")
	}
	if r.Expression != "1d20" {
		t.Errorf("expression = %q, want %q", r.Expression, "1d20")
	}
	if len(r.Rolls) != 1 {
		t.Errorf("len(Rolls) = %d, want 1", len(r.Rolls))
	}
	if r.Rolls[0] < 1 || r.Rolls[0] > 20 {
		t.Errorf("roll = %d, want 1-20", r.Rolls[0])
	}
	if r.Total != r.Rolls[0] {
		t.Errorf("total = %d, want %d", r.Total, r.Rolls[0])
	}
}

func TestRoll_MultipleDice(t *testing.T) {
	r := Roll("[3d6]")
	if r == nil {
		t.Fatal("expected result, got nil")
	}
	if len(r.Rolls) != 3 {
		t.Errorf("len(Rolls) = %d, want 3", len(r.Rolls))
	}
	sum := 0
	for _, v := range r.Rolls {
		if v < 1 || v > 6 {
			t.Errorf("roll = %d, want 1-6", v)
		}
		sum += v
	}
	if r.Total != sum {
		t.Errorf("total = %d, want %d", r.Total, sum)
	}
}

func TestRoll_WithModifier(t *testing.T) {
	r := Roll("[2d6+3]")
	if r == nil {
		t.Fatal("expected result, got nil")
	}
	if r.Modifier != 3 {
		t.Errorf("modifier = %d, want 3", r.Modifier)
	}
	sum := 3
	for _, v := range r.Rolls {
		sum += v
	}
	if r.Total != sum {
		t.Errorf("total = %d, want %d", r.Total, sum)
	}
}

func TestRoll_NegativeModifier(t *testing.T) {
	r := Roll("[1d20-2]")
	if r == nil {
		t.Fatal("expected result, got nil")
	}
	if r.Modifier != -2 {
		t.Errorf("modifier = %d, want -2", r.Modifier)
	}
}

func TestRoll_DropLowest(t *testing.T) {
	r := Roll("[4d6dl1]")
	if r == nil {
		t.Fatal("expected result, got nil")
	}
	if len(r.Rolls) != 3 {
		t.Errorf("len(Rolls) = %d, want 3 (kept)", len(r.Rolls))
	}
	if len(r.Dropped) != 1 {
		t.Errorf("len(Dropped) = %d, want 1", len(r.Dropped))
	}
	// Dropped value should be <= smallest kept value
	for _, k := range r.Rolls {
		if r.Dropped[0] > k {
			t.Errorf("dropped %d > kept %d", r.Dropped[0], k)
		}
	}
}

func TestRoll_KeepHighest(t *testing.T) {
	r := Roll("[4d6kh3]")
	if r == nil {
		t.Fatal("expected result, got nil")
	}
	if len(r.Rolls) != 3 {
		t.Errorf("len(Rolls) = %d, want 3", len(r.Rolls))
	}
	if len(r.Dropped) != 1 {
		t.Errorf("len(Dropped) = %d, want 1", len(r.Dropped))
	}
}

func TestRoll_DropHighest(t *testing.T) {
	r := Roll("[4d6dh1]")
	if r == nil {
		t.Fatal("expected result, got nil")
	}
	if len(r.Rolls) != 3 {
		t.Errorf("len(Rolls) = %d, want 3", len(r.Rolls))
	}
	if len(r.Dropped) != 1 {
		t.Errorf("len(Dropped) = %d, want 1", len(r.Dropped))
	}
}

func TestRoll_KeepLowest(t *testing.T) {
	r := Roll("[4d6kl1]")
	if r == nil {
		t.Fatal("expected result, got nil")
	}
	if len(r.Rolls) != 1 {
		t.Errorf("len(Rolls) = %d, want 1", len(r.Rolls))
	}
	if len(r.Dropped) != 3 {
		t.Errorf("len(Dropped) = %d, want 3", len(r.Dropped))
	}
}

func TestRoll_Invalid(t *testing.T) {
	cases := []string{
		"not a dice expression",
		"d20",
		"1d",
		"",
		"hello",
	}
	for _, expr := range cases {
		r := Roll(expr)
		if r != nil {
			t.Errorf("Roll(%q) = %+v, want nil", expr, r)
		}
	}
}

func TestProcess(t *testing.T) {
	// Non-dice text should pass through unchanged
	out := Process("hello world")
	if out != "hello world" {
		t.Errorf("Process(plain text) = %q, want %q", out, "hello world")
	}

	// Dice expressions get replaced
	out = Process("I roll [1d20] for initiative")
	if strings.Contains(out, "[1d20]") {
		t.Errorf("dice expression was not replaced: %q", out)
	}
	if !strings.Contains(out, "1d20") {
		t.Errorf("result should contain original expression: %q", out)
	}

	// Multiple dice expressions
	out = Process("[1d20] and [2d6+3]")
	if strings.Contains(out, "[1d20]") || strings.Contains(out, "[2d6+3]") {
		t.Errorf("not all dice expressions replaced: %q", out)
	}
}

func TestResult_String(t *testing.T) {
	r := Result{
		Expression: "2d6+3",
		Rolls:      []int{4, 5},
		Modifier:   3,
		Total:      12,
	}
	s := r.String()
	if !strings.Contains(s, "2d6+3") {
		t.Errorf("String() missing expression: %q", s)
	}
	if !strings.Contains(s, "4+5") {
		t.Errorf("String() missing rolls: %q", s)
	}
	if !strings.Contains(s, "+3") {
		t.Errorf("String() missing modifier: %q", s)
	}
	if !strings.Contains(s, "= 12") {
		t.Errorf("String() missing total: %q", s)
	}
}

func TestResult_String_WithDropped(t *testing.T) {
	r := Result{
		Expression: "4d6dl1",
		Rolls:      []int{3, 4, 5},
		Dropped:    []int{1},
		Total:      12,
	}
	s := r.String()
	if !strings.Contains(s, "dropped: 1") {
		t.Errorf("String() missing dropped: %q", s)
	}
}

func TestResult_String_NegativeModifier(t *testing.T) {
	r := Result{
		Expression: "1d20-2",
		Rolls:      []int{15},
		Modifier:   -2,
		Total:      13,
	}
	s := r.String()
	if !strings.Contains(s, "-2") {
		t.Errorf("String() missing negative modifier: %q", s)
	}
}
