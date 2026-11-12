package i18n

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

// Ter is the interface for types that render localized or static text in a context.
// UI components accept Ter so they can render translated keys, static text, or computed labels.
type Ter interface {
	T(ctx context.Context, args ...any) string
}

// Ker is the interface for types that identify a translation key by its name.
type Ker interface {
	K() string
}

// TKer combines Ter and Ker for handles that both identify a key and render its text.
type TKer interface {
	Ter
	Ker
}

// Tfunc adapts an ordinary function to the Ter interface.
type Tfunc func(ctx context.Context, args ...any) string

// T calls f(ctx, args...) to satisfy the Ter interface.
func (f Tfunc) T(ctx context.Context, args ...any) string {
	return f(ctx, args...)
}

// NoneString is the default text rendered for nil optional values.
var NoneString = " - "

type static string

func (s static) T(context.Context, ...any) string {
	return string(s)
}

// Static returns a Ter that always renders s without translation.
func Static(s string) Ter {
	return static(s)
}

type staticOptional struct {
	s *string
}

func (o staticOptional) T(context.Context, ...any) string {
	if o.s == nil {
		return NoneString
	}
	return *o.s
}

// StaticOptional returns a Ter rendering *s, or NoneString when s is nil.
// NoneString is read at render time.
func StaticOptional(s *string) Ter {
	return staticOptional{s: s}
}

// StaticInt returns a Ter that renders n formatted as a decimal string.
func StaticInt(n int) Ter {
	return static(strconv.Itoa(n))
}

// StaticInt32 returns a Ter that renders n formatted as a decimal string.
func StaticInt32(n int32) Ter {
	return static(strconv.FormatInt(int64(n), 10))
}

// StaticInt64 returns a Ter that renders n formatted as a decimal string.
func StaticInt64(n int64) Ter {
	return static(strconv.FormatInt(n, 10))
}

// StaticFloat64 returns a Ter that renders f formatted according to fmt, prec, and bitSize.
func StaticFloat64(f float64, fmt byte, prec, bitSize int) Ter {
	return static(strconv.FormatFloat(f, fmt, prec, bitSize))
}

// StaticBool returns a Ter that renders b formatted as "true" or "false".
func StaticBool(b bool) Ter {
	return static(strconv.FormatBool(b))
}

// StaticDate returns a Ter that renders t formatted as time.DateOnly ("2006-01-02").
func StaticDate(t time.Time) Ter {
	return static(t.Format(time.DateOnly))
}

type staticStringer struct {
	s fmt.Stringer
}

func (ss staticStringer) T(context.Context, ...any) string {
	if ss.s == nil {
		return NoneString
	}
	return ss.s.String()
}

// StaticStringer returns a Ter rendering s.String() at render time, or NoneString when s is nil.
func StaticStringer(s fmt.Stringer) Ter {
	return staticStringer{s: s}
}

type staticOptionalInt struct {
	n    *int
	none string
}

func (o staticOptionalInt) T(context.Context, ...any) string {
	if o.n == nil {
		return o.none
	}
	return strconv.Itoa(*o.n)
}

// StaticOptionalInt returns a Ter rendering *n formatted as a decimal string, or none when n is nil.
func StaticOptionalInt(n *int, none string) Ter {
	return staticOptionalInt{n: n, none: none}
}

type staticOptionalInt32 struct {
	n    *int32
	none string
}

func (o staticOptionalInt32) T(context.Context, ...any) string {
	if o.n == nil {
		return o.none
	}
	return strconv.FormatInt(int64(*o.n), 10)
}

// StaticOptionalInt32 returns a Ter rendering *n formatted as a decimal string, or none when n is nil.
func StaticOptionalInt32(n *int32, none string) Ter {
	return staticOptionalInt32{n: n, none: none}
}

type staticOptionalInt64 struct {
	n    *int64
	none string
}

func (o staticOptionalInt64) T(context.Context, ...any) string {
	if o.n == nil {
		return o.none
	}
	return strconv.FormatInt(*o.n, 10)
}

// StaticOptionalInt64 returns a Ter rendering *n formatted as a decimal string, or none when n is nil.
func StaticOptionalInt64(n *int64, none string) Ter {
	return staticOptionalInt64{n: n, none: none}
}

type staticOptionalFloat64 struct {
	f       *float64
	fmt     byte
	prec    int
	bitSize int
	none    string
}

func (o staticOptionalFloat64) T(context.Context, ...any) string {
	if o.f == nil {
		return o.none
	}
	return strconv.FormatFloat(*o.f, o.fmt, o.prec, o.bitSize)
}

// StaticOptionalFloat64 returns a Ter rendering *f formatted according to fmt, prec, and bitSize, or none when f is nil.
func StaticOptionalFloat64(f *float64, fmt byte, prec, bitSize int, none string) Ter {
	return staticOptionalFloat64{f: f, fmt: fmt, prec: prec, bitSize: bitSize, none: none}
}
