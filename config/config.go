// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

import "time"

type Config struct {
	Poll            time.Duration `config:"poll"`
	StateFile       string        `config:"state_file"`
	HomeSeerLogPath string        `config:"homeseer_log_path"`
	LogBatchSize    int           `config:"log_batch_size"`
}

var DefaultConfig = Config{
	Poll:            5 * time.Second,
	StateFile:       "homeseerbeat_state.json",
	HomeSeerLogPath: "/usr/local/HomeSeer/Data/HomeSeerLog.hsd",
	LogBatchSize:    1000,
}
