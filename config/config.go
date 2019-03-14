// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

import "time"

type Config struct {
	Poll            time.Duration `config:"poll"`
	StatePath       string        `config:"state_path"`
	HomeSeerLogPath string        `config:"homeseer_log_path"`
	LogBatchSize    int           `config:"log_batch_size"`
}

var DefaultConfig = Config{
	Poll:            5 * time.Second,
	StatePath:       "./homeseerbeat_state.json",
	HomeSeerLogPath: "/usr/local/HomeSeer/Data/HomeSeerLog.hsd",
	LogBatchSize:    1000,
}
