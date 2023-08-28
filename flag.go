package mcli

import (
	"encoding"
	"encoding/json"
	"flag"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Modifier represents an option to a flag, it sets the flag to be
// deprecated, hidden, or required. In a `cli` tag, modifiers appears as
// the first segment, starting with a `#` character.
//
// Fow now the following modifiers are available:
//
//	D - marks a flag or argument as deprecated, "DEPRECATED" will be showed in help
//	R - marks a flag or argument as required, "REQUIRED" will be showed in help
//	H - marks a flag as hidden, see below for more about hidden flags
//
// Hidden flags won't be showed in help, except that when a special flag
// "--mcli-show-hidden" is provided.
//
// Modifier `H` shall not be used for an argument, else it panics.
// An argument must be showed in help to tell user how to use the program
// correctly.
//
// Some modifiers cannot be used together, else it panics, e.g.
//
//	H & R - a required flag must appear in help to tell user to set it
//	D & R - a required flag must not be deprecated, it does not make sense,
//	        but makes user confused
type Modifier byte

func (m Modifier) apply(f *_flag) {
	switch byte(m) {
	case 'D':
		f.deprecated = true
	case 'H':
		f.hidden = true
	case 'R':
		f.required = true
	}
}

type textValue interface {
	encoding.TextMarshaler
	encoding.TextUnmarshaler
}

var (
	flagGetterTyp = reflect.TypeOf((*flag.Getter)(nil)).Elem()
	flagValueTyp  = reflect.TypeOf((*flag.Value)(nil)).Elem()
	textValueTyp  = reflect.TypeOf((*textValue)(nil)).Elem()
)

// _flag implements flag.Value.
type _flag struct {
	name        string
	short       string
	description string
	defValue    string
	envNames    []string
	_tags
	_value

	isGlobal           bool
	hasDefault         bool
	deprecated         bool
	hidden             bool
	required           bool
	nonflag            bool
	completionFunction string
}

type _tags struct {
	cliTag          string
	defaultValueTag string
	envTag          string
}

type _value struct {
	rv reflect.Value
}

func (f *_flag) Get() any {
	if f.rv.Type().Implements(flagGetterTyp) {
		return f.rv.Interface().(flag.Getter).Get()
	}
	if f.rv.CanAddr() && f.rv.Addr().Type().Implements(flagGetterTyp) {
		return f.rv.Addr().Interface().(flag.Getter).Get()
	}
	if f.rv.Kind() == reflect.Pointer {
		if f.rv.Elem().IsValid() {
			return f.rv.Elem().Interface()
		}
		zero := reflect.New(f.rv.Type().Elem()).Elem()
		return zero.Interface()
	}
	return f.rv.Interface()
}

func (f *_flag) String() string {
	return formatValue(f.rv)
}

func formatValue(rv reflect.Value) string {
	if rv.Kind() == reflect.Ptr && rv.IsNil() {
		rv = reflect.New(rv.Type().Elem())
	}
	if rv.Type().Implements(flagValueTyp) {
		return rv.Interface().(flag.Value).String()
	}
	if rv.CanAddr() && rv.Addr().Type().Implements(flagValueTyp) {
		return rv.Addr().Interface().(flag.Value).String()
	}
	if rv.Type().Implements(textValueTyp) {
		b, _ := rv.Interface().(textValue).MarshalText()
		return string(b)
	}
	if rv.CanAddr() && rv.Addr().Type().Implements(textValueTyp) {
		b, _ := rv.Addr().Interface().(textValue).MarshalText()
		return string(b)
	}
	if rv.Kind() == reflect.Pointer {
		return formatValueOfBasicTypePtr(rv)
	}
	return formatValueOfBasicType(rv)
}

func formatValueOfBasicTypePtr(rv reflect.Value) string {
	if rv.Elem().IsValid() {
		return formatValueOfBasicType(rv.Elem())
	}
	zero := reflect.New(rv.Type().Elem()).Elem()
	return formatValueOfBasicType(zero)
}

func formatValueOfBasicType(rv reflect.Value) string {
	switch rv.Kind() {
	case reflect.Bool:
		return strconv.FormatBool(rv.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if rv.Type() == reflect.TypeOf(time.Duration(0)) {
			return rv.Interface().(time.Duration).String()
		}
		return strconv.FormatInt(rv.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(rv.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(rv.Float(), 'g', -1, 64)
	case reflect.String:
		return rv.String()
	case reflect.Slice, reflect.Map:
		if rv.Len() == 0 {
			return ""
		}
		b, _ := json.Marshal(rv.Interface())
		return string(b)
	default:
		panic(fmt.Sprintf("mcli: unsupported value type: %v", rv.Type()))
	}
}

func (f *_flag) Set(s string) error {
	return applyValue(f.rv, s)
}

func applyValue(rv reflect.Value, s string) error {
	if s == "" {
		return nil
	}
	if rv.Kind() == reflect.Ptr && rv.IsNil() {
		rv.Set(reflect.New(rv.Type().Elem()))
	}
	if rv.Type().Implements(flagValueTyp) {
		return rv.Interface().(flag.Value).Set(s)
	}
	if rv.CanAddr() && rv.Addr().Type().Implements(flagValueTyp) {
		return rv.Addr().Interface().(flag.Value).Set(s)
	}
	if rv.Type().Implements(textValueTyp) {
		return rv.Interface().(textValue).UnmarshalText([]byte(s))
	}
	if rv.CanAddr() && rv.Addr().Type().Implements(textValueTyp) {
		return rv.Addr().Interface().(textValue).UnmarshalText([]byte(s))
	}

	return applyValueOfBasicType(rv, s)
}

func applyValueOfBasicType(rv reflect.Value, s string) error {
	if isSupportedBasicTypePtr(rv.Type()) {
		rv.Set(reflect.New(rv.Type().Elem()))
		return applyValueOfBasicType(rv.Elem(), s)
	}

	if isIntegerValue(rv) {
		return applyIntegerValue(rv, s)
	}
	switch rv.Kind() {
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return err
		}
		rv.SetBool(b)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return err
		}
		rv.SetFloat(f)
	case reflect.String:
		rv.SetString(s)
	case reflect.Slice: // []basicType
		e := reflect.New(rv.Type().Elem()).Elem()
		if err := applyValue(e, s); err != nil {
			return err
		}
		rv.Set(reflect.Append(rv, e))
	case reflect.Map: // map[string]basicType
		if rv.IsNil() {
			rv.Set(reflect.MakeMap(rv.Type()))
		}
		parts := append(strings.SplitN(s, "=", 2), "")
		k := parts[0]
		val := reflect.New(rv.Type().Elem()).Elem()
		if err := applyValue(val, parts[1]); err != nil {
			return err
		}
		rv.SetMapIndex(reflect.ValueOf(k), val)
	default:
		panic(fmt.Sprintf("mcli: unsupported value type: %v", rv.Type()))
	}
	return nil
}

func isIntegerValue(rv reflect.Value) bool {
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	}
	return false
}

func applyIntegerValue(rv reflect.Value, s string) error {
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var i int64
		var d time.Duration
		var err error
		if rv.Type() == reflect.TypeOf(time.Duration(0)) {
			d, err = time.ParseDuration(s)
			i = int64(d)
		} else {
			i, err = strconv.ParseInt(s, 10, 64)
		}
		if err != nil {
			return err
		}
		rv.SetInt(i)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return err
		}
		rv.SetUint(u)
	}
	return nil
}

func (f *_flag) isBooleanPtr() bool {
	return f.rv.Kind() == reflect.Pointer &&
		f.rv.Type().Elem().Kind() == reflect.Bool
}

func (f *_flag) isBoolean() bool {
	return f.rv.Kind() == reflect.Bool
}

func (f *_flag) isSlice() bool {
	return f.rv.Kind() == reflect.Slice
}

func (f *_flag) isMap() bool {
	return f.rv.Kind() == reflect.Map
}

func (f *_flag) isCompositeType() bool {
	kind := f.rv.Kind()
	return kind == reflect.Slice || kind == reflect.Map
}

func (f *_flag) isString() bool {
	return f.rv.Kind() == reflect.String
}

func (f *_flag) isZero() bool {
	typ := f.rv.Type()
	if isFlagValueImpl(f.rv) {
		zero := zeroFlagValueStr(f.rv)
		return f.String() == zero
	}
	if isTextValueImpl(f.rv) {
		zero := zeroTextValueStr(f.rv)
		return f.String() == zero
	}
	// Check comparable values.
	if typ.Comparable() {
		return reflect.Zero(typ).Interface() == f.rv.Interface()
	}
	// Else it must be a slice or a map.
	return f.rv.Len() == 0
}

func (f *_flag) helpName() string {
	if f.nonflag {
		return fmt.Sprintf("argument '%s'", f.name)
	}
	return fmt.Sprintf("flag '-%s'", f.name)
}

func (f *_flag) usageName() string {
	if f.isBoolean() || f.isBooleanPtr() {
		return ""
	}
	if isFlagValueImpl(f.rv) {
		return "value"
	}
	return usageName(f.rv.Type())
}

func usageName(typ reflect.Type) string {
	if isSupportedBasicTypePtr(typ) {
		return usageName(typ.Elem())
	}
	switch typ.Kind() {
	case reflect.Bool:
		return "bool"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if typ == reflect.TypeOf(time.Duration(0)) {
			return "duration"
		}
		return "int"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "uint"
	case reflect.Float32, reflect.Float64:
		return "float"
	case reflect.String:
		return "string"
	case reflect.Slice:
		elemName := usageName(typ.Elem())
		return "[]" + elemName
	case reflect.Map:
		elemName := usageName(typ.Elem())
		return "map[string]" + elemName
	default:
		return "value"
	}
}

func (f *_flag) getUsage(hasShortFlag bool) (prefix, usage string) {
	if f.nonflag {
		prefix += "  " + f.name
	} else if f.short != "" && f.name != "" {
		prefix += fmt.Sprintf("  -%s, --%s", f.short, f.name)
	} else if len(f.name) == 1 || !hasShortFlag {
		prefix += fmt.Sprintf("  -%s", f.name)
	} else {
		prefix += fmt.Sprintf("      --%s", f.name)
	}
	name, usage := unquoteUsage(f)
	if name != "" {
		prefix += " " + name
	}
	var modifiers []string
	if f.required {
		modifiers = append(modifiers, "REQUIRED")
	}
	if f.deprecated {
		modifiers = append(modifiers, "DEPRECATED")
	}
	if f.hidden {
		modifiers = append(modifiers, "HIDDEN")
	}
	if len(modifiers) > 0 {
		prefix += fmt.Sprintf(" (%s)", strings.Join(modifiers, ", "))
	}
	if f.hasDefault {
		var dftStr string
		if f.isString() {
			dftStr = fmt.Sprintf("(default %q)", f.defValue)
		} else {
			dftStr = fmt.Sprintf("(default %v)", f.defValue)
		}
		usage = spaceJoin(usage, dftStr)
	}
	if len(f.envNames) > 0 {
		envStr := fmt.Sprintf(`(env "%s")`, strings.Join(f.envNames, `", "`))
		usage = spaceJoin(usage, envStr)
	}
	return
}

func unquoteUsage(f *_flag) (name, usage string) {
	usage = f.description
	runes := []rune(f.description)
	const backQuote = '`'
	for i := 0; i < len(runes); i++ {
		if runes[i] == backQuote {
			for j := i + 1; j < len(runes); j++ {
				if runes[j] == backQuote {
					name = string(runes[i+1 : j])
					usage = string(runes[:i]) + name + string(runes[j+1:])
					return name, usage
				}
			}
			break // Only one back quote; use type name.
		}
	}
	if name != "" {
		return
	}

	// Try single quote.
	const backSlash = '\\'
	const singleQuote = '\''
	builder := strings.Builder{}
nextChar:
	for i := 0; i < len(runes); i++ {
		if name == "" && runes[i] == singleQuote && i > 0 && runes[i-1] != backSlash {
			for j := i + 1; j < len(runes); j++ {
				if runes[j] == singleQuote {
					name = string(runes[i+1 : j])
					builder.WriteString(name)
					i = j
					continue nextChar
				}
			}
		}
		if runes[i] == backSlash && i < len(runes)-1 && runes[i+1] == singleQuote {
			builder.WriteRune(singleQuote)
			i++
			continue
		}
		builder.WriteRune(runes[i])
	}
	usage = builder.String()
	if name == "" {
		name = f.usageName()
	}
	return
}

func (f *_flag) validate() error {
	if f.name == "" {
		return newProgramingError("cannot parse name from cli tag %q", f.cliTag)
	}
	if f.hidden && f.nonflag {
		return newProgramingError("shall not set an argument to be hidden, %s", f.name)
	}
	if f.hidden && f.required {
		return newProgramingError("modifiers H & R shall not be used together, %s", f.helpName())
	}
	if f.deprecated && f.required {
		return newProgramingError("modifiers D & R shall not be used together, %s", f.helpName())
	}
	if !isSupportedType(f.rv) {
		return newProgramingError("unsupported value type %v for %s", f.rv.Type(), f.helpName())
	}
	if f.defaultValueTag != "" && !isFlagValueImpl(f.rv) && !isTextValueImpl(f.rv) {
		if f.isSlice() {
			return newProgramingError("default value is unsupported for slice type, %s", f.helpName())
		}
		if f.isMap() {
			return newProgramingError("default value is unsupported for map type, %s", f.helpName())
		}
	}
	if f.envTag != "" {
		if f.isSlice() {
			return newProgramingError("env is unsupported for slice type, %s", f.helpName())
		}
		if f.isMap() {
			return newProgramingError("env is unsupported for map type, %s", f.helpName())
		}
	}
	return nil
}

func parseFlags(isGlobal bool, fs *flag.FlagSet, rv reflect.Value, flagMap map[string]*_flag) (flags, nonflags []*_flag, err error) {
	p := &flagParser{
		fs:      fs,
		flagMap: flagMap,
	}
	rt := rv.Type()

	for i := 0; i < rt.NumField(); i++ {
		ft := rt.Field(i)
		fv := rv.Field(i)
		cliTag := strings.TrimSpace(ft.Tag.Get("cli"))
		if cliTag == "" {
			cliTag = strings.TrimSpace(ft.Tag.Get("mcli"))
		}
		defaultValue := strings.TrimSpace(ft.Tag.Get("default"))
		envTag := strings.TrimSpace(ft.Tag.Get("env"))
		cmplTag := strings.TrimSpace(ft.Tag.Get("cmpl"))

		isGlobalFlag := isGlobal
		if ft.Name == "GlobalFlags" && rt == reflect.TypeOf(withGlobalFlagArgs{}) {
			isGlobalFlag = true
		}

		err = p.parseField(ft, fv, isGlobalFlag, cliTag, defaultValue, envTag, cmplTag)
		if err != nil {
			return nil, nil, err
		}
	}
	if err = p.validateNonflags(); err != nil {
		return nil, nil, err
	}
	p.sortFlags()
	return p.flags, p.nonflags, nil
}

type flagParser struct {
	fs      *flag.FlagSet
	flagMap map[string]*_flag

	flags    []*_flag
	nonflags []*_flag
}

func (p *flagParser) appendFlag(f *_flag) {
	if f.name != "" {
		p.flagMap[f.name] = f
	}
	if f.short != "" {
		p.flagMap[f.short] = f
	}
	p.flags = append(p.flags, f)
}

func (p *flagParser) addToFlagSet(f *_flag, fv reflect.Value) {
	fs := p.fs
	if f.isBoolean() {
		ptr := fv.Addr().Interface().(*bool)
		fs.BoolVar(ptr, f.name, f.rv.Bool(), f.description)
		if f.short != "" {
			fs.BoolVar(ptr, f.short, f.rv.Bool(), f.description)
		}
		return
	}
	if f.isBooleanPtr() {
		ptr := new(bool)
		fs.BoolVar(ptr, f.name, false, f.description)
		if f.short != "" {
			fs.BoolVar(ptr, f.short, false, f.description)
		}
		return
	}
	fs.Var(f, f.name, f.description)
	if f.short != "" {
		fs.Var(f, f.short, f.description)
	}
}

func (p *flagParser) tidyFieldValue(ft reflect.StructField, fv reflect.Value, cliTag string) (reflect.Value, bool) {
	if ft.PkgPath != "" || isIgnoreTag(cliTag) {
		return fv, false
	}
	if fv.IsValid() && fv.Kind() == reflect.Interface {
		fv = fv.Elem()
	}
	if fv.IsValid() && fv.Kind() == reflect.Ptr &&
		!fv.IsNil() && fv.Elem().Kind() == reflect.Struct {
		fv = fv.Elem()
	}
	if !fv.IsValid() {
		return fv, false
	}
	return fv, true
}

func (p *flagParser) parseField(
	ft reflect.StructField, fv reflect.Value,
	isGlobalFlag bool,
	cliTag, defaultValue, envTag string, cmplTag string) error {

	fv, ok := p.tidyFieldValue(ft, fv, cliTag)
	if !ok {
		return nil
	}

	// Got a struct field, parse it recursively.
	if fv.Kind() == reflect.Struct && !isFlagValueImpl(fv) && !isTextValueImpl(fv) {
		subFlags, subNonflags, subErr := parseFlags(isGlobalFlag, p.fs, fv, p.flagMap)
		if subErr != nil {
			return subErr
		}
		for _, f := range subFlags {
			p.appendFlag(f)
		}
		p.nonflags = append(p.nonflags, subNonflags...)
		return nil
	}
	if cliTag == "" {
		return nil
	}

	// Parse the flag.
	var f *_flag
	f, err := p.parseFlag(isGlobalFlag, cliTag, defaultValue, envTag, fv, cmplTag)
	if err != nil {
		return err
	}
	if f == nil || f.name == "" {
		return nil
	}
	if f.nonflag {
		p.nonflags = append(p.nonflags, f)
		return nil
	}

	p.appendFlag(f)
	p.addToFlagSet(f, fv)
	return nil
}

var spaceRE = regexp.MustCompile(`\s+`)

func (p *flagParser) parseFlag(isGlobal bool, cliTag, defaultValue, envTag string, rv reflect.Value, cmplTag string) (*_flag, error) {
	f := &_flag{
		_tags: _tags{
			cliTag:          cliTag,
			defaultValueTag: defaultValue,
			envTag:          envTag,
		},
		_value:             _value{rv},
		isGlobal:           isGlobal,
		completionFunction: cmplTag,
	}

	p.parseCliTag(f, cliTag)

	if f.name == "" {
		f.name = f.short
	}
	if f.short == f.name {
		f.short = ""
	}
	if err := f.validate(); err != nil {
		return nil, err
	}
	if defaultValue != "" {
		err := f.Set(defaultValue)
		if err != nil {
			return nil, newProgramingError("invalid default value %q for %s: %v", defaultValue, f.helpName(), err)
		}
		f.defValue = defaultValue
		f.hasDefault = !f.isZero()
	}
	if envTag != "" {
		f.envNames = splitByComma(envTag)
	}
	return f, nil
}

func (p *flagParser) parseCliTag(f *_flag, cliTag string) {
	const (
		modifier = iota
		short
		long
		description
		stop
	)
	parts := strings.SplitN(cliTag, ",", 4)
	st := modifier
	for i := 0; i < len(parts) && st < stop; i++ {
		p := strings.TrimSpace(parts[i])
		switch st {
		case modifier:
			st = short
			if strings.HasPrefix(p, "#") {
				for _, x := range p[1:] {
					Modifier(x).apply(f)
				}
				continue
			}
			i--
		case short:
			if strings.HasPrefix(p, "-") {
				st = long
				p = strings.TrimLeft(p, "-")
				if len(p) == 1 {
					f.short = p
				} else {
					i--
				}
			} else {
				st = description
				f.nonflag = true
				f.name = p
			}
		case long:
			st = description
			if strings.HasPrefix(p, "-") {
				p = strings.TrimLeft(p, "-")
				// Allow split flag name and description by spaces.
				sParts := spaceRE.Split(p, 2)
				f.name = sParts[0]
				newParts := append(parts[:i:i], sParts...)
				newParts = append(newParts, parts[i+1:]...)
				parts = newParts
				continue
			}
			f.name = f.short
			i--
		case description:
			st = stop
			p = strings.TrimSpace(strings.Join(parts[i:], ","))
			f.description = p
		}
	}
}

func (p *flagParser) validateNonflags() error {
	var compositeTypeArg *_flag
	for _, f := range p.nonflags {
		if compositeTypeArg != nil {
			return newProgramingError("%s after composite type %s will never get a value, you may define it as a flag", f.helpName(), compositeTypeArg.helpName())
		}
		if f.isCompositeType() {
			compositeTypeArg = f
		}
	}
	return nil
}

func (p *flagParser) sortFlags() {
	sort.Slice(p.flags, func(i, j int) bool {
		return strings.ToLower(p.flags[i].name) < strings.ToLower(p.flags[j].name)
	})
}

func isIgnoreTag(tag string) bool {
	parts := strings.Split(tag, ",")
	return strings.TrimSpace(parts[0]) == "-"
}

func isSupportedType(rv reflect.Value) bool {
	if _, ok := rv.Interface().(bool); ok {
		return true
	}
	if isFlagValueImpl(rv) || isTextValueImpl(rv) {
		return true
	}
	if isSupportedBasicTypePtr(rv.Type()) {
		return true
	}
	if isSupportedBasicType(rv.Kind()) {
		return true
	}
	if rv.Kind() == reflect.Slice && isSupportedBasicType(rv.Type().Elem().Kind()) {
		return true
	}
	if rv.Kind() == reflect.Map &&
		rv.Type().Key().Kind() == reflect.String &&
		isSupportedBasicType(rv.Type().Elem().Kind()) {
		return true
	}
	return false
}

func isFlagValueImpl(rv reflect.Value) bool {
	return rv.Type().Implements(flagValueTyp) ||
		(rv.CanAddr() && rv.Addr().Type().Implements(flagValueTyp))
}

func zeroFlagValueStr(rv reflect.Value) string {
	typ := rv.Type()
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	zero := reflect.New(typ)
	zero.Elem().Set(reflect.Zero(typ))
	return zero.Interface().(flag.Value).String()
}

func isTextValueImpl(rv reflect.Value) bool {
	return rv.Type().Implements(textValueTyp) ||
		(rv.CanAddr() && rv.Addr().Type().Implements(textValueTyp))
}

func zeroTextValueStr(rv reflect.Value) string {
	typ := rv.Type()
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	zero := reflect.New(typ)
	zero.Elem().Set(reflect.Zero(typ))
	b, _ := zero.Interface().(textValue).MarshalText()
	return string(b)
}

func isSupportedBasicTypePtr(typ reflect.Type) bool {
	return typ.Kind() == reflect.Pointer && isSupportedBasicType(typ.Elem().Kind())
}

func isSupportedBasicType(kind reflect.Kind) bool {
	switch kind {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.String:
		return true
	}
	return false
}

func splitByComma(value string) []string {
	value = strings.TrimSpace(value)
	parts := strings.Split(value, ",")
	out := parts[:0]
	for _, x := range parts {
		x = strings.TrimSpace(x)
		if x != "" {
			out = append(out, x)
		}
	}
	return out
}

func hasBoolFlag(name string, args []string) bool {
	for _, a := range args {
		if !strings.HasPrefix(a, "-") || !strings.Contains(a, name) {
			continue
		}
		a = strings.TrimLeft(a, "-")
		a = strings.SplitN(a, "=", 2)[0]
		if a == name {
			return true
		}
	}
	return false
}

func spaceJoin(strList ...string) string {
	result := ""
	for _, s := range strList {
		s = strings.TrimSpace(s)
		if result == "" {
			result = s
		} else {
			result += " " + s
		}
	}
	return result
}
