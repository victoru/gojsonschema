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
// description      Contains const string and messages.
//
// created          01-01-2015

package gojsonschema

const (
	STRING_NUMBER                     = "number"
	STRING_ARRAY_OF_STRINGS           = "array of strings"
	STRING_ARRAY_OF_SCHEMAS           = "array of schemas"
	STRING_SCHEMA                     = "schema"
	STRING_SCHEMA_OR_ARRAY_OF_STRINGS = "schema or array of strings"
	STRING_PROPERTIES                 = "properties"
	STRING_DEPENDENCY                 = "dependency"
	STRING_PROPERTY                   = "property"

	STRING_CONTEXT_ROOT         = "#"
	STRING_ROOT_SCHEMA_PROPERTY = "#"

	STRING_UNDEFINED = "undefined"

	ERROR_MESSAGE_X_IS_NOT_A_VALID_TYPE                = `%s is not a valid type`
	ERROR_MESSAGE_X_TYPE_IS_DUPLICATED                 = `%s type is duplicated`
	ERROR_MESSAGE_X_MUST_BE_OF_TYPE_Y                  = `%s must be of type %s`
	ERROR_MESSAGE_X_MUST_BE_A_Y                        = `%s must be of a %s`
	ERROR_MESSAGE_X_MUST_BE_AN_Y                       = `%s must be of an %s`
	ERROR_MESSAGE_X_ITEMS_MUST_BE_UNIQUE               = `%s items must be unique`
	ERROR_MESSAGE_X_ITEMS_MUST_BE_TYPE_Y               = `%s items must be %s`
	ERROR_MESSAGE_NEW_SCHEMA_DOCUMENT_INVALID_ARGUMENT = `Invalid argument, must be a JSON string, a JSON reference string or a map[string]interface{}`

	ERROR_MESSAGE_INTERNAL                          = `internal error %s`
	ERROR_MESSAGE_GET_HTTP_BAD_STATUS               = `Could not read schema from HTTP, response status is %d`
	ERROR_MESSAGE_INVALID_REGEX_PATTERN             = `Invalid regex pattern '%s'`
	ERROR_MESSAGE_X_MUST_BE_VALID_REGEX             = `%s must be a valid regex`
	ERROR_MESSAGE_X_MUST_BE_GREATER_OR_TO_0         = `%s must be greater than or equal to 0`
	ERROR_MESSAGE_X_CANNOT_BE_GREATER_THAN_Y        = `%s cannot be greater than %s`
	ERROR_MESSAGE_X_MUST_BE_STRICTLY_GREATER_THAN_0 = `%s must be strictly greater than 0`
	ERROR_MESSAGE_X_CANNOT_BE_USED_WITHOUT_Y        = `%s cannot be used without %s`
	ERROR_MESSAGE_REFERENCE_X_MUST_BE_CANONICAL     = `Reference %s must be canonical`
)
