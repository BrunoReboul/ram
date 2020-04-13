// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the 'License');
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an 'AS IS' BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package validater helps to validate struct fields
package validater

import (
	"fmt"
	"log"
	"reflect"
	"strings"
)

const tagKeyName = "valid"

// validater interface
type validater interface {
	validate(interface{}) (bool, error)
}

// defaultValidater is always valid
type defaultValidater struct {
}

// validate interface returns true for a valid field, false and why in the error otherwise
func (v defaultValidater) validate(val interface{}) (bool, error) {
	return true, nil
}

// isNotZeroValueValidater do not accept zero value
type isNotZeroValueValidater struct {
}

// validate interface returns true for a valid field, false and why in the error otherwise
func (v isNotZeroValueValidater) validate(value interface{}) (bool, error) {
	typ := reflect.TypeOf(value)
	kind := typ.Kind()
	switch kind {
	case reflect.String:
		l := len(value.(string))
		if l == 0 {
			return false, fmt.Errorf("should NOT be a zero value")
		}

	}
	return true, nil
}

func getValidater(kind reflect.Kind, tagValue string) validater {
	tagValueParts := strings.Split(tagValue, ",")
	tagPrefix := tagValueParts[0]
	switch tagPrefix {
	case "isNotZeroValue":
		return isNotZeroValueValidater{}
	}
	return defaultValidater{}
}

// getValidationErrors recursively loop through a struct to find validation errors
func getValidationErrors(structure interface{}, pedigree string) []error {
	errs := []error{}
	if structure == nil {
		return errs
	}
	value := reflect.ValueOf(structure)
	if value.Kind() == reflect.Interface || value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return []error{fmt.Errorf("type %s is not a struct", value.Kind())}
	}

	for i := 0; i < value.NumField(); i++ {
		valueField := value.Field(i)
		typeField := value.Type().Field(i)
		if valueField.Kind() == reflect.Interface {
			valueField = valueField.Elem()
		}
		if typeField.Tag.Get(tagKeyName) != "-" &&
			(valueField.Kind() == reflect.Struct || (valueField.Kind() == reflect.Ptr && valueField.Elem().Kind() == reflect.Struct)) {
			// log.Printf("Explore %s %s", typeField.Type.Kind(), typeField.Name)
			childErrs := getValidationErrors(valueField.Interface(), fmt.Sprintf("%s/%s", pedigree, typeField.Name))
			errs = append(errs, childErrs...)
		} else {
			// log.Printf("%s %s %s %s", pedigree, typeField.Name, typeField.Type.Kind(), typeField.Tag.Get(tagKeyName))
			validater := getValidater(typeField.Type.Kind(), typeField.Tag.Get(tagKeyName))
			ok, err := validater.validate(valueField.Interface())
			if !ok {
				errs = append(errs, fmt.Errorf("Validater error %s '%s' %v", pedigree, typeField.Name, err))
			}
		}
	}
	return errs
}

// ValidateStruct validates the fields of a struct
func ValidateStruct(structure interface{}, pedigree string) (err error) {
	errors := getValidationErrors(structure, pedigree)
	if len(errors) > 0 {
		for _, err := range errors {
			log.Println(err)
		}
		err = fmt.Errorf("Error, settings validation failed")
		return err
	}
	return nil
}
