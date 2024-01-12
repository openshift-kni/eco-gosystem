package powermanagementhelper

import (
	"errors"
	"math"
	"sort"
)

// Min returns the minimum value of the input array.
func Min(input []float64) (min float64, err error) {
	if len(input) < 1 {
		return math.NaN(), errors.New("input array must have at least 1 element")
	}

	min = input[0]
	for i := 1; i < len(input); i++ {
		if input[i] < min {
			min = input[i]
		}
	}

	return min, nil
}

// Max returns the maximum value of the input array.
func Max(input []float64) (max float64, err error) {
	if len(input) < 1 {
		return math.NaN(), errors.New("input array must have at least 1 element")
	}

	max = input[0]
	for i := 1; i < len(input); i++ {
		if input[i] > max {
			max = input[i]
		}
	}

	return max, nil
}

// Mean computes the mean value of the input array.
func Mean(input []float64) (mean float64, err error) {
	if len(input) < 1 {
		return math.NaN(), errors.New("input array must have at least 1 element")
	}

	mean = 0
	numElements := len(input)

	for i := 0; i < len(input); i++ {
		mean += input[i]
	}

	return mean / float64(numElements), nil
}

// StdDev computes the population standard deviation of the input array.
func StdDev(input []float64) (stdev float64, err error) {
	if len(input) < 1 {
		return math.NaN(), errors.New("input array must have at least 1 element")
	}

	mean, _ := Mean(input)

	v := 0.0
	for _, x := range input {
		v += (x - mean) * (x - mean)
	}

	return math.Sqrt(v / float64(len(input))), nil
}

// Median computes the median value of the input array.
func Median(input []float64) (median float64, err error) {
	if len(input) < 1 {
		return math.NaN(), errors.New("input array must have at least 1 element")
	}

	numElements := len(input)

	// sort a copy of the input array
	inputCopy := make([]float64, numElements)
	copy(inputCopy, input)

	sort.Float64s(inputCopy)

	if numElements%2 == 1 {
		median = inputCopy[numElements/2]
	} else {
		median = (inputCopy[numElements/2] + inputCopy[numElements/2-1]) / 2
	}

	return median, nil
}
