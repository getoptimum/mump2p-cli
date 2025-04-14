package service

import (
	"context"
	"log"

	"github.com/getoptimum/optimum-p2p/pkg/config"
	"github.com/getoptimum/optimum-p2p/pkg/logger"
	"github.com/getoptimum/optimum-p2p/pkg/service/p2p_proxier"
)

var srv *p2p_proxier.Service

func GetP2PService(configPath string) *p2p_proxier.Service {
	if srv != nil {
		return srv
	}

	ctx := context.Background()
	// TODO:: define logger configuration in config
	appLog := logger.NewAppSLogger("mump2p")
	// TODO:: config change
	appConf, err := config.InitConf(configPath)
	if err != nil {
		log.Fatalf("unable to load config: %v", err)
	}
	// TODO:: p2p_proxier make config modular
	srv = p2p_proxier.NewService(ctx, appLog, appConf)
	go srv.Run()
	return srv
}
