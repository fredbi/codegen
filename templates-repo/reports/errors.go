// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package reports

// reportError is the type of the sentinel errors this package declares.
//
// A string type keeps a sentinel a constant, so nothing may reassign it. It compares by value,
// which is what [errors.Is] needs to find it in a chain of wrapped errors.
type reportError string

// Error implements the error interface.
func (e reportError) Error() string { return string(e) }

// ErrReport is matched by every error this package reports.
//
// Errors wrap the cause as well, so a caller may match on a template parse error with
// [errors.Is] and [errors.As] all the same.
const ErrReport reportError = "templates report"
