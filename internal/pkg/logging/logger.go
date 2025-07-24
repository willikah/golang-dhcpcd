package logging

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
)

var Logger *logrus.Logger

// LogConfig represents logging configuration
type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"` // json, text, simple, or compact
}

// CompactFormatter implements a custom formatter for compact logging
type CompactFormatter struct {
	ShowTime bool
}

// Format renders a single log entry
func (f *CompactFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	// Add timestamp if required
	if f.ShowTime {
		b.WriteString(fmt.Sprintf("[%s]", entry.Time.Format("15:04:05")))
	}

	// Add log level
	level := strings.ToUpper(entry.Level.String())
	b.WriteString(fmt.Sprintf("[%s]", level))

	// Add component and interface in brackets
	component, hasComponent := entry.Data["component"]
	iface, hasInterface := entry.Data["interface"]

	if hasComponent {
		b.WriteString(fmt.Sprintf("[%s]", component))
	}
	if hasInterface {
		b.WriteString(fmt.Sprintf("[%s]", iface))
	}

	// Add space before message
	b.WriteString(" ")

	// Add message
	b.WriteString(entry.Message)

	// Add remaining fields in sorted order (excluding component and interface)
	remainingFields := make(map[string]interface{})
	for k, v := range entry.Data {
		if k != "component" && k != "interface" {
			remainingFields[k] = v
		}
	}

	if len(remainingFields) > 0 {
		b.WriteString(" (")

		// Sort fields for consistent output
		keys := make([]string, 0, len(remainingFields))
		for k := range remainingFields {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		first := true
		for _, key := range keys {
			if !first {
				b.WriteString(", ")
			}
			b.WriteString(fmt.Sprintf("%s=%v", key, remainingFields[key]))
			first = false
		}
		b.WriteString(")")
	}

	b.WriteByte('\n')
	return b.Bytes(), nil
} // InitLogger initializes the global logger with the provided configuration
func InitLogger(config LogConfig) {
	Logger = logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(config.Level)
	if err != nil {
		// Default to info if invalid level
		level = logrus.InfoLevel
		Logger.Warnf("Invalid log level '%s', defaulting to 'info'", config.Level)
	}
	Logger.SetLevel(level)

	// Set output format
	switch strings.ToLower(config.Format) {
	case "json":
		Logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
		})
	case "simple":
		Logger.SetFormatter(&CompactFormatter{ShowTime: false})
	case "compact":
		Logger.SetFormatter(&CompactFormatter{ShowTime: true})
	case "text", "":
		Logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		})
	default:
		Logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		})
		Logger.Warnf("Invalid log format '%s', defaulting to 'text'", config.Format)
	}

	// Set output to stdout
	Logger.SetOutput(os.Stdout)

	Logger.Infof("Logger initialized with level: %s, format: %s", level.String(), config.Format)
}

// GetLogger returns the global logger instance
func GetLogger() *logrus.Logger {
	if Logger == nil {
		// Initialize with default config if not already initialized
		InitLogger(LogConfig{
			Level:  "info",
			Format: "text",
		})
	}
	return Logger
}

// Helper functions for common logging patterns
func WithComponent(component string) *logrus.Entry {
	return GetLogger().WithField("component", component)
}

func WithInterface(iface string) *logrus.Entry {
	return GetLogger().WithField("interface", iface)
}

func WithComponentAndInterface(component, iface string) *logrus.Entry {
	return GetLogger().WithFields(logrus.Fields{
		"component": component,
		"interface": iface,
	})
}

func WithError(err error) *logrus.Entry {
	return GetLogger().WithError(err)
}
