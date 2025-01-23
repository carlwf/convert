// Package convert provides functionality for converting values between different
// units of measurement.
//
// Usage:
//
// To convert a value from one unit to another, use the ToValue function:
//
//	val, err := convert.ToValue(value, from, to), where from and to are strings
//
// To convert a value from one unit to another and get the result as a JSON
//
//	jsonBytes, err := convert.ToJson(value, from, to), where from and to are strings
//
// To get a list of all units in a given category, use the UnitsByCategory function:
//
//	units := convert.UnitsByCategory("length")
//
// To get a list of all categories of units, use the Categories function:
//
//	categories := convert.Categories()
//
// To add/update a Converter to/in the store, use the AddConverter function:
//
//	c := &MyConverter{...}
//	convert.AddConverter(c)
//
// To remove a Converter from the store, use the RemoveConverter function:
//
//	convert.RemoveConverter("myconverter")
//
// To clear all Converters from the store, use the Clear function:
//
//	convert.Clear()
//
// To add/update Converters from files, use the AddFromFiles function:
//
//	convert.AddFromFiles(myConverterReader, "path/to/converters/*.json")
//
// The Converter interface defines the methods that must be implemented by a
// unit of measurement (UOM) converter. The ToValue and ToJson functions use
// this interface to perform the conversion.
package convert

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

var (
	ErrUnknownUnit       = errors.New("unknown unit")
	ErrCategoryMismatch  = errors.New("units are not in the same category")
	ErrMissingData       = errors.New("missing data")
	ErrZeroNotAllowed    = errors.New("zero not allowed")
	ErrIncompatibleUnits = errors.New("incompatible units")
)

// A Converter represents a unit of measurement (UOM) that can be converted to
// another UOM that has the same base UOM.
type Converter interface {
	Convert(value float64, toUOM Converter) (float64, error)
	Name() string
	Symbol() string
	Category() string
	BaseUOM() string
}

// ToValue converts val from the unit specified by from to the unit
// specified by to. It returns the converted value and nil, or 0 and an error.
func ToValue(val float64, from, to string) (float64, error) {
	f, ok := store.get(strings.ToLower(from))
	if !ok {
		return 0, Error(ErrUnknownUnit, from)
	}
	t, ok := store.get(strings.ToLower(to))
	if !ok {
		return 0, Error(ErrUnknownUnit, to)
	}
	return f.Convert(val, t)
}

// ToJson converts val from the unit specified by from to the unit specified by
// to and returns the result as a JSON formatted byte slice with information
// about the conversion.
func ToJson(val float64, from, to string) ([]byte, error) {
	type response struct {
		Ok         bool    `json:"ok"`
		Message    string  `json:"message,omitempty"`
		Result     float64 `json:"result,omitempty"`
		Category   string  `json:"category,omitempty"`
		From       string  `json:"from,omitempty"`
		FromSymbol string  `json:"fromsymbol,omitempty"`
		To         string  `json:"to,omitempty"`
		ToSymbol   string  `json:"tosymbol,omitempty"`
		BaseUOM    string  `json:"baseuom,omitempty"`
	}

	f, _ := store.get(strings.ToLower(from))
	t, _ := store.get(strings.ToLower(to))
	val, err := ToValue(val, from, to)
	if err != nil {
		msg := err.Error()
		resp := response{
			Ok:      false,
			Message: msg,
		}
		return json.Marshal(resp)
	}

	resp := response{
		Ok:         true,
		Message:    "success",
		Result:     val,
		Category:   t.Category(),
		From:       from,
		FromSymbol: f.Symbol(),
		To:         to,
		ToSymbol:   t.Symbol(),
		BaseUOM:    t.BaseUOM(),
	}
	return json.Marshal(resp)
}

// converterStore is a thread-safe in-memory store for Converters.
type converterStore struct {
	mu   sync.RWMutex
	data map[string]Converter
}

// store is the global instance of the converterStore.
var store = converterStore{data: make(map[string]Converter)}

type ConverterReader interface {
	ReadFile(string) ([]Converter, error)
}

func AddFromFiles(reader ConverterReader, path string) error {
	files, err := filepath.Glob(path)
	if err != nil {
		return err
	}

	for _, file := range files {
		if strings.HasSuffix(file, ".json") {
			cs, err := reader.ReadFile(file)
			if err != nil {
				return err
			}
			for _, c := range cs {
				store.add(c)
			}
		}
	}
	return nil
}

// get retrieves a Converter from store based on the provided name. It
// returns the Converter and a boolean indicating whether it was found in the
// store.
func (s *converterStore) get(name string) (Converter, bool) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if c, ok := store.data[strings.ToLower(name)]; ok {
		return c, true
	}

	return nil, false
}

// AddConverter adds/updates a Converter to/in the store.
func (s *converterStore) add(c Converter) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.data[strings.ToLower(c.Name())] = c
}

// RemoveConverter removes a Converter from the cache based on the provided
// name. If the Converter is not in the cache, it does nothing.
func (s *converterStore) remove(name string) {
	store.mu.Lock()
	defer store.mu.Unlock()
	delete(store.data, strings.ToLower(name))
}

// Clear removes all Converters from the cache.
func (s *converterStore) clear() {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.data = make(map[string]Converter)
}

// Categories returns a list of all categories of Converters in the cache.
func Categories() []string {
	store.mu.RLock()
	defer store.mu.RUnlock()

	categories := make(map[string]bool)
	for _, c := range store.data {
		categories[c.Category()] = true
	}

	result := make([]string, 0, len(categories))
	for c := range categories {
		result = append(result, c)
	}

	slices.SortFunc(result, func(a, b string) int {
		return strings.Compare(strings.ToLower(a), strings.ToLower(b))
	})

	return result
}

type Uom struct {
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Category string `json:"category"`
	BaseUOM  string `json:"baseUOM"`
}

func UnitsByCategory(category string) []Uom {
	units := make([]Uom, 0, len(store.data))
	for _, c := range store.data {
		if c.Category() == category {
			units = append(units, Uom{
				Name:     c.Name(),
				Symbol:   c.Symbol(),
				Category: c.Category(),
				BaseUOM:  c.BaseUOM(),
			})
		}
	}

	slices.SortFunc(units, func(a, b Uom) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})

	return units

}

func Error(err error, msg string) error {
	if err != nil {
		return errors.New(err.Error() + ": " + msg)
	}
	return err
}
