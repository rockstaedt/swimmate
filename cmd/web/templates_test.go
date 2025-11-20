package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rockstaedt/swimmate/internal/models"
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
		{"January", 1, "Jan"},
		{"February", 2, "Feb"},
		{"March", 3, "Mar"},
		{"April", 4, "Apr"},
		{"May", 5, "May"},
		{"June", 6, "Jun"},
		{"July", 7, "Jul"},
		{"August", 8, "Aug"},
		{"September", 9, "Sep"},
		{"October", 10, "Oct"},
		{"November", 11, "Nov"},
		{"December", 12, "Dec"},
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

func TestWithPartial(t *testing.T) {
	tests := []struct {
		name           string
		originalData   templateData
		partial        interface{}
		expectedData   templateData
	}{
		{
			name: "with swim data",
			originalData: templateData{
				Version: "1.0.0",
				Data:    "original",
			},
			partial: &models.Swim{
				Id:         1,
				Date:       time.Now(),
				DistanceM:  1500,
				Assessment: 2,
			},
			expectedData: templateData{
				Version: "1.0.0",
				Data:    "original",
				Partial: &models.Swim{
					Id:         1,
					Date:       time.Now(),
					DistanceM:  1500,
					Assessment: 2,
				},
			},
		},
		{
			name: "with string partial",
			originalData: templateData{
				Version: "1.0.0",
				Data:    "original",
			},
			partial: "test string",
			expectedData: templateData{
				Version: "1.0.0",
				Data:    "original",
				Partial: "test string",
			},
		},
		{
			name: "with nil partial",
			originalData: templateData{
				Version:         "1.0.0",
				Data:            "original",
				IsAuthenticated: true,
			},
			partial: nil,
			expectedData: templateData{
				Version:         "1.0.0",
				Data:            "original",
				IsAuthenticated: true,
				Partial:         nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := withPartial(tt.originalData, tt.partial)
			assert.Equal(t, tt.expectedData.Version, result.Version)
			assert.Equal(t, tt.expectedData.Data, result.Data)
			assert.Equal(t, tt.expectedData.IsAuthenticated, result.IsAuthenticated)
			if tt.partial != nil {
				assert.NotNil(t, result.Partial)
			} else {
				assert.Nil(t, result.Partial)
			}
		})
	}
}

func TestNewTemplateData(t *testing.T) {
	tests := []struct {
		name                string
		setupSession        func(*application, *http.Request) *http.Request
		data                interface{}
		expectAuthenticated bool
		expectFlash         bool
	}{
		{
			name: "with authenticated user",
			setupSession: func(app *application, r *http.Request) *http.Request {
				ctx, _ := app.sessionManager.Load(r.Context(), "")
				app.sessionManager.Put(ctx, "authenticatedUserID", 1)
				return r.WithContext(ctx)
			},
			data:                "test data",
			expectAuthenticated: true,
			expectFlash:         false,
		},
		{
			name: "without authenticated user",
			setupSession: func(app *application, r *http.Request) *http.Request {
				ctx, _ := app.sessionManager.Load(r.Context(), "")
				return r.WithContext(ctx)
			},
			data:                "test data",
			expectAuthenticated: false,
			expectFlash:         false,
		},
		{
			name: "with flash message",
			setupSession: func(app *application, r *http.Request) *http.Request {
				ctx, _ := app.sessionManager.Load(r.Context(), "")
				app.sessionManager.Put(ctx, "flashText", "Success!")
				app.sessionManager.Put(ctx, "flashType", "flash-success")
				return r.WithContext(ctx)
			},
			data:                nil,
			expectAuthenticated: false,
			expectFlash:         true,
		},
		{
			name: "with empty flash text",
			setupSession: func(app *application, r *http.Request) *http.Request {
				ctx, _ := app.sessionManager.Load(r.Context(), "")
				app.sessionManager.Put(ctx, "flashText", "")
				return r.WithContext(ctx)
			},
			data:                nil,
			expectAuthenticated: false,
			expectFlash:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApplication()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r = tt.setupSession(app, r)

			result := app.newTemplateData(r, tt.data)

			assert.Equal(t, "test", result.Version)
			assert.Equal(t, tt.data, result.Data)
			assert.Equal(t, tt.expectAuthenticated, result.IsAuthenticated)

			if tt.expectFlash {
				assert.NotNil(t, result.Flash)
				assert.NotEmpty(t, result.Flash.Text)
			} else {
				assert.Nil(t, result.Flash)
			}

			assert.NotEmpty(t, result.CurrentDate)
		})
	}
}

func TestNewTemplateCache(t *testing.T) {
	cache, err := newTemplateCache()

	assert.NoError(t, err, "newTemplateCache should not return an error")
	assert.NotEmpty(t, cache, "cache should not be empty")

	expectedTemplates := []string{
		"home.tmpl",
		"login.tmpl",
		"about.tmpl",
		"swims.tmpl",
		"yearly-figures.tmpl",
		"swim-create.tmpl",
		"swim-edit.tmpl",
	}

	for _, tmpl := range expectedTemplates {
		t.Run("has template "+tmpl, func(t *testing.T) {
			assert.Contains(t, cache, tmpl, "cache should contain "+tmpl)
			assert.NotNil(t, cache[tmpl], tmpl+" should not be nil")
		})
	}

	t.Run("templates have base", func(t *testing.T) {
		for name, tmpl := range cache {
			baseTemplate := tmpl.Lookup("base")
			assert.NotNil(t, baseTemplate, "template "+name+" should have base")
		}
	})
}
