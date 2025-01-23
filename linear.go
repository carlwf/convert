package convert

import (
	"encoding/json"
	"os"
)

// linearConverter implements Converter and contains the logic and attributes for the
// linear conversion of UOM.
type linearConverter struct {
	name     string
	symbol   string
	baseuom  string
	category string
	factor   float64 // factor to convert to the base UOM for this category. cannot be 0 - protect in MakeLinearUOM.
	offset   float64 // offset to convert to the base UOM for this category.
}

func LinearConverter(name, symbol, baseunit, category string, factor, offset float64) (linearConverter, error) {
	if name == "" || baseunit == "" || category == "" {
		return linearConverter{}, ErrMissingData
	}
	if factor == 0 {
		return linearConverter{}, ErrZeroNotAllowed
	}

	newUnit := linearConverter{
		name:     name,
		symbol:   symbol,
		baseuom:  baseunit,
		category: category,
		factor:   factor,
		offset:   offset,
	}
	return newUnit, nil
}

// Convert converts val from the converter type defined in from from to that defined
// in to and returns the converted value and nil, or 0 and an error.
func (from linearConverter) Convert(val float64, to Converter) (float64, error) {
	if from.BaseUOM() != to.BaseUOM() || from.Category() != to.Category() {
		return 0, ErrIncompatibleUnits
	}

	tto, ok := to.(linearConverter)
	if !ok {
		return 0, ErrIncompatibleUnits
	}
	return ((val*from.factor + from.offset) - tto.offset) / tto.factor, nil
	// return ((val*from.factor + from.offset) - tto.offset) / from.factor, nil
}

// Name returns the name of the unit.
func (u linearConverter) Name() string {
	return u.name
}

// Symbol returns the symbol of the unit.
func (u linearConverter) Symbol() string {
	return u.symbol
}

// Category returns the category of the unit Converter.
func (f linearConverter) Category() string {
	return f.category
}

// BaseUOM returns the base unit of the unit Converter.
func (f linearConverter) BaseUOM() string {
	return f.baseuom
}

// #
// #
// #
// fileLayout represents the structure of json files that contains Converter data for linear UOMs.
type fileLayout struct {
	Category    string `json:"category"`
	Description string `json:"description"`
	BaseUnit    string `json:"baseunit"`
	Units       []struct {
		Name     string  `json:"name"`
		Symbol   string  `json:"symbol"`
		BaseUnit string  `json:"baseunit"`
		Factor   float64 `json:"factor"`
		Offset   float64 `json:"offset"`
	} `json:"units"`
}

// LinUOMReader returns a new instance of fileLayout that can be used to read
// linear uom data from a json file.
func LinearReader() *fileLayout {
	return new(fileLayout)
}

func (fl *fileLayout) ReadFile(filename string) ([]Converter, error) {
	var converters []Converter
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	err = json.NewDecoder(f).Decode(&fl)
	if err != nil {
		return nil, err
	}
	for _, u := range fl.Units {
		newUnit, err := LinearConverter(u.Name, u.Symbol, fl.BaseUnit, fl.Category, u.Factor, u.Offset)
		if err != nil {
			return nil, err
		}
		converters = append(converters, newUnit)
	}
	return converters, nil
}
