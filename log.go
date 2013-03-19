/* This implements an alternative logger to the one found in the standard
 * library with support for more logging levels, formatters and outputs.
 * The main goal is to provide easy and flexible way to handle new outputs and formats
 * Author: Robert Zaremba
 *
 * https://github.com/scale-it/go-log
 */
package log

import (
	"errors"
	"fmt"
	"io"
	"log"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Represents how critical the logged
// message is.
type Level uint8

var Levels = struct {
	Trace   Level
	Debug   Level
	Info    Level
	Warning Level
	Error   Level
	Fatal   Level
}{0, 10, 20, 30, 40, 50}

// Verbose names of the levels
var levelStrings = map[Level]string{
	Levels.Trace:   "TRACE",
	Levels.Debug:   "DEBUG",
	Levels.Info:    "INFO",
	Levels.Warning: "WARN",
	Levels.Error:   "ERROR",
	Levels.Fatal:   "FATAL",
}

// Verbose and colored names of the levels
var levelCStrings = map[Level]string{
	Levels.Trace:   levelStrings[Levels.Trace],
	Levels.Debug:   levelStrings[Levels.Debug],
	Levels.Info:    AnsiEscape(MAGENTA, levelStrings[Levels.Info], OFF),
	Levels.Warning: AnsiEscape(YELLOW, levelStrings[Levels.Warning], OFF),
	Levels.Error:   AnsiEscape(RED, levelStrings[Levels.Error], OFF),
	Levels.Fatal:   AnsiEscape(RED, BOLD, levelStrings[Levels.Fatal], OFF),
}

// Returns an log Level which name match given string.
// If there is no such Level, then Levels.Debug is returned
func String2Level(level string) (Level, error) {
	if level == "" {
		return Levels.Debug, errors.New("level is empty")
	}
	for li, ls := range levelStrings {
		if ls == level {
			return li, nil
		}
	}
	return Levels.Debug, errors.New("Wrong log level " + level)
}

type Formatter interface {
	Format(Level, string) []byte
}

type output struct {
	writer io.Writer
	level  Level
	fmt    Formatter
}

// The Logger
type Logger struct {
	mtx     sync.Mutex
	outputs []output
}

// Instantiate a new Logger
func New() *Logger {
	return &Logger{sync.Mutex{}, make([]output, 0)}
}

// Standard Formatter
type StdFormatter struct {
	Prefix  string // prefix to write at beginning of each line
	Flag    int    // format flags - based flags from std log package
	Colored bool   // use colored level names
}

func (this StdFormatter) Format(level Level, msg string) []byte {
	var slevel string
	var ok bool
	var out []string

	// adding time info
	if this.Flag&(log.Ldate|log.Ltime|log.Lmicroseconds) != 0 {
		now := time.Now()
		if this.Flag&log.Ldate != 0 {
			out = append(out, fmt.Sprintf("%v-%02d-%02d", now.Year(), now.Month(), now.Day()))
		}
		if this.Flag&(log.Lmicroseconds) != 0 {
			out = append(out, fmt.Sprintf("%02d:%02d:%02d.%06d", now.Hour(), now.Minute(), now.Second(), now.Nanosecond()/1000000))
		} else if this.Flag&(log.Ltime) != 0 {
			out = append(out, fmt.Sprintf("%02d:%02d:%02d", now.Hour(), now.Minute(), now.Second()))
		}

	}

	// adding level info
	if this.Colored {
		slevel, ok = levelCStrings[level]
	} else {
		slevel, ok = levelStrings[level]
	}
	if !ok {
		slevel = strconv.Itoa(int(level))
	}
	out = append(out, slevel)

	out = append(out, this.Prefix)

	// adding caller info. It's quiet exepnsive
	if this.Flag&(log.Lshortfile|log.Llongfile) != 0 {
		if _, file, line, ok := runtime.Caller(2); ok { // 2: calldepth
			if this.Flag&log.Lshortfile != 0 {
				for i := len(file) - 1; i > 0; i-- {
					if file[i] == '/' {
						file = file[i+1:]
						break
					}
				}
			}
			out = append(out, fmt.Sprintf("%s:%d", file, line))
		} else {
			out = append(out, "???")
		}
	}

	out = append(out, msg)
	return []byte(strings.Join(out, " "))
}

/* LOGGER
 * ------
 */

// Adds an ouput, specifying the maximum log Level
// you want to be written to this output. For instance,
// if you pass Warning for level, all logs of type
// Warning, Error, and Fatal would be logged to this output.
func (this *Logger) AddOutput(writer io.Writer, level Level, fm Formatter) {
	this.mtx.Lock()
	this.outputs = append(this.outputs, output{writer, level, fm})
	this.mtx.Unlock()
}

// Logs a message for the given level. Most callers will likely
// prefer to use one of the provided convenience functions (Debug, Info...).
func (this *Logger) Log(level Level, msg string) {
	this.mtx.Lock()
	defer this.mtx.Unlock()
	for _, output := range this.outputs {
		if output.level <= level {
			output.writer.Write(output.fmt.Format(level, msg))
		}
	}
}

// Logs a formatted message message for the given level.
// Wrapper around Log method
func (this *Logger) Logf(level Level, format string, v ...interface{}) {
	this.Log(level, fmt.Sprintf(format+"\n", v...))
}

// Convenience function
func (this *Logger) Trace(format string, v ...interface{}) {
	// TODO: split the string
	this.Log(Levels.Trace, fmt.Sprintf(format+"\n", v...))
}

// Convenience function
func (this *Logger) Debug(format string, v ...interface{}) {
	this.Log(Levels.Debug, fmt.Sprintf(format+"\n", v...))
}

// Convenience function
func (this *Logger) Info(format string, v ...interface{}) {
	this.Log(Levels.Info, fmt.Sprintf(format+"\n", v...))
}

// Convenience function
func (this *Logger) Warning(format string, v ...interface{}) {
	this.Log(Levels.Warning, fmt.Sprintf(format+"\n", v...))
}

// Convenience function
func (this *Logger) Error(format string, v ...interface{}) {
	this.Log(Levels.Error, fmt.Sprintf(format+"\n", v...))
}

// Convenience function, will not terminate the program
func (this *Logger) Fatal(format string, v ...interface{}) {
	this.Log(Levels.Fatal, fmt.Sprintf(format+"\n", v...))
}
