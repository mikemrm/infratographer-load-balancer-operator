package srv

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/chart"
	"k8s.io/client-go/rest"
)

// Server holds options for server connectivity and settings
type Server struct {
	Gin             *gin.Engine
	Context         context.Context
	StreamName      string
	Logger          *zap.SugaredLogger
	KubeClient      *rest.Config
	JetstreamClient nats.JetStreamContext
	Debug           bool
	Prefix          string
	Subjects        []string
	Subscriptions   []*nats.Subscription
	Chart           *chart.Chart
	ChartPath       string
	ValuesPath      string
}

// Run will start the server queue connections and healthcheck endpoints
func (s *Server) Run(ctx context.Context) error {
	if err := s.configureSubscribers(); err != nil {
		s.Logger.Errorw("unable to configure subscribers", "error", err)
		return err
	}

	s.configureHealthcheck()

	go func() {
		if err := s.Gin.Run(viper.GetString("healthcheck-port")); err != nil {
			s.Logger.Errorw("unable to start healthcheck server", "error", err)
		}
	}()

	s.Logger.Infow("starting server")

	return nil
}

func (s *Server) configureSubscribers() error {
	for _, subject := range s.Subjects {
		hash := md5.Sum([]byte(subject))

		subscription, err := s.JetstreamClient.QueueSubscribe(fmt.Sprintf("%s.%s", s.Prefix, subject), "loadbalanceroperator-workers"+hex.EncodeToString(hash[:]), s.messageRouter, nats.BindStream(s.StreamName))
		if err != nil {
			s.Logger.Errorw("unable to subscribe to queue", "queue", subject, "error", err)
			return err
		}

		s.Subscriptions = append(s.Subscriptions, subscription)
	}

	return nil
}
