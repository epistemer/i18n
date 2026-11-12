package i18n

import (
	"strings"
	"testing"
)

func TestRegistry_IdenticalDuplicateIsFine(t *testing.T) {
	r := newRegistry()
	a := r.add("a", "A", []byte{1})
	b := r.add("a", "A", []byte{1})
	if a.e != b.e {
		t.Error("an identical duplicate should share the entry")
	}
	if err := r.freeze(); err != nil {
		t.Errorf("freeze: %v", err)
	}
}

func TestRegistry_ProblemsAreReportedTogether(t *testing.T) {
	r := newRegistry()
	r.add("a", "A", nil)
	r.add("a", "Other", nil)
	r.add("m", "M", []byte{1})
	r.add("m", "M", []byte{2})
	r.add("", "orphan text", nil)
	err := r.freeze()
	if err == nil {
		t.Fatal("freeze: no error")
	}
	for _, want := range []string{`"a" registered twice, with texts "A" and "Other"`, `"m" registered twice, with different meta`, `empty key with text "orphan text"`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q lacks %q", err, want)
		}
	}
	if got := r.m["a"].text; got != "A" {
		t.Errorf("first registration should win, got %q", got)
	}
}

func TestRegistry_KeyAfterLoadPanics(t *testing.T) {
	r := newRegistry()
	mustLoad(t, newT(t, testDB(t), r))
	defer func() {
		msg, _ := recover().(string)
		if !strings.Contains(msg, `Key("late.key") called after Load`) {
			t.Errorf("panic = %q", msg)
		}
	}()
	r.add("late.key", "Late", nil)
}

func TestLoad_FailsOnRegistryProblemsBeforeWriting(t *testing.T) {
	r := release("a", "A", "a", "B")
	tr := newT(t, testDB(t), r)
	if err := tr.Load(); err == nil {
		t.Fatal("Load: no error")
	}
	if ce := catalogOf(t, tr, "a"); ce != nil {
		t.Error("Load wrote to the store despite registry problems")
	}
}

func TestDynamicWrap(t *testing.T) {
	k := DynamicWrap("some.key")
	if k.K() != "some.key" {
		t.Errorf("K() = %q", k.K())
	}
}
