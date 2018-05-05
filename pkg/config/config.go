package config

import (
	"github.com/spf13/viper"
)

const (
	KeyInformerCacheSyncTimeoutSeconds = "informer_cache_sync_timeout_seconds"
	KeyScope                           = "scope"
)

func init() {
	viper.SetEnvPrefix("K8S_QUEUE_ENTRY_OPERATOR")
	viper.SetDefault(KeyInformerCacheSyncTimeoutSeconds, "30")
	viper.SetDefault(KeyScope, "")
	viper.AutomaticEnv()
}
