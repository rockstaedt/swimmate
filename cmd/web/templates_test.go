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

func TestDiv(t *testing.T) {
	tests := []struct {
		name     string
		a        int
		b        int
		expected int
	}{
		{"positive division", 10, 2, 5},
		{"division with remainder", 10, 3, 3},
		{"division by zero", 10, 0, 0},
		{"zero divided by number", 0, 5, 0},
		{"both zero", 0, 0, 0},
		{"negative dividend", -10, 2, -5},
		{"negative divisor", 10, -2, -5},
		{"both negative", -10, -2, 5},
		{"large numbers", 1000000, 1000, 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := div(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAtoi(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"zero", "0", 0},
		{"positive number", "42", 42},
		{"negative number", "-42", -42},
		{"large number", "1234567", 1234567},
		{"invalid string", "abc", 0},
		{"empty string", "", 0},
		{"mixed alphanumeric", "123abc", 0},
		{"with spaces", " 42 ", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := atoi(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSlice(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		start    int
		end      int
		expected string
	}{
		{"normal slice", "hello", 1, 4, "ell"},
		{"from beginning", "hello", 0, 3, "hel"},
		{"to end", "hello", 2, 5, "llo"},
		{"full string", "hello", 0, 5, "hello"},
		{"start negative", "hello", -1, 3, ""},
		{"start out of bounds", "hello", 10, 15, ""},
		{"end negative treats as length", "hello", 0, -1, "hello"},
		{"end beyond length", "hello", 0, 10, "hello"},
		{"start equals end", "hello", 2, 2, ""},
		{"start greater than end", "hello", 3, 1, ""},
		{"empty string", "", 0, 0, ""},
		{"date year extraction", "2025-11-18", 0, 4, "2025"},
		{"date month extraction", "2025-11-18", 5, 7, "11"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := slice(tt.s, tt.start, tt.end)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMonthAbbr(t *testing.T) {
	tests := []struct {
		name     string
		month    int
		expected string
	}{
		{"January", 1, "J"},
		{"February", 2, "F"},
		{"March", 3, "M"},
		{"April", 4, "A"},
		{"May", 5, "M"},
		{"June", 6, "J"},
		{"July", 7, "J"},
		{"August", 8, "A"},
		{"September", 9, "S"},
		{"October", 10, "O"},
		{"November", 11, "N"},
		{"December", 12, "D"},
		{"zero (invalid)", 0, ""},
		{"thirteen (invalid)", 13, ""},
		{"negative (invalid)", -1, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := monthAbbr(tt.month)
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
