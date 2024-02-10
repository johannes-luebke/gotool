package log

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	toolio "github.com/johannes-luebke/gotool/pkg/io"
)

const (
	defaultMaxFiles = 5
	defaultFileName = "app"
)

var (
	Log *slog.Logger // global logger

	logFolder string // log folder path
	logFile   string // log file path
)

type Options struct {
	UserDir     string // User directory. Log file is stored in <UserDir>/log
	Prefix      string // Prefix for log file name. <Prefix>.log.json
	ShowDebug   bool   // Show debug logs
	MaxLogFiles int    // Maximum number of log files
}

func Start(logOpts *Options) error {
	// Check log options
	if logOpts.UserDir == "" {
		return fmt.Errorf("user directory cannot be empty")
	}
	if logOpts.Prefix == "" {
		logOpts.Prefix = defaultFileName
	}
	if logOpts.MaxLogFiles < 1 {
		logOpts.MaxLogFiles = defaultMaxFiles
	}
	// Get log file path
	logFolder = filepath.Join(logOpts.UserDir, "log")
	logFileName := fmt.Sprintf("%s.log.json", logOpts.Prefix)
	logFile = filepath.Join(logFolder, logFileName)

	// Create log folder if it doesn't exist
	if _, err := os.Stat(logFolder); os.IsNotExist(err) {
		err = os.MkdirAll(logFolder, toolio.Perm755)
		if err != nil {
			return err
		}
	}
	// Roll log file
	err := rollLogFile(logFile, logOpts)
	if err != nil {
		return err
	}
	// Open log file
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, toolio.Perm666)
	if err != nil {
		return err
	}
	// Set log level
	logLevel := slog.LevelInfo
	if logOpts.ShowDebug {
		logLevel = slog.LevelDebug
	}
	// Create logger
	writer := io.MultiWriter(os.Stderr, f)
	jsonHandler := slog.NewJSONHandler(writer, &slog.HandlerOptions{Level: logLevel, AddSource: true})
	Log = slog.New(jsonHandler)

	Log.Debug("Successfully initialized the Logger.", "log file", logFile, "logger level", logLevel)
	return nil
}

func Must(logOpts *Options) {
	err := Start(logOpts)
	if err != nil {
		panic(err)
	}
}

// Handles log rotation.
//
//	On startup, a new log file is being created.
//	If the log file already exists, it is renamed to `<name>.log.json.1`.
//	If the new log file would exceed the maximum number of log files, the oldest log file is deleted.
func rollLogFile(logFile string, logOpts *Options) error {
	// Ignore if log file doesn't exist
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		return nil
	}
	// Get log number
	parts := strings.Split(logFile, ".")
	suffix := parts[len(parts)-1]
	logNumber := 0
	if suffix != "json" {
		var err error
		logNumber, err = strconv.Atoi(suffix)
		if err != nil {
			return err
		}
	}
	// Get next log number
	nextLogNumber := logNumber + 1
	// Delete old log file
	if nextLogNumber == logOpts.MaxLogFiles {
		err := os.Remove(logFile)
		if err != nil {
			return err
		}
		return nil
	}
	// Get new log file name
	var newLogFile string
	if logNumber == 0 {
		newLogFile = strings.Join(parts, ".") + ".1"
	} else {
		newLogFile = strings.Join(parts[:len(parts)-1], ".") + "." + strconv.Itoa(nextLogNumber)
	}
	// Rollover older log file
	err := rollLogFile(newLogFile, logOpts)
	if err != nil {
		return err
	}
	// Rename log file
	err = os.Rename(logFile, newLogFile)
	if err != nil {
		return err
	}

	return nil
}

// Returns the logs from the log file.
//
// Each line of the log file is a json object,
// which is unmarshalled into a map.
func GetLogs() ([]map[string]interface{}, error) {
	// Open log file
	file, err := os.Open(logFile)
	if err != nil {
		Log.Error("Failed to open the log file.", "error", err, "log file", logFile)
		return nil, err
	}
	defer file.Close()
	// Read log file
	logs := make([]map[string]interface{}, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		l := make(map[string]interface{})
		err := json.Unmarshal(scanner.Bytes(), &l)
		if err != nil {
			Log.Error("Failed to unmarshal log", "error", err)
			return nil, err
		}
		l["_ERROR"] = l["level"] == "ERROR"
		l["_WARN"] = l["level"] == "WARN"
		l["_INFO"] = l["level"] == "INFO"
		l["_DEBUG"] = l["level"] == "DEBUG"
		logs = append(logs, l)
	}
	// Check for errors
	if err := scanner.Err(); err != nil {
		Log.Error("Failed to read the log file.", "error", err, "log file", logFile)
		return nil, err
	}

	return logs, nil
}
