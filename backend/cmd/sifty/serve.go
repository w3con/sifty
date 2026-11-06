package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/w3con/sifty/internal/background"
	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/dirs"
	"github.com/w3con/sifty/internal/inspect"
	"github.com/w3con/sifty/internal/server"
	"github.com/w3con/sifty/internal/store"
	"github.com/w3con/sifty/internal/worker"
)

func runServe(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dataDir := fs.String("data-dir", "", "local data directory (overrides $SIFTY_DATA_DIR)")
	distDir := fs.String("dist", "", "path to the built React frontend dist/ directory")
	workersDir := fs.String("workers-dir", "workers", "path to the workers/ directory (default: workers/ in the current directory)")
	// Preview and the deploy loop bind the process to an internal loopback port
	// while browsers use a stable public port (or the reverse). Both default to
	// the configured runtime port, so normal `sifty serve` behaviour is unchanged.
	bindPort := fs.Int("port", 0, "HTTP port override (0 = runtime.yaml port)")
	publicPort := fs.Int("public-port", 0, "port browsers use for Host/Origin checks (0 = bind port)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *bindPort < 0 || *bindPort > 65535 || *publicPort < 0 || *publicPort > 65535 {
		fmt.Fprintln(stderr, "sifty serve: ports must be between 1 and 65535")
		return 2
	}

	d, err := dirs.Resolve(*dataDir)
	if err != nil {
		fmt.Fprintf(stderr, "sifty serve: %v\n", err)
		return 1
	}
	if err := d.Ensure(); err != nil {
		fmt.Fprintf(stderr, "sifty serve: ensure data dir: %v\n", err)
		return 1
	}

	rawCfg, err := config.Load(d)
	if err != nil {
		fmt.Fprintf(stderr, "sifty serve: load config: %v\n", err)
		return 1
	}
	cfgHolder := config.NewHolder(rawCfg)

	// Resolve the listener and the port browsers use. A preview proxy forwards
	// browser traffic from the public port to an internal bind port; the security
	// middleware checks Host/Origin against the public port in that case.
	listenPort := rawCfg.Runtime.Port
	if *bindPort > 0 {
		listenPort = *bindPort
	}
	if listenPort < 1 || listenPort > 65535 {
		fmt.Fprintf(stderr, "sifty serve: invalid listen port %d\n", listenPort)
		return 1
	}
	servePort := listenPort
	if *publicPort > 0 {
		servePort = *publicPort
	}

	s, err := store.Open(d.Store())
	if err != nil {
		fmt.Fprintf(stderr, "sifty serve: open store: %v\n", err)
		return 1
	}
	defer func() { _ = s.DB.Close() }()

	deps := &server.Deps{Store: s, Dirs: d, Cfg: cfgHolder}
	deps.CollectFn = func(source string, force bool) (int64, string, error) {
		return background.TriggerAsync(s, d, cfgHolder.Get(), background.TriggerOpts{
			Source: source,
			Force:  force,
		})
	}
	mux := server.New(servePort, func(m *http.ServeMux) {
		server.RegisterRoutes(m, deps)
	})

	// Serve the built frontend if a dist directory is configured.
	resolvedDist := *distDir
	if resolvedDist == "" {
		// Default: look for frontend/dist relative to the binary's working directory.
		resolvedDist = "frontend/dist"
	}
	staticH, err := server.StaticHandler(resolvedDist)
	if err != nil {
		fmt.Fprintf(stderr, "sifty serve: frontend not found at %s — build it first: npm --prefix frontend run build\n", resolvedDist)
		return 1
	}
	mux.Handle("/", staticH)

	handler := server.Handler(servePort, mux)

	// Build the Playwright inspect function. The worker runs as a Node subprocess.
	// resolvedWorkersDir is the workers/ directory; the playwright script is one level below it.
	resolvedWorkersDir, _ := filepath.Abs(*workersDir)
	rawInspectFn := newPlaywrightInspectFn(resolvedWorkersDir)
	// Wrap with the browser discipline: per-host pacing, global concurrency limit and per-host
	// cooldown all live here in the long-lived Go process, not in the ephemeral Node subprocess.
	discipline := background.NewBrowserDiscipline(cfgHolder, background.SystemClock{})
	inspectFn := discipline.Wrap(rawInspectFn)

	// Start background workers.
	assessW := background.NewAssessmentWorker(s, d, cfgHolder, nil, nil)
	prepW := background.NewPreparationWorker(s, d, cfgHolder, nil, nil, inspectFn)
	sched := background.NewDailyScheduler(s, d, cfgHolder, nil, nil)
	assessW.Start()
	prepW.Start()
	sched.Start()
	defer func() {
		assessW.Stop()
		prepW.Stop()
		sched.Stop()
	}()

	addr := fmt.Sprintf("127.0.0.1:%d", listenPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Fprintf(stderr, "sifty serve: listen %s: %v\n", addr, err)
		return 1
	}

	srv := &http.Server{Handler: handler}
	fmt.Fprintf(stdout, "Catalogue: http://127.0.0.1:%d\n", servePort)
	log.SetOutput(stderr)

	// Shutdown on SIGINT or SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()

	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(stderr, "sifty serve: %v\n", err)
		return 1
	}
	return 0
}

// newPlaywrightInspectFn returns an InspectFn that invokes the Playwright worker subprocess.
// workersDir is the path to the workers/ directory (containing playwright/, stub/, etc.).
// The worker runs with Dir set to the parent of workersDir (the backend root) so that Node
// can resolve the playwright package via the standard node_modules search path.
func newPlaywrightInspectFn(workersDir string) background.InspectFn {
	backendRoot := filepath.Dir(workersDir)
	scriptPath := filepath.Join(workersDir, "playwright", "index.mjs")
	return func(ctx context.Context, applyURL string) (inspect.Questionnaire, error) {
		w := worker.Worker{
			Command: "node",
			Args:    []string{scriptPath},
			Dir:     backendRoot,
			Timeout: 90 * time.Second,
		}
		return inspect.InspectForm(ctx, w, inspect.Params{URL: applyURL})
	}
}
