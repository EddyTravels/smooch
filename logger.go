package smooch

type Logger interface {
	Debugw(msg string, keysAndValues ...interface{})
	Infow(msg string, keysAndValues ...interface{})
	Errorw(msg string, keysAndValues ...interface{})
}

type nopLogger struct{}

func (nl *nopLogger) Debugw(msg string, keysAndValues ...interface{}) {}

func (nl *nopLogger) Infow(msg string, keysAndValues ...interface{}) {}

func (nl *nopLogger) Errorw(msg string, keysAndValues ...interface{}) {}
