package beater

import (
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/legrego/homeseerbeat/config"
	"github.com/legrego/homeseerbeat/readers"
)

// Homeseerbeat configuration.
type Homeseerbeat struct {
	done   chan struct{}
	config config.Config
	client beat.Client
}

// New creates an instance of homeseerbeat.
func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
	c := config.DefaultConfig
	if err := cfg.Unpack(&c); err != nil {
		return nil, fmt.Errorf("Error reading config file: %v", err)
	}

	bt := &Homeseerbeat{
		done:   make(chan struct{}),
		config: c,
	}
	return bt, nil
}

// Run starts homeseerbeat.
func (bt *Homeseerbeat) Run(b *beat.Beat) error {
	logp.Info("homeseerbeat is running! Hit CTRL-C to stop it.")

	var err error
	bt.client, err = b.Publisher.Connect()
	if err != nil {
		return err
	}

	readers.InitLogReader(bt.config.StateFile, bt.config.HomeSeerLogPath)

	ticker := time.NewTicker(bt.config.Poll)
	for {
		select {
		case <-bt.done:
			readers.CloseLogReader()
			return nil
		case <-ticker.C:
		}

		results, err := readers.ReadLogs(bt.config.StateFile, bt.config.LogBatchSize)
		if err != nil {
			return err
		}

		for _, result := range results {
			bt.client.Publish(beat.Event{
				Timestamp: result.Datetime,
				Fields: common.MapStr{
					"event.id":       result.ID,
					"event.module":   result.LogType,
					"event.created":  time.Now(),
					"event.severity": result.LogPriority,
					"message":        result.LogEntry,
				},
			})
		}

		logp.Info("Events sent")
	}
}

// Stop stops homeseerbeat.
func (bt *Homeseerbeat) Stop() {
	bt.client.Close()
	close(bt.done)
}
