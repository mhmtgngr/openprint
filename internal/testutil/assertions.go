// Package testutil provides custom assertion helpers for cleaner test code.
package testutil

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Assert provides custom assertion methods.
// These methods panic on failure so they can be tested with assert.Panics.
type Assert struct {
	t *testing.T
}

// NewAssert creates a new assertion helper.
func NewAssert(t *testing.T) *Assert {
	return &Assert{t: t}
}

// NoError asserts that an error is nil.
func (a *Assert) NoError(err error, msgAndArgs ...interface{}) {
	if err != nil {
		a.t.Helper()
		panic(fmt.Sprintf("NoError failed: %v %s", err, fmt.Sprint(msgAndArgs...)))
	}
}

// Error asserts that an error is not nil.
func (a *Assert) Error(err error, msgAndArgs ...interface{}) {
	if err == nil {
		a.t.Helper()
		panic("Error failed: expected error, got nil " + fmt.Sprint(msgAndArgs...))
	}
}

// ErrorIs asserts that an error matches a target error.
func (a *Assert) ErrorIs(err error, target error, msgAndArgs ...interface{}) {
	if !errors.Is(err, target) {
		a.t.Helper()
		panic(fmt.Sprintf("ErrorIs failed: expected %v, got %v %s", target, err, fmt.Sprint(msgAndArgs...)))
	}
}

// ErrorType asserts that an error is of a specific type.
func (a *Assert) ErrorType(err error, targetType error, msgAndArgs ...interface{}) {
	a.t.Helper()

	if err == nil {
		panic(fmt.Sprintf("ErrorType failed: expected error type %T, got nil %s", targetType, fmt.Sprint(msgAndArgs...)))
	}

	targetTypeVal := reflect.TypeOf(targetType)
	errType := reflect.TypeOf(err)

	if !errType.AssignableTo(targetTypeVal) {
		panic(fmt.Sprintf("ErrorType failed: expected error type %T, got %T %s", targetType, err, fmt.Sprint(msgAndArgs...)))
	}
}

// Equals asserts that two values are equal.
func (a *Assert) Equals(expected, actual interface{}, msgAndArgs ...interface{}) {
	a.t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		panic(fmt.Sprintf("Equals failed:\nexpected: %#v (%T)\nactual:   %#v (%T) %s",
			expected, expected, actual, actual, fmt.Sprint(msgAndArgs...)))
	}
}

// NotEquals asserts that two values are not equal.
func (a *Assert) NotEquals(notExpected, actual interface{}, msgAndArgs ...interface{}) {
	a.t.Helper()
	if reflect.DeepEqual(notExpected, actual) {
		panic(fmt.Sprintf("NotEquals failed: values are equal: %#v %s", actual, fmt.Sprint(msgAndArgs...)))
	}
}

// True asserts that a condition is true.
func (a *Assert) True(condition bool, msgAndArgs ...interface{}) {
	a.t.Helper()
	if !condition {
		panic("True failed: condition is false " + fmt.Sprint(msgAndArgs...))
	}
}

// False asserts that a condition is false.
func (a *Assert) False(condition bool, msgAndArgs ...interface{}) {
	a.t.Helper()
	if condition {
		panic("False failed: condition is true " + fmt.Sprint(msgAndArgs...))
	}
}

// Nil asserts that a value is nil.
func (a *Assert) Nil(value interface{}, msgAndArgs ...interface{}) {
	a.t.Helper()
	if value != nil {
		// Check for typed nil (e.g., (*string)(nil))
		rv := reflect.ValueOf(value)
		if rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface || rv.Kind() == reflect.Slice || rv.Kind() == reflect.Map || rv.Kind() == reflect.Chan || rv.Kind() == reflect.Func {
			if !rv.IsNil() {
				panic(fmt.Sprintf("Nil failed: expected nil, got %#v %s", value, fmt.Sprint(msgAndArgs...)))
			}
			// Typed nil is considered nil
			return
		}
		panic(fmt.Sprintf("Nil failed: expected nil, got %#v %s", value, fmt.Sprint(msgAndArgs...)))
	}
}

// NotNil asserts that a value is not nil.
func (a *Assert) NotNil(value interface{}, msgAndArgs ...interface{}) {
	a.t.Helper()
	if value == nil {
		panic("NotNil failed: value is nil " + fmt.Sprint(msgAndArgs...))
	}
	// Also check for typed nil
	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface || rv.Kind() == reflect.Slice || rv.Kind() == reflect.Map || rv.Kind() == reflect.Chan || rv.Kind() == reflect.Func {
		if rv.IsNil() {
			panic("NotNil failed: value is nil " + fmt.Sprint(msgAndArgs...))
		}
	}
}

// Contains asserts that a string contains a substring.
func (a *Assert) Contains(s, substr string, msgAndArgs ...interface{}) {
	a.t.Helper()
	if !strings.Contains(s, substr) {
		panic(fmt.Sprintf("Contains failed: %q does not contain %q %s", s, substr, fmt.Sprint(msgAndArgs...)))
	}
}

// NotContains asserts that a string does not contain a substring.
func (a *Assert) NotContains(s, substr string, msgAndArgs ...interface{}) {
	a.t.Helper()
	if strings.Contains(s, substr) {
		panic(fmt.Sprintf("NotContains failed: %q contains %q %s", s, substr, fmt.Sprint(msgAndArgs...)))
	}
}

// ContainsAll asserts that a string contains all substrings.
func (a *Assert) ContainsAll(s string, substrs []string, msgAndArgs ...interface{}) {
	a.t.Helper()
	for _, substr := range substrs {
		if !strings.Contains(s, substr) {
			panic(fmt.Sprintf("ContainsAll failed: %q does not contain %q %s", s, substr, fmt.Sprint(msgAndArgs...)))
		}
	}
}

// Len asserts that a slice/map/channel/string has the expected length.
func (a *Assert) Len(obj interface{}, expectedLength int, msgAndArgs ...interface{}) {
	a.t.Helper()

	val := reflect.ValueOf(obj)
	var actualLength int

	switch val.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map, reflect.Chan, reflect.String:
		actualLength = val.Len()
	default:
		panic(fmt.Sprintf("Len failed: type %T does not have a length %s", obj, fmt.Sprint(msgAndArgs...)))
	}

	if actualLength != expectedLength {
		panic(fmt.Sprintf("Len failed: expected length %d, got %d %s", expectedLength, actualLength, fmt.Sprint(msgAndArgs...)))
	}
}

// Empty asserts that a slice/map/channel/string is empty.
func (a *Assert) Empty(obj interface{}, msgAndArgs ...interface{}) {
	a.t.Helper()
	a.Len(obj, 0, msgAndArgs...)
}

// NotEmpty asserts that a slice/map/channel/string is not empty.
func (a *Assert) NotEmpty(obj interface{}, msgAndArgs ...interface{}) {
	a.t.Helper()

	val := reflect.ValueOf(obj)
	var length int

	switch val.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map, reflect.Chan, reflect.String:
		length = val.Len()
	default:
		panic(fmt.Sprintf("NotEmpty failed: type %T does not have a length %s", obj, fmt.Sprint(msgAndArgs...)))
	}

	if length == 0 {
		panic("NotEmpty failed: value is empty " + fmt.Sprint(msgAndArgs...))
	}
}

// Greater asserts that first > second.
func (a *Assert) Greater(first, second interface{}, msgAndArgs ...interface{}) {
	a.t.Helper()

	firstVal := reflect.ValueOf(first)
	secondVal := reflect.ValueOf(second)

	if !firstVal.CanInt() || !secondVal.CanInt() {
		panic("Greater failed: values must be comparable integers " + fmt.Sprint(msgAndArgs...))
	}

	if firstVal.Int() <= secondVal.Int() {
		panic(fmt.Sprintf("Greater failed: %v is not greater than %v %s", first, second, fmt.Sprint(msgAndArgs...)))
	}
}

// Less asserts that first < second.
func (a *Assert) Less(first, second interface{}, msgAndArgs ...interface{}) {
	a.t.Helper()

	firstVal := reflect.ValueOf(first)
	secondVal := reflect.ValueOf(second)

	if !firstVal.CanInt() || !secondVal.CanInt() {
		panic("Less failed: values must be comparable integers " + fmt.Sprint(msgAndArgs...))
	}

	if firstVal.Int() >= secondVal.Int() {
		panic(fmt.Sprintf("Less failed: %v is not less than %v %s", first, second, fmt.Sprint(msgAndArgs...)))
	}
}

// WithinDuration asserts that two times are within a duration of each other.
func (a *Assert) WithinDuration(expected, actual time.Time, delta time.Duration, msgAndArgs ...interface{}) {
	a.t.Helper()

	diff := expected.Sub(actual)
	if diff < 0 {
		diff = -diff
	}

	if diff > delta {
		panic(fmt.Sprintf("WithinDuration failed: expected %v, got %v (difference %v, max allowed %v) %s",
			expected, actual, diff, delta, fmt.Sprint(msgAndArgs...)))
	}
}

// Panics asserts that the function panics.
func (a *Assert) Panics(f func(), msgAndArgs ...interface{}) {
	a.t.Helper()

	didPanic := false
	var panicValue interface{}

	func() {
		defer func() {
			if panicValue = recover(); panicValue != nil {
				didPanic = true
			}
		}()
		f()
	}()

	if !didPanic {
		panic("Panics failed: function did not panic " + fmt.Sprint(msgAndArgs...))
	}
}

// NotPanics asserts that the function does not panic.
func (a *Assert) NotPanics(f func(), msgAndArgs ...interface{}) {
	a.t.Helper()

	var panicValue interface{}

	func() {
		defer func() {
			if panicValue = recover(); panicValue != nil {
				panic(fmt.Sprintf("NotPanics failed: function panicked with %v %s", panicValue, fmt.Sprint(msgAndArgs...)))
			}
		}()
		f()
	}()
}

// Eventually asserts that a condition will eventually become true.
func (a *Assert) Eventually(condition func() bool, timeout time.Duration, msgAndArgs ...interface{}) {
	a.t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		if condition() {
			return
		}

		select {
		case <-ctx.Done():
			panic(fmt.Sprintf("Eventually failed: condition never became true within %v %s", timeout, fmt.Sprint(msgAndArgs...)))
		case <-ticker.C:
		}
	}
}

// Consistently asserts that a condition remains true for a duration.
func (a *Assert) Consistently(condition func() bool, duration time.Duration, msgAndArgs ...interface{}) {
	a.t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		if !condition() {
			panic(fmt.Sprintf("Consistently failed: condition became false before %v elapsed %s", duration, fmt.Sprint(msgAndArgs...)))
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// FileExists asserts that a file exists.
func (a *Assert) FileExists(path string, msgAndArgs ...interface{}) {
	a.t.Helper()

	fullPath := filepath.Clean(path)
	if _, err := filepath.Abs(fullPath); err != nil {
		panic(fmt.Sprintf("FileExists failed: invalid path %q: %v %s", path, err, fmt.Sprint(msgAndArgs...)))
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			panic(fmt.Sprintf("FileExists failed: file does not exist: %q %s", path, fmt.Sprint(msgAndArgs...)))
		}
		panic(fmt.Sprintf("FileExists failed: error checking file %q: %v %s", path, err, fmt.Sprint(msgAndArgs...)))
	}

	if info.IsDir() {
		panic(fmt.Sprintf("FileExists failed: path is a directory, not a file: %q %s", path, fmt.Sprint(msgAndArgs...)))
	}
}

// DirExists asserts that a directory exists.
func (a *Assert) DirExists(path string, msgAndArgs ...interface{}) {
	a.t.Helper()

	fullPath := filepath.Clean(path)
	if _, err := filepath.Abs(fullPath); err != nil {
		panic(fmt.Sprintf("DirExists failed: invalid path %q: %v %s", path, err, fmt.Sprint(msgAndArgs...)))
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			panic(fmt.Sprintf("DirExists failed: directory does not exist: %q %s", path, fmt.Sprint(msgAndArgs...)))
		}
		panic(fmt.Sprintf("DirExists failed: error checking directory %q: %v %s", path, err, fmt.Sprint(msgAndArgs...)))
	}

	if !info.IsDir() {
		panic(fmt.Sprintf("DirExists failed: path is a file, not a directory: %q %s", path, fmt.Sprint(msgAndArgs...)))
	}
}

// FileContains asserts that a file contains a substring.
func (a *Assert) FileContains(path, substr string, msgAndArgs ...interface{}) {
	a.t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("FileContains failed: error reading file %q: %v %s", path, err, fmt.Sprint(msgAndArgs...)))
	}

	if !bytes.Contains(content, []byte(substr)) {
		panic(fmt.Sprintf("FileContains failed: file %q does not contain %q %s", path, substr, fmt.Sprint(msgAndArgs...)))
	}
}

// JSONEq asserts that two JSON strings are equivalent.
func (a *Assert) JSONEq(expected, actual string, msgAndArgs ...interface{}) {
	a.t.Helper()

	// Simple check - in production, use a proper JSON comparison
	expected = strings.TrimSpace(expected)
	actual = strings.TrimSpace(actual)

	// Remove whitespace for simple comparison
	// Note: This is a simplified implementation. For proper JSON equality,
	// you would want to parse and compare the JSON objects.
	if expected != actual {
		// Try to give a helpful error message
		panic(fmt.Sprintf("JSONEq failed:\nexpected: %s\nactual:   %s %s", expected, actual, fmt.Sprint(msgAndArgs...)))
	}
}

// HTTPSuccess asserts that an HTTP response has a success status code (2xx).
func (a *Assert) HTTPSuccess(resp interface{ StatusCode() int }, msgAndArgs ...interface{}) {
	a.t.Helper()

	status := resp.StatusCode()
	if status < 200 || status >= 300 {
		panic(fmt.Sprintf("HTTPSuccess failed: expected 2xx status, got %d %s", status, fmt.Sprint(msgAndArgs...)))
	}
}

// HTTPStatus asserts that an HTTP response has a specific status code.
func (a *Assert) HTTPStatus(resp interface{ StatusCode() int }, expectedStatus int, msgAndArgs ...interface{}) {
	a.t.Helper()

	status := resp.StatusCode()
	if status != expectedStatus {
		panic(fmt.Sprintf("HTTPStatus failed: expected status %d, got %d %s", expectedStatus, status, fmt.Sprint(msgAndArgs...)))
	}
}

// HTTPHeader asserts that an HTTP response has a specific header value.
func (a *Assert) HTTPHeader(resp interface{ Header() http.Header }, key, expectedValue string, msgAndArgs ...interface{}) {
	a.t.Helper()

	actualValue := resp.Header().Get(key)
	if actualValue != expectedValue {
		panic(fmt.Sprintf("HTTPHeader failed: expected header %q=%q, got %q %s", key, expectedValue, actualValue, fmt.Sprint(msgAndArgs...)))
	}
}

// HTTPBodyContains asserts that an HTTP response body contains a substring.
func (a *Assert) HTTPBodyContains(resp interface{ Body() []byte }, substr string, msgAndArgs ...interface{}) {
	a.t.Helper()

	body := resp.Body()
	if !bytes.Contains(body, []byte(substr)) {
		panic(fmt.Sprintf("HTTPBodyContains failed: body does not contain %q\nbody: %s %s", substr, string(body), fmt.Sprint(msgAndArgs...)))
	}
}

// Retryable wraps a function and retries it on failure.
type Retryable struct {
	maxAttempts int
	delay       time.Duration
	assert      *Assert
}

// NewRetryable creates a new retryable assertion helper.
func NewRetryable(assert *Assert, maxAttempts int, delay time.Duration) *Retryable {
	return &Retryable{
		maxAttempts: maxAttempts,
		delay:       delay,
		assert:      assert,
	}
}

// EventuallyTrue retries a function until it returns true or max attempts reached.
func (r *Retryable) EventuallyTrue(fn func() bool, msgAndArgs ...interface{}) {
	r.assert.t.Helper()

	for i := 0; i < r.maxAttempts; i++ {
		if fn() {
			return
		}
		if i < r.maxAttempts-1 {
			time.Sleep(r.delay)
		}
	}

	panic(fmt.Sprintf("EventuallyTrue failed: condition never became true after %d attempts %s",
		r.maxAttempts, fmt.Sprint(msgAndArgs...)))
}

// EventuallyNoError retries a function until it returns no error or max attempts reached.
func (r *Retryable) EventuallyNoError(fn func() error, msgAndArgs ...interface{}) {
	r.assert.t.Helper()

	var lastErr error
	for i := 0; i < r.maxAttempts; i++ {
		if err := fn(); err == nil {
			return
		} else {
			lastErr = err
		}
		if i < r.maxAttempts-1 {
			time.Sleep(r.delay)
		}
	}

	panic(fmt.Sprintf("EventuallyNoError failed: %v after %d attempts %s",
		lastErr, r.maxAttempts, fmt.Sprint(msgAndArgs...)))
}

// AllSatisfy asserts that all elements in a slice satisfy a predicate.
func (a *Assert) AllSatisfy(slice interface{}, predicate func(interface{}) bool, msgAndArgs ...interface{}) {
	a.t.Helper()

	val := reflect.ValueOf(slice)
	if val.Kind() != reflect.Slice && val.Kind() != reflect.Array {
		panic("AllSatisfy failed: input is not a slice/array " + fmt.Sprint(msgAndArgs...))
	}

	for i := 0; i < val.Len(); i++ {
		elem := val.Index(i).Interface()
		if !predicate(elem) {
			panic(fmt.Sprintf("AllSatisfy failed: element at index %d (%#v) does not satisfy predicate %s",
				i, elem, fmt.Sprint(msgAndArgs...)))
		}
	}
}

// NoneSatisfy asserts that no elements in a slice satisfy a predicate.
func (a *Assert) NoneSatisfy(slice interface{}, predicate func(interface{}) bool, msgAndArgs ...interface{}) {
	a.t.Helper()

	val := reflect.ValueOf(slice)
	if val.Kind() != reflect.Slice && val.Kind() != reflect.Array {
		panic("NoneSatisfy failed: input is not a slice/array " + fmt.Sprint(msgAndArgs...))
	}

	for i := 0; i < val.Len(); i++ {
		elem := val.Index(i).Interface()
		if predicate(elem) {
			panic(fmt.Sprintf("NoneSatisfy failed: element at index %d (%#v) satisfies predicate %s",
				i, elem, fmt.Sprint(msgAndArgs...)))
		}
	}
}

// ContainsExactly asserts that a slice contains exactly the specified elements.
func (a *Assert) ContainsExactly(slice, expected interface{}, msgAndArgs ...interface{}) {
	a.t.Helper()

	sliceVal := reflect.ValueOf(slice)
	expectedVal := reflect.ValueOf(expected)

	if sliceVal.Kind() != reflect.Slice && sliceVal.Kind() != reflect.Array {
		panic("ContainsExactly failed: first argument is not a slice/array " + fmt.Sprint(msgAndArgs...))
	}

	if expectedVal.Kind() != reflect.Slice && expectedVal.Kind() != reflect.Array {
		panic("ContainsExactly failed: second argument is not a slice/array " + fmt.Sprint(msgAndArgs...))
	}

	if sliceVal.Len() != expectedVal.Len() {
		panic(fmt.Sprintf("ContainsExactly failed: length mismatch: got %d, want %d %s",
			sliceVal.Len(), expectedVal.Len(), fmt.Sprint(msgAndArgs...)))
	}

	// Check each expected element exists in slice
	for i := 0; i < expectedVal.Len(); i++ {
		expectedElem := expectedVal.Index(i).Interface()
		found := false
		for j := 0; j < sliceVal.Len(); j++ {
			if reflect.DeepEqual(sliceVal.Index(j).Interface(), expectedElem) {
				found = true
				break
			}
		}
		if !found {
			panic(fmt.Sprintf("ContainsExactly failed: element %#v not found in slice %s",
				expectedElem, fmt.Sprint(msgAndArgs...)))
		}
	}
}

// KeyExists asserts that a map contains a key.
func (a *Assert) KeyExists(m interface{}, key interface{}, msgAndArgs ...interface{}) {
	a.t.Helper()

	val := reflect.ValueOf(m)
	if val.Kind() != reflect.Map {
		panic("KeyExists failed: input is not a map " + fmt.Sprint(msgAndArgs...))
	}

	keyVal := reflect.ValueOf(key)
	if !val.MapIndex(keyVal).IsValid() {
		panic(fmt.Sprintf("KeyExists failed: key %#v does not exist in map %s", key, fmt.Sprint(msgAndArgs...)))
	}
}

// KeyNotExists asserts that a map does not contain a key.
func (a *Assert) KeyNotExists(m interface{}, key interface{}, msgAndArgs ...interface{}) {
	a.t.Helper()

	val := reflect.ValueOf(m)
	if val.Kind() != reflect.Map {
		panic("KeyNotExists failed: input is not a map " + fmt.Sprint(msgAndArgs...))
	}

	keyVal := reflect.ValueOf(key)
	if val.MapIndex(keyVal).IsValid() {
		panic(fmt.Sprintf("KeyNotExists failed: key %#v exists in map %s", key, fmt.Sprint(msgAndArgs...)))
	}
}

// MapLen asserts that a map has the expected length.
func (a *Assert) MapLen(m interface{}, expectedLength int, msgAndArgs ...interface{}) {
	a.t.Helper()

	val := reflect.ValueOf(m)
	if val.Kind() != reflect.Map {
		panic("MapLen failed: input is not a map " + fmt.Sprint(msgAndArgs...))
	}

	actualLength := val.Len()
	if actualLength != expectedLength {
		panic(fmt.Sprintf("MapLen failed: expected length %d, got %d %s",
			expectedLength, actualLength, fmt.Sprint(msgAndArgs...)))
	}
}

// Helpers that delegate to testify for convenience

// Require is an alias for require package methods.
type Require struct {
	t *testing.T
}

// NewRequire creates a new require helper that stops tests on failure.
func NewRequire(t *testing.T) *Require {
	return &Require{t: t}
}

// NoError is a require alias that stops the test on error.
func (r *Require) NoError(err error, msgAndArgs ...interface{}) {
	r.t.Helper()
	require.NoError(r.t, err, msgAndArgs...)
}

// Equals is a require alias that stops the test on equality failure.
func (r *Require) Equals(expected, actual interface{}, msgAndArgs ...interface{}) {
	r.t.Helper()
	require.Equal(r.t, expected, actual, msgAndArgs...)
}

// Nil is a require alias that stops the test if value is not nil.
func (r *Require) Nil(value interface{}, msgAndArgs ...interface{}) {
	r.t.Helper()
	require.Nil(r.t, value, msgAndArgs...)
}

// NotNil is a require alias that stops the test if value is nil.
func (r *Require) NotNil(value interface{}, msgAndArgs ...interface{}) {
	r.t.Helper()
	require.NotNil(r.t, value, msgAndArgs...)
}

// True is a require alias that stops the test if condition is false.
func (r *Require) True(condition bool, msgAndArgs ...interface{}) {
	r.t.Helper()
	require.True(r.t, condition, msgAndArgs...)
}

// Convenience functions that create assertion helpers automatically

func AssertNoError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	NewAssert(t).NoError(err, msgAndArgs...)
}

func AssertEquals(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	NewAssert(t).Equals(expected, actual, msgAndArgs...)
}

func AssertTrue(t *testing.T, condition bool, msgAndArgs ...interface{}) {
	t.Helper()
	NewAssert(t).True(condition, msgAndArgs...)
}

func AssertFalse(t *testing.T, condition bool, msgAndArgs ...interface{}) {
	t.Helper()
	NewAssert(t).False(condition, msgAndArgs...)
}

func AssertNil(t *testing.T, value interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	NewAssert(t).Nil(value, msgAndArgs...)
}

func AssertNotNil(t *testing.T, value interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	NewAssert(t).NotNil(value, msgAndArgs...)
}

func AssertContains(t *testing.T, s, substr string, msgAndArgs ...interface{}) {
	t.Helper()
	NewAssert(t).Contains(s, substr, msgAndArgs...)
}

func AssertLen(t *testing.T, obj interface{}, expectedLength int, msgAndArgs ...interface{}) {
	t.Helper()
	NewAssert(t).Len(obj, expectedLength, msgAndArgs...)
}
