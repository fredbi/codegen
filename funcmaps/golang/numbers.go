// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package golang

import (
	"cmp"
	"math"
	"math/big"
	"strconv"
)

// isInteger takes any type that may have an integer representation (including []byte, string, etc.) and
// returns true if it is in fact an integer.
//
// Pointers are resolved and nil is not considered an integer value.
func isInteger(arg any) bool {
	in, ok := derefArg(arg)
	if !ok {
		return false
	}

	switch val := in.(type) {
	case int8, int16, int32, int, int64, uint8, uint16, uint32, uint, uint64, big.Int:
		return true
	case float64:
		return math.Round(val) == val
	case float32:
		return math.Round(float64(val)) == float64(val)
	case string:
		_, err := strconv.ParseInt(val, 10, 64)
		return err == nil
	case []byte:
		_, err := strconv.ParseInt(string(val), 10, 64)
		return err == nil
	case big.Rat:
		return val.IsInt()
	case big.Float:
		return val.IsInt()
	default:
		return false
	}
}

// gt0 is like the builtin "gt 0".
//
// Its arg is de-dereferenced if this is a pointer.
//
// It may return true only for numerical values.
func gt0(arg any) bool {
	in, ok := derefArg(arg)
	if !ok {
		return false
	}

	switch val := in.(type) {
	case int8:
		return cmp.Compare(val, 0) > 0
	case int16:
		return cmp.Compare(val, 0) > 0
	case int32:
		return cmp.Compare(val, 0) > 0
	case int:
		return cmp.Compare(val, 0) > 0
	case int64:
		return cmp.Compare(val, 0) > 0
	case uint8:
		return cmp.Compare(val, 0) > 0
	case uint16:
		return cmp.Compare(val, 0) > 0
	case uint32:
		return cmp.Compare(val, 0) > 0
	case uint:
		return cmp.Compare(val, 0) > 0
	case uint64:
		return cmp.Compare(val, 0) > 0
	case float64:
		return cmp.Compare(val, 0) > 0
	case float32:
		return cmp.Compare(val, 0) > 0
	case big.Rat:
		return val.IsInt()
	case big.Float:
		return val.IsInt()
	default:
		return false
	}
}
