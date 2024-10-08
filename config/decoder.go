package config

import (
	"gopkg.in/yaml.v3"
	"io/fs"
)

// Decoder of YAML files for aconfig.
type Decoder struct {
	fsys fs.FS
}

// New YAML decoder for aconfig.
func New() *Decoder { return &Decoder{} }

// Format of the decoder.
func (d *Decoder) Format() string {
	return "yaml"
}

// DecodeFile implements aconfig.FileDecoder.
func (d *Decoder) DecodeFile(filename string) (map[string]interface{}, error) {
	f, err := d.fsys.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var raw map[string]interface{}
	if err := yaml.NewDecoder(f).Decode(&raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// DecodeFile implements aconfig.FileDecoder.
func (d *Decoder) Init(fsys fs.FS) {
	d.fsys = fsys
}
