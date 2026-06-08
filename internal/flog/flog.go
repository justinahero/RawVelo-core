package flog

import (
	"fmt"
	"os"
	"sync/atomic"
	"time"
)

type Level int

const None Level = -1
const (
	Debug Level = iota
	Info
	Warn
	Error
	Fatal
)

var (
	minLevel    = Info
	logCh       = make(chan string, 1024)
	droppedMsgs atomic.Int64 // باگ ۷ fix: شمارنده پیامهای دور انداخته‌شده
)

func init() {
	// باگ ۴ fix: goroutine drain رو بدون قید و شرط اینجا شروع میکنیم
	// قبلاً فقط داخل SetLevel شروع میشد و اگه SetLevel صدا زده نمیشد، پیامها بی‌سروصدا دور انداخته میشدن
	go func() {
		for msg := range logCh {
			fmt.Fprint(os.Stdout, msg)
		}
	}()
}

func SetLevel(l int) {
	minLevel = Level(l)
}

func logf(level Level, format string, args ...any) {
	if level < minLevel || minLevel == None {
		return
	}

	for _, arg := range args {
		if err, ok := arg.(error); ok {
			err = WErr(err)
			if err == nil {
				return
			}
		}
	}

	// اگه پیامهای دور انداخته‌شده داریم، اونا رو قبل از این پیام گزارش بده
	if dropped := droppedMsgs.Swap(0); dropped > 0 {
		now := time.Now().Format("2006-01-02 15:04:05.000")
		warn := fmt.Sprintf("%s [WARN] %d log message(s) dropped due to back-pressure\n", now, dropped)
		select {
		case logCh <- warn:
		default:
			droppedMsgs.Add(dropped + 1) // نتونستیم warn رو هم بفرستیم
		}
	}

	now := time.Now().Format("2006-01-02 15:04:05.000")
	line := fmt.Sprintf("%s [%s] %s\n", now, level.String(), fmt.Sprintf(format, args...))

	// باگ ۷ fix: Error و Fatal بلاکینگ ارسال میشن تا هرگز از دست نرن
	// بقیه سطحها non-blocking هستن با شمارنده drop
	if level >= Error {
		logCh <- line
	} else {
		select {
		case logCh <- line:
		default:
			droppedMsgs.Add(1)
		}
	}
}

func (l Level) String() string {
	switch l {
	case Debug:
		return "DEBUG"
	case Info:
		return "INFO"
	case Warn:
		return "WARN"
	case Error:
		return "ERROR"
	case Fatal:
		return "FATAL"
	case None:
		return "None"
	default:
		return "UNKNOWN"
	}
}

func Debugf(format string, args ...any) { logf(Debug, format, args...) }
func Infof(format string, args ...any)  { logf(Info, format, args...) }
func Warnf(format string, args ...any)  { logf(Warn, format, args...) }
func Errorf(format string, args ...any) { logf(Error, format, args...) }
func Fatalf(format string, args ...any) {
	logf(Fatal, format, args...)
	// flush logs (optional: small sleep to let goroutine write)
	time.Sleep(10 * time.Millisecond)
	os.Exit(1)
}

func Close() { close(logCh) }
