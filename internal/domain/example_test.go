package domain_test

import (
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"testing/quick"

	"github.com/RachidChabane/shallnot/internal/domain"
)

const parkingStatement = "WHEN a stay lasts 15 minutes or less THE SYSTEM SHALL charge 0.00."

func returns(literalType domain.LiteralType, value string) domain.ExpectedOutcome {
	return domain.ExpectedOutcome{Kind: domain.OutcomeReturns, Value: domain.Literal{Type: literalType, Value: value}}
}

func argument(name string, literalType domain.LiteralType, value string) domain.Argument {
	return domain.Argument{Name: name, Value: domain.Literal{Type: literalType, Value: value}}
}

func TestExampleExtraction(t *testing.T) {
	t.Run("splits the joined statement at each marker and keeps the prose before the first [verifies SN-90~1]", func(t *testing.T) {
		statement := parkingStatement +
			" - Example: quote(entry=2026-03-02T10:00, exit=2026-03-02T10:15) => 0.00" +
			" - Example: quote(entry=2026-03-02T10:00, exit=2026-03-02T10:00) => 0.00"
		set := domain.ParseExamples(statement)
		if set.Prose != parkingStatement || len(set.Examples) != 2 || len(set.Malformed) != 0 {
			t.Fatalf("got %+v", set)
		}
		first := set.Examples[0]
		want := domain.Example{
			Index: 1, Operation: "quote",
			Args: []domain.Argument{
				argument("entry", domain.LiteralDatetime, "2026-03-02T10:00"),
				argument("exit", domain.LiteralDatetime, "2026-03-02T10:15"),
			},
			Expect: returns(domain.LiteralDecimal, "0.00"),
		}
		if !reflect.DeepEqual(first, want) || set.Examples[1].Index != 2 {
			t.Fatalf("got %+v, want %+v", first, want)
		}
	})

	t.Run("reads a marker with any list bullet or none [verifies SN-90~1]", func(t *testing.T) {
		for _, bullet := range []string{"- ", "* ", "+ ", ""} {
			set := domain.ParseExamples("Prose. " + bullet + "Example: f() => true")
			if set.Prose != "Prose." || len(set.Examples) != 1 {
				t.Errorf("bullet %q: got %+v", bullet, set)
			}
		}
	})

	t.Run("reads a statement made of examples only [verifies SN-90~1]", func(t *testing.T) {
		set := domain.ParseExamples("Example: f() => 1")
		if set.Prose != "" || len(set.Examples) != 1 {
			t.Fatalf("got %+v", set)
		}
	})

	t.Run("a statement without a marker is all prose and has no example [verifies SN-90~1]", func(t *testing.T) {
		for _, statement := range []string{parkingStatement, "for example: f() => 1", "a Counterexample: f() => 1", "an Example f() => 1"} {
			set := domain.ParseExamples(statement)
			if len(set.Examples)+len(set.Malformed) != 0 || set.Prose != statement {
				t.Errorf("%q: got %+v", statement, set)
			}
		}
	})

	t.Run("text after the last example's outcome is not dropped but malformed [verifies SN-90~1]", func(t *testing.T) {
		set := domain.ParseExamples("Prose. Example: f() => 1 and then more prose")
		if len(set.Examples) != 0 || len(set.Malformed) != 1 || set.Malformed[0].Index != 1 {
			t.Fatalf("got %+v", set)
		}
	})

	t.Run("a malformed segment keeps its position among the markers [verifies SN-90~1]", func(t *testing.T) {
		set := domain.ParseExamples("Prose. Example: f() => 1 Example: broken Example: g() => 2")
		if len(set.Examples) != 2 || set.Examples[0].Index != 1 || set.Examples[1].Index != 3 ||
			len(set.Malformed) != 1 || set.Malformed[0].Index != 2 || set.Malformed[0].Text != "broken" {
			t.Fatalf("got %+v", set)
		}
	})
}

func TestExampleGrammar(t *testing.T) {
	valid := []struct {
		name    string
		segment string
		want    domain.Example
	}{
		{"no argument", "ping() => null", domain.Example{Operation: "ping", Expect: returns(domain.LiteralNull, "null")}},
		{"integer", "double(n=-4) => 8", domain.Example{
			Operation: "double", Args: []domain.Argument{argument("n", domain.LiteralInteger, "-4")}, Expect: returns(domain.LiteralInteger, "8")}},
		{"decimal keeps its text", "price(amount=10.50) => 10.50", domain.Example{
			Operation: "price", Args: []domain.Argument{argument("amount", domain.LiteralDecimal, "10.50")}, Expect: returns(domain.LiteralDecimal, "10.50")}},
		{"booleans", "both(a=true, b=false) => false", domain.Example{
			Operation: "both",
			Args:      []domain.Argument{argument("a", domain.LiteralBoolean, "true"), argument("b", domain.LiteralBoolean, "false")},
			Expect:    returns(domain.LiteralBoolean, "false")}},
		{"datetime with seconds and offset", "at(t=2026-03-02T10:00:30+01:00) => 1", domain.Example{
			Operation: "at", Args: []domain.Argument{argument("t", domain.LiteralDatetime, "2026-03-02T10:00:30+01:00")}, Expect: returns(domain.LiteralInteger, "1")}},
		{"datetime with fraction and Z", "at(t=2026-03-02T10:00:30.250Z) => 1", domain.Example{
			Operation: "at", Args: []domain.Argument{argument("t", domain.LiteralDatetime, "2026-03-02T10:00:30.250Z")}, Expect: returns(domain.LiteralInteger, "1")}},
		{"string with a comma, an arrow and escapes", `say(text="a, b => \"c\"\\") => "x"`, domain.Example{
			Operation: "say", Args: []domain.Argument{argument("text", domain.LiteralString, `a, b => "c"\`)}, Expect: returns(domain.LiteralString, "x")}},
		{"raises", "withdraw(amount=500) => raises InsufficientFunds", domain.Example{
			Operation: "withdraw", Args: []domain.Argument{argument("amount", domain.LiteralInteger, "500")},
			Expect: domain.ExpectedOutcome{Kind: domain.OutcomeRaises, Value: domain.Literal{Type: domain.LiteralName, Value: "InsufficientFunds"}}}},
		{"free spacing", "  f ( a = 1 ,b= 2 )   =>   3 ", domain.Example{
			Operation: "f", Args: []domain.Argument{argument("a", domain.LiteralInteger, "1"), argument("b", domain.LiteralInteger, "2")}, Expect: returns(domain.LiteralInteger, "3")}},
		{"a returned value that starts like the raises keyword", "f() => raisesx", domain.Example{}},
	}
	for _, sample := range valid[:len(valid)-1] {
		t.Run("reads "+sample.name+" [verifies SN-91~1]", func(t *testing.T) {
			set := domain.ParseExamples("Example: " + sample.segment)
			if len(set.Malformed) != 0 || len(set.Examples) != 1 {
				t.Fatalf("got %+v", set)
			}
			want := sample.want
			want.Index = 1
			if !reflect.DeepEqual(set.Examples[0], want) {
				t.Fatalf("got %+v, want %+v", set.Examples[0], want)
			}
		})
	}

	malformed := map[string]string{
		"empty":                  "",
		"no call":                "=> 1",
		"no outcome":             "f()",
		"no arrow":               "f() 1",
		"unclosed call":          "f(a=1 => 1",
		"missing binding":        "f(a) => 1",
		"missing value":          "f(a=) => 1",
		"trailing comma":         "f(a=1,) => 1",
		"missing comma":          "f(a=1 b=2) => 1",
		"duplicate argument":     "f(a=1, a=2) => 1",
		"unquoted word":          "f(a=hello) => 1",
		"date without time":      "f(a=2026-03-02) => 1",
		"impossible datetime":    "f(a=2026-13-40T10:00) => 1",
		"unclosed string":        `f(a="x) => 1`,
		"unknown escape":         `f(a="\q") => 1`,
		"lone backslash":         `f(a="x\`,
		"raises without a name":  "f() => raises",
		"raises with a literal":  "f() => raises 12",
		"operation with a dot":   "a.b() => 1",
		"operation with a digit": "1f() => 1",
		"two outcomes":           "f() => 1 2",
		"text after the outcome": "f() => 1 because",
		"argument name quoted":   `f("a"=1) => 1`,
		"outcome is a call":      "f() => g()",
		"plus sign":              "f(a=+1) => 1",
		"exponent":               "f(a=1e3) => 1",
	}
	for name, segment := range malformed {
		t.Run("rejects "+name+" [verifies SN-91~1]", func(t *testing.T) {
			set := domain.ParseExamples("Example: " + segment)
			if len(set.Examples) != 0 || len(set.Malformed) != 1 || set.Malformed[0].Reason == "" {
				t.Fatalf("got %+v", set)
			}
		})
	}

	t.Run("a returned value that starts like the raises keyword is a malformed literal, not a raise [verifies SN-91~1]", func(t *testing.T) {
		set := domain.ParseExamples("Example: f() => raisesx")
		if len(set.Examples) != 0 || len(set.Malformed) != 1 {
			t.Fatalf("got %+v", set)
		}
	})
}

// render writes an example in the grammar, with random spacing.
func render(random *rand.Rand, example domain.Example) string {
	blank := func() string { return strings.Repeat(" ", random.Intn(3)) }
	var arguments []string
	for _, argument := range example.Args {
		arguments = append(arguments, argument.Name+blank()+"="+blank()+renderLiteral(argument.Value))
	}
	outcome := renderLiteral(example.Expect.Value)
	if example.Expect.Kind == domain.OutcomeRaises {
		outcome = "raises " + example.Expect.Value.Value
	}
	return example.Operation + blank() + "(" + blank() + strings.Join(arguments, blank()+","+blank()) + blank() + ")" + blank() + "=>" + blank() + outcome
}

func renderLiteral(literal domain.Literal) string {
	if literal.Type != domain.LiteralString {
		return literal.Value
	}
	escaped := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\t", `\t`).Replace(literal.Value)
	return `"` + escaped + `"`
}

var (
	sampleLiterals = []domain.Literal{
		{Type: domain.LiteralDatetime, Value: "2026-03-02T10:00"},
		{Type: domain.LiteralDatetime, Value: "2026-03-02T10:00:15.5+02:00"},
		{Type: domain.LiteralDecimal, Value: "0.00"},
		{Type: domain.LiteralDecimal, Value: "-12.50"},
		{Type: domain.LiteralInteger, Value: "7"},
		{Type: domain.LiteralInteger, Value: "-1"},
		{Type: domain.LiteralBoolean, Value: "true"},
		{Type: domain.LiteralBoolean, Value: "false"},
		{Type: domain.LiteralNull, Value: "null"},
		{Type: domain.LiteralString, Value: ""},
		{Type: domain.LiteralString, Value: "a, b"},
		{Type: domain.LiteralString, Value: `quote " and \ backslash`},
		{Type: domain.LiteralString, Value: "Example: => (x)"},
		{Type: domain.LiteralString, Value: "tab\tline\nbreak"},
	}
	sampleNames = []string{"a", "entry", "exit_time", "_x", "N1"}
)

func randomExample(random *rand.Rand) domain.Example {
	example := domain.Example{Operation: sampleNames[random.Intn(len(sampleNames))]}
	for _, name := range random.Perm(len(sampleNames))[:random.Intn(len(sampleNames)+1)] {
		example.Args = append(example.Args, domain.Argument{Name: sampleNames[name], Value: sampleLiterals[random.Intn(len(sampleLiterals))]})
	}
	if random.Intn(3) == 0 {
		example.Expect = domain.ExpectedOutcome{Kind: domain.OutcomeRaises, Value: domain.Literal{Type: domain.LiteralName, Value: "Boom" + sampleNames[random.Intn(len(sampleNames))]}}
	} else {
		example.Expect = domain.ExpectedOutcome{Kind: domain.OutcomeReturns, Value: sampleLiterals[random.Intn(len(sampleLiterals))]}
	}
	return example
}

func TestExampleRoundTrip(t *testing.T) {
	t.Run("an example written in the grammar is read back as it was written [verifies SN-91~1]", func(t *testing.T) {
		property := func(seed int64) bool {
			random := rand.New(rand.NewSource(seed))
			example := randomExample(random)
			text := render(random, example)
			if strings.Contains(text, domain.ExampleMarker) {
				return true
			}
			set := domain.ParseExamples(domain.ExampleMarker + " " + text)
			example.Index = 1
			return len(set.Malformed) == 0 && len(set.Examples) == 1 && reflect.DeepEqual(set.Examples[0], example)
		}
		if err := quick.Check(property, &quick.Config{MaxCount: 2000}); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("several examples after the prose are read in order [verifies SN-90~1]", func(t *testing.T) {
		property := func(seed int64) bool {
			random := rand.New(rand.NewSource(seed))
			var statement strings.Builder
			statement.WriteString(parkingStatement)
			var want []domain.Example
			for index := 1; index <= 1+random.Intn(4); index++ {
				example := randomExample(random)
				text := render(random, example)
				if strings.Contains(text, domain.ExampleMarker) {
					continue
				}
				example.Index = len(want) + 1
				want = append(want, example)
				statement.WriteString(fmt.Sprintf(" %s Example: %s", []string{"-", "*", "+", ""}[random.Intn(4)], text))
			}
			set := domain.ParseExamples(statement.String())
			return set.Prose == parkingStatement && len(set.Malformed) == 0 && reflect.DeepEqual(set.Examples, want)
		}
		if err := quick.Check(property, nil); err != nil {
			t.Fatal(err)
		}
	})
}

func FuzzParseExamples(f *testing.F) {
	for _, seed := range []string{
		"", "Example:", "Prose. - Example: f() => 1", "Example: f(a=1, b=\"x\") => raises Boom",
		"Example: f(a=2026-03-02T10:00) => 0.00 Example: g(", `Example: f(a="\`, "- Example: - Example: Example:",
		"Example: f(a=1 => 1", "Example: =>", "\xff Example: \xfe",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, statement string) {
		set := domain.ParseExamples(statement)
		markers := len(set.Examples) + len(set.Malformed)
		last := 0
		for _, example := range set.Examples {
			if example.Index <= last || example.Index > markers || example.Operation == "" {
				t.Fatalf("invalid example %+v in %+v", example, set)
			}
			last = example.Index
		}
		for _, malformed := range set.Malformed {
			if malformed.Index < 1 || malformed.Index > markers || malformed.Reason == "" {
				t.Fatalf("invalid malformed example %+v in %+v", malformed, set)
			}
		}
		if markers == 0 && set.Prose != strings.TrimSpace(statement) {
			t.Fatalf("a statement without a marker lost its prose: %+v", set)
		}
	})
}
