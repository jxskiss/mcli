package mcli

import (
	"flag"
	"reflect"
	"unsafe"
)

func tidyFlags(fs *flag.FlagSet, flags []*_flag, nonflagArgs []string) {
	m := make(map[string]*_flag)
	for _, f := range flags {
		m[f.name] = f
		if f.short != "" {
			m[f.short] = f
		}
	}

	// This is awkward, but we can not simply call flag.Value's Set
	// method, the Set operation may be not idempotent.
	// Thus, we unsafely modify FlagSet's unexported internal data,
	// this may break in a future Go release.

	actual := _flagSet_getActual(fs)
	formal := _flagSet_getFormal(fs)
	fs.Visit(func(ff *flag.Flag) {
		f := m[ff.Name]
		if f == nil {
			return
		}

		// Special processing for *bool value.
		if f.isBooleanPtr() {
			f.rv.Set(reflect.New(f.rv.Type().Elem()))
			f.rv.Elem().SetBool(ff.Value.String() == "true")
		}

		if f.name != ff.Name {
			formal[f.name].Value = ff.Value
			actual[f.name] = formal[f.name]
		}
		if f.short != "" && f.short != ff.Name {
			formal[f.short].Value = ff.Value
			actual[f.short] = formal[f.short]
		}
	})

	if len(nonflagArgs) > 0 {
		_flagSet_setArgs(fs, nonflagArgs)
	}
}

var (
	_flagSet_actual_offset uintptr
	_flagSet_formal_offset uintptr
	_flagSet_args_offset   uintptr
	_flagSetMapType        = reflect.TypeOf(map[string]*flag.Flag{})
)

func init() {
	typ := reflect.TypeOf(flag.FlagSet{})
	actualField, ok1 := typ.FieldByName("actual")
	formalField, ok2 := typ.FieldByName("formal")
	if !ok1 || !ok2 {
		panic("mcli: cannot find flag.FlagSet fields actual/formal")
	}
	argsField, ok3 := typ.FieldByName("args")
	if !ok3 {
		panic("mcli: cannot find flag.FlagSet field args")
	}
	if actualField.Type != _flagSetMapType || formalField.Type != _flagSetMapType {
		panic("mcli: type of flag.FlagSet fields actual/formal is not map[string]*flag.Flag")
	}
	_flagSet_actual_offset = actualField.Offset
	_flagSet_formal_offset = formalField.Offset
	_flagSet_args_offset = argsField.Offset
}

func _flagSet_getActual(fs *flag.FlagSet) map[string]*flag.Flag {
	return *(*map[string]*flag.Flag)(unsafe.Pointer(uintptr(unsafe.Pointer(fs)) + _flagSet_actual_offset))
}

func _flagSet_getFormal(fs *flag.FlagSet) map[string]*flag.Flag {
	return *(*map[string]*flag.Flag)(unsafe.Pointer(uintptr(unsafe.Pointer(fs)) + _flagSet_formal_offset))
}

func _flagSet_setArgs(fs *flag.FlagSet, args []string) {
	*(*[]string)(unsafe.Pointer(uintptr(unsafe.Pointer(fs)) + _flagSet_args_offset)) = args
}
