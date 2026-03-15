package selfupdate

var log Logger = &emptyLogger{}

// SetLogger redirects all logs to the given logger.
// By default logs are discarded.
func SetLogger(logger Logger) {
	log = logger
}

// Logger interface. Compatible with standard log.Logger.
type Logger interface {
	Print(v ...interface{})
	Printf(format string, v ...interface{})
}

type emptyLogger struct{}

func (l *emptyLogger) Print(v ...interface{})                 {}
func (l *emptyLogger) Printf(format string, v ...interface{}) {}
