package cmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/getoptimum/mump2p-cli/internal/auth"
	"github.com/getoptimum/mump2p-cli/internal/config"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/spf13/cobra"
)

type Snapshot struct {
	Algorithm           string             `json:"algorithm"`
	ActiveNodes         int                `json:"active_nodes"`
	UnhealthyNodes      int                `json:"unhealthy_nodes"`
	PublishedMessages   int64              `json:"published_messages"`
	DeliveredMessages   int64              `json:"delivered_messages"`
	DuplicateMessages   int64              `json:"duplicate_messages"`
	AverageDelaySeconds float64            `json:"average_delay_seconds"`
	P75                 float64            `json:"p75"`
	P95                 float64            `json:"p95"`
	TotalBytesMoved     uint64             `json:"total_bytes_moved"`
	BloatFactor         float64            `json:"bloat_factor"`
	IdealByteComplexity uint64             `json:"ideal_byte_complexity"`
	LastUpdated         time.Time          `json:"last_updated"`
	Messages            map[string]MsgInfo `json:"messages"`
	WindowSeconds       int                `json:"window_seconds"`
}

type MsgInfo struct {
	Topic      string              `json:"topic"`
	DelaySec   float64             `json:"delay_sec"`
	PeersSeen  map[string]struct{} `json:"peers_seen"`
	BytesMoved uint64              `json:"bytes_moved"`
}

var (
	tracerServiceURL string
	tracerWindow     string
	tracerTickMs     int
	tracerRows       int
)

var tracerCmd = &cobra.Command{
	Use:   "tracer",
	Short: "Interactive tracer dashboard",
}

var (
	loadEndpoint2 string
	loadTopic     string
	loadSize      int
	loadCount     int
	loadInterval  int
)

var tracerDashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Open the tracer TUI dashboard",
	RunE: func(cmd *cobra.Command, args []string) error {
		baseURL := resolveServiceURL(tracerServiceURL)

		jwtToken, err := resolveJWT()
		if err != nil {
			return err
		}

		if tracerTickMs <= 0 {
			tracerTickMs = 500
		}
		tick := time.Duration(tracerTickMs) * time.Millisecond
		if tracerWindow == "" {
			tracerWindow = "10s"
		}
		if tracerRows <= 0 {
			tracerRows = 12
		}

		return runTracerDashboard(baseURL, jwtToken, tracerWindow, tick, tracerRows)
	},
}

var tracerResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset tracer statistics on the proxy",
	RunE: func(cmd *cobra.Command, args []string) error {
		baseURL := resolveServiceURL(tracerServiceURL)

		jwtToken, err := resolveJWT()
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return resetStats(ctx, baseURL, jwtToken)
	},
}

var tracerLoadCmd = &cobra.Command{
	Use:   "load",
	Short: "Generate random traffic to the proxy for tracer",
	RunE: func(cmd *cobra.Command, args []string) error {
		baseURL := resolveServiceURL(tracerServiceURL)

		jwtToken, clientID, err := resolveJWTAndClientID()
		if err != nil {
			return err
		}

		if loadTopic == "" {
			loadTopic = "demo"
		}
		if loadSize <= 0 {
			loadSize = 850_000
		}
		if loadCount <= 0 {
			loadCount = 50
		}
		if loadInterval < 0 {
			loadInterval = 500
		}
		if err := proxySubscribe(baseURL, jwtToken, loadTopic, clientID, 1); err != nil {
		}
		if loadEndpoint2 != "" && loadEndpoint2 != baseURL {
			_ = proxySubscribe(loadEndpoint2, jwtToken, loadTopic, clientID, 1)
		}
		for i := 0; i < loadCount; i++ {
			_ = proxyPublishRandom(baseURL, jwtToken, clientID, loadTopic, uint64(loadSize))
			if loadEndpoint2 != "" {
				_ = proxyPublishRandom(loadEndpoint2, jwtToken, clientID, loadTopic, uint64(loadSize))
			}
			if loadInterval > 0 {
				time.Sleep(time.Duration(loadInterval) * time.Millisecond)
			}
		}
		fmt.Println("load generation completed")
		return nil
	},
}

func init() {
	// Attach commands
	rootCmd.AddCommand(tracerCmd)
	tracerCmd.AddCommand(tracerDashboardCmd)
	tracerCmd.AddCommand(tracerResetCmd)
	tracerCmd.AddCommand(tracerLoadCmd)

	// Flags shared by tracer subcommands
	tracerCmd.PersistentFlags().StringVar(&tracerServiceURL, "service-url", "", "Override the default service URL")

	// Dashboard-only flags
	tracerDashboardCmd.Flags().StringVar(&tracerWindow, "window", "10s", "Sliding window size (e.g. 10s, 1m)")
	tracerDashboardCmd.Flags().IntVar(&tracerTickMs, "tick-ms", 500, "UI refresh tick in milliseconds")
	tracerDashboardCmd.Flags().IntVar(&tracerRows, "rows", 12, "Max recent message rows to show")

	// Load flags
	tracerLoadCmd.Flags().StringVar(&loadEndpoint2, "endpoint2", "", "Optional second proxy endpoint for comparison")
	tracerLoadCmd.Flags().StringVar(&loadTopic, "topic", "demo", "Topic to publish to")
	tracerLoadCmd.Flags().IntVar(&loadSize, "size", 850000, "Random message size in bytes")
	tracerLoadCmd.Flags().IntVar(&loadCount, "count", 50, "Number of messages to send")
	tracerLoadCmd.Flags().IntVar(&loadInterval, "interval-ms", 500, "Interval between messages in milliseconds")
}

type history struct {
	values []float64
	limit  int
	mu     sync.Mutex
}

func newHistory(limit int) *history { return &history{limit: limit} }
func (h *history) push(v float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.values = append(h.values, v)
	if len(h.values) > h.limit {
		h.values = h.values[len(h.values)-h.limit:]
	}
}
func (h *history) snapshot() []float64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]float64, len(h.values))
	copy(out, h.values)
	return out
}

func humanBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
func pct(a, b int) int {
	if a+b == 0 {
		return 0
	}
	return int(float64(a) / float64(a+b) * 100)
}
func safeDiv(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}

func streamSnapshots(ctx context.Context, base, jwt, window string, out chan<- Snapshot, statusCh chan<- string, backoffStart, backoffMax time.Duration) {
	defer close(out)
	url := fmt.Sprintf("%s/api/v1/tracer/stream?window=%s", strings.TrimRight(base, "/"), window)
	client := &http.Client{Timeout: 0}
	backoff := backoffStart

	for {
		if ctx.Err() != nil {
			return
		}
		statusCh <- fmt.Sprintf("[Connecting] %s", url)

		req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
		req.Header.Set("Accept", "text/event-stream")
		if !IsAuthDisabled() && jwt != "" {
			req.Header.Set("Authorization", "Bearer "+jwt)
		}

		resp, err := client.Do(req)
		if err != nil {
			statusCh <- fmt.Sprintf("[Error] %v -> retrying in %s", err, backoff)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return
			}
			if backoff < backoffMax {
				backoff *= 2
				if backoff > backoffMax {
					backoff = backoffMax
				}
			}
			continue
		}
		if resp.StatusCode != http.StatusOK {
			_ = resp.Body.Close()
			statusCh <- fmt.Sprintf("[HTTP %d] -> retrying in %s", resp.StatusCode, backoff)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return
			}
			if backoff < backoffMax {
				backoff *= 2
				if backoff > backoffMax {
					backoff = backoffMax
				}
			}
			continue
		}

		backoff = backoffStart
		statusCh <- "[Connected] Streaming… (press 'q' to quit)"

		reader := bufio.NewReader(resp.Body)
		var dataBuf bytes.Buffer
		for {
			if ctx.Err() != nil {
				_ = resp.Body.Close()
				return
			}
			line, err := reader.ReadBytes('\n')
			if err != nil {
				_ = resp.Body.Close()
				if err == io.EOF {
					statusCh <- "[Disconnected] server closed stream; reconnecting…"
				} else {
					statusCh <- fmt.Sprintf("[Read error] %v; reconnecting…", err)
				}
				break
			}
			trim := bytes.TrimRight(line, "\r\n")
			if len(trim) == 0 {
				if dataBuf.Len() > 0 {
					var snap Snapshot
					if json.Unmarshal(dataBuf.Bytes(), &snap) == nil {
						out <- snap
					}
					dataBuf.Reset()
				}
				continue
			}
			if bytes.HasPrefix(trim, []byte("data:")) {
				data := bytes.TrimSpace(trim[len("data:"):])
				if dataBuf.Len() > 0 {
					dataBuf.WriteByte('\n')
				}
				dataBuf.Write(data)
			}
		}
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return
		}
		if backoff < backoffMax {
			backoff *= 2
			if backoff > backoffMax {
				backoff = backoffMax
			}
		}
	}
}

func runTracerDashboard(baseURL, jwt, window string, tick time.Duration, maxRows int) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := ui.Init(); err != nil {
		fmt.Println("UI init error:", err)
		return nil
	}
	defer ui.Close()

	header := widgets.NewParagraph()
	header.Title = " Optimum Tracer Dashboard "
	header.Text = "initializing…"
	header.BorderStyle = ui.NewStyle(ui.ColorCyan)

	nodesGauge := widgets.NewGauge()
	nodesGauge.Title = " Active vs Unhealthy "
	nodesGauge.Percent = 0
	nodesGauge.BarColor = ui.ColorGreen
	nodesGauge.Label = "—"
	nodesGauge.BorderStyle = ui.NewStyle(ui.ColorWhite)

	msgBars := widgets.NewBarChart()
	msgBars.Title = " Messages "
	msgBars.Labels = []string{"Published", "Delivered", "Dupes"}
	msgBars.Data = []float64{0, 0, 0}
	msgBars.BarWidth = 9
	msgBars.BarGap = 3
	msgBars.BorderStyle = ui.NewStyle(ui.ColorWhite)
	msgBars.NumStyles = []ui.Style{
		ui.NewStyle(ui.ColorBlack),
		ui.NewStyle(ui.ColorBlack),
		ui.NewStyle(ui.ColorBlack),
	}

	latBars := widgets.NewBarChart()
	latBars.Title = " Latency (ms) "
	latBars.Labels = []string{"avg", "p75", "p95"}
	latBars.Data = []float64{0, 0, 0}
	latBars.BarWidth = 7
	latBars.BarGap = 3
	latBars.BorderStyle = ui.NewStyle(ui.ColorWhite)
	latBars.NumStyles = []ui.Style{
		ui.NewStyle(ui.ColorBlack),
		ui.NewStyle(ui.ColorBlack),
		ui.NewStyle(ui.ColorBlack),
	}

	bytesBox := widgets.NewParagraph()
	bytesBox.Title = " Bytes, Bloat & Ratios "
	bytesBox.Text = "Waiting for data…"
	bytesBox.BorderStyle = ui.NewStyle(ui.ColorWhite)

	table := widgets.NewTable()
	table.Title = " Recent Messages (slowest first) "
	table.TextStyle = ui.NewStyle(ui.ColorWhite)
	table.RowSeparator = false
	table.Rows = [][]string{{"Topic", "Delay(ms)", "Peers", "Bytes", "Delivered/Published"}}
	table.BorderStyle = ui.NewStyle(ui.ColorWhite)

	throughputSpark := widgets.NewSparklineGroup()
	thSpark := widgets.NewSparkline()
	thSpark.Title = "Throughput: Delivered/s"
	thSpark.LineColor = ui.ColorYellow
	throughputSpark.Title = " Activity "
	throughputSpark.Sparklines = []*widgets.Sparkline{thSpark}
	throughputHist := newHistory(120)

	latencySpark := widgets.NewSparklineGroup()
	latSpark := widgets.NewSparkline()
	latSpark.Title = "Latency avg (ms)"
	latSpark.LineColor = ui.ColorCyan
	latencySpark.Title = " Latency Trend "
	latencySpark.Sparklines = []*widgets.Sparkline{latSpark}
	latHist := newHistory(120)

	status := widgets.NewParagraph()
	status.Title = " Status "
	status.Text = "—"
	status.BorderStyle = ui.NewStyle(ui.ColorCyan)

	help := widgets.NewParagraph()
	help.Text = "q / Ctrl+C = quit • r = reset stats • Resize = adaptive layout"
	help.Border = false
	help.TextStyle = ui.NewStyle(ui.ColorCyan)

	grid := ui.NewGrid()
	resize := func() {
		w, h := ui.TerminalDimensions()
		grid.SetRect(0, 0, w, h)
		grid.Set(
			ui.NewRow(0.28,
				ui.NewCol(0.60,
					ui.NewRow(0.40, header),
					ui.NewRow(0.60,
						ui.NewCol(0.50, nodesGauge),
						ui.NewCol(0.50, msgBars),
					),
				),
				ui.NewCol(0.40,
					ui.NewRow(0.60, latBars),
					ui.NewRow(0.40, bytesBox),
				),
			),
			ui.NewRow(0.52,
				ui.NewCol(0.60, table),
				ui.NewCol(0.40,
					ui.NewRow(0.48, throughputSpark),
					ui.NewRow(0.48, latencySpark),
				),
			),
			ui.NewRow(0.20,
				ui.NewCol(0.70, status),
				ui.NewCol(0.30, help),
			),
		)
	}
	resize()

	snapCh := make(chan Snapshot, 4)
	statusCh := make(chan string, 8)

	go streamSnapshots(ctx, strings.TrimRight(baseURL, "/"), jwt, window, snapCh, statusCh, 2*time.Second, 30*time.Second)

	var (
		lastDelivered int64
		lastTickTime  = time.Now()
		lastReset     time.Time
		mu            sync.Mutex
	)

	updateUI := func(s Snapshot) {
		mu.Lock()
		defer mu.Unlock()

		header.Text = fmt.Sprintf(
			" Source: %s  •  Window: %s  •  Algo: %s  •  Updated: %s",
			baseURL,
			window,
			s.Algorithm,
			s.LastUpdated.Format("15:04:05"),
		)

		nodesGauge.Title = fmt.Sprintf(" Active vs Unhealthy  (%d / %d unhealthy) ", s.ActiveNodes, s.UnhealthyNodes)
		nodesGauge.Percent = pct(s.ActiveNodes, s.UnhealthyNodes)
		nodesGauge.Label = fmt.Sprintf("%d%% healthy", nodesGauge.Percent)
		switch {
		case nodesGauge.Percent >= 80:
			nodesGauge.BarColor = ui.ColorGreen
		case nodesGauge.Percent >= 50:
			nodesGauge.BarColor = ui.ColorYellow
		default:
			nodesGauge.BarColor = ui.ColorRed
		}

		msgBars.Data = []float64{
			float64(s.PublishedMessages),
			float64(s.DeliveredMessages),
			float64(s.DuplicateMessages),
		}

		avg := s.AverageDelaySeconds * 1000
		p75 := s.P75 * 1000
		p95 := s.P95 * 1000
		latBars.Data = []float64{avg, p75, p95}

		deliveredPerPeer := safeDiv(float64(s.DeliveredMessages), float64(s.PublishedMessages))

		bytesBox.Text = fmt.Sprintf(
			"Total Bytes:        %s\nBloat Factor:       %.2f\nIdeal Complexity:   %s\nDelivered/Published %.3f",
			humanBytes(s.TotalBytesMoved),
			s.BloatFactor,
			humanBytes(s.IdealByteComplexity),
			deliveredPerPeer,
		)

		type pair struct {
			Topic string
			M     MsgInfo
		}
		var msgs []pair
		for _, m := range s.Messages {
			msgs = append(msgs, pair{Topic: m.Topic, M: m})
		}
		sort.Slice(msgs, func(i, j int) bool { return msgs[i].M.DelaySec > msgs[j].M.DelaySec })

		tableRows := [][]string{{"Topic", "Delay(ms)", "Peers", "Bytes", "Delivered/Published"}}
		for i, p := range msgs {
			if i >= maxRows {
				break
			}
			tableRows = append(tableRows, []string{
				ellipsize(p.Topic, 40),
				fmt.Sprintf("%.2f", p.M.DelaySec*1000),
				fmt.Sprintf("%d", len(p.M.PeersSeen)),
				humanBytes(p.M.BytesMoved),
				fmt.Sprintf("%.3f", deliveredPerPeer),
			})
		}
		table.Rows = tableRows

		now := time.Now()
		deltaDelivered := s.DeliveredMessages - lastDelivered
		elapsed := now.Sub(lastTickTime).Seconds()
		if elapsed > 0 {
			rate := float64(deltaDelivered) / elapsed
			if !math.IsNaN(rate) && !math.IsInf(rate, 0) {
				throughputHist.push(rate)
				thSpark.Data = throughputHist.snapshot()
			}
		}
		lastDelivered = s.DeliveredMessages
		lastTickTime = now

		if avg >= 0 {
			latHist.push(avg)
			latSpark.Data = latHist.snapshot()
		}
	}

	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	uiEvents := ui.PollEvents()
	for {
		select {
		case <-ctx.Done():
			return nil
		case ev := <-uiEvents:
			switch ev.ID {
			case "<Resize>":
				resize()
				ui.Clear()
				ui.Render(grid)
			case "q", "<C-c>":
				return nil
			case "r":
				if time.Since(lastReset) < 2*time.Second {
					status.Text = "Reset pressed too quickly; wait a moment…"
					ui.Render(grid)
					break
				}
				lastReset = time.Now()
				go func() {
					statusCh <- "[Reset] Sending reset to server…"
					if err := resetStats(ctx, baseURL, jwt); err != nil {
						statusCh <- fmt.Sprintf("[Reset error] %v", err)
						return
					}
					mu.Lock()
					lastDelivered = 0
					lastTickTime = time.Now()
					throughputHist.values = nil
					thSpark.Data = throughputHist.snapshot()
					latHist.values = nil
					latSpark.Data = latHist.snapshot()
					mu.Unlock()
					statusCh <- "[Reset] Done."
				}()
			}
		case s, ok := <-snapCh:
			if !ok {
				status.Text = "Stream ended. Waiting for reconnect… (press 'q' to quit)"
				ui.Render(grid)
				continue
			}
			updateUI(s)
			ui.Render(grid)
		case st := <-statusCh:
			status.Text = st
			ui.Render(grid)
		case <-ticker.C:
			ui.Render(grid)
		}
	}
}

func ellipsize(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 1 {
		return "…"
	}
	return s[:max-1] + "…"
}

func resetStats(ctx context.Context, base, jwt string) error {
	url := fmt.Sprintf("%s/api/v1/tracer/reset", strings.TrimRight(base, "/"))
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	if !IsAuthDisabled() && jwt != "" {
		req.Header.Set("Authorization", "Bearer "+jwt)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("reset failed: HTTP %d %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

// resolveServiceURL matches the pattern used in publish and other commands:
// it starts from config.ServiceUrl and allows overriding with a flag.
func resolveServiceURL(override string) string {
	base := config.LoadConfig().ServiceUrl
	if override != "" {
		base = override
	}
	return base
}

func loadTokenAndClaims(storagePath string) (tokenStr string, claims *auth.TokenClaims, err error) {
	authClient := auth.NewClient()
	storage := auth.NewStorageWithPath(storagePath)

	token, err := authClient.GetValidToken(storage)
	if err != nil {
		return "", nil, fmt.Errorf("authentication required: %v", err)
	}

	parser := auth.NewTokenParser()
	claims, err = parser.ParseToken(token.Token)
	if err != nil {
		return "", nil, fmt.Errorf("error parsing token: %v", err)
	}
	if !claims.IsActive {
		return "", nil, fmt.Errorf("your account is inactive, please contact support")
	}

	return token.Token, claims, nil
}

// resolveJWT returns the bearer token to use for HTTP calls:
// - when auth is enabled, it loads the stored token and validates it (same flow as publish)
// - when auth is disabled, it returns an empty string (no Authorization header).
func resolveJWT() (string, error) {
	if IsAuthDisabled() {
		return "", nil
	}

	tokenStr, _, err := loadTokenAndClaims(GetAuthPath())
	return tokenStr, err
}

// resolveJWTAndClientID is like resolveJWT but also returns the clientID to use:
// - when auth is enabled, clientID comes from the token claims
// - when auth is disabled, clientID must be provided via the global --client-id flag.
func resolveJWTAndClientID() (string, string, error) {
	if IsAuthDisabled() {
		clientID := GetClientID()
		if clientID == "" {
			return "", "", fmt.Errorf("--client-id is required when using --disable-auth")
		}
		return "", clientID, nil
	}

	tokenStr, claims, err := loadTokenAndClaims(GetAuthPath())
	if err != nil {
		return "", "", err
	}
	return tokenStr, claims.ClientID, nil
}

func proxyPublishRandom(base, jwt, clientID, topic string, length uint64) error {
	url := strings.TrimRight(base, "/") + "/api/v1/publish"
	body := map[string]any{
		"client_id":      clientID,
		"topic":          topic,
		"message_length": length,
	}
	reqBytes, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", url, bytes.NewReader(reqBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if !IsAuthDisabled() && jwt != "" {
		req.Header.Set("Authorization", "Bearer "+jwt)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("publish error: HTTP %d %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

func proxySubscribe(base, jwt, topic, clientID string, threshold int) error {
	url := strings.TrimRight(base, "/") + "/api/v1/subscribe"
	body := map[string]any{
		"client_id": clientID,
		"topic":     topic,
		"threshold": threshold,
	}
	reqBytes, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", url, bytes.NewReader(reqBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if !IsAuthDisabled() && jwt != "" {
		req.Header.Set("Authorization", "Bearer "+jwt)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck
	// ignore non-2xx errors intentionally; load can still proceed
	return nil
}
