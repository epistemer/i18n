// Command basic is a small web app that translates its text with i18n, and
// takes its default locale from a Mutable cfgx setting. Its settings'
// labels are translation keys too, generated through genx.
package main

//go:generate go tool genx

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/epistemer/i18n"
	"github.com/epistemer/i18n/examples/basic/internal/app"
	"github.com/epistemer/i18n/examples/basic/internal/settings"
	"github.com/spf13/cobra"
)

// English is the language the code's default texts are written in.
var English = i18n.Locale{Code: "en", NativeName: "English", FlagCode: "us"}

// The generated settings type satisfies the library's Config interface.
var _ i18n.Config = (*settings.Serve)(nil)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	root := &cobra.Command{Use: "basic", Short: "An i18n example"}
	settings.BindRoot(root)
	root.AddCommand(settings.ServeCommand(&cobra.Command{Use: "serve", Short: "Run the HTTP server"}, serve))
	if err := root.ExecuteContext(ctx); err != nil {
		stop()
		os.Exit(1)
	}
}

// Setup makes sure the example's Arabic locale exists, then loads the
// translations. Creating the locale first keeps Load from warning that a
// configured default locale is missing.
func Setup(s *settings.Serve) (*i18n.Translator, error) {
	tr := i18n.New(s.Store().DB(), English, s, i18n.WithLogger(slog.Default()))
	if err := tr.CreateLocale(i18n.Locale{Code: "ar", NativeName: "العربية", FlagCode: "eg", RTL: true}); err != nil && !errors.Is(err, i18n.ErrLocaleExists) {
		return nil, err
	}
	if err := tr.Load(); err != nil {
		return nil, err
	}
	return tr, nil
}

func serve(ctx context.Context, s *settings.Serve) error {
	tr, err := Setup(s)
	if err != nil {
		return err
	}
	srv := &http.Server{Addr: s.ServerAddr(), Handler: tr.Middleware()(app.Handler(tr))}
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = srv.Shutdown(shutdown)
		case <-done: // ListenAndServe failed
		}
	}()
	slog.Info("serving", "addr", s.ServerAddr())
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
