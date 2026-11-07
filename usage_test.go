package mcli

import (
	"bytes"
	"flag"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestQuoteUsageName(t *testing.T) {
	app := NewApp()
	app.HelpWidth = 80
	var args struct {
		Arg1 string        `cli:"-a, this is escaped quoted words \\'not name\\' and this is the 'name' and more quoted words \\'actually no need to quote this\\'"`
		Arg2 string        "cli:\"-b, using backtick to quote name `value` some more description\""
		Arg3 time.Duration "cli:\"-c, only one backtick, `some thing, using type name\""
	}
	app.AddRoot(func(ctx *Context) {
		ctx.Parse(&args)
		ctx.PrintHelp()
	})

	var buf bytes.Buffer
	app.getFlagSet().SetOutput(&buf)

	defer mockOSArgs("run")()
	app.Run()

	got := buf.String()
	assert.Contains(t, got, `-a <name>        this is escaped quoted words 'not name' and this is the name`+"\n                   "+`and more quoted words 'actually no need to quote this'`)
	assert.Contains(t, got, "-b <value>       using backtick to quote name value some more description")
	assert.Contains(t, got, "-c <duration>    only one backtick, `some thing, using type name")
}

func TestUsageEnvVariables(t *testing.T) {
	app := NewApp()
	var args struct {
		CNServiceSecret    string `mcli:"#E, CN service account secret blah blahblah blah blahblahblah" env:"CN_SERVICE_BEARER_SECRET"`
		I18NServiceSecret  string `mcli:"#E, I18N service account secret" env:"I18N_SERVICE_BEARER_SECRET"`
		CNPersonalSecret   string `mcli:"#E, CN personal account secret" env:"CN_PERSON_BEARER_SECRET"`
		I18NPersonalSecret string `mcli:"#E, I18N personal account secret" env:"I18N_PERSON_BEARER_SECRET"`
	}
	app.AddRoot(func(ctx *Context) {
		ctx.MustParse(&args)
		ctx.PrintHelp()
	})

	var buf bytes.Buffer
	app.getFlagSet().SetOutput(&buf)

	defer mockOSArgs("run")()
	app.Run()

	got := buf.String()
	assert.Contains(t, got, "Usage:\n  run\n\nEnvironment Variables:\n")
	assert.Contains(t, got, "  - CN_SERVICE_BEARER_SECRET <string>\n    CN service account secret blah blahblah blah blahblahblah\n")
	assert.Contains(t, got, "  - I18N_SERVICE_BEARER_SECRET <string>\n    I18N service account secret\n")
	assert.Contains(t, got, "  - CN_PERSON_BEARER_SECRET <string>\n    CN personal account secret\n")
	assert.Contains(t, got, "  - I18N_PERSON_BEARER_SECRET <string>\n    I18N personal account secret\n\n")
}

func TestUsageSharedAlignment(t *testing.T) {
	app := NewApp()
	app.SetGlobalFlags(&struct {
		Configuration string `cli:"--configuration, Global configuration"`
	}{})
	var args struct {
		Verbose bool   `cli:"-v, --verbose, Enable verbose output"`
		Long    string `cli:"--very-long-configuration-name, First line\nSecond line"`
		Target  string `cli:"target, Deployment target"`
	}
	var out bytes.Buffer
	app.getFlagSet().SetOutput(&out)
	_, err := app.parseArgs(&args, WithArgs([]string{"--help"}), WithErrorHandling(flag.ContinueOnError))
	assert.ErrorIs(t, err, flag.ErrHelp)
	lines := strings.Split(out.String(), "\n")
	column := -1
	for _, description := range []string{"Enable verbose output", "First line", "Second line", "Deployment target", "Global configuration"} {
		found := false
		for _, line := range lines {
			if pos := strings.Index(line, description); pos >= 0 {
				if column < 0 {
					column = pos
				}
				assert.Equal(t, column, pos, description)
				found = true
				break
			}
		}
		assert.True(t, found, description)
	}
	assert.Contains(t, out.String(), "      --very-long-configuration-name <string>\n")
}

func TestUsageCompactMetadata(t *testing.T) {
	app := NewApp()
	app.HelpWidth = 80
	var args struct {
		Format string `cli:"-f, --format, Output format" default:"json" env:"TOOL_FORMAT"`
		Output string `cli:"--output" default:"stdout" env:"TOOL_OUTPUT"`
	}
	var out bytes.Buffer
	app.getFlagSet().SetOutput(&out)
	_, err := app.parseArgs(&args, WithArgs([]string{"--help"}), WithErrorHandling(flag.ContinueOnError),
		WithEnums(map[string][]string{"format": {"json", "yaml"}}))
	assert.ErrorIs(t, err, flag.ErrHelp)
	assert.Contains(t, out.String(), `(default: "json") (env: TOOL_FORMAT)`+"\n                          "+`(choices: json, yaml)`)
	assert.Contains(t, out.String(), `(default: "stdout") (env: TOOL_OUTPUT)`)
}

func TestHelpWidthFallback(t *testing.T) {
	var out bytes.Buffer
	assert.Equal(t, -1, resolveHelpWidth(&out, 0))
	assert.Equal(t, 42, resolveHelpWidth(&out, 42))
	assert.Equal(t, -1, resolveHelpWidth(&out, -1))
	file, err := os.CreateTemp(t.TempDir(), "help")
	if !assert.NoError(t, err) {
		return
	}
	defer file.Close()
	assert.Equal(t, -1, resolveHelpWidth(file, 0))
}

func TestWrapHelpText(t *testing.T) {
	for _, tt := range []struct {
		name, text string
		width      int
		want       string
	}{
		{"words", "one two three four", 9, "one two\nthree\nfour"},
		{"manual lines", "one two\n\n  - three four five", 12, "one two\n\n  - three\n  four five"},
		{"long token", "https://example.com/long/path next", 12, "https://example.com/long/path\nnext"},
		{"unicode", "中文 参数 描述", 9, "中文 参数\n描述"},
		{"disabled", "one two three", -1, "one two three"},
		{"narrow", "one two", 1, "one\ntwo"},
	} {
		t.Run(tt.name, func(t *testing.T) { assert.Equal(t, tt.want, wrapHelpText(tt.text, tt.width)) })
	}
}

func TestUsageConfiguredWidth(t *testing.T) {
	for _, width := range []int{30, 40, 80, -1} {
		app := NewApp()
		app.HelpWidth = width
		var args struct {
			Name string `cli:"-n, --name, Select a name for the current deployment" default:"main" env:"TOOL_NAME"`
		}
		var out bytes.Buffer
		app.getFlagSet().SetOutput(&out)
		_, err := app.parseArgs(&args, WithArgs([]string{"-h"}), WithErrorHandling(flag.ContinueOnError))
		assert.ErrorIs(t, err, flag.ErrHelp)
		assert.Contains(t, out.String(), "-n, --name <string>")
		if width > 0 {
			for _, line := range strings.Split(out.String(), "\n") {
				assert.LessOrEqual(t, len(line), width, line)
			}
		}
		if width < 0 {
			assert.Contains(t, out.String(), "Select a name for the current deployment")
			assert.Contains(t, out.String(), `(default: "main") (env: TOOL_NAME)`)
		}
	}
}

func TestHelpHangingIndent(t *testing.T) {
	var out bytes.Buffer
	printWithAlignment(&out, []usageItem{{prefix: strings.Repeat("x", 31), description: "description"}}, 30, -1)
	assert.Equal(t, strings.Repeat("x", 31)+"\n"+strings.Repeat(" ", 34)+"description\n", out.String())
	out.Reset()
	printWithAlignment(&out, []usageItem{{prefix: "  -v", description: "description"}}, 6, 40)
	assert.Equal(t, "  -v\n    description\n", out.String())
	out.Reset()
	printWithAlignment(&out, []usageItem{{prefix: "  a", description: "description"}}, 6, 40)
	assert.Equal(t, "  a\n    description\n", out.String())
}

func TestHelpPreservesExamplesAndFooter(t *testing.T) {
	app := NewApp()
	app.HelpWidth = 20
	example := "tool --name a-very-long-name --output a-very-long-path"
	footer := "https://example.com/a-very-long-documentation-path"
	app.HelpFooter = footer
	var out bytes.Buffer
	app.getFlagSet().SetOutput(&out)
	_, err := app.parseArgs(&struct{}{}, WithArgs([]string{"-h"}), WithErrorHandling(flag.ContinueOnError), WithExamples(example))
	assert.ErrorIs(t, err, flag.ErrHelp)
	assert.Contains(t, out.String(), example)
	assert.Contains(t, out.String(), footer)
}

func TestHelpMinimumWidthEnvironmentDescription(t *testing.T) {
	app := NewApp()
	app.HelpWidth = 4
	var args struct {
		Token string `cli:"#E, one two three" env:"TOOL_TOKEN"`
	}
	var out bytes.Buffer
	app.getFlagSet().SetOutput(&out)
	_, err := app.parseArgs(&args, WithArgs([]string{"-h"}), WithErrorHandling(flag.ContinueOnError))
	assert.ErrorIs(t, err, flag.ErrHelp)
	assert.Contains(t, out.String(), "\n    one two three\n")
}

func TestHelpUnicodeAlignment(t *testing.T) {
	var out bytes.Buffer
	printWithAlignment(&out, []usageItem{{prefix: "  中文", description: "one"}, {prefix: "  -a", description: "two"}}, 0, 80)
	assert.Equal(t, "  中文    one\n  -a      two\n", out.String())
}

func TestHelpMetadataItemsStayTogether(t *testing.T) {
	for _, tt := range []struct {
		name  string
		width int
		items []string
		want  string
	}{
		{"fits", 40, []string{`(default: "hello world")`, `(env: TOOL)`}, "  -f\n    (default: \"hello world\") (env: TOOL)\n"},
		{"wrap between items", 30, []string{`(default: "hello world")`, `(env: TOOL)`, `(choices: one, two)`}, "  -f\n    (default: \"hello world\")\n    (env: TOOL)\n    (choices: one, two)\n"},
		{"oversized item", 15, []string{`(choices: one, two, three)`, `(env: X)`}, "  -f\n    (choices: one, two, three)\n    (env: X)\n"},
		{"unicode", 28, []string{`(default: "中文")`, `(env: X)`}, "  -f\n    (default: \"中文\")\n    (env: X)\n"},
		{"disabled", -1, []string{`(default: "hello world")`, `(choices: one, two)`}, "  -f\n          (default: \"hello world\") (choices: one, two)\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			printWithAlignment(&out, []usageItem{{prefix: "  -f", appendixes: tt.items}}, 6, tt.width)
			assert.Equal(t, tt.want, out.String())
		})
	}
}

func TestWrapHelpTabs(t *testing.T) {
	assert.Equal(t, "Heading\n\talpha\n\tbeta\n\tgamma", wrapHelpText("Heading\n\talpha beta gamma", 16))
	var out bytes.Buffer
	printWithAlignment(&out, []usageItem{{prefix: "  -a", description: "\talpha beta gamma"}}, 6, 20)
	assert.Equal(t, "  -a\n    \talpha beta\n    \tgamma\n", out.String())
}

func TestWrapHelpEmoji(t *testing.T) {
	for _, text := range []string{"🇺🇸 🇺🇸", "1️⃣ 2️⃣", "☀️ ☀️", "👩‍💻 👨‍💻", "é é"} {
		t.Run(text, func(t *testing.T) {
			width := 4
			want := strings.ReplaceAll(text, " ", "\n")
			if text == "é é" {
				want = text
			}
			assert.Equal(t, want, wrapHelpText(text, width))
		})
	}
}

func TestHelpEmojiMetadataAndAlignment(t *testing.T) {
	var out bytes.Buffer
	printWithAlignment(&out, []usageItem{{prefix: "  -a", appendixes: []string{"(env: 🇺🇸)", "(env: 1️⃣)"}}}, 6, 22)
	assert.Equal(t, "  -a\n    (env: 🇺🇸)\n    (env: 1️⃣)\n", out.String())
	out.Reset()
	printWithAlignment(&out, []usageItem{{prefix: "  🇺🇸", description: "one"}, {prefix: "  -a", description: "two"}}, 0, 80)
	assert.Equal(t, "  🇺🇸      one\n  -a      two\n", out.String())
}

func TestHelpTextWidth(t *testing.T) {
	for _, tt := range []struct {
		text         string
		column, want int
	}{
		{"\tx", 0, 9}, {"\tx", 4, 5}, {"\tx", 8, 9},
		{"🇺🇸", 0, 2}, {"1️⃣", 0, 2}, {"#⃣", 0, 2}, {"*️⃣", 0, 2},
		{"☀️", 0, 2}, {"☀︎", 0, 1}, {"👩‍💻", 0, 2}, {"👍🏽", 0, 2},
		{"é", 0, 1}, {"中文", 0, 4},
	} {
		assert.Equal(t, tt.want, helpTextWidth(tt.text, tt.column), tt.text)
	}
	assert.Equal(t, "a\n\tx\nb", wrapHelpItems([]string{"a", "\tx", "b"}, 8, 0))
}

func TestHelpTabIndentedApplicationDescription(t *testing.T) {
	app := NewApp()
	app.Description = "Heading\n\talpha beta gamma delta epsilon"
	app.HelpWidth = 16
	var out bytes.Buffer
	app.getFlagSet().SetOutput(&out)
	_, err := app.parseArgs(&struct{}{}, WithArgs([]string{"-h"}), WithErrorHandling(flag.ContinueOnError))
	assert.ErrorIs(t, err, flag.ErrHelp)
	assert.Contains(t, out.String(), "Heading\n\talpha beta gamma delta\n\tepsilon\n")
}

func TestHelpMinimumWidth(t *testing.T) {
	var out bytes.Buffer
	for _, tt := range []struct{ requested, want int }{{1, 30}, {20, 30}, {29, 30}, {30, 30}, {40, 40}, {80, 80}, {0, -1}, {-1, -1}} {
		assert.Equal(t, tt.want, resolveHelpWidth(&out, tt.requested))
	}
}

func TestUsageDynamicColumnBudget(t *testing.T) {
	for _, tt := range []struct{ width, column int }{{120, 34}, {80, 26}, {60, 20}, {40, 4}, {30, 4}} {
		app := NewApp()
		app.HelpWidth = tt.width
		app.SetGlobalFlags(&struct {
			Global bool `cli:"--glob, Description global"`
		}{})
		var args struct {
			Target   string `cli:"target, Description target"`
			Alpha    string `cli:"-a, --alpha, Description alpha"`
			Medium   bool   `cli:"--abcdefgh, Description medium"`
			Wide     bool   `cli:"--wideparamnameflag, Description wide"`
			VeryWide bool   `cli:"--abcdefghijklmn, Description verywide"`
			Maximum  bool   `cli:"--abcdefghijklmnopqrstuv, Description maximum"`
		}
		var out bytes.Buffer
		app.getFlagSet().SetOutput(&out)
		_, err := app.parseArgs(&args, WithArgs([]string{"-h"}), WithErrorHandling(flag.ContinueOnError))
		assert.ErrorIs(t, err, flag.ErrHelp)
		for _, line := range strings.Split(out.String(), "\n") {
			if index := strings.Index(line, "Description"); index >= 0 {
				assert.Equal(t, tt.column, index, line)
			}
		}
		if tt.width <= 40 {
			assert.Contains(t, out.String(), "\n  --abcdefgh\n    Description medium\n")
			assert.NotContains(t, out.String(), "\n      --")
		}
	}
}

func TestHelpNonTerminalOutput(t *testing.T) {
	reader, writer, err := os.Pipe()
	if !assert.NoError(t, err) {
		return
	}
	defer reader.Close()
	defer writer.Close()
	assert.Equal(t, -1, resolveHelpWidth(writer, 0))
	assert.Equal(t, 42, resolveHelpWidth(writer, 42))

	text := strings.Repeat("word ", 24) + "end"
	for _, width := range []int{0, -1, 40} {
		app := NewApp()
		app.HelpWidth = width
		app.Description = text
		var args struct {
			Name string `cli:"--name"`
		}
		var out bytes.Buffer
		app.getFlagSet().SetOutput(&out)
		_, err := app.parseArgs(&args, WithArgs([]string{"-h"}), WithErrorHandling(flag.ContinueOnError))
		assert.ErrorIs(t, err, flag.ErrHelp)
		if width <= 0 {
			assert.Contains(t, out.String(), text)
		} else {
			assert.NotContains(t, out.String(), text)
		}
	}
}
