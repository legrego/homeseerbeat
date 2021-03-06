package readers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/paths"

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
func InitLogReader(stateFile string, dataPath string) error {
	logp.Info("Initilizing reader. Opening HomeSeer database at %s", dataPath)

	var err error
	db, err = sql.Open("sqlite3", dataPath)
	if err != nil {
		logp.Error(err)
		return err
	}

	statePath := paths.Resolve(paths.Data, "")
	logp.Info("Creating state file directory at %s", statePath)
	err = os.MkdirAll(statePath, 0750)
	if err != nil {
		return fmt.Errorf("Failed to created state file dir %s: %v", statePath, err)
	}

	return nil
}

// ReadLogs reads the provided HomeSeer log file
func ReadLogs(stateFile string, batchSize int) ([]LogEntry, error) {

	logp.Info("Starting log read operation")

	state, err := getState(stateFile)
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

		correctedTime, _ := correctTimestamp(entry.Datetime)
		if correctedTime != nil {
			entry.Datetime = *correctedTime
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

		setState(stateFile, *nextState)
	}

	logp.Info("Finished reading %d log entries. Last ID read: %d", len(results), lastID)

	return results, nil
}

func correctTimestamp(timestamp time.Time) (*time.Time, error) {
	// HomeSeer stores timestamps in the user's preferred timezone, but without any timezone information.
	// Therefore, we need to read the timestamp out as UTC, and then manually append the correct timezone to the end.
	// NOTE: This assumes that the preferred timezone stored in HomeSeer matches the timezone of the local system
	zonename, _ := time.Now().In(time.Local).Zone()
	dateAsStr := timestamp.Format(time.RFC822)
	adjustedDateStr := dateAsStr
	if strings.HasSuffix(dateAsStr, " UTC") {
		adjustedDateStr = strings.Replace(dateAsStr, " UTC", " "+zonename, 1)
		adjustedDate, err := time.Parse(time.RFC822, adjustedDateStr)
		if err != nil {
			logp.Error(err)
			return nil, err
		}
		return &adjustedDate, nil
	}

	err := fmt.Errorf("expected %s to end in ' UTC'", dateAsStr)
	logp.Error(err)
	return nil, err
}

// CloseLogReader shuts down the reader
func CloseLogReader() {
	logp.Info("Shutting down HS log reader")
	if db != nil {
		defer db.Close()
	}
}

func getState(stateFile string) (readerState, error) {
	var state readerState

	stateFileLocation := paths.Resolve(paths.Data, stateFile)

	if fileExists(stateFileLocation) {
		jsonFile, err := os.Open(stateFileLocation)
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

func setState(stateFile string, state readerState) error {
	file, err := json.MarshalIndent(state, "", " ")
	if err != nil {
		return err
	}

	stateFileLocation := paths.Resolve(paths.Data, stateFile)

	err = ioutil.WriteFile(stateFileLocation, file, 0644)
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
