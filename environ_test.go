package environ_test

import (
	"reflect"
	"testing"

	"github.com/metrumresearchgroup/environ"
)

func TestMarshalUnmarshal(t *testing.T) {
	orig := environ.FromOS()

	if orig.Len() == 0 {
		t.Fatal("orig empty, cannot test")
	}

	marshaled, err := orig.MarshalJSON()

	if err != nil {
		t.Fatalf("error in MarshalJSON(): %v", err)
	}

	if len(marshaled) == 0 {
		t.Fatal("marshaled empty")
	}

	var unmarshaled = new(environ.Environ)

	err = unmarshaled.UnmarshalJSON(marshaled)

	if err != nil {
		t.Fatalf("error in UnmarshalJSON(): %v", err)
	}

	if !reflect.DeepEqual(orig.AsSlice(), unmarshaled.AsSlice()) {
		t.Fatalf("expected orig to match unmarshaled, orig: %v, unmarshaled: %v", orig.AsSlice(), unmarshaled.AsSlice())
	}
}

func TestGetSetUnset(t *testing.T) {
	e := environ.New([]string{"A=A"})

	e.Set("B", "B")
	e.Set("B", "Bee")

	if !reflect.DeepEqual(e.AsSlice(), []string{"A=A", "B=Bee"}) {
		t.Fatalf("B not updated to Bee")
	}

	e.Set("A", "Apple")

	got := e.Get("B")
	if got != "Bee" {
		t.Fatalf("B not returning value Bee")
	}

	e.Unset("B")
	e.Unset("Z")

	got = e.Get("B")
	if got != "" {
		t.Fatalf("B not unset")
	}
}

func TestComments(t *testing.T) {
	e := environ.New([]string{"A=", "#C", "", "B=B"})
	if !reflect.DeepEqual(e.AsSlice(), []string{"A=", "B=B"}) {
		t.Fatalf("unexpected slice: %v", e.AsSlice())
	}
}

func TestSkipNoEquals(t *testing.T) {
	e := environ.New([]string{"A", "B=B", "C="})
	if !reflect.DeepEqual(e.AsSlice(), []string{"B=B", "C="}) {
		t.Fatalf("unexpected slice: %v", e.AsSlice())
	}
}

func TestCatchBadRegex(t *testing.T) {
	e := environ.New([]string{"A", "B=B", "C="})
	missing, err := e.Drop(`unsupported\K`)
	if err == nil {
		t.Fatalf("expected an error which did not occur")
	}
	if !reflect.DeepEqual(missing, []string{`unsupported\K`}) {
		t.Fatalf("missing had unexpected result. actual: %v", missing)
	}

	missing, err = e.Keep(`unsupported\K`)
	if err == nil {
		t.Fatalf("expected an error which did not occur")
	}
	if !reflect.DeepEqual(missing, []string{`unsupported\K`}) {
		t.Fatalf("missing had unexpected result. actual: %v", missing)
	}
}

func TestKeepDrop(t *testing.T) {
	env := environ.New([]string{"A=A", "B=B", "C=C", "D=D", "A_A=AA", "A_B=AB"})

	missing, err := env.Keep("A", "B", "C", "E", "A_.*", "B_.*")
	if err != nil {
		t.Fatalf("unexpected an error: %v", err)
	}

	if !reflect.DeepEqual(env.AsSlice(), []string{"A=A", "A_A=AA", "A_B=AB", "B=B", "C=C"}) {
		t.Fatalf("didn't keep correct values: %v", env.AsSlice())
	}

	if !reflect.DeepEqual(missing, []string{"B_.*", "E"}) {
		t.Fatalf("missing is missing a value (either 'B_*' or 'E'): %v", missing)
	}

	missing, err = env.Drop("B", "D", "A_.*")
	if err != nil {
		t.Fatalf("found unexpected err: %v", err)
	}

	if !reflect.DeepEqual(env.AsSlice(), []string{"A=A", "C=C"}) {
		t.Fatalf("didn't drop correct values")
	}

	if !reflect.DeepEqual(missing, []string{"D"}) {
		t.Fatalf("D was not missing")
	}
}

func TestKeys(t *testing.T) {
	env := environ.New([]string{"A=A", "B", "C="})
	keys := env.Keys()
	if !reflect.DeepEqual(keys, []string{"A", "C"}) {
		t.Fatalf("unexpected slice: %v", env.AsSlice())
	}
}

func TestUnmarshalFailure(t *testing.T) {
	env := new(environ.Environ)
	err := env.UnmarshalJSON([]byte(`{"A":"A","B"}`))
	if err == nil {
		t.Fatalf("nil err")
	}
	if err.Error() != "invalid character '}' after object key" {
		t.Fatalf("error incorrect, got: %v", err)
	}
}
