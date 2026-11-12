package i18n

import (
	"context"
	"fmt"
	"testing"
	"time"
)

type customStringer struct {
	val string
}

func (c customStringer) String() string {
	return c.val
}

func TestTfunc(t *testing.T) {
	ctx := context.WithValue(context.Background(), struct{}{}, "val")
	f := Tfunc(func(c context.Context, args ...any) string {
		if len(args) > 0 {
			return fmt.Sprintf("called:%v", args[0])
		}
		return "called:no-args"
	})

	var ter Ter = f
	if got := ter.T(ctx, "arg1"); got != "called:arg1" {
		t.Errorf("Tfunc.T(ctx, arg1) = %q, want %q", got, "called:arg1")
	}
	if got := ter.T(ctx); got != "called:no-args" {
		t.Errorf("Tfunc.T(ctx) = %q, want %q", got, "called:no-args")
	}
}

func TestStaticAdapters(t *testing.T) {
	ctx := context.Background()

	strVal := "hello"
	intVal := 42
	intNegVal := -7
	int32Val := int32(100)
	int64Val := int64(9876543210)
	floatVal := 3.14159
	fixedDate := time.Date(2026, time.November, 12, 15, 4, 5, 0, time.UTC)

	tests := []struct {
		name    string
		adapter Ter
		want    string
	}{
		{
			name:    "Static regular string",
			adapter: Static("welcome"),
			want:    "welcome",
		},
		{
			name:    "Static empty string",
			adapter: Static(""),
			want:    "",
		},
		{
			name:    "StaticOptional non-nil",
			adapter: StaticOptional(&strVal),
			want:    "hello",
		},
		{
			name:    "StaticOptional nil renders NoneString",
			adapter: StaticOptional(nil),
			want:    NoneString,
		},
		{
			name:    "StaticInt positive",
			adapter: StaticInt(intVal),
			want:    "42",
		},
		{
			name:    "StaticInt negative",
			adapter: StaticInt(intNegVal),
			want:    "-7",
		},
		{
			name:    "StaticInt zero",
			adapter: StaticInt(0),
			want:    "0",
		},
		{
			name:    "StaticInt32",
			adapter: StaticInt32(int32Val),
			want:    "100",
		},
		{
			name:    "StaticInt64",
			adapter: StaticInt64(int64Val),
			want:    "9876543210",
		},
		{
			name:    "StaticFloat64 fixed 2 prec",
			adapter: StaticFloat64(floatVal, 'f', 2, 64),
			want:    "3.14",
		},
		{
			name:    "StaticFloat64 general fmt",
			adapter: StaticFloat64(floatVal, 'g', -1, 64),
			want:    "3.14159",
		},
		{
			name:    "StaticBool true",
			adapter: StaticBool(true),
			want:    "true",
		},
		{
			name:    "StaticBool false",
			adapter: StaticBool(false),
			want:    "false",
		},
		{
			name:    "StaticDate",
			adapter: StaticDate(fixedDate),
			want:    "2026-11-12",
		},
		{
			name:    "StaticStringer non-nil",
			adapter: StaticStringer(customStringer{val: "custom-text"}),
			want:    "custom-text",
		},
		{
			name:    "StaticStringer nil renders NoneString",
			adapter: StaticStringer(nil),
			want:    NoneString,
		},
		{
			name:    "StaticOptionalInt non-nil",
			adapter: StaticOptionalInt(&intVal, "n/a"),
			want:    "42",
		},
		{
			name:    "StaticOptionalInt nil",
			adapter: StaticOptionalInt(nil, "n/a"),
			want:    "n/a",
		},
		{
			name:    "StaticOptionalInt32 non-nil",
			adapter: StaticOptionalInt32(&int32Val, "none"),
			want:    "100",
		},
		{
			name:    "StaticOptionalInt32 nil",
			adapter: StaticOptionalInt32(nil, "none"),
			want:    "none",
		},
		{
			name:    "StaticOptionalInt64 non-nil",
			adapter: StaticOptionalInt64(&int64Val, "nil-val"),
			want:    "9876543210",
		},
		{
			name:    "StaticOptionalInt64 nil",
			adapter: StaticOptionalInt64(nil, "nil-val"),
			want:    "nil-val",
		},
		{
			name:    "StaticOptionalFloat64 non-nil",
			adapter: StaticOptionalFloat64(&floatVal, 'f', 2, 64, "empty"),
			want:    "3.14",
		},
		{
			name:    "StaticOptionalFloat64 nil",
			adapter: StaticOptionalFloat64(nil, 'f', 2, 64, "empty"),
			want:    "empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.adapter.T(ctx, "ignored-arg-1", "ignored-arg-2")
			if got != tt.want {
				t.Errorf("%s: got %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestStaticOptional_RenderTimeNoneString(t *testing.T) {
	ctx := context.Background()
	original := NoneString
	t.Cleanup(func() {
		NoneString = original
	})

	optStr := StaticOptional(nil)
	optStringer := StaticStringer(nil)

	if got := optStr.T(ctx); got != " - " {
		t.Errorf("initial StaticOptional(nil) = %q, want %q", got, " - ")
	}
	if got := optStringer.T(ctx); got != " - " {
		t.Errorf("initial StaticStringer(nil) = %q, want %q", got, " - ")
	}

	NoneString = "N/A"
	if got := optStr.T(ctx); got != "N/A" {
		t.Errorf("after modifying NoneString, StaticOptional(nil) = %q, want %q", got, "N/A")
	}
	if got := optStringer.T(ctx); got != "N/A" {
		t.Errorf("after modifying NoneString, StaticStringer(nil) = %q, want %q", got, "N/A")
	}
}

type testTKer struct{}

func (testTKer) T(ctx context.Context, args ...any) string {
	return "tker"
}

func (testTKer) K() string {
	return "key"
}

func TestInterfaces(t *testing.T) {
	var _ Ter = testTKer{}
	var _ Ker = testTKer{}
	var _ TKer = testTKer{}
}
