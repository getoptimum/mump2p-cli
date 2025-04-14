package cmd

import (
	"fmt"

	"github.com/getoptimum/optcli/internal/service"
	"github.com/spf13/cobra"
)

var (
	subTopic     string
	subAlgorithm string
)

var subscribeCmd = &cobra.Command{
	Use:   "subscribe",
	Short: "Subscribe to a topic directly using OptimumP2P service",
	RunE: func(cmd *cobra.Command, args []string) error {
		s := service.GetP2PService(ConfigPath)

		totalOpt, totalGossip := s.TotalP2PNodes()
		// one of the algo either optimump2p or gossibsub
		subscribedOpt, subscribedGossip := s.SubscribeToTopic(subTopic, []string{subAlgorithm})

		fmt.Println("✅ Subscribed")
		fmt.Printf("Topic: %s\n", subTopic)
		fmt.Printf("Total Nodes: Optimum=%d, Gossip=%d\n", totalOpt, totalGossip)
		fmt.Printf("Subscribed:  Optimum=%d, Gossip=%d\n", subscribedOpt, subscribedGossip)

		// TODO:: discuss, callback messages (block code)
		return nil
	},
}

func init() {
	subscribeCmd.Flags().StringVar(&subTopic, "topic", "", "Topic to subscribe to")
	subscribeCmd.Flags().StringVar(&subAlgorithm, "algorithm", "", "Protocol to use (optimump2p or gossipsub)")
	subscribeCmd.MarkFlagRequired("topic")
	subscribeCmd.MarkFlagRequired("algorithm")
	rootCmd.AddCommand(subscribeCmd)
}
