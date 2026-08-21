// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package golang

import (
	"encoding/json"
	"strings"

	"github.com/go-openapi/inflect"
)

// asJSON marshals data to a compact JSON string.
func asJSON(data any) (string, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

// asPrettyJSON marshals data to an indented JSON string.
func asPrettyJSON(data any) (string, error) {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func pluralizeFirstWord(arg string) string {
	sentence := strings.Split(arg, " ")
	if len(sentence) == 1 {
		return inflect.Pluralize(arg)
	}

	return inflect.Pluralize(sentence[0]) + " " + strings.Join(sentence[1:], " ")
}
