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
	svg "github.com/h2non/go-is-svg"
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

type image string

// validate returns nil if img points to a supported image file, otherwise
// it returns an appropriate error
func (img *image) validate() error {
	imgStr := string(*img)
	// not providing an image is allowed:
	if imgStr == "" {
		return nil
	}

	buf, err := os.ReadFile(imgStr)
	if err != nil {
		return fmt.Errorf("file %q appears unaccesible", imgStr)
	}
	if !filetype.IsImage(buf) || svg.IsSVG(buf) {
		return fmt.Errorf("file %q is not an image file", imgStr)
	}
	return nil
}

type measurement string

// validate makes sure entryMeasurement follows one of the allowed patterns
func (m *measurement) validate() error {
	parseError := fmt.Errorf("invalid geometry value %q", *m)
	if len(string(*m)) == 1 {
		return parseError
	}

	mStr := string(*m)

	if mStr == "" {
		return nil
	} else if mStr[0] == '%' {
		uI, _ := strconv.ParseUint(mStr[1:], 10, 32)
		if uI == 0 || uI > 100 {
			return parseError
		}
	} else if mStr[len(mStr)-1:] == "%" {
		uI, _ := strconv.ParseUint(mStr[:len(mStr)-1], 10, 32)
		if uI == 0 || uI > 100 {
			return parseError
		}
	} else if mStr[len(mStr)-2:] == "px" {
		uI, _ := strconv.ParseUint(mStr[:len(mStr)-2], 10, 32)
		if uI == 0 {
			return parseError
		}
	} else {
		return parseError
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
		return nil, err
	}

	ini, err := ini.LoadSources(
		ini.LoadOptions{
			AllowNonUniqueSections: true,
			Insensitive:            true,
		},
		f)
	if err != nil {
		return nil, err
	}

	return ini, nil
}

type settings struct {
	Background image
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
			return fmt.Errorf("unknown key %s in section %s", key.Name(), sec.Name())
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
	// t := v.Type()
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
				return err
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
