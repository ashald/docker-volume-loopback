package context

import (
	"fmt"
	"github.com/oklog/ulid"
	"github.com/sirupsen/logrus"
	"io"
	"math/rand"
	"time"
)

const (
	Error   = 0
	Warning = 1
	Info    = 2
	Debug   = 3
	Trace   = 4
)

var allLevels = []logrus.Level{
	logrus.ErrorLevel,
	logrus.WarnLevel,
	logrus.InfoLevel,
	logrus.DebugLevel,
	logrus.TraceLevel,
}

type Fields map[string]interface{}

type Context struct {
	Trace string
	index uint64 // given Context is intended to be used in single-threaded fashion so we use uint instead of ULID

	level  logrus.Level
	fields map[string]interface{}
}

func Init(level int, output io.Writer) {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(output)
	logrus.SetLevel(convertLevel(level))
}

func New() (ctx *Context) {
	ctx = new(Context)
	ctx.Trace = newULID()
	ctx.fields = make(map[string]interface{})
	return
}

func (ctx *Context) Copy() *Context {
	newContext := new(Context)
	newContext.Trace = ctx.Trace
	newContext.index = ctx.index
	newContext.level = ctx.level
	newContext.fields = make(map[string]interface{})

	for k, v := range ctx.fields {
		newContext.fields[k] = v
	}

	return newContext
}

func (ctx *Context) Derived() (derived *Context) {
	ctx.index += 1
	derived = New()
	derived.Trace = fmt.Sprintf("%s.%d", ctx.Trace, ctx.index)
	return
}

func (ctx *Context) Level(level int) *Context {
	newContext := ctx.Copy()
	newContext.level = convertLevel(level)

	return newContext
}

func (ctx *Context) Field(name string, value interface{}) *Context {
	ctx.fields[name] = value
	return ctx
}

func (ctx *Context) Message(message string) {
	ctx.fields["::trace"] = ctx.Trace
	if ctx.level == logrus.PanicLevel {
		panic(fmt.Sprintf("Failed to log because level is not provided: %s", message))
	}
	logrus.WithFields(ctx.fields).Log(ctx.level, message)
}

func convertLevel(level int) logrus.Level {
	if level < 0 || level >= len(allLevels) {
		panic(
			fmt.Sprintf(
				"Wrong log level '%d' - only '0' (least verbose) to '%d' (most verbose) supported",
				level, len(allLevels)-1))
	}

	return allLevels[level]
}

func newULID() string {
	t := time.Now()
	entropy := ulid.Monotonic(rand.New(rand.NewSource(t.UnixNano())), 0)
	value := ulid.MustNew(ulid.Timestamp(t), entropy)
	return value.String()
}
