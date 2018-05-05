package config

import (
	"github.com/spf13/viper"
	"os"
	"testing"
)

func Test_init_informerCacheSyncTimeoutSeconds_default(t *testing.T) {
	actual := viper.GetInt(KeyInformerCacheSyncTimeoutSeconds)

	expected := 30

	if actual != expected {
		t.Errorf("got: %v, want: %v", actual, expected)
	}
}

func Test_init_informerCacheSyncTimeoutSeconds_set(t *testing.T) {
	envKey := "K8S_QUEUE_ENTRY_OPERATOR_INFORMER_CACHE_SYNC_TIMEOUT_SECONDS"

	old := os.Getenv(envKey)
	defer func() {
		os.Setenv(envKey, old)
	}()
	os.Setenv(envKey, "42")

	actual := viper.GetInt(KeyInformerCacheSyncTimeoutSeconds)

	expected := 42

	if actual != expected {
		t.Errorf("got: %v, want: %v", actual, expected)
	}
}

func Test_init_scope(t *testing.T) {
	envKey := "K8S_QUEUE_ENTRY_OPERATOR_SCOPE"

	old := os.Getenv(envKey)
	defer func() {
		os.Setenv(envKey, old)
	}()
	os.Setenv(envKey, "dev")

	actual := viper.GetString(KeyScope)

	expected := "dev"

	if actual != expected {
		t.Errorf("got: %s, want: %s", actual, expected)
	}
}
