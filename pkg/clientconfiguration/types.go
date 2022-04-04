package clientconfiguration

import (
	"github.com/underpin-korea/protocol/livekit"
)

type ClientConfigurationManager interface {
	GetConfiguration(clientInfo *livekit.ClientInfo) *livekit.ClientConfiguration
}
