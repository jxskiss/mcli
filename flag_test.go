package mcli

import (
	"bytes"
	"flag"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_flag_DefaultValue(t *testing.T) {
	resetDefaultApp()
	var args struct {
		A bool          `cli:"-a" default:"true"`
		B string        `cli:"-b" default:"astr"`
		D time.Duration `cli:"-d" default:"1.5s"`

		Arg1 string `cli:"arg1" default:"arg1str"`
		Arg2 string `cli:"arg2" default:"arg2str"`
	}
	fs, err := Parse(&args, WithErrorHandling(flag.ContinueOnError),
		WithArgs([]string{"-d", "1000ms", "cmdlineArg1"}))
	assert.Nil(t, err)
	assert.Equal(t, true, args.A)
	assert.Equal(t, "astr", args.B)
	assert.Equal(t, 1000*time.Millisecond, args.D)
	assert.Equal(t, "cmdlineArg1", args.Arg1)
	assert.Equal(t, "arg2str", args.Arg2)

	var buf bytes.Buffer
	fs.SetOutput(&buf)
	fs.Usage()

	got := buf.String()
	assert.Contains(t, got, "FLAGS:")
	assert.Contains(t, got, "  -a")
	assert.Contains(t, got, "(default true)")
	assert.Contains(t, got, "  -b string")
	assert.Contains(t, got, `(default "astr")`)
	assert.Contains(t, got, "  -d duration")
	assert.Contains(t, got, "(default 1.5s)")
	assert.Contains(t, got, "ARGUMENTS:")
	assert.Contains(t, got, "  arg1 string")
	assert.Contains(t, got, `(default "arg1str")`)
	assert.Contains(t, got, "  arg2 string")
	assert.Contains(t, got, `(default "arg2str")`)
}

func Test_flag_CompositeType(t *testing.T) {
	table := [][]interface{}{
		{&struct {
			S1 []uint8 `cli:"s1"`
		}{}, "s1 []uint"},
		{&struct {
			S2 []time.Duration `cli:"s2"`
		}{}, "s2 []duration"},
		{&struct {
			M1 map[string]float64 `cli:"m1"`
		}{}, "m1 map[string]float"},
		{&struct {
			M2 map[string]time.Duration `cli:"m2"`
		}{}, "m2 map[string]duration"},
	}

	for _, row := range table {
		resetDefaultApp()
		args := row[0]
		fs, err := Parse(args, WithErrorHandling(flag.ContinueOnError), WithArgs([]string{}))
		assert.Nil(t, err)

		var buf bytes.Buffer
		fs.SetOutput(&buf)
		fs.Usage()

		got := buf.String()
		want := row[1]
		assert.Contains(t, got, want)
	}
}

type MyMap map[string]string

func Test_flag_Map(t *testing.T) {
	resetDefaultApp()
	var args struct {
		M1 map[string]string        `cli:"-m1"`
		M2 MyMap                    `cli:"-m2"`
		M3 map[string]time.Duration `cli:"-m3"`
	}
	fs, err := Parse(&args, WithErrorHandling(flag.ContinueOnError),
		WithArgs([]string{
			"-m1", "key1=val1",
			"-m1", "key2=val2",
			"-m2", "key3=val3",
			"-m2", "key4=val4",
			"-m3", "key5=1s",
			"-m3", "key6=100ms",
		}))
	_ = fs
	assert.Nil(t, err)
	assert.Equal(t, map[string]string{"key1": "val1", "key2": "val2"}, args.M1)
	assert.Equal(t, MyMap{"key3": "val3", "key4": "val4"}, args.M2)
	assert.Equal(t, map[string]time.Duration{"key5": time.Second, "key6": 100 * time.Millisecond}, args.M3)
}
