// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package golang

import "fmt"

func dict(values ...any) (map[string]any, error) {
	const pair = 2

	if len(values)%pair != 0 {
		return nil, fmt.Errorf("expected even number of arguments, got %d", len(values))
	}

	dict := make(map[string]any, len(values)/pair)
	for i := 0; i < len(values)-1; i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, fmt.Errorf("expected string key, got %+v", values[i])
		}
		dict[key] = values[i+1] // bounds checked by the modulo guard above
	}

	return dict, nil
}
