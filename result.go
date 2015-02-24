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
// description      Result and ResultError implementations.
//
// created          01-01-2015

package gojsonschema

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type ResultError struct {
	Context *JSONContext // Tree like notation of the part that failed the validation. ex (root).a.b ...
	Value   interface{}  // Value given by the JSON file that is the source of the error

	Reason      string      //JSON schema keyword responsible for this error
	Requirement interface{} // the schema attribute's requirement that caused this error
}

func (v ResultError) String() string {
	var l []string
	l = append(l, fmt.Sprintf("%s", v.Reason))
	if v.Requirement != nil {
		l = append(l, fmt.Sprintf("%s", v.Requirement))
	}

	return fmt.Sprintf("%s: %s", v.Context.String(), strings.Join(l, ","))
}

// sort by score descending
type resultsByScore []*Result

func (r resultsByScore) Len() int           { return len(r) }
func (r resultsByScore) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }
func (r resultsByScore) Less(i, j int) bool { return r[i].score > r[j].score }

// returns the best result based on the highest non repeating score.
func getBestResult(results resultsByScore) *Result {
	if len(results) > 1 {
		sort.Sort(results)
		if results[0].score != results[1].score {
			return results[0]
		}
	}
	return nil
}

type Result struct {
	errors []ResultError
	// Scores how well the validation matched. Useful in generating
	// better error messages for anyOf and oneOf.
	score int
}

func (v *Result) Valid() bool {
	return len(v.errors) == 0
}

func (v *Result) Errors() ResultErrors {
	return v.errors
}

// AddError adds a context JSON schema error to Result using the failing schema
// attribute as the reason
func (v *Result) AddError(
	context *JSONContext,
	reason string,
	requirement interface{},
	value interface{},
) {
	rerr := ResultError{
		Context:     context,
		Reason:      reason,
		Requirement: requirement,
		Value:       value,
	}
	v.errors = append(v.errors, rerr)
	v.score -= 2 // results in a net -1 when added to the +1 we get at the end of the validation function
}

// Used to copy errors from a sub-schema to the main one
func (v *Result) mergeErrors(otherResult *Result) {
	v.errors = append(v.errors, otherResult.Errors()...)
	v.score += otherResult.score
}

func (v *Result) incrementScore() {
	v.score++
}

// ResultErrors is a collection of JSON schema errors
type ResultErrors []ResultError

func (rerrs ResultErrors) Error() string {
	return fmt.Sprintf("%d fields with validation error(s)", len(rerrs))
}

// Map parses ResultErrors into a map object for simpler error parsing/handling
func (rerrs ResultErrors) Map() map[string][]interface{} {
	var jmap = make(map[string][]interface{})
	for _, rerr := range rerrs {
		var errStack = []interface{}{rerr.Reason}
		if rerr.Requirement != nil {
			errStack = append(errStack, rerr.Requirement)
		}

		jmap[rerr.Context.String()] = append(jmap[rerr.Context.String()], errStack)
	}
	return jmap
}

func (rerrs ResultErrors) MarshalJSON() ([]byte, error) {
	return json.Marshal(rerrs.Map())
}

// ResultErrorsMarshalerFunc is the function used when json.Marshal is called
// on ResultErrors. It's been set as package variable to allow importing packages
// to alter the default behavior when marshaling ResultErrors.
var ResultErrorsMarshalerFunc = func(rerrs ResultErrors) ([]byte, error) {
	return json.Marshal(rerrs.Map())
}
