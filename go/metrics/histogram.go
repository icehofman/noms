// Copyright 2017 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package metrics

import (
	"fmt"
	"strings"
	"time"

	"github.com/attic-labs/noms/go/d"
	humanize "github.com/dustin/go-humanize"
)

// Histogram is a shameless and low-rent knock of the chromium project's
// histogram:
//   https://chromium.googlesource.com/chromium/src/base/+/master/metrics/histogram.h
//
// It logically stores a running histogram of uint64 values and shares some
// important features of its inspiration:
//   * It acccepts a correctness deficit in return for not needing to lock.
//     IOW, concurrent calls to Sample may clobber each other.
//   * It trades compactness and ease of arithmatic across histograms for
//     precision. Samples lose precision up to the range of the values which
//     are stored in a bucket
//
// Only implemented: Log2-based histogram
type Histogram struct {
	buckets  [bucketCount]uint64
	ToString ToStringFunc
}

type ToStringFunc func(v uint64) string

func identToString(v uint64) string {
	return fmt.Sprintf("%d", v)
}

const bucketCount = 63

// Sample adds a uint64 data point to the histogram
func (h *Histogram) Sample(v uint64) {
	d.PanicIfTrue(v == 0)
	pot := 0

	for v > 0 {
		v = v >> 1
		pot++
	}

	h.buckets[pot-1]++
}

// SampleTime is a convenience wrapper around Sample which internally type
// asserts the time.Duration to a uint64
func (h *Histogram) SampleTime(d time.Duration) {
	h.Sample(uint64(d))
}

// SampleTime is a convenience wrapper around Sample which internally type
// asserts the int to a uint64
func (h *Histogram) SampleLen(l int) {
	h.Sample(uint64(l))
}

func (h Histogram) bucketVal(bucket int) uint64 {
	return 1 << (uint64(bucket))
}

// The bucket sum is reported as the mid-point value of a bucket multiplied by
// the number of samples in the bucket
func (h Histogram) bucketSum(bucket int) uint64 {
	return h.buckets[bucket] * (h.bucketVal(bucket) + h.bucketVal(bucket+1)) / 2
}

// Sum return the sum of sampled values, given that each sample is clamped to
// the mid-point value of the bucket in which it is recorded.
func (h Histogram) Sum() uint64 {
	sum := uint64(0)
	for i := 0; i < bucketCount; i++ {
		sum += h.bucketSum(i)
	}
	return sum
}

// Add returns a new Histogram which is the result of adding this and other
// bucket-wise.
func (h *Histogram) Add(other Histogram) {
	for i := 0; i < bucketCount; i++ {
		h.buckets[i] += other.buckets[i]
	}
}

// Delta returns a new Histogram whcih is the result of subtracting other from
// this bucket-wise. The intent is to capture changes in the state of histogram
// which is collecting samples over some time period. It will panic if any
// bucket from other is larger than the corresponding bucket in this.
func (h Histogram) Delta(other Histogram) Histogram {
	nh := Histogram{}
	for i := 0; i < bucketCount; i++ {
		c := h.buckets[i]
		l := other.buckets[i]
		d.PanicIfTrue(l > c)
		nh.buckets[i] = c - l
	}
	return nh
}

// Mean returns 0 if there are no samples, and h.Sum()/h.Samples otherwise.
func (h Histogram) Mean() uint64 {
	samples := h.Samples()
	if samples == 0 {
		return 0
	}

	return h.Sum() / samples
}

// Samples returns the number of samples contained in the histogram
func (h Histogram) Samples() uint64 {
	s := uint64(0)
	for i := 0; i < bucketCount; i++ {
		s += h.buckets[i]
	}
	return s
}

func (h Histogram) String() string {
	f := h.ToString
	if f == nil {
		f = identToString
	}
	return fmt.Sprintf("Mean: %s, Sum: %s, Samples: %d", f(h.Mean()), f(h.Sum()), h.Samples())
}

func NewTimeHistogram() Histogram {
	return Histogram{ToString: timeToString}
}

func timeToString(v uint64) string {
	return time.Duration(v).String()
}

// ByteHistogram stringifies values using humanize over byte values
func NewByteHistogram() Histogram {
	return Histogram{ToString: humanize.Bytes}
}

const colWidth = 100

// Report returns an ASCII graph of the non-zero range of normalized buckets.
// IOW, it returns a basic graph of the histogram
func (h Histogram) Report() string {
	ts := h.ToString
	if ts == nil {
		ts = identToString
	}

	maxSamples := uint64(0)
	foundFirstNonEmpty := false
	firstNonEmpty := 0
	lastNonEmpty := 0
	for i := 0; i < bucketCount; i++ {
		samples := h.buckets[i]

		if samples > 0 {
			lastNonEmpty = i
			if !foundFirstNonEmpty {
				foundFirstNonEmpty = true
				firstNonEmpty = i
			}
		}

		if samples > maxSamples {
			maxSamples = samples
		}
	}

	if maxSamples == 0 {
		return ""
	}

	val := uint64(1)

	p := func(bucket int) string {
		samples := h.buckets[bucket]
		val := h.bucketVal(bucket)
		adj := samples * colWidth / maxSamples
		return fmt.Sprintf("%s> %s: (%d)", strings.Repeat("-", int(adj)), ts(val), samples)
	}

	lines := make([]string, 0)
	for i := 0; i < bucketCount; i++ {
		if i >= firstNonEmpty && i <= lastNonEmpty {
			lines = append(lines, p(i))
		}

		val = val << 1
	}

	return strings.Join(lines, "\n")
}
