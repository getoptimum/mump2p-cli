package cmd

import (
	"fmt"

	"github.com/getoptimum/optcli/internal/service"
	"github.com/spf13/cobra"
)

var (
	pubTopic     string
	pubMessage   string
	pubAlgorithm string
)

const (
	defaultMessageSize = 2 << 20 // 2MB
)

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish a message to the P2P network using direct service calls",
	RunE: func(cmd *cobra.Command, args []string) error {
		s := service.GetP2PService(ConfigPath)
		messageSize := int64(len(pubMessage))
		// TODO:: discuss default message size
		if messageSize == 0 {
			messageSize = defaultMessageSize // fallback default size
		}

		optNode, gossipNode := s.SendRandomMessage(pubTopic, messageSize, []string{pubAlgorithm})
		fmt.Println("✅ Published")
		fmt.Printf("Topic: %s\n", pubTopic)
		fmt.Printf("Optimum node: %s\n", optNode)
		fmt.Printf("Gossip node:  %s\n", gossipNode)
		return nil
	},
}

func init() {
	publishCmd.Flags().StringVar(&pubTopic, "topic", "", "Topic to publish to")
	publishCmd.Flags().StringVar(&pubMessage, "message", "", "Message string (used to estimate message size)")
	publishCmd.Flags().StringVar(&pubAlgorithm, "algorithm", "", "Protocol to use (optimump2p or gossipsub)")
	publishCmd.MarkFlagRequired("topic")
	publishCmd.MarkFlagRequired("algorithm")
	rootCmd.AddCommand(publishCmd)
}
