// Copyright 2015 xeipuuv ( https://github.com/xeipuuv )
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// author           xeipuuv
// author-github    https://github.com/xeipuuv
// author-mail      xeipuuv@gmail.com
//
// repository-name  gojsonschema
// repository-desc  An implementation of JSON Schema, based on IETF's draft v4 - Go language.
//
// description      Extends Schema and subSchema, implements the validation phase.
//
// created          28-02-2013

package gojsonschema

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

func Validate(ls JSONLoader, ld JSONLoader) (*Result, error) {

	var err error

	// load schema

	schema, err := NewSchema(ls)
	if err != nil {
		return nil, err
	}

	// begine validation

	return schema.Validate(ld)

}

func (v *Schema) Validate(l JSONLoader) (*Result, error) {

	// load document

	root, err := l.loadJSON()
	if err != nil {
		return nil, err
	}

	// begin validation

	result := &Result{}
	context := newJsonContext(STRING_CONTEXT_ROOT, nil)
	v.rootSchema.validateRecursive(v.rootSchema, root, result, context)

	return result, nil

}

func (v *subSchema) subValidateWithContext(document interface{}, context *jsonContext) *Result {
	result := &Result{}
	v.validateRecursive(v, document, result, context)
	return result
}

// Walker function to validate the json recursively against the subSchema
func (v *subSchema) validateRecursive(currentSubSchema *subSchema, currentNode interface{}, result *Result, context *jsonContext) {

	internalLog("validateRecursive %s", context.String())
	internalLog(" %v", currentNode)

	// Handle referenced schemas, returns directly when a $ref is found
	if currentSubSchema.refSchema != nil {
		v.validateRecursive(currentSubSchema.refSchema, currentNode, result, context)
		return
	}

	// Check for null value
	if currentNode == nil {

		if currentSubSchema.types.IsTyped() && !currentSubSchema.types.Contains(TYPE_NULL) {
			result.addError(
				context,
				KEY_TYPE,
				currentSubSchema.types.String(),
				currentNode,
				fmt.Sprintf(invalidTypeErrorMessage, currentSubSchema.types.String()),
			)
			return
		}

		currentSubSchema.validateSchema(currentSubSchema, currentNode, result, context)
		v.validateCommon(currentSubSchema, currentNode, result, context)

	} else { // Not a null value

		rValue := reflect.ValueOf(currentNode)
		rKind := rValue.Kind()

		switch rKind {

		// Slice => JSON array

		case reflect.Slice:

			if currentSubSchema.types.IsTyped() && !currentSubSchema.types.Contains(TYPE_ARRAY) {
				result.addError(
					context,
					KEY_TYPE,
					currentSubSchema.types.String(),
					currentNode,
					fmt.Sprintf(invalidTypeErrorMessage, currentSubSchema.types.String()),
				)
				return
			}

			castCurrentNode := currentNode.([]interface{})

			currentSubSchema.validateSchema(currentSubSchema, castCurrentNode, result, context)

			v.validateArray(currentSubSchema, castCurrentNode, result, context)
			v.validateCommon(currentSubSchema, castCurrentNode, result, context)

		// Map => JSON object

		case reflect.Map:
			if currentSubSchema.types.IsTyped() && !currentSubSchema.types.Contains(TYPE_OBJECT) {
				result.addError(
					context,
					KEY_TYPE,
					currentSubSchema.types.String(),
					currentNode,
					fmt.Sprintf(invalidTypeErrorMessage, currentSubSchema.types.String()),
				)
				return
			}

			castCurrentNode, ok := currentNode.(map[string]interface{})
			if !ok {
				castCurrentNode = convertDocumentNode(currentNode).(map[string]interface{})
			}

			currentSubSchema.validateSchema(currentSubSchema, castCurrentNode, result, context)

			v.validateObject(currentSubSchema, castCurrentNode, result, context)
			v.validateCommon(currentSubSchema, castCurrentNode, result, context)

			for _, pSchema := range currentSubSchema.propertiesChildren {
				nextNode, ok := castCurrentNode[pSchema.property]
				if ok {
					subContext := newJsonContext(pSchema.property, context)
					v.validateRecursive(pSchema, nextNode, result, subContext)
				}
			}

		// Simple JSON values : string, number, boolean

		case reflect.Bool:

			if currentSubSchema.types.IsTyped() && !currentSubSchema.types.Contains(TYPE_BOOLEAN) {
				result.addError(
					context,
					KEY_TYPE,
					currentSubSchema.types.String(),
					currentNode,
					fmt.Sprintf(invalidTypeErrorMessage, currentSubSchema.types.String()),
				)
				return
			}

			value := currentNode.(bool)

			currentSubSchema.validateSchema(currentSubSchema, value, result, context)
			v.validateNumber(currentSubSchema, value, result, context)
			v.validateCommon(currentSubSchema, value, result, context)
			v.validateString(currentSubSchema, value, result, context)

		case reflect.String:

			if currentSubSchema.types.IsTyped() && !currentSubSchema.types.Contains(TYPE_STRING) {
				result.addError(
					context,
					KEY_TYPE,
					currentSubSchema.types.String(),
					currentNode,
					fmt.Sprintf(invalidTypeErrorMessage, currentSubSchema.types.String()),
				)
				return
			}

			value := currentNode.(string)

			currentSubSchema.validateSchema(currentSubSchema, value, result, context)
			v.validateNumber(currentSubSchema, value, result, context)
			v.validateCommon(currentSubSchema, value, result, context)
			v.validateString(currentSubSchema, value, result, context)

		case reflect.Float64:

			value := currentNode.(float64)

			// Note: JSON only understand one kind of numeric ( can be float or int )
			// JSON subSchema make a distinction between fload and int
			// An integer can be a number, but a number ( with decimals ) cannot be an integer
			isInteger := isFloat64AnInteger(value)
			validType := currentSubSchema.types.Contains(TYPE_NUMBER) || (isInteger && currentSubSchema.types.Contains(TYPE_INTEGER))

			if currentSubSchema.types.IsTyped() && !validType {
				result.addError(
					context,
					KEY_TYPE,
					currentSubSchema.types.String(),
					currentNode,
					fmt.Sprintf(invalidTypeErrorMessage, currentSubSchema.types.String()),
				)
				return
			}

			currentSubSchema.validateSchema(currentSubSchema, value, result, context)
			v.validateNumber(currentSubSchema, value, result, context)
			v.validateCommon(currentSubSchema, value, result, context)
			v.validateString(currentSubSchema, value, result, context)
		}
	}

	result.incrementScore()
}

// Different kinds of validation there, subSchema / common / array / object / string...
func (v *subSchema) validateSchema(currentSubSchema *subSchema, currentNode interface{}, result *Result, context *jsonContext) {

	internalLog("validateSchema %s", context.String())
	internalLog(" %v", currentNode)

	if len(currentSubSchema.anyOf) > 0 {

		validatedAnyOf := false
		// list of results that the best is later determined from
		var results []*Result

		for _, anyOfSchema := range currentSubSchema.anyOf {
			if !validatedAnyOf {
				validationResult := anyOfSchema.subValidateWithContext(currentNode, context)
				validatedAnyOf = validationResult.Valid()
				results = append(results, validationResult)
			}
		}
		if !validatedAnyOf {
			if bestValidationResult := getBestResult(results); bestValidationResult != nil {
				// add error messages of closest matching subSchema as
				// that's probably the one the user was trying to match
				result.mergeErrors(bestValidationResult)
			} else {
				result.addError(
					context,
					KEY_ANY_OF,
					marshalSubSchemas(currentSubSchema.anyOf),
					currentNode,
					fmt.Sprintf(invalidAnyOfErrorMessage, currentSubSchema.anyOf),
				)
			}
		}
	}

	if len(currentSubSchema.oneOf) > 0 {
		var results []*Result
		var nbValidated int

		for _, oneOfSchema := range currentSubSchema.oneOf {
			validationResult := oneOfSchema.subValidateWithContext(currentNode, context)
			if validationResult.Valid() {
				nbValidated++
			} else {
				results = append(results, validationResult)
			}
		}

		if nbValidated != 1 {
			var bestValidationResult *Result
			if nbValidated == 0 {
				bestValidationResult = getBestResult(results)
			}

			if bestValidationResult != nil {
				// add error messages of closest matching subSchema as
				// that's probably the one the user was trying to match
				result.mergeErrors(bestValidationResult)
			} else {
				result.addError(
					context,
					KEY_ONE_OF,
					marshalSubSchemas(currentSubSchema.oneOf),
					currentNode,
					fmt.Sprintf(invalidOneOfErrorMessage, currentSubSchema.oneOf),
				)
			}
		}

	}

	if len(currentSubSchema.allOf) > 0 {
		var nbValidated int
		for _, allOfSchema := range currentSubSchema.allOf {
			validationResult := allOfSchema.subValidateWithContext(currentNode, context)
			if validationResult.Valid() {
				nbValidated++
			}
			result.mergeErrors(validationResult)
		}

		if nbValidated != len(currentSubSchema.allOf) {
			result.addError(
				context,
				KEY_ALL_OF,
				marshalSubSchemas(currentSubSchema.allOf),
				currentNode,
				fmt.Sprintf(invalidAllOfErrorMessage, currentSubSchema.allOf),
			)
		}
	}

	if currentSubSchema.not != nil {
		validationResult := currentSubSchema.not.subValidateWithContext(currentNode, context)
		if validationResult.Valid() {
			result.addError(
				context,
				KEY_NOT,
				marshalSubSchema(currentSubSchema.not),
				currentNode,
				fmt.Sprintf(invalidNotErrorMessage, currentSubSchema.not),
			)
		}
	}

	if currentSubSchema.dependencies != nil && len(currentSubSchema.dependencies) > 0 {
		if isKind(currentNode, reflect.Map) {
			for elementKey := range currentNode.(map[string]interface{}) {
				if dependency, ok := currentSubSchema.dependencies[elementKey]; ok {
					switch dependency := dependency.(type) {

					//TODO: how does will this cause schema dependencies to render?
					case []string:
						for _, dependOnKey := range dependency {
							if _, dependencyResolved := currentNode.(map[string]interface{})[dependOnKey]; !dependencyResolved {
								result.addError(
									newJsonContext(elementKey, context),
									KEY_DEPENDENCIES,
									dependency,
									currentNode,
									fmt.Sprintf(invalidDependencyErrorMessage, dependOnKey),
								)
							}
						}

					case *subSchema:
						dependency.validateRecursive(dependency, currentNode, result, context)

					}
				}
			}
		}
	}

	result.incrementScore()
}

func (v *subSchema) validateCommon(currentSubSchema *subSchema, value interface{}, result *Result, context *jsonContext) {

	internalLog("validateCommon %s", context.String())
	internalLog(" %v", value)

	// enum:
	if len(currentSubSchema.enum) > 0 {
		has, err := currentSubSchema.ContainsEnum(value)
		if err != nil {
			result.addError(
				context,
				KEY_ENUM,
				currentSubSchema.enum,
				value,
				fmt.Sprintf(ERROR_MESSAGE_INTERNAL, err),
			)
		}
		if !has {
			result.addError(
				context,
				KEY_ENUM,
				currentSubSchema.enum,
				value,
				fmt.Sprintf(invalidEnumErrorMessage, strings.Join(currentSubSchema.enum, ",")),
			)
		}
	}

	result.incrementScore()
}

func (v *subSchema) validateArray(currentSubSchema *subSchema, value []interface{}, result *Result, context *jsonContext) {

	internalLog("validateArray %s", context.String())
	internalLog(" %v", value)

	nbItems := len(value)

	// TODO explain
	if currentSubSchema.itemsChildrenIsSingleSchema {
		for i := range value {
			subContext := newJsonContext(strconv.Itoa(i), context)
			validationResult := currentSubSchema.itemsChildren[0].subValidateWithContext(value[i], subContext)
			result.mergeErrors(validationResult)
		}
	} else {
		if currentSubSchema.itemsChildren != nil && len(currentSubSchema.itemsChildren) > 0 {

			nbItems := len(currentSubSchema.itemsChildren)
			nbValues := len(value)

			if nbItems == nbValues {
				for i := 0; i != nbItems; i++ {
					subContext := newJsonContext(strconv.Itoa(i), context)
					validationResult := currentSubSchema.itemsChildren[i].subValidateWithContext(value[i], subContext)
					result.mergeErrors(validationResult)
				}
			} else if nbItems < nbValues {
				switch currentSubSchema.additionalItems.(type) {
				case bool:
					if !currentSubSchema.additionalItems.(bool) {
						result.addError(
							context,
							KEY_ADDITIONAL_ITEMS,
							currentSubSchema.additionalItems,
							value,
							fmt.Sprintf(invalidAdditionalItemsErrorMessage, false),
						)
					}
				case *subSchema:
					additionalItemSchema := currentSubSchema.additionalItems.(*subSchema)
					for i := nbItems; i != nbValues; i++ {
						subContext := newJsonContext(strconv.Itoa(i), context)
						//TODO: see if this can be used in other rules that require validation and context modification
						validationResult := additionalItemSchema.subValidateWithContext(value[i], subContext)
						result.mergeErrors(validationResult)
					}
				}
			}
		}
	}

	// minItems & maxItems
	if currentSubSchema.minItems != nil {
		if nbItems < *currentSubSchema.minItems {
			result.addError(
				context,
				KEY_MIN_ITEMS,
				currentSubSchema.minItems,
				value,
				fmt.Sprintf(invalidMinItemsErrorMessage, *currentSubSchema.minItems),
			)
		}
	}
	if currentSubSchema.maxItems != nil {
		if nbItems > *currentSubSchema.maxItems {
			result.addError(
				context,
				KEY_MAX_ITEMS,
				currentSubSchema.maxItems,
				value,
				fmt.Sprintf(invalidMaxItemsErrorMessage, *currentSubSchema.maxItems),
			)
		}
	}

	// uniqueItems:
	if currentSubSchema.uniqueItems != nil && *currentSubSchema.uniqueItems {
		var stringifiedItems []string
		for _, v := range value {
			vString, err := marshalToJsonString(v)
			if err != nil {
				//TODO: better handling of errors like this? should this come back as a schema error?
				result.addError(
					context,
					KEY_UNIQUE_ITEMS,
					nil, // since the name is self explanatory and the requirement is subjective
					value,
					fmt.Sprintf(ERROR_MESSAGE_INTERNAL, err),
				)
			}
			if isStringInSlice(stringifiedItems, *vString) {
				result.addError(
					context,
					KEY_UNIQUE_ITEMS,
					nil,
					value,
					invalidUniqueItemsErrorMessage,
				) //TODO: check if needed , TYPE_ARRAY)
			}
			stringifiedItems = append(stringifiedItems, *vString)
		}
	}

	result.incrementScore()
}

func (v *subSchema) validateObject(currentSubSchema *subSchema, value map[string]interface{}, result *Result, context *jsonContext) {

	internalLog("validateObject %s", context.String())
	internalLog(" %v", value)

	// minProperties & maxProperties:
	if currentSubSchema.minProperties != nil {
		if len(value) < *currentSubSchema.minProperties {
			result.addError(
				context,
				KEY_MIN_PROPERTIES,
				currentSubSchema.minProperties,
				value,
				fmt.Sprintf(invalidMinProperties, *currentSubSchema.minProperties),
			)
		}
	}
	if currentSubSchema.maxProperties != nil {
		if len(value) > *currentSubSchema.maxProperties {
			result.addError(
				context,
				KEY_MAX_PROPERTIES,
				currentSubSchema.maxProperties,
				value,
				fmt.Sprintf(invalidMaxProperties, *currentSubSchema.maxProperties),
			)
		}
	}

	// required:
	for _, requiredProperty := range currentSubSchema.required {
		_, ok := value[requiredProperty]
		if ok {
			result.incrementScore()
		} else {
			result.addError(
				newJsonContext(requiredProperty, context),
				KEY_REQUIRED,
				nil, // self explanatory and subjective
				EmptyProperty,
				invalidRequiredErrorMessage,
			)
		}
	}

	// additionalProperty & patternProperty:
	if currentSubSchema.additionalProperties != nil {

		switch currentSubSchema.additionalProperties.(type) {
		case bool:

			if !currentSubSchema.additionalProperties.(bool) {

				for pk := range value {

					found := false
					for _, spValue := range currentSubSchema.propertiesChildren {
						if pk == spValue.property {
							found = true
						}
					}

					pp_has, pp_match := v.validatePatternProperty(currentSubSchema, pk, value[pk], result, context)

					if found {

						if pp_has && !pp_match {
							result.addError(
								newJsonContext(pk, context),
								KEY_ADDITIONAL_PROPERTIES,
								currentSubSchema.patternProperties,
								EmptyProperty,
								fmt.Sprintf(invalidAdditionalPropertyErrorMessage, pk),
							)
						}

					} else {

						if !pp_has || !pp_match {
							result.addError(
								newJsonContext(pk, context),
								KEY_ADDITIONAL_PROPERTIES,
								nil, //TODO: we should show additionalProperties and patternProperties here...
								EmptyProperty,
								fmt.Sprintf(invalidAdditionalPropertyErrorMessage, pk),
							)
						}
					}
				}
			}

		case *subSchema:

			additionalPropertiesSchema := currentSubSchema.additionalProperties.(*subSchema)
			for pk := range value {

				found := false
				for _, spValue := range currentSubSchema.propertiesChildren {
					if pk == spValue.property {
						found = true
					}
				}

				pp_has, pp_match := v.validatePatternProperty(currentSubSchema, pk, value[pk], result, context)

				if found {

					//TODO double check
					if pp_has && !pp_match {
						validationResult := additionalPropertiesSchema.subValidateWithContext(value[pk], context)
						result.mergeErrors(validationResult)
					}

				} else {

					if !pp_has || !pp_match {
						validationResult := additionalPropertiesSchema.subValidateWithContext(value[pk], context)
						result.mergeErrors(validationResult)
					}

				}

			}
		}
	} else {

		for pk := range value {

			pp_has, pp_match := v.validatePatternProperty(currentSubSchema, pk, value[pk], result, context)

			if pp_has && !pp_match {

				result.addError(
					newJsonContext(pk, context),
					KEY_PATTERN_PROPERTIES,
					currentSubSchema.patternProperties,
					value,
					fmt.Sprintf(invalidPatternPropertyErrorMessage, currentSubSchema.PatternPropertiesString()),
				)
			}

		}
	}

	result.incrementScore()
}

func (v *subSchema) validatePatternProperty(currentSubSchema *subSchema, key string, value interface{}, result *Result, context *jsonContext) (has bool, matched bool) {

	internalLog("validatePatternProperty %s", context.String())
	internalLog(" %s %v", key, value)

	has = false

	validatedkey := false

	for pk, pv := range currentSubSchema.patternProperties {
		if matches, _ := regexp.MatchString(pk, key); matches {
			has = true
			subContext := newJsonContext(key, context)
			validationResult := pv.subValidateWithContext(value, subContext)
			result.mergeErrors(validationResult)
			if validationResult.Valid() {
				validatedkey = true
			}
		}
	}

	if !validatedkey {
		return has, false
	}

	result.incrementScore()

	return has, true
}

func (v *subSchema) validateString(currentSubSchema *subSchema, value interface{}, result *Result, context *jsonContext) {

	internalLog("validateString %s", context.String())
	internalLog(" %v", value)

	// Ignore non strings
	if !isKind(value, reflect.String) {
		return
	}

	stringValue := value.(string)

	// minLength & maxLength:
	if currentSubSchema.minLength != nil {
		if utf8.RuneCount([]byte(stringValue)) < *currentSubSchema.minLength {
			result.addError(
				context,
				KEY_MIN_LENGTH,
				currentSubSchema.minLength,
				value,
				fmt.Sprintf(invalidMinLengthErrorMessage, *currentSubSchema.minLength),
			)
		}
	}
	if currentSubSchema.maxLength != nil {
		if utf8.RuneCount([]byte(stringValue)) > *currentSubSchema.maxLength {
			result.addError(
				context,
				KEY_MAX_LENGTH,
				currentSubSchema.maxLength,
				value,
				fmt.Sprintf(invalidMaxLengthErrorMessage, *currentSubSchema.maxLength),
			)
		}
	}

	// pattern:
	if currentSubSchema.pattern != nil {
		if !currentSubSchema.pattern.MatchString(stringValue) {
			result.addError(
				context,
				KEY_PATTERN,
				currentSubSchema.pattern,
				value,
				fmt.Sprintf(invalidPatternErrorMessage, currentSubSchema.pattern),
			)
		}
	}

	result.incrementScore()
}

func (v *subSchema) validateNumber(currentSubSchema *subSchema, value interface{}, result *Result, context *jsonContext) {

	internalLog("validateNumber %s", context.String())
	internalLog(" %v", value)

	// Ignore non numbers
	if !isKind(value, reflect.Float64) {
		return
	}

	float64Value := value.(float64)

	// multipleOf:
	if currentSubSchema.multipleOf != nil {
		if !isFloat64AnInteger(float64Value / *currentSubSchema.multipleOf) {
			result.addError(
				context,
				KEY_MULTIPLE_OF,
				currentSubSchema.multipleOf,
				resultErrorFormatNumber(float64Value),
				fmt.Sprintf(invalidMultipleOfErrorMessage, resultErrorFormatNumber(*currentSubSchema.multipleOf)),
			)
		}
	}

	//maximum & exclusiveMaximum:
	if currentSubSchema.maximum != nil {
		if currentSubSchema.exclusiveMaximum != nil && *currentSubSchema.exclusiveMaximum {
			if float64Value >= *currentSubSchema.maximum {
				result.addError(
					context,
					KEY_EXCLUSIVE_MAXIMUM,
					currentSubSchema.maximum,
					resultErrorFormatNumber(float64Value),
					fmt.Sprintf(invalidExclusiveMaximumErrorMessage, resultErrorFormatNumber(*currentSubSchema.maximum)),
				)
			}
		} else {
			if float64Value > *currentSubSchema.maximum {
				result.addError(
					context,
					KEY_MAXIMUM,
					currentSubSchema.maximum,
					resultErrorFormatNumber(float64Value),
					fmt.Sprintf(invalidMaximumErrorMessage, resultErrorFormatNumber(*currentSubSchema.maximum)),
				)
			}
		}
	}

	//minimum & exclusiveMinimum:
	if currentSubSchema.minimum != nil {
		if currentSubSchema.exclusiveMinimum != nil && *currentSubSchema.exclusiveMinimum {
			if float64Value <= *currentSubSchema.minimum {
				result.addError(
					context,
					KEY_EXCLUSIVE_MINIMUM,
					currentSubSchema.minimum,
					resultErrorFormatNumber(float64Value),
					fmt.Sprintf(invalidExclusiveMinimumErrorMessage,
						resultErrorFormatNumber(*currentSubSchema.minimum)))
			}
		} else {
			if float64Value < *currentSubSchema.minimum {
				result.addError(
					context,
					KEY_MINIMUM,
					currentSubSchema.minimum,
					resultErrorFormatNumber(float64Value),
					fmt.Sprintf(invalidMinimumErrorMessage,
						resultErrorFormatNumber(*currentSubSchema.minimum)),
				)
			}
		}
	}

	result.incrementScore()
}
