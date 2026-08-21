// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package cover

import (
	"fmt"
	"io"
	"maps"
	"math/rand/v2"
	"slices"
	"sync"
	"sync/atomic"
)

// Profile counts the lines the templates of one repository reach when they run.
//
// It is built while the repository is, and counts from then on. Counting is safe for concurrent
// use: a counter is an [sync/atomic.Int64], and the lines a template holds are known before any of
// it runs, so nothing is written to the index once the repository is sealed.
type Profile struct {
	prefix string

	// mu guards the index while templates are instrumented, never while they run.
	mu    sync.Mutex
	files map[string]*fileCounters
}

// fileCounters holds one counter per line of an asset.
type fileCounters struct {
	lengths map[int]int
	counts  map[int]*atomic.Int64
}

// NewProfile builds a profile of the templates of a repository.
//
// prefix is prepended to the path of every asset, so that the profile reads as an import path go
// list resolves, so go tool cover finds the templates.
func NewProfile(prefix string) *Profile {
	return &Profile{
		prefix: prefix,
		files:  make(map[string]*fileCounters),
	}
}

// Flush writes the profile, in the format go test produces and go tool cover reads.
//
// Counters are left as they are, so a caller may write the profile more than once, and go on
// running templates afterwards.
func (p *Profile) Flush(w io.Writer) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, err := fmt.Fprintln(w, "mode: count"); err != nil {
		return err
	}

	for _, assetPath := range slices.Sorted(maps.Keys(p.files)) {
		counters := p.files[assetPath]

		for _, line := range slices.Sorted(maps.Keys(counters.counts)) {
			// a block covers the whole line: the parser gives where a token starts, never how far
			// it reaches, and a block of no width makes go tool cover write broken html
			_, err := fmt.Fprintf(w, "%s:%d.%d,%d.%d %d %d\n",
				p.prefix+assetPath,
				line, 1,
				line, counters.lengths[line]+1,
				1,
				counters.counts[line].Load(),
			)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Lines reports how many lines are counted, and how many of them ran.
func (p *Profile) Lines() (counted, reached int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, counters := range p.files {
		for _, count := range counters.counts {
			counted++
			if count.Load() > 0 {
				reached++
			}
		}
	}

	return counted, reached
}

// register records a line of an asset, so that it is reported whether it runs or not.
func (p *Profile) register(assetPath string, line, length int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	counters, known := p.files[assetPath]
	if !known {
		counters = &fileCounters{lengths: make(map[int]int), counts: make(map[int]*atomic.Int64)}
		p.files[assetPath] = counters
	}

	if _, counted := counters.counts[line]; counted {
		return
	}

	counters.lengths[line] = length
	counters.counts[line] = new(atomic.Int64)
}

// counterFor returns the counter of a line, once the templates are instrumented.
func (p *Profile) counterFor(assetPath string, line int) *atomic.Int64 {
	p.mu.Lock()
	defer p.mu.Unlock()

	counters, known := p.files[assetPath]
	if !known {
		return nil
	}

	return counters.counts[line]
}

// callbackName draws the name of the function a template calls to count a line.
//
// It is drawn at random so that it cannot collide with a function of the map a caller supplies.
// Nothing reads the name once the template holds it.
func callbackName() string {
	const (
		alphabet = "abcdefghijklmnopqrstuvwxyz"
		length   = 12
	)

	name := make([]byte, 0, len("cover")+length)
	name = append(name, "cover"...)

	for range length {
		name = append(name, alphabet[rand.IntN(len(alphabet))]) //nolint:gosec // a name, not a secret
	}

	return string(name)
}
