// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package repo

// repoError is the type of the sentinel errors this package declares.
//
// A string type keeps a sentinel a constant, so nothing may reassign it. It compares by value,
// which is what [errors.Is] needs to find it in a chain of wrapped errors.
type repoError string

// Error implements the error interface.
func (e repoError) Error() string { return string(e) }

// ErrTemplateRepo is matched by every error this package reports.
//
// Errors wrap the cause as well, so a caller may match on a parse error or on an [io/fs] error
// with [errors.Is] and [errors.As] all the same.
const ErrTemplateRepo repoError = "template repository"
