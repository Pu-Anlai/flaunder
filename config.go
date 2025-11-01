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
	"gopkg.in/ini.v1"
)

type entry struct {
	Name           string
	Image          image
	ImageHeight    measurement
	imageHeightAbs int
	ImageWidth     measurement
	imageWidthAbs  int
	Command        string
}

type font string

// validate checks if path exists and returns an error if it doesn't
func (f *font) validate() error {
	fStr := string(*f)
	fontFile, err := os.ReadFile(fStr)
	if err != nil {
		return &fileAccessError{
			path: fStr,
		}
	}
	if !filetype.IsFont(fontFile) {
		return &fileAccessError{
			path:     fStr,
			fileType: "font",
		}
	}
	return nil
}

type measurement string

// validate makes sure entryMeasurement follows one of the allowed patterns
func (m *measurement) validate() error {
	if len(string(*m)) == 1 {
		return &iniParseError{value: string(*m)}
	}

	mStr := string(*m)

	if mStr == "" {
		return nil
	} else if mStr[0] == '%' {
		uI, _ := strconv.ParseUint(mStr[1:], 10, 32)
		if uI == 0 || uI > 100 {
			return &iniParseError{value: string(*m)}
		}
	} else if mStr[len(mStr)-1:] == "%" {
		uI, _ := strconv.ParseUint(mStr[:len(mStr)-1], 10, 32)
		if uI == 0 || uI > 100 {
			return &iniParseError{value: string(*m)}
		}
	} else if mStr[len(mStr)-2:] == "px" {
		uI, _ := strconv.ParseUint(mStr[:len(mStr)-2], 10, 32)
		if uI == 0 {
			return &iniParseError{value: string(*m)}
		}
	} else {
		return &iniParseError{value: string(*m)}
	}

	return nil
}

type option interface {
	validate() error
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
	Background image
	Font       font
}

type config struct {
	settings *settings
	entries  []entry
}

// readIntoStruct reads an ini section sec into a struct. If the section
// contains a key that does not match a struct field, an error will be returned.
func readIntoStruct[T any](sec *ini.Section, dst *T) error {
	v := reflect.ValueOf(dst).Elem()
	fieldMap := make(map[string]reflect.StructField, 7)
	fields := reflect.VisibleFields(v.Type())
	for i := range fields {
		fieldMap[strings.ToLower(fields[i].Name)] = fields[i]
	}
	for _, key := range sec.Keys() {
		field, ok := fieldMap[key.Name()]
		if !ok {
			return &iniParseError{key: key.Name(), section: sec.Name()}
		}
		fieldVal := v.FieldByIndex(field.Index)
		if fieldVal.Kind() != reflect.String {
			panic(fmt.Sprintf("wrong field type: %+v", field))
		}
		fieldVal.SetString(key.Value())
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
