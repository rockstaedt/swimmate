package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNumberFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected string
	}{
		{"zero", 0, "0"},
		{"single digit", 5, "5"},
		{"two digits", 42, "42"},
		{"three digits", 999, "999"},
		{"one thousand", 1000, "1,000"},
		{"ten thousand", 10000, "10,000"},
		{"hundred thousand", 100000, "100,000"},
		{"one million", 1000000, "1,000,000"},
		{"arbitrary large", 1234567, "1,234,567"},
		{"negative single digit", -5, "-5"},
		{"negative two digits", -42, "-42"},
		{"negative three digits", -999, "-999"},
		{"negative one thousand", -1000, "-1,000"},
		{"negative ten thousand", -10000, "-10,000"},
		{"negative million", -1234567, "-1,234,567"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := numberFormat(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSub(t *testing.T) {
	tests := []struct {
		name     string
		a        int
		b        int
		expected int
	}{
		{"positive numbers", 10, 5, 5},
		{"result zero", 5, 5, 0},
		{"result negative", 5, 10, -5},
		{"negative first operand", -10, 5, -15},
		{"negative second operand", 10, -5, 15},
		{"both negative", -10, -5, -5},
		{"large numbers", 1000000, 999999, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sub(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAdd(t *testing.T) {
	tests := []struct {
		name     string
		a        int
		b        int
		expected int
	}{
		{"positive numbers", 10, 5, 15},
		{"with zero first", 0, 5, 5},
		{"with zero second", 10, 0, 10},
		{"both zero", 0, 0, 0},
		{"negative first operand", -10, 5, -5},
		{"negative second operand", 10, -5, 5},
		{"both negative", -10, -5, -15},
		{"large numbers", 1000000, 999999, 1999999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := add(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSeq(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected []int
	}{
		{"zero", 0, []int{}},
		{"one", 1, []int{1}},
		{"five", 5, []int{1, 2, 3, 4, 5}},
		{"ten", 10, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := seq(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		name     string
		a        int
		b        int
		expected int
	}{
		{"first smaller", 5, 10, 5},
		{"second smaller", 10, 5, 5},
		{"equal", 5, 5, 5},
		{"both zero", 0, 0, 0},
		{"negative first", -5, 10, -5},
		{"negative second", 10, -5, -5},
		{"both negative", -10, -5, -10},
		{"large numbers", 1000000, 999999, 999999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := min(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEmptyStars(t *testing.T) {
	tests := []struct {
		name       string
		assessment int
		expected   int
	}{
		{"zero stars", 0, 2},
		{"one star", 1, 1},
		{"two stars", 2, 0},
		{"three stars (capped at max)", 3, 0},
		{"five stars (capped at max)", 5, 0},
		{"negative (edge case)", -1, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := emptyStars(tt.assessment)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewFlash(t *testing.T) {
	tests := []struct {
		name          string
		text          string
		flashType     string
		expectedFlash *Flash
	}{
		{
			name:      "with text and type",
			text:      "Success!",
			flashType: "flash-success",
			expectedFlash: &Flash{
				Text: "Success!",
				Type: "flash-success",
			},
		},
		{
			name:      "with text, default type",
			text:      "Something happened",
			flashType: "",
			expectedFlash: &Flash{
				Text: "Something happened",
				Type: "flash-success",
			},
		},
		{
			name:      "with text and error type",
			text:      "Error occurred",
			flashType: "flash-error",
			expectedFlash: &Flash{
				Text: "Error occurred",
				Type: "flash-error",
			},
		},
		{
			name:          "empty text returns nil",
			text:          "",
			flashType:     "flash-success",
			expectedFlash: nil,
		},
		{
			name:          "empty text and type returns nil",
			text:          "",
			flashType:     "",
			expectedFlash: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := newFlash(tt.text, tt.flashType)
			assert.Equal(t, tt.expectedFlash, result)
		})
	}
}
