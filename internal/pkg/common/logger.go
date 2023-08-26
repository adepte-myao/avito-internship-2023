package common

type Logger interface {
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	Fatal(args ...interface{})

	Debugw(msg string, args ...interface{})
	Infow(msg string, args ...interface{})
	Warnw(msg string, args ...interface{})
	Errorw(msg string, args ...interface{})
	Fatalw(msg string, args ...interface{})
}
