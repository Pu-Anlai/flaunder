package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/h2non/filetype"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"gopkg.in/ini.v1"
)

type entry struct {
	Name       string
	Icon       icon
	IconHeight measurement
	Command    string
}

type font struct {
	path   string
	height float64
	face   *text.GoTextFace
}

func (f *font) setBaseField(v string) {
	f.path = v
}

// validate checks if path exists and returns an error if it doesn't
func (f *font) validate() error {
	// not providing a font is allowed
	if f.path == "" {
		return nil
	}

	fontFile, err := os.ReadFile(f.path)
	if err != nil {
		return &fileAccessError{
			path: f.path,
		}
	}
	if !filetype.IsFont(fontFile) {
		return &fileAccessError{
			path:     f.path,
			fileType: "font",
		}
	}

	return nil
}

type measurement struct {
	value string
	abs   float64
}

func (m *measurement) setBaseField(v string) {
	m.value = v
}

// validate makes sure entryMeasurement follows one of the allowed patterns
func (m *measurement) validate() error {
	if len(m.value) == 1 {
		return &iniParseError{value: m.value}
	}

	if m.value == "" {
		return nil
	} else if m.value[0] == '%' {
		uI, _ := strconv.ParseUint(m.value[1:], 10, 32)
		if uI == 0 || uI > 100 {
			return &iniParseError{value: m.value}
		}
	} else if m.value[len(m.value)-1:] == "%" {
		uI, _ := strconv.ParseUint(m.value[:len(m.value)-1], 10, 32)
		if uI == 0 || uI > 100 {
			return &iniParseError{value: m.value}
		}
	} else if m.value[len(m.value)-2:] == "px" {
		uI, _ := strconv.ParseUint(m.value[:len(m.value)-2], 10, 32)
		if uI == 0 {
			return &iniParseError{value: m.value}
		}
	} else {
		return &iniParseError{value: m.value}
	}

	return nil
}

type option interface {
	validate() error
}

type complexOption interface {
	setBaseField(string)
}

// readIniFile reads an ini file with a set of preset options
func readIniFile(path string) (*ini.File, error) {
	f, err := os.ReadFile(path)
	if err != nil {
		return nil, &fileAccessError{path: path, fileType: "ini"}
	}

	ini, err := ini.LoadSources(
		ini.LoadOptions{
			AllowNonUniqueSections: true,
			Insensitive:            true,
		},
		f)
	if err != nil {
		return nil, &fileAccessError{path: path, fileType: "ini"}
	}

	return ini, nil
}

type settings struct {
	Background                icon
	Font                      font
	FontSize                  int
	ImageTitlePadding         measurement
	TopPadding, BottomPadding measurement
	LeftPadding, RightPadding measurement
}

type config struct {
	settings *settings
	entries  []entry
}

// readIntoStruct reads an ini section sec into a struct. If the section
// contains a key that does not match a struct field, an error will be returned.
func readIntoStruct[T any](sec *ini.Section, dst *T) error {
	// get the value that dst points to
	v := reflect.ValueOf(dst).Elem()
	// create a map where we can look up fields of the struct dst by their names
	fields := reflect.VisibleFields(v.Type())
	fieldMap := make(map[string]reflect.StructField, len(fields))
	for i := range fields {
		fieldMap[strings.ToLower(fields[i].Name)] = fields[i]
	}

	// loop over the ini section that was passed (sec)
	for _, key := range sec.Keys() {
		// if there is an ini key that is not in our struct dst, return an error
		field, ok := fieldMap[key.Name()]
		if !ok {
			return &iniParseError{key: key.Name(), section: sec.Name()}
		}
		// now that we have the StructField corresponding to the key name in the
		// ini file, we can look up the value of that field in our instance v
		fieldVal := v.FieldByIndex(field.Index)
		// we're supporting string and int fields as well as those implementing
		// complexOption
		switch fieldVal.Kind() {
		case reflect.String:
			fieldVal.SetString(key.Value())
		case reflect.Int:
			if val, err := strconv.Atoi(key.Value()); err != nil {
				return &iniParseError{key: key.Name(), section: sec.Name()}
			} else {
				fieldVal.SetInt(int64(val))
			}
		default:
			// if the field is not a built-in type, check if it implements
			// complexOption
			if opt, ok := fieldVal.Addr().Interface().(complexOption); ok {
				opt.setBaseField(key.Value())
			} else {
				// if none of that works, we'll panic, this should be
				// preventable on the code level
				panic(fmt.Sprintf("wrong field type: %+v", field))
			}
		}
	}
	return nil
}

// parseConfigFile reads confFile and returns a config struct containing all app
// entries as well as general settings. If there any errors are encountered when
// parsing the file, an error will be returned.
func parseConfigFile(confFile *ini.File) (*config, error) {
	var c config
	for _, sec := range confFile.Sections() {
		switch sec.Name() {
		case "default":
			continue
		case "global":
			if c.settings != nil {
				return nil, errors.New("global section in config not unique")
			}
			set := new(settings)
			if err := readIntoStruct(sec, set); err != nil {
				return nil, err
			}
			c.settings = set
		default:
			e := entry{}
			if err := readIntoStruct(sec, &e); err != nil {
				return nil, err
			}
			c.entries = append(c.entries, e)
		}
	}
	return &c, nil
}

// validateConfig validates settings and all entries in conf
func validateConfig(conf *config) error {
	if err := validateSettings(conf.settings); err != nil {
		return err
	}
	if len(conf.entries) == 0 {
		return errors.New("at least one application must be specified in config")
	}

	for i := range conf.entries {
		if err := validateSettings(&conf.entries[i]); err != nil {
			return err
		}
	}
	return nil
}

// validateSettings runs validate on all option fields in the object c
func validateSettings[T settings | entry](s *T) error {
	v := reflect.ValueOf(s).Elem()
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if !f.CanInterface() {
			continue
		}
		// we're *only* supporting validate on pointer receivers so skip
		// non-pointers
		if !f.CanAddr() {
			continue
		}

		if opt, ok := f.Addr().Interface().(option); ok {
			if err := opt.validate(); err != nil {
				return &iniParseError{key: f.Type().Name(), value: f.String()}
			}
		}
	}
	return nil
}

// getConfig reads the config file at path, creates a config struct based on its
// content and returns a pointer to the config
func getConfig() (*config, error) {
	home, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	ini, err := readIniFile(filepath.Join(home, "fyne-idim", "config"))
	if err != nil {
		return nil, err
	}
	conf, err := parseConfigFile(ini)
	if err != nil {
		return nil, err
	}
	if err = validateConfig(conf); err != nil {
		return nil, err
	}

	return conf, nil
}
