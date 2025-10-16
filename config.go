package main

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/h2non/filetype"
	svg "github.com/h2non/go-is-svg"
	"gopkg.in/ini.v1"
)

type entry struct {
	Name        string
	Image       entryImage
	ImageHeight entryMeasurement
	ImageWidth  entryMeasurement
	Command     string
}

type entryImage string

// validate returns nil if img points to a supported image file, otherwise
// it returns an appropriate error
func (img *entryImage) validate() error {
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

type entryMeasurement string

// validate
func (m *entryMeasurement) validate() error {
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

type entryOption interface {
	validate() error
}

// validateEntryOption runs validate() on entryOption
func validateEntryOption(eO entryOption) error {
	if err := eO.validate(); err != nil {
		return err
	} else {
		return nil
	}
}

type settings struct {
	Background string
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

// parseConfig reads confFile and returns a config struct containing all app
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

// validateEntries makes sure that every entry in slice is a valid reference to
// an existing application. If it is not, an error is returned.
func validateEntries(e []entry) error {
	if len(e) == 0 {
		return errors.New("at least one application must be specified in config")
	}

	for i := range e {
		entryOptions := []entryOption{
			&e[i].ImageHeight,
			&e[i].ImageWidth,
			&e[i].Image,
		}
		for j := range entryOptions {
			if err := entryOptions[j].validate(); err != nil {
				return err
			}
		}
	}
	return nil
}
