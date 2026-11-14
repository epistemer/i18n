package i18n

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// labelSetup loads a greeting that takes a label, a few labels, and ar next
// to en, with every key translated into Arabic.
func labelSetup(t testing.TB) (*Translator, func(name string) *key) {
	r := release(
		"hello", "Hello %s",
		"server", "Server",
		"failed", "Failed: %s",
		"unreachable", "Cannot reach %s",
		"count", "%s: %d",
	)
	tr := newT(t, testDB(t), r)
	mustLoad(t, tr)
	addLocale(t, tr, "ar")
	putOverride(t, tr, "ar", "hello", "مرحبا %s")
	putOverride(t, tr, "ar", "server", "الخادم")
	putOverride(t, tr, "ar", "failed", "فشل: %s")
	putOverride(t, tr, "ar", "unreachable", "تعذر الوصول إلى %s")
	putOverride(t, tr, "ar", "count", "%s: %d")
	return tr, func(name string) *key { return &key{name: name, e: r.m[name]} }
}

func TestArgs_LabelInLocale(t *testing.T) {
	tr, k := labelSetup(t)
	hello, server := k("hello"), k("server")

	if got := hello.T(in(tr, "ar"), server); got != "مرحبا الخادم" {
		t.Errorf("a label argument renders in the request's locale: %q", got)
	}
	if got := hello.T(in(tr, "en"), server); got != "Hello Server" {
		t.Errorf("source locale: %q", got)
	}
	if got := T(in(tr, "ar"), "hello", DynamicWrap("server")); got != "مرحبا الخادم" {
		t.Errorf("T by name with a DynamicWrap argument: %q", got)
	}
}

func TestArgs_WithoutTranslator(t *testing.T) {
	_, k := labelSetup(t)
	if got := k("hello").T(context.Background(), k("server")); got != "Hello Server" {
		t.Errorf("without a translator, the label's code text: %q", got)
	}
}

// A label whose own text takes a label: both levels render in the locale.
func TestArgs_Nested(t *testing.T) {
	tr, k := labelSetup(t)
	failed, unreachable, server := k("failed"), k("unreachable"), k("server")
	reason := Tfunc(func(ctx context.Context, _ ...any) string {
		return unreachable.T(ctx, server)
	})

	if got := failed.T(in(tr, "ar"), reason); got != "فشل: تعذر الوصول إلى الخادم" {
		t.Errorf("nested labels in the locale: %q", got)
	}
	if got := failed.T(context.Background(), reason); got != "Failed: Cannot reach Server" {
		t.Errorf("nested labels without a translator: %q", got)
	}
}

func TestArgs_Mixed(t *testing.T) {
	tr, k := labelSetup(t)
	args := []any{k("server"), 3}
	if got := k("count").T(in(tr, "ar"), args...); got != "الخادم: 3" {
		t.Errorf("mixed arguments: %q", got)
	}
	if _, ok := args[0].(*key); !ok {
		t.Errorf("the caller's slice was modified: %#v", args[0])
	}
	if got := k("hello").T(in(tr, "ar"), StaticInt(7)); got != "مرحبا 7" {
		t.Errorf("a Static adapter: %q", got)
	}
}

// labelError is both a label and an error, as an errors package value is:
// the label wins.
type labelError struct{}

func (labelError) Error() string                    { return "log text" }
func (labelError) T(context.Context, ...any) string { return "person text" }

type nilLabel struct{ s string }

func (n *nilLabel) T(context.Context, ...any) string { return n.s }

func TestArgs_TerBeatsErrorAndNil(t *testing.T) {
	tr, k := labelSetup(t)
	ctx := in(tr, "en")
	if got := k("failed").T(ctx, labelError{}); got != "Failed: person text" {
		t.Errorf("a Ter that is also an error: %q", got)
	}
	if got := k("failed").T(ctx, (*nilLabel)(nil)); got != "Failed: <nil>" {
		t.Errorf("a nil label pointer: %q", got)
	}
	if got := k("failed").T(ctx, Tfunc(nil)); got != "Failed: <nil>" {
		t.Errorf("a nil Tfunc: %q", got)
	}
	boom := Tfunc(func(context.Context, ...any) string { panic("boom") })
	func() {
		defer func() {
			if p := recover(); p != "boom" {
				t.Errorf("a label's own panic propagates, got %v", p)
			}
		}()
		k("failed").T(ctx, boom)
	}()
}

// stringerFunc is an argument that isn't a label, whose String runs code.
type stringerFunc func() string

func (f stringerFunc) String() string { return f() }

// No code from an argument runs inside T's read transaction: a label's T
// renders before it, and fmt calls String after it. Here the argument writes
// enough to make bbolt grow its file, which takes the mmap lock that every
// open read transaction holds: if the argument ran inside T's View, T would
// never return.
func TestArgs_OutsideTransaction(t *testing.T) {
	for name, arg := range map[string]func(write func() string) any{
		"Ter":      func(w func() string) any { return Tfunc(func(context.Context, ...any) string { return w() }) },
		"Stringer": func(w func() string) any { return stringerFunc(w) },
	} {
		t.Run(name, func(t *testing.T) {
			tr, k := labelSetup(t)
			big := strings.Repeat("x", 16<<20)
			errc := make(chan error, 1)
			write := func() string {
				errc <- tr.Set("ar", "server", big)
				return "written"
			}
			done := make(chan string, 1)
			go func() { done <- k("hello").T(in(tr, "ar"), arg(write)) }()
			select {
			case got := <-done:
				if err := <-errc; err != nil {
					t.Fatal(err)
				}
				if got != "مرحبا written" {
					t.Errorf("got %q", got)
				}
			case <-time.After(30 * time.Second):
				// Not t.Fatal or panic: testing runs the store's cleanup
				// either way, and closing the store waits for the stuck
				// reader.
				fmt.Fprintf(os.Stderr, "FAIL: TestArgs_OutsideTransaction/%s: T deadlocked: the argument ran inside T's transaction\n", name)
				os.Exit(1)
			}
		})
	}
}

// TestArgs_Stress renders nested label arguments while other goroutines Set
// and Reset the same keys. It runs for I18N_STRESS (a Go duration), or for a
// moment by default. Run it under -race: I18N_STRESS=10m go test -race -run
// TestArgs_Stress -timeout 15m.
func TestArgs_Stress(t *testing.T) {
	d := 500 * time.Millisecond
	if s := os.Getenv("I18N_STRESS"); s != "" {
		var err error
		if d, err = time.ParseDuration(s); err != nil {
			t.Fatal(err)
		}
	}
	tr, k := labelSetup(t)
	failed, unreachable, server := k("failed"), k("unreachable"), k("server")
	reason := Tfunc(func(ctx context.Context, _ ...any) string {
		return unreachable.T(ctx, server)
	})

	// Every text a writer can leave, per key: the reader accepts any
	// combination, and nothing else.
	servers := []string{"الخادم", "الخادم ٢", "Server"}
	reasons := []string{"تعذر الوصول إلى %s", "لا يمكن الوصول إلى %s"}
	stop := make(chan struct{})
	var wg sync.WaitGroup
	var writes, reads atomic.Int64
	var failure atomic.Value

	for w := range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; ; i++ {
				select {
				case <-stop:
					return
				default:
				}
				var err error
				switch (i + w) % 5 {
				case 0:
					err = tr.Set("ar", "server", "الخادم ٢")
				case 1:
					_, err = tr.Reset("ar", "server")
				case 2:
					err = tr.Set("ar", "server", "الخادم")
				case 3:
					err = tr.Set("ar", "unreachable", reasons[i/5%2])
				case 4:
					// A text of up to 4 MB on a key the readers don't use,
					// so the file grows under the readers. bbolt remaps
					// only when its mapping doubles, so this happens early in
					// a run; TestArgs_OutsideTransaction covers the remap
					// itself.
					pad := strings.Repeat("x", (i%64+1)<<16)
					err = tr.Set("ar", "count", "%s: %d"+pad)
				}
				if err != nil {
					failure.CompareAndSwap(nil, err)
					return
				}
				writes.Add(1)
			}
		}()
	}
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := in(tr, "ar")
			for {
				select {
				case <-stop:
					return
				default:
				}
				got := failed.T(ctx, reason)
				ok := false
				for _, r := range reasons {
					for _, s := range servers {
						if got == "فشل: "+fmt.Sprintf(r, s) {
							ok = true
						}
					}
				}
				if !ok {
					failure.CompareAndSwap(nil, fmt.Errorf("unexpected text %q", got))
					return
				}
				reads.Add(1)
			}
		}()
	}

	deadline := time.After(d)
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	start := time.Now()
wait:
	for {
		select {
		case <-deadline:
			break wait
		case now := <-tick.C:
			if failure.Load() != nil {
				break wait
			}
			if s := int(now.Sub(start).Seconds()); s%60 == 0 {
				t.Logf("%ds: %d reads, %d writes", s, reads.Load(), writes.Load())
			}
		}
	}
	close(stop)
	wg.Wait()
	if err, _ := failure.Load().(error); err != nil {
		t.Fatal(err)
	}
	if reads.Load() == 0 || writes.Load() == 0 {
		t.Fatalf("nothing ran: %d reads, %d writes", reads.Load(), writes.Load())
	}
	t.Logf("%v: %d reads, %d writes", d, reads.Load(), writes.Load())
}
