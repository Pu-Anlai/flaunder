package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"gopkg.in/ini.v1"
)

type iniParseError struct {
	key     string
	value   string
	section string
}

func (e *iniParseError) Error() string {
	if e.section == "" {
		return fmt.Sprintf("%s: cannot parse key %q (value %q)", e.section, e.key, e.value)
	} else {
		return fmt.Sprintf("cannot parse key %q (value %q)", e.key, e.value)
	}
}

type fileAccessError struct {
	path     string
	fileType string
}

func (e *fileAccessError) Error() string {
	if e.fileType == "" {
		return fmt.Sprintf("cannot access file %q", e.path)
	} else {
		return fmt.Sprintf("cannot access %s file %q", e.fileType, e.path)
	}
}

type entry struct {
	Name       string
	Icon       icon
	IconHeight measurement
	Command    string
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
	Background                icon
	BackgroundScale           bool
	Font                      font
	FontSize                  int
	IconTitlePadding          measurement
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
			return &iniParseError{key: key.Name(), value: key.Value(), section: sec.Name()}
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
				return &iniParseError{key: key.Name(), value: key.Value(), section: sec.Name()}
			} else {
				fieldVal.SetInt(int64(val))
			}
		case reflect.Bool:
			if val, err := strconv.ParseBool(key.Value()); err != nil {
				return &iniParseError{key: key.Name(), value: key.Value(), section: sec.Name()}
			} else {
				fieldVal.SetBool(val)
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

// getConfigPath returns the path to the config file
func getConfigPath() (string, error) {
	home, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "flunder", "config"), nil
}

// getConfig reads the config file at path, creates a config struct based on its
// content and returns a pointer to the config
func getConfig(path string) (*config, error) {
	ini, err := readIniFile(path)
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
