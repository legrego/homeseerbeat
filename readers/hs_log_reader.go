package readers

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"os"
	"time"

	"github.com/elastic/beats/libbeat/logp"

	_ "github.com/mattn/go-sqlite3"
)

type LogEntry struct {
	ID            int
	Datetime      time.Time
	LogFraction   int
	LogType       string
	LogEntry      string
	LogFrom       string
	LogPriority   int
	ErrorCode     int
	LogClassColor string
}

type readerState struct {
	LastID int
}

var db *sql.DB

// InitLogReader initializes this data reader
func InitLogReader(dataPath string) error {
	logp.Info("Initilizing reader. Opening HomeSeer database at %s", dataPath)

	var err error
	db, err = sql.Open("sqlite3", dataPath)
	if err != nil {
		logp.Error(err)
		return err
	}
	return nil
}

// ReadLogs reads the provided HomeSeer log file
func ReadLogs(statePath string, batchSize int) ([]LogEntry, error) {

	logp.Info("Starting log read operation")

	state, err := getState(statePath)
	if err != nil {
		logp.Error(err)
		return nil, err
	}

	logp.Info("Retrieving logs with an ID > %d, using a batch size of %d", state.LastID, batchSize)

	stmt, err := db.Prepare("SELECT ID, Log_DateTime, Log_Time_Fraction, Log_Type, Log_Entry, Log_From, Log_Priority, ErrorCode, Log_ClassColor FROM Log WHERE ID > ? ORDER BY ID LIMIT ?")
	if err != nil {
		logp.Error(err)
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(state.LastID, batchSize)

	var results []LogEntry
	for rows.Next() {
		entry := &LogEntry{}
		err = rows.Scan(&entry.ID, &entry.Datetime, &entry.LogFraction, &entry.LogType, &entry.LogEntry, &entry.LogFrom, &entry.LogPriority, &entry.ErrorCode, &entry.LogClassColor)
		if err != nil {
			logp.Error(err)
			return nil, err
		}
		results = append(results, *entry)
	}

	count := len(results)
	lastID := state.LastID
	if count > 0 {
		lastID = results[count-1].ID

		nextState := &readerState{
			LastID: lastID,
		}

		setState(statePath, *nextState)
	}

	logp.Info("Finished reading %d log entries. Last ID read: %d", len(results), lastID)

	return results, nil
}

// CloseLogReader shuts down the reader
func CloseLogReader() {
	logp.Info("Shutting down HS log reader")
	if db != nil {
		defer db.Close()
	}
}

func getState(statePath string) (readerState, error) {
	var state readerState

	if fileExists(statePath) {
		jsonFile, err := os.Open(statePath)
		if err != nil {
			logp.Error(err)
			return state, err
		}
		defer jsonFile.Close()

		byteValue, err := ioutil.ReadAll(jsonFile)
		if err != nil {
			logp.Error(err)
			return state, err
		}

		err = json.Unmarshal(byteValue, &state)
		if err != nil {
			logp.Error(err)
			return state, err
		}
	} else {
		state.LastID = -1
	}

	return state, nil
}

func setState(statePath string, state readerState) error {
	file, err := json.MarshalIndent(state, "", " ")
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(statePath, file, 0644)
	if err != nil {
		return err
	}
	return nil
}

func fileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
