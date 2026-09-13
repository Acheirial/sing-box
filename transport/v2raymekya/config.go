package v2raymekya

import (
	"github.com/sagernet/sing-box/transport/v2raymkcp"
)

type Config struct {
	KCP                            v2raymkcp.Config
	URL                            string
	H2PoolSize                     int
	MaxWriteDelay                  int
	MaxRequestSize                 int
	PollingIntervalInitial         int
	MaxWriteSize                   int
	MaxWriteDurationMs             int
	MaxSimultaneousWriteConnection int
	PacketWritingBuffer            int
}
