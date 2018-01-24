//go:generate stringer -type=LoggerLevel
package recipe

import (
	"log"
	"os"
	"fmt"
	"github.com/fatih/color"
)

type LoggerLevel int

const (
	DebugL LoggerLevel = iota
	InfoL
	WarningL
	ErrorL
	FatalL
)

type Logger struct {
	l *log.Logger
	Level LoggerLevel
}

func NewLogger(prefix string) *Logger {
	l := log.New(os.Stderr, color.HiWhiteString(prefix), log.LstdFlags)
	return &Logger{l:l, Level:InfoL}
}

func (l *Logger) Debug(format string, v ...interface{}) {
	if l.Level <= DebugL {
		l.l.Printf(color.BlueString("(D): ")+format, v...)
	}
}

func (l *Logger) Info(format string, v ...interface{}) {
	if l.Level <= InfoL {
		l.l.Printf(color.HiWhiteString("(I): ")+format, v...)
	}
}

func (l *Logger) Warning(format string, v ...interface{}) {
	if l.Level <= WarningL {
		l.l.Printf(color.YellowString("(W): ")+format, v...)
	}
}

func (l *Logger) Error(format string, v ...interface{}) {
	if l.Level <= ErrorL {
		l.l.Printf(color.RedString("(E): ")+format, v...)
	}
}

func (l *Logger) Fatal(v ...interface{}) {
	l.l.Print(color.MagentaString("(F): ") + fmt.Sprint(v...))
	os.Exit(1)
}
