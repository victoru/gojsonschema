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
	Context     *jsonContext // Tree like notation of the part that failed the validation. ex (root).a.b ...
	Description string       // A human readable error message
	Value       interface{}  // Value given by the JSON file that is the source of the error

	Attribute   string      //schema keyword responsible for this error
	Requirement interface{} // the schema attribute's requirement that caused this error
}

func (v ResultError) String() string {
	var t = []interface{}{v.Context.String()}
	var s string = "%s: %s"
	for _, ee := range strings.SplitN(v.Description, ".", 2) {
		t = append(t, ee)
	}
	if len(t[2:]) != 0 {
		s += "[%v]"
	}
	return fmt.Sprintf(s, t...)
}

// sort by score descending
type ResultsByScore []*Result

func (r ResultsByScore) Len() int           { return len(r) }
func (r ResultsByScore) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }
func (r ResultsByScore) Less(i, j int) bool { return r[i].score > r[j].score }

// returns the best result based on the highest non repeating score.
func getBestResult(results ResultsByScore) *Result {
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

func (v *Result) Errors() []ResultError {
	return v.errors
}

//TODO: remove need for description. apps should handle their own descriptions
func (v *Result) addError(
	context *jsonContext,
	attr string,
	requirement interface{},
	value interface{},
	description string,
) {
	rerr := ResultError{
		Context:     context,
		Attribute:   attr,
		Requirement: requirement,
		Value:       value,
		Description: description,
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

func (v *Result) MarshalJSON() ([]byte, error) {
	return ResultMarshalerFunc(v)
}

// ResultMarshalerFunc is the function used when json.Marshal is called on *Result.
// Set as package variable to allow importing packages to override *Result's
// default marshaling
var ResultMarshalerFunc = func(res *Result) ([]byte, error) {
	var jmap = make(map[string][]interface{})
	for _, rerr := range res.Errors() {
		var errStack []interface{}
		errStack = append(errStack, rerr.Attribute)
		if rerr.Requirement != nil {
			errStack = append(errStack, rerr.Requirement)
		}

		jmap[rerr.Context.String()] = append(jmap[rerr.Context.String()], errStack)
	}

	return json.Marshal(jmap)
}
