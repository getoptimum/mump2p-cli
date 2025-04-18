package cmd

// TODO::
// lets fix the gateway socket and add listener
// add dumper for local (flag --path) also exporter for now lets add --webhook (post all the trace on given endpoint)
// either @har_preet or @bergasov do it
// var (
// 	subTopic     string
// 	useWebSocket bool
// )

// var subscribeCmd = &cobra.Command{
// 	Use:   "subscribe",
// 	Short: "Subscribe to a topic via HTTP API (and optionally WebSocket)",
// 	RunE: func(cmd *cobra.Command, args []string) error {
// 		cfg, err := config.LoadConfig(ConfigFile)
// 		if err != nil {
// 			return fmt.Errorf("error loading config: %v", err)
// 		}

// 		authClient := auth.NewClient(cfg.Domain, cfg.ClientID, cfg.Audience, cfg.Scope)
// 		storage := auth.NewStorage()
// 		token, err := authClient.GetValidToken(storage)
// 		if err != nil {
// 			return fmt.Errorf("authentication required: %v", err)
// 		}

// 		parser := auth.NewTokenParser()
// 		claims, err := parser.ParseToken(token.Token)
// 		if err == nil {
// 			limiter, err := ratelimit.NewRateLimiter(claims)
// 			if err == nil {
// 				_ = limiter.RecordSubscribe()
// 			}
// 		}

// 		// signal handling
// 		sigChan := make(chan os.Signal, 1)
// 		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

// 		// webSocket mode
// 		if useWebSocket {
// 			fmt.Println("Opening WebSocket listener...")
// 			wsURL := strings.Replace(cfg.ServiceUrl, "http", "ws", 1) + "/ws/api/v1"

// 			header := http.Header{}
// 			header.Set("Authorization", "Bearer "+token.Token)

// 			conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
// 			if err != nil {
// 				return fmt.Errorf("websocket connection failed: %v", err)
// 			}
// 			defer conn.Close()

// 			fmt.Println("Listening for messages (WebSocket)... Press Ctrl+C to exit")

// 			go func() {
// 				for {
// 					_, msg, err := conn.ReadMessage()
// 					if err != nil {
// 						fmt.Printf("❌ WebSocket read error: %v\n", err)
// 						return
// 					}
// 					fmt.Printf("%s\n", string(msg))
// 				}
// 			}()

// 			<-sigChan
// 			fmt.Println("\nWebSocket closed")
// 			return nil
// 		}

// 		// HTTP polling mode
// 		fmt.Println("Starting HTTP subscribe polling... Press Ctrl+C to exit")
// 		ticker := time.NewTicker(time.Duration(refreshSeconds) * time.Second)
// 		defer ticker.Stop()

// 		tokenRefresh := time.NewTicker(15 * time.Minute)
// 		defer tokenRefresh.Stop()

// 		for {
// 			select {
// 			case <-ticker.C:
// 				// refresh data
// 				reqBody := fmt.Sprintf(`{"topic": "%s", "protocol": ["%s"], "message_size": 0}`, subTopic, subAlgorithm)
// 				request, err := http.NewRequest("POST", cfg.ServiceUrl+"/api/subscribe", strings.NewReader(reqBody))
// 				if err != nil {
// 					fmt.Println("❌ Failed to build request:", err)
// 					continue
// 				}
// 				request.Header.Set("Authorization", "Bearer "+token.Token)
// 				request.Header.Set("Content-Type", "application/json")

// 				resp, err := http.DefaultClient.Do(request)
// 				if err != nil {
// 					fmt.Println("❌ HTTP error:", err)
// 					continue
// 				}
// 				body, _ := io.ReadAll(resp.Body)
// 				resp.Body.Close()

// 				if resp.StatusCode != 200 {
// 					fmt.Printf("❌ Subscribe error: %s\n", string(body))
// 					continue
// 				}
// 				fmt.Printf("[%s] Subscribed successfully: %s\n", time.Now().Format(time.Kitchen), string(body))

// 			case <-tokenRefresh.C:
// 				newToken, err := authClient.GetValidToken(storage)
// 				if err != nil {
// 					fmt.Printf("Token refresh failed: %v\n", err)
// 				} else {
// 					token = newToken
// 					fmt.Println("Token refreshed")
// 				}

// 			case <-sigChan:
// 				fmt.Println("\nExiting HTTP polling")
// 				return nil
// 			}
// 		}
// 	},
// }

// func init() {
// 	subscribeCmd.Flags().StringVar(&subTopic, "topic", "", "Topic to subscribe to")
// 	subscribeCmd.Flags().BoolVar(&useWebSocket, "websocket", false, "Enable WebSocket stream after subscribing")
// 	subscribeCmd.MarkFlagRequired("topic")
// 	subscribeCmd.MarkFlagRequired("algorithm")
// 	rootCmd.AddCommand(subscribeCmd)
// }
