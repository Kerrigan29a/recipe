//go:generate stringer -type=LoggerLevel
package recipe

import (
	"fmt"
	"github.com/fatih/color"
	"log"
	"os"
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
	l     *log.Logger
	Level LoggerLevel
}

func NewLogger(prefix string) *Logger {
	l := log.New(os.Stderr, color.HiWhiteString(prefix), log.LstdFlags) //|log.Lshortfile)
	return &Logger{l: l, Level: InfoL}
}

func (l *Logger) Debug(format string, v ...interface{}) {
	if l.Level <= DebugL {
		l.l.Output(2, fmt.Sprintf(color.BlueString("(D): ")+format, v...))
	}
}

func (l *Logger) Info(format string, v ...interface{}) {
	if l.Level <= InfoL {
		l.l.Output(2, fmt.Sprintf(color.HiWhiteString("(I): ")+format, v...))
	}
}

func (l *Logger) Warning(format string, v ...interface{}) {
	if l.Level <= WarningL {
		l.l.Output(2, fmt.Sprintf(color.YellowString("(W): ")+format, v...))
	}
}

func (l *Logger) Error(format string, v ...interface{}) {
	if l.Level <= ErrorL {
		l.l.Output(2, fmt.Sprintf(color.RedString("(E): ")+format, v...))
	}
}

func (l *Logger) Fatal(v ...interface{}) {
	l.l.Output(2, fmt.Sprintf(color.MagentaString("(F): ")+fmt.Sprint(v...)))
	os.Exit(1)
}
