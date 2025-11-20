package cmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	tracerServiceURL  string
	tracerWindow      string
	tracerTickMs      int
	tracerRows        int
	dashboardTopic    string
	dashboardSize     int
	dashboardCount    int
	dashboardInterval int
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

		jwtToken, clientID, err := resolveJWTAndClientID()
		if err != nil {
			return err
		}

		return runTracerDashboard(baseURL, jwtToken, clientID, tracerWindow, tick, tracerRows, dashboardTopic, dashboardSize, dashboardCount, dashboardInterval)
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
		_ = proxySubscribe(baseURL, jwtToken, loadTopic, clientID, 1)
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
	rootCmd.AddCommand(tracerCmd)
	tracerCmd.AddCommand(tracerDashboardCmd)
	tracerCmd.AddCommand(tracerResetCmd)
	tracerCmd.AddCommand(tracerLoadCmd)

	tracerCmd.PersistentFlags().StringVar(&tracerServiceURL, "service-url", "", "Override the default service URL")

	tracerDashboardCmd.Flags().StringVar(&tracerWindow, "window", "10s", "Sliding window size (e.g. 10s, 1m)")
	tracerDashboardCmd.Flags().IntVar(&tracerTickMs, "tick-ms", 500, "UI refresh tick in milliseconds")
	tracerDashboardCmd.Flags().IntVar(&tracerRows, "rows", 12, "Max recent message rows to show")
	tracerDashboardCmd.Flags().StringVar(&dashboardTopic, "topic", "demo", "Topic to publish messages to (auto-publish enabled)")
	tracerDashboardCmd.Flags().IntVar(&dashboardSize, "size", 102400, "Random message size in bytes for auto-publish")
	tracerDashboardCmd.Flags().IntVar(&dashboardCount, "count", 60, "Number of messages to auto-publish")
	tracerDashboardCmd.Flags().IntVar(&dashboardInterval, "interval-ms", 500, "Interval between auto-published messages in milliseconds")

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
func (h *history) reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.values = nil
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
		msgCount := 0
		for {
			if ctx.Err() != nil {
				_ = resp.Body.Close()
				return
			}
			line, err := reader.ReadBytes('\n')
			if err != nil {
				_ = resp.Body.Close()
				if err == io.EOF {
					statusCh <- fmt.Sprintf("[Disconnected] server closed stream after %d messages; reconnecting…", msgCount)
				} else {
					statusCh <- fmt.Sprintf("[Read error] %v; reconnecting…", err)
				}
				break
			}
			trim := bytes.TrimRight(line, "\r\n")
			if len(trim) == 0 {
				if dataBuf.Len() > 0 {
					var snap Snapshot
					if err := json.Unmarshal(dataBuf.Bytes(), &snap); err == nil {
						msgCount++
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

func runTracerDashboard(baseURL, jwt, clientID, window string, tick time.Duration, maxRows int, topic string, size, count, interval int) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := ui.Init(); err != nil {
		return fmt.Errorf("failed to init tracer UI: %w", err)
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

	table := widgets.NewTable()
	table.Title = " Recent Messages (slowest first) "
	table.TextStyle = ui.NewStyle(ui.ColorWhite)
	table.RowSeparator = false
	table.Rows = [][]string{{"Topic", "Delay(ms)", "Peers", "Bytes", "Delivered/Published"}}
	table.BorderStyle = ui.NewStyle(ui.ColorWhite)

	status := widgets.NewParagraph()
	status.Title = " Status "
	status.Text = "—"
	status.BorderStyle = ui.NewStyle(ui.ColorCyan)

	help := widgets.NewParagraph()
	help.Text = "q / Ctrl+C = quit • r = reset stats • Resize = adaptive layout"
	help.Border = false
	help.TextStyle = ui.NewStyle(ui.ColorCyan)

	resize := func() {
		w, h := ui.TerminalDimensions()
		header.SetRect(0, 0, w, 3)
		status.SetRect(0, h-3, w, h)
		nodesGauge.SetRect(0, 3, w/2, 8)
		msgBars.SetRect(w/2, 3, w, 8)
		table.SetRect(0, 8, w, h-3)
	}
	resize()

	snapCh := make(chan Snapshot, 4)
	statusCh := make(chan string, 8)

	go streamSnapshots(ctx, strings.TrimRight(baseURL, "/"), jwt, window, snapCh, statusCh, 2*time.Second, 30*time.Second)

	if count > 0 {
		go func() {
			statusCh <- fmt.Sprintf("[Auto-publish] Starting: %d messages to topic '%s'", count, topic)
			for i := 0; i < count; i++ {
				select {
				case <-ctx.Done():
					return
				default:
					if err := proxyPublishRandom(baseURL, jwt, clientID, topic, uint64(size)); err != nil {
						statusCh <- fmt.Sprintf("[Auto-publish error] %v", err)
					}
					if i < count-1 {
						time.Sleep(time.Duration(interval) * time.Millisecond)
					}
				}
			}
			statusCh <- fmt.Sprintf("[Auto-publish] Completed: %d messages sent", count)
		}()
	}

	var (
		lastReset time.Time
		mu        sync.Mutex
	)

	updateUI := func(s Snapshot) {
		mu.Lock()
		defer mu.Unlock()

		header.Text = fmt.Sprintf(
			" Source: %s  •  Window: %s  •  Algo: %s  •  Updated: %s",
			baseURL,
			window,
			s.Algorithm,
			s.LastUpdated.Format("15:04:05"), // 24-hour format HH:MM:SS
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

		deliveredPerPeer := safeDiv(float64(s.DeliveredMessages), float64(s.PublishedMessages))

		// Calculate latency metrics
		avgMs := s.AverageDelaySeconds * 1000
		p75Ms := s.P75 * 1000
		p95Ms := s.P95 * 1000

		// Update table title with latency metrics
		table.Title = fmt.Sprintf(
			" Recent Messages • P95: %.2fms • P75: %.2fms • Avg: %.2fms ",
			p95Ms, p75Ms, avgMs,
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
				ui.Render(header, status, nodesGauge, msgBars, table)
			case "q", "<C-c>":
				return nil
			case "r":
				if time.Since(lastReset) < 2*time.Second {
					status.Text = "Reset pressed too quickly; wait a moment…"
					ui.Render(header, status, nodesGauge, msgBars, table)
					break
				}
				lastReset = time.Now()
				go func() {
					statusCh <- "[Reset] Sending reset to server…"
					if err := resetStats(ctx, baseURL, jwt); err != nil {
						statusCh <- fmt.Sprintf("[Reset error] %v", err)
						return
					}
					statusCh <- "[Reset] Done."
				}()
			}
		case s, ok := <-snapCh:
			if !ok {
				status.Text = "Stream ended. Waiting for reconnect… (press 'q' to quit)"
				ui.Render(header, status, nodesGauge, msgBars, table)
				continue
			}
			updateUI(s)
			ui.Render(header, status, nodesGauge, msgBars, table)
		case st := <-statusCh:
			status.Text = st
		case <-ticker.C:
			ui.Render(header, status, nodesGauge, msgBars, table)
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
	if jwt != "" {
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

func resolveJWT() (string, error) {
	if IsAuthDisabled() {
		return "", nil
	}

	tokenStr, _, err := loadTokenAndClaims(GetAuthPath())
	return tokenStr, err
}

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
	reqBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encoding request body: %w", err)
	}
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
	reqBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encoding request body: %w", err)
	}
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
	return nil
}
