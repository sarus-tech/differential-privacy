//
// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package dpagg

import (
	"math"
	"reflect"
	"testing"

	"github.com/google/differential-privacy/go/noise"
	"github.com/google/differential-privacy/go/rand"
	"github.com/google/go-cmp/cmp"
)

func TestNewBoundedMeanFloat64(t *testing.T) {
	for _, tc := range []struct {
		desc string
		opt  *BoundedMeanFloat64Options
		want *BoundedMeanFloat64
	}{
		{"MaxPartitionsContributed is not set",
			&BoundedMeanFloat64Options{
				Epsilon:                      ln3,
				Delta:                        tenten,
				Lower:                        -1,
				Upper:                        5,
				Noise:                        noNoise{},
				MaxContributionsPerPartition: 2,
			},
			&BoundedMeanFloat64{
				lower:    -1,
				upper:    5,
				state:    defaultState,
				midPoint: 2,
				Count: Count{
					epsilon:         ln3 * 0.5,
					delta:           tenten * 0.5,
					l0Sensitivity:   1,
					lInfSensitivity: 2,
					Noise:           noNoise{},
					noiseKind:       noise.Unrecognised,
					count:           0,
					state:           defaultState,
				},
				NormalizedSum: BoundedSumFloat64{
					epsilon:         ln3 * 0.5,
					delta:           tenten * 0.5,
					l0Sensitivity:   1,
					lInfSensitivity: 6,
					lower:           -3,
					upper:           3,
					Noise:           noNoise{},
					noiseKind:       noise.Unrecognised,
					sum:             0,
					state:           defaultState,
				},
			}},
		{"Noise is not set",
			&BoundedMeanFloat64Options{
				Epsilon:                      ln3,
				Delta:                        0,
				Lower:                        -1,
				Upper:                        5,
				MaxContributionsPerPartition: 2,
				MaxPartitionsContributed:     1,
			},
			&BoundedMeanFloat64{
				lower:    -1,
				upper:    5,
				state:    defaultState,
				midPoint: 2,
				Count: Count{
					epsilon:         ln3 * 0.5,
					delta:           0,
					l0Sensitivity:   1,
					lInfSensitivity: 2,
					noiseKind:       noise.LaplaceNoise,
					Noise:           noise.Laplace(),
					count:           0,
					state:           defaultState,
				},
				NormalizedSum: BoundedSumFloat64{
					epsilon:         ln3 * 0.5,
					delta:           0,
					l0Sensitivity:   1,
					lInfSensitivity: 6,
					lower:           -3,
					upper:           3,
					noiseKind:       noise.LaplaceNoise,
					Noise:           noise.Laplace(),
					sum:             0,
					state:           defaultState,
				},
			}},
	} {
		bm, err := NewBoundedMeanFloat64(tc.opt)
		if err != nil {
			t.Fatalf("Couldn't initialize bm: %v", err)
		}
		if !reflect.DeepEqual(bm, tc.want) {
			t.Errorf("NewBoundedMeanFloat64: when %s got %+v, want %+v", tc.desc, bm, tc.want)
		}
	}
}

func TestBMNoInputFloat64(t *testing.T) {
	bmf := getNoiselessBMF(t)
	got, err := bmf.Result()
	if err != nil {
		t.Fatalf("Couldn't compute dp result: %v", err)
	}
	// count = 0 => returns midPoint = 2
	want := 2.0
	if !ApproxEqual(got, want) {
		t.Errorf("BoundedMean: when there is no input data got=%f, want=%f", got, want)
	}
}

func TestBMAddFloat64(t *testing.T) {
	bmf := getNoiselessBMF(t)
	bmf.Add(1.5)
	bmf.Add(2.5)
	bmf.Add(3.5)
	bmf.Add(4.5)
	got, err := bmf.Result()
	if err != nil {
		t.Fatalf("Couldn't compute dp result: %v", err)
	}
	want := 3.0
	if !ApproxEqual(got, want) {
		t.Errorf("Add: when dataset with elements inside boundaries got %f, want %f", got, want)
	}
}

func TestBMAddFloat64IgnoresNaN(t *testing.T) {
	bmf := getNoiselessBMF(t)
	bmf.Add(1)
	bmf.Add(math.NaN())
	bmf.Add(3)
	got, err := bmf.Result()
	if err != nil {
		t.Fatalf("Couldn't compute dp result: %v", err)
	}
	want := 2.0
	if !ApproxEqual(got, want) {
		t.Errorf("Add: when dataset contains NaN got %f, want %f", got, want)
	}
}

func TestBMReturnsEntryIfSingleEntryIsAddedFloat64(t *testing.T) {
	bmf := getNoiselessBMF(t)
	// lower = -1, upper = 5
	// normalized sum inside BM has bounds: lower = -3, upper = 3
	// midPoint = 2

	bmf.Add(1.2345)
	got, err := bmf.Result()
	if err != nil {
		t.Fatalf("Couldn't compute dp result: %v", err)
	}
	want := 1.2345 // single entry
	if !ApproxEqual(got, want) {
		t.Errorf("BoundedMean: when dataset contains single entry got %f, want %f", got, want)
	}
}

func TestBMClampFloat64(t *testing.T) {
	bmf := getNoiselessBMF(t)
	// lower = -1, upper = 5
	// normalized sum inside BM has bounds: lower = -3, upper = 3
	// midPoint = 2

	bmf.Add(3.5)  // clamp(3.5) - midPoint = 3.5 - 2 = 1.5 -> to normalizedSum
	bmf.Add(8.3)  // clamp(8.3) - midPoint = 5 - 2 = 3 -> to normalizedSum
	bmf.Add(-7.5) // clamp(-7.5) - midPoint = -1 - 2 = -3 -> to normalizedSum
	got, err := bmf.Result()
	if err != nil {
		t.Fatalf("Couldn't compute dp result: %v", err)
	}
	want := 2.5
	if !ApproxEqual(got, want) { // clamp (normalizedSum/count + mid point) = 1.5 / 3 + 2 = 2.5
		t.Errorf("Add: when dataset with elements outside boundaries got %f, want %f", got, want)
	}
}

func TestBoundedMeanFloat64ResultSetsStateCorrectly(t *testing.T) {
	bm := getNoiselessBMF(t)
	_, err := bm.Result()
	if err != nil {
		t.Fatalf("Couldn't compute dp result: %v", err)
	}

	if bm.state != resultReturned {
		t.Errorf("BoundedMeanFloat64 should have its state set to ResultReturned, got %v, want ResultReturned", bm.state)
	}
}

func TestBMNoiseIsCorrectlyCalledFloat64(t *testing.T) {
	bmf := getMockBMF(t)
	bmf.Add(1)
	bmf.Add(2)
	got, _ := bmf.Result() // will fail if parameters are wrong
	want := 5.0
	// clamp((noised normalizedSum / noised count) + mid point) = clamp((-1 + 100) / (2 + 10) + 2) = 5
	if !ApproxEqual(got, want) {
		t.Errorf("Add: when dataset = {1, 2} got %f, want %f", got, want)
	}
}

func TestBMReturnsResultInsideProvidedBoundariesFloat64(t *testing.T) {
	lower := rand.Uniform() * 100
	upper := lower + rand.Uniform()*100

	bmf, err := NewBoundedMeanFloat64(&BoundedMeanFloat64Options{
		Epsilon:                      ln3,
		MaxPartitionsContributed:     1,
		MaxContributionsPerPartition: 1,
		Lower:                        lower,
		Upper:                        upper,
		Noise:                        noise.Laplace(),
	})
	if err != nil {
		t.Fatalf("Couldn't initialize mean: %v", err)
	}

	for i := 0; i <= 1000; i++ {
		bmf.Add(rand.Uniform() * 300 * rand.Sign())
	}

	res, err := bmf.Result()
	if err != nil {
		t.Fatalf("Couldn't compute dp result: %v", err)
	}
	if res < lower {
		t.Errorf("BoundedMean: result is outside of boundaries, got %f, want to be >= %f", res, lower)
	}

	if res > upper {
		t.Errorf("BoundedMean: result is outside of boundaries, got %f, want to be <= %f", res, upper)
	}
}

type mockBMNoise struct {
	t *testing.T
	noise.Noise
}

// AddNoiseInt64 checks that the parameters passed are the ones we expect.
func (mn mockBMNoise) AddNoiseInt64(x, l0, lInf int64, eps, del float64) (int64, error) {
	if x != 2 && x != 0 {
		// AddNoiseInt64 is initially called with a placeholder value of 0, so we don't want to fail when that happens
		mn.t.Errorf("AddNoiseInt64: for parameter x got %d, want %d", x, 2)
	}
	if l0 != 1 {
		mn.t.Errorf("AddNoiseInt64: for parameter l0Sensitivity got %d, want %d", l0, 1)
	}
	if lInf != 1 {
		mn.t.Errorf("AddNoiseInt64: for parameter lInfSensitivity got %d, want %d", lInf, 5)
	}
	if !ApproxEqual(eps, ln3*0.5) {
		mn.t.Errorf("AddNoiseInt64: for parameter epsilon got %f, want %f", eps, ln3*0.5)
	}
	if !ApproxEqual(del, tenten*0.5) {
		mn.t.Errorf("AddNoiseInt64: for parameter delta got %f, want %f", del, tenten*0.5)
	}
	return x + 10, nil
}

// AddNoiseFloat64 checks that the parameters passed are the ones we expect.
func (mn mockBMNoise) AddNoiseFloat64(x float64, l0 int64, lInf, eps, del float64) (float64, error) {
	if !ApproxEqual(x, -1.0) && !ApproxEqual(x, 0.0) {
		// AddNoiseFloat64 is initially called with a placeholder value of 0, so we don't want to fail when that happens
		mn.t.Errorf("AddNoiseFloat64: for parameter x got %f, want %f", x, 0.0)
	}
	if l0 != 1 {
		mn.t.Errorf("AddNoiseFloat64: for parameter l0Sensitivity got %d, want %d", l0, 1)
	}
	if !ApproxEqual(lInf, 3.0) && !ApproxEqual(lInf, 1.0) {
		// AddNoiseFloat64 is initially called with an lInf of 1, so we don't want to fail when that happens
		mn.t.Errorf("AddNoiseFloat64: for parameter lInfSensitivity got %f, want %f", lInf, 3.0)
	}
	if !ApproxEqual(eps, ln3*0.5) {
		mn.t.Errorf("AddNoiseFloat64: for parameter epsilon got %f, want %f", eps, ln3*0.5)
	}
	if !ApproxEqual(del, tenten*0.5) {
		mn.t.Errorf("AddNoiseFloat64: for parameter delta got %f, want %f", del, tenten*0.5)
	}
	return x + 100, nil
}

func getNoiselessBMF(t *testing.T) *BoundedMeanFloat64 {
	t.Helper()
	bm, err := NewBoundedMeanFloat64(&BoundedMeanFloat64Options{
		Epsilon:                      ln3,
		Delta:                        tenten,
		MaxPartitionsContributed:     1,
		MaxContributionsPerPartition: 1,
		Lower:                        -1,
		Upper:                        5,
		Noise:                        noNoise{},
	})
	if err != nil {
		t.Fatalf("Couldn't get noiseless BMF")
	}
	return bm
}

func getMockBMF(t *testing.T) *BoundedMeanFloat64 {
	t.Helper()
	bm, err := NewBoundedMeanFloat64(&BoundedMeanFloat64Options{
		Epsilon:                      ln3,
		Delta:                        tenten,
		MaxPartitionsContributed:     1,
		MaxContributionsPerPartition: 1,
		Lower:                        -1,
		Upper:                        5,
		Noise:                        mockBMNoise{t: t},
	})
	if err != nil {
		t.Fatalf("Couldn't get mock BMF")
	}
	return bm
}

func TestMergeBoundedMeanFloat64(t *testing.T) {
	bm1 := getNoiselessBMF(t)
	bm2 := getNoiselessBMF(t)
	bm1.Add(1)
	bm1.Add(2)
	bm1.Add(3.5)
	bm1.Add(4)
	bm2.Add(4.5)
	err := bm1.Merge(bm2)
	if err != nil {
		t.Fatalf("Couldn't merge bm1 and bm2: %v", err)
	}
	got, err := bm1.Result()
	if err != nil {
		t.Fatalf("Couldn't compute dp result: %v", err)
	}
	want := 3.0
	if !ApproxEqual(got, want) {
		t.Errorf("Merge: when merging 2 instances of BoundedMean got %f, want %f", got, want)
	}
	if bm2.state != merged {
		t.Errorf("Merge: when merging 2 instances of BoundedMean for bm2.state got %v, want Merged", bm2.state)
	}
}

func TestCheckMergeBoundedMeanFloat64Compatibility(t *testing.T) {
	for _, tc := range []struct {
		desc    string
		opt1    *BoundedMeanFloat64Options
		opt2    *BoundedMeanFloat64Options
		wantErr bool
	}{
		{"same options",
			&BoundedMeanFloat64Options{
				Epsilon:                      ln3,
				Delta:                        tenten,
				MaxPartitionsContributed:     1,
				Lower:                        -1,
				Upper:                        5,
				Noise:                        noise.Gaussian(),
				MaxContributionsPerPartition: 2,
			},
			&BoundedMeanFloat64Options{
				Epsilon:                      ln3,
				Delta:                        tenten,
				MaxPartitionsContributed:     1,
				Lower:                        -1,
				Upper:                        5,
				Noise:                        noise.Gaussian(),
				MaxContributionsPerPartition: 2,
			},
			false},
		{"different epsilon",
			&BoundedMeanFloat64Options{
				Epsilon:                      ln3,
				Delta:                        tenten,
				MaxPartitionsContributed:     1,
				Lower:                        -1,
				Upper:                        5,
				Noise:                        noise.Gaussian(),
				MaxContributionsPerPartition: 2,
			},
			&BoundedMeanFloat64Options{
				Epsilon:                      2,
				Delta:                        tenten,
				MaxPartitionsContributed:     1,
				Lower:                        -1,
				Upper:                        5,
				Noise:                        noise.Gaussian(),
				MaxContributionsPerPartition: 2,
			},
			true},
		{"different delta",
			&BoundedMeanFloat64Options{
				Epsilon:                      ln3,
				Delta:                        tenten,
				MaxPartitionsContributed:     1,
				Lower:                        -1,
				Upper:                        5,
				Noise:                        noise.Gaussian(),
				MaxContributionsPerPartition: 2,
			},
			&BoundedMeanFloat64Options{
				Epsilon:                      ln3,
				Delta:                        tenfive,
				MaxPartitionsContributed:     1,
				Lower:                        -1,
				Upper:                        5,
				Noise:                        noise.Gaussian(),
				MaxContributionsPerPartition: 2,
			},
			true},
		{"different MaxPartitionsContributed",
			&BoundedMeanFloat64Options{
				Epsilon:                      ln3,
				Delta:                        tenten,
				MaxPartitionsContributed:     1,
				Lower:                        -1,
				Upper:                        5,
				Noise:                        noise.Gaussian(),
				MaxContributionsPerPartition: 2,
			},
			&BoundedMeanFloat64Options{
				Epsilon:                      ln3,
				Delta:                        tenten,
				MaxPartitionsContributed:     2,
				Lower:                        -1,
				Upper:                        5,
				Noise:                        noise.Gaussian(),
				MaxContributionsPerPartition: 2,
			},
			true},
		{"different lower bound",
			&BoundedMeanFloat64Options{
				Epsilon:                      ln3,
				Delta:                        tenten,
				MaxPartitionsContributed:     1,
				Lower:                        -1,
				Upper:                        5,
				Noise:                        noise.Gaussian(),
				MaxContributionsPerPartition: 2,
			},
			&BoundedMeanFloat64Options{
				Epsilon:                      ln3,
				Delta:                        tenten,
				MaxPartitionsContributed:     1,
				Lower:                        0,
				Upper:                        5,
				Noise:                        noise.Gaussian(),
				MaxContributionsPerPartition: 2,
			},
			true},
		{"different upper bound",
			&BoundedMeanFloat64Options{
				Epsilon:                      ln3,
				Delta:                        tenten,
				MaxPartitionsContributed:     1,
				Lower:                        -1,
				Upper:                        5,
				Noise:                        noise.Gaussian(),
				MaxContributionsPerPartition: 2,
			},
			&BoundedMeanFloat64Options{
				Epsilon:                      ln3,
				Delta:                        tenten,
				MaxPartitionsContributed:     1,
				Lower:                        -1,
				Upper:                        6,
				Noise:                        noise.Gaussian(),
				MaxContributionsPerPartition: 2,
			},
			true},
		{"different noise",
			&BoundedMeanFloat64Options{
				Epsilon:                      ln3,
				Delta:                        tenten,
				MaxPartitionsContributed:     1,
				Lower:                        -1,
				Upper:                        5,
				Noise:                        noise.Gaussian(),
				MaxContributionsPerPartition: 2,
			},
			&BoundedMeanFloat64Options{
				Epsilon:                      ln3,
				Delta:                        0,
				MaxPartitionsContributed:     1,
				Lower:                        -1,
				Upper:                        5,
				Noise:                        noise.Laplace(),
				MaxContributionsPerPartition: 2,
			},
			true},
		{"different maxContributionsPerPartition",
			&BoundedMeanFloat64Options{
				Epsilon:                      ln3,
				Delta:                        tenten,
				MaxPartitionsContributed:     1,
				Lower:                        -1,
				Upper:                        5,
				Noise:                        noise.Gaussian(),
				MaxContributionsPerPartition: 2,
			},
			&BoundedMeanFloat64Options{
				Epsilon:                      ln3,
				Delta:                        tenten,
				MaxPartitionsContributed:     1,
				Lower:                        -1,
				Upper:                        5,
				Noise:                        noise.Gaussian(),
				MaxContributionsPerPartition: 5,
			},
			true},
	} {
		bm1, err := NewBoundedMeanFloat64(tc.opt1)
		if err != nil {
			t.Fatalf("Couldn't initialize bm1: %v", err)
		}
		bm2, err := NewBoundedMeanFloat64(tc.opt2)
		if err != nil {
			t.Fatalf("Couldn't initialize bm2: %v", err)
		}

		if err := checkMergeBoundedMeanFloat64(bm1, bm2); (err != nil) != tc.wantErr {
			t.Errorf("CheckMerge: when %s for err got %v, wantErr %t", tc.desc, err, tc.wantErr)
		}
	}
}

// Tests that checkMergeBoundedMeanFloat64() returns errors correctly with different BoundedMeanFloat64 aggregation states.
func TestCheckMergeBoundedMeanFloat64StateChecks(t *testing.T) {
	for _, tc := range []struct {
		state1  aggregationState
		state2  aggregationState
		wantErr bool
	}{
		{defaultState, defaultState, false},
		{resultReturned, defaultState, true},
		{defaultState, resultReturned, true},
		{serialized, defaultState, true},
		{defaultState, serialized, true},
		{defaultState, merged, true},
		{merged, defaultState, true},
	} {
		bm1 := getNoiselessBMF(t)
		bm2 := getNoiselessBMF(t)

		bm1.state = tc.state1
		bm2.state = tc.state2

		if err := checkMergeBoundedMeanFloat64(bm1, bm2); (err != nil) != tc.wantErr {
			t.Errorf("CheckMerge: when states [%v, %v] for err got %v, wantErr %t", tc.state1, tc.state2, err, tc.wantErr)
		}
	}
}

func TestBMEquallyInitializedFloat64(t *testing.T) {
	for _, tc := range []struct {
		desc  string
		bm1   *BoundedMeanFloat64
		bm2   *BoundedMeanFloat64
		equal bool
	}{
		{
			"equal parameters",
			&BoundedMeanFloat64{
				lower:         0,
				upper:         2,
				midPoint:      1,
				NormalizedSum: BoundedSumFloat64{},
				Count:         Count{},
				state:         defaultState},
			&BoundedMeanFloat64{
				lower:         0,
				upper:         2,
				midPoint:      1,
				NormalizedSum: BoundedSumFloat64{},
				Count:         Count{},
				state:         defaultState},
			true,
		},
		{
			"different lower",
			&BoundedMeanFloat64{
				lower:         -1,
				upper:         2,
				midPoint:      1,
				NormalizedSum: BoundedSumFloat64{},
				Count:         Count{},
				state:         defaultState},
			&BoundedMeanFloat64{
				lower:         0,
				upper:         2,
				midPoint:      1,
				NormalizedSum: BoundedSumFloat64{},
				Count:         Count{},
				state:         defaultState},
			false,
		},
		{
			"different upper",
			&BoundedMeanFloat64{
				lower:         0,
				upper:         2,
				midPoint:      1,
				NormalizedSum: BoundedSumFloat64{},
				Count:         Count{},
				state:         defaultState},
			&BoundedMeanFloat64{
				lower:         0,
				upper:         3,
				midPoint:      1,
				NormalizedSum: BoundedSumFloat64{},
				Count:         Count{},
				state:         defaultState},
			false,
		},
		{
			"different count",
			&BoundedMeanFloat64{
				lower:         0,
				upper:         2,
				midPoint:      1,
				NormalizedSum: BoundedSumFloat64{},
				Count:         Count{epsilon: ln3},
				state:         defaultState},
			&BoundedMeanFloat64{
				lower:         0,
				upper:         2,
				midPoint:      1,
				NormalizedSum: BoundedSumFloat64{},
				Count:         Count{epsilon: 1},
				state:         defaultState},
			false,
		},
		{
			"different normalizedSum",
			&BoundedMeanFloat64{
				lower:         0,
				upper:         2,
				midPoint:      1,
				NormalizedSum: BoundedSumFloat64{epsilon: ln3},
				Count:         Count{},
				state:         defaultState},
			&BoundedMeanFloat64{
				lower:         0,
				upper:         2,
				midPoint:      1,
				NormalizedSum: BoundedSumFloat64{epsilon: 1},
				Count:         Count{},
				state:         defaultState},
			false,
		},
		{
			"different state",
			&BoundedMeanFloat64{
				lower:         0,
				upper:         2,
				midPoint:      1,
				NormalizedSum: BoundedSumFloat64{},
				Count:         Count{},
				state:         defaultState},
			&BoundedMeanFloat64{
				lower:         0,
				upper:         2,
				midPoint:      1,
				NormalizedSum: BoundedSumFloat64{},
				Count:         Count{},
				state:         merged},
			false,
		},
	} {
		if bmEquallyInitializedFloat64(tc.bm1, tc.bm2) != tc.equal {
			t.Errorf("bmEquallyInitializedFloat64: when %s got %t, want %t", tc.desc, !tc.equal, tc.equal)
		}
	}
}

func compareBoundedMeanFloat64(bm1, bm2 *BoundedMeanFloat64) bool {
	return bm1.lower == bm2.lower &&
		bm1.upper == bm2.upper &&
		compareCount(&bm1.Count, &bm2.Count) &&
		compareBoundedSumFloat64(&bm1.NormalizedSum, &bm2.NormalizedSum) &&
		bm1.midPoint == bm2.midPoint &&
		bm1.state == bm2.state
}

// Tests that serialization for BoundedMeanFloat64 works as expected.
func TestBMFloat64Serialization(t *testing.T) {
	for _, tc := range []struct {
		desc string
		opts *BoundedMeanFloat64Options
	}{
		{"default options", &BoundedMeanFloat64Options{
			Epsilon:                      ln3,
			Lower:                        0,
			Upper:                        1,
			Delta:                        0,
			MaxContributionsPerPartition: 1,
		}},
		{"non-default options", &BoundedMeanFloat64Options{
			Lower:                        -100,
			Upper:                        555,
			Epsilon:                      ln3,
			Delta:                        1e-5,
			MaxPartitionsContributed:     5,
			MaxContributionsPerPartition: 6,
			Noise:                        noise.Gaussian(),
		}},
	} {
		bm, err := NewBoundedMeanFloat64(tc.opts)
		if err != nil {
			t.Fatalf("Couldn't initialize bm: %v", err)
		}
		bmUnchanged, err := NewBoundedMeanFloat64(tc.opts)
		if err != nil {
			t.Fatalf("Couldn't initialize bmUnchanged: %v", err)
		}
		bytes, err := encode(bm)
		if err != nil {
			t.Fatalf("encode(BoundedMeanFloat64) error: %v", err)
		}
		bmUnmarshalled := new(BoundedMeanFloat64)
		if err := decode(bmUnmarshalled, bytes); err != nil {
			t.Fatalf("decode(BoundedMeanFloat64) error: %v", err)
		}
		// Check that encoding -> decoding is the identity function.
		if !cmp.Equal(bmUnchanged, bmUnmarshalled, cmp.Comparer(compareBoundedMeanFloat64)) {
			t.Errorf("decode(encode(_)): when %s got %+v, want %+v", tc.desc, bmUnmarshalled, bmUnchanged)
		}
		if bm.state != serialized {
			t.Errorf("BoundedMean should have its state set to Serialized, got %v , want Serialized", bm.state)
		}
	}
}

// Tests that GobEncode() returns errors correctly with different BoundedMeanFloat64 aggregation states.
func TestBoundedMeanFloat64SerializationStateChecks(t *testing.T) {
	for _, tc := range []struct {
		state   aggregationState
		wantErr bool
	}{
		{defaultState, false},
		{merged, true},
		{serialized, false},
		{resultReturned, true},
	} {
		bm := getNoiselessBMF(t)
		bm.state = tc.state

		if _, err := bm.GobEncode(); (err != nil) != tc.wantErr {
			t.Errorf("GobEncode: when state %v for err got %v, wantErr %t", tc.state, err, tc.wantErr)
		}
	}
}
