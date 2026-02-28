// Package testutil tests for assertion utilities
package testutil

import (
	"errors"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAssert(t *testing.T) {
	a := NewAssert(t)
	assert.NotNil(t, a)
	assert.NotNil(t, a.t)
}

func TestAssert_NoError(t *testing.T) {
	t.Run("no error", func(t *testing.T) {
		a := NewAssert(t)
		a.NoError(nil) // Should not panic
	})

	t.Run("with error", func(t *testing.T) {
		a := NewAssert(t)
		err := errors.New("test error")

		// Should panic/fail
		assert.Panics(t, func() {
			a.NoError(err)
		})
	})
}

func TestAssert_Error(t *testing.T) {
	t.Run("with error", func(t *testing.T) {
		a := NewAssert(t)
		err := errors.New("test error")
		a.Error(err) // Should not panic
	})

	t.Run("no error", func(t *testing.T) {
		a := NewAssert(t)
		assert.Panics(t, func() {
			a.Error(nil)
		})
	})
}

func TestAssert_ErrorIs(t *testing.T) {
	t.Run("matching error", func(t *testing.T) {
		a := NewAssert(t)
		// Use the same error instance for matching test
		baseErr := errors.New("test")
		a.ErrorIs(baseErr, baseErr) // Same instance, should match
	})

	t.Run("not matching", func(t *testing.T) {
		a := NewAssert(t)
		err1 := errors.New("error 1")
		err2 := errors.New("error 2")

		assert.Panics(t, func() {
			a.ErrorIs(err1, err2)
		})
	})
}

func TestAssert_ErrorType(t *testing.T) {
	customErr := &customError{message: "custom"}

	t.Run("correct type", func(t *testing.T) {
		a := NewAssert(t)
		a.ErrorType(customErr, &customError{})
	})

	t.Run("wrong type", func(t *testing.T) {
		a := NewAssert(t)
		otherErr := errors.New("other")

		assert.Panics(t, func() {
			a.ErrorType(otherErr, &customError{})
		})
	})

	t.Run("nil error", func(t *testing.T) {
		a := NewAssert(t)
		assert.Panics(t, func() {
			a.ErrorType(nil, &customError{})
		})
	})
}

type customError struct {
	message string
}

func (e *customError) Error() string {
	return e.message
}

func TestAssert_Equals(t *testing.T) {
	a := NewAssert(t)

	t.Run("equal values", func(t *testing.T) {
		a.Equals("test", "test")
		a.Equals(42, 42)
		a.Equals(true, true)
	})

	t.Run("not equal", func(t *testing.T) {
		assert.Panics(t, func() {
			a.Equals("test1", "test2")
		})

		assert.Panics(t, func() {
			a.Equals(1, 2)
		})
	})
}

func TestAssert_NotEquals(t *testing.T) {
	a := NewAssert(t)

	t.Run("not equal", func(t *testing.T) {
		a.NotEquals("test1", "test2")
		a.NotEquals(1, 2)
	})

	t.Run("equal", func(t *testing.T) {
		assert.Panics(t, func() {
			a.NotEquals("test", "test")
		})
	})
}

func TestAssert_True(t *testing.T) {
	a := NewAssert(t)

	t.Run("true condition", func(t *testing.T) {
		a.True(true)
		a.True(1 == 1)
	})

	t.Run("false condition", func(t *testing.T) {
		assert.Panics(t, func() {
			a.True(false)
		})

		assert.Panics(t, func() {
			a.True(1 == 2)
		})
	})
}

func TestAssert_False(t *testing.T) {
	a := NewAssert(t)

	t.Run("false condition", func(t *testing.T) {
		a.False(false)
		a.False(1 == 2)
	})

	t.Run("true condition", func(t *testing.T) {
		assert.Panics(t, func() {
			a.False(true)
		})
	})
}

func TestAssert_Nil(t *testing.T) {
	a := NewAssert(t)

	t.Run("nil value", func(t *testing.T) {
		var ptr *string
		a.Nil(ptr)
		a.Nil(nil)
	})

	t.Run("non-nil", func(t *testing.T) {
		str := "test"
		assert.Panics(t, func() {
			a.Nil(str)
		})
	})
}

func TestAssert_NotNil(t *testing.T) {
	a := NewAssert(t)

	t.Run("non-nil value", func(t *testing.T) {
		str := "test"
		a.NotNil(str)

		ptr := &str
		a.NotNil(ptr)
	})

	t.Run("nil", func(t *testing.T) {
		var ptr *string
		assert.Panics(t, func() {
			a.NotNil(ptr)
		})
	})
}

func TestAssert_Contains(t *testing.T) {
	a := NewAssert(t)

	t.Run("contains substring", func(t *testing.T) {
		a.Contains("hello world", "lo wo")
		a.Contains("test", "es")
	})

	t.Run("does not contain", func(t *testing.T) {
		assert.Panics(t, func() {
			a.Contains("hello", "world")
		})
	})
}

func TestAssert_NotContains(t *testing.T) {
	a := NewAssert(t)

	t.Run("does not contain", func(t *testing.T) {
		a.NotContains("hello", "world")
	})

	t.Run("contains", func(t *testing.T) {
		assert.Panics(t, func() {
			a.NotContains("hello world", "lo wo")
		})
	})
}

func TestAssert_ContainsAll(t *testing.T) {
	a := NewAssert(t)

	t.Run("contains all", func(t *testing.T) {
		a.ContainsAll("hello world test", []string{"hello", "world", "test"})
	})

	t.Run("missing one", func(t *testing.T) {
		assert.Panics(t, func() {
			a.ContainsAll("hello world", []string{"hello", "missing"})
		})
	})
}

func TestAssert_Len(t *testing.T) {
	a := NewAssert(t)

	t.Run("correct length", func(t *testing.T) {
		a.Len("test", 4)
		a.Len([]int{1, 2, 3}, 3)
		a.Len(map[string]int{"a": 1}, 1)
	})

	t.Run("wrong length", func(t *testing.T) {
		assert.Panics(t, func() {
			a.Len("test", 5)
		})
	})

	t.Run("non-length type", func(t *testing.T) {
		assert.Panics(t, func() {
			a.Len(42, 2)
		})
	})
}

func TestAssert_Empty(t *testing.T) {
	a := NewAssert(t)

	a.Empty("")
	a.Empty([]int{})
	a.Empty(map[string]int{})
}

func TestAssert_NotEmpty(t *testing.T) {
	a := NewAssert(t)

	a.NotEmpty("test")
	a.NotEmpty([]int{1})
	a.NotEmpty(map[string]int{"a": 1})
}

func TestAssert_Greater(t *testing.T) {
	a := NewAssert(t)

	t.Run("greater", func(t *testing.T) {
		a.Greater(5, 3)
		a.Greater(100, 50)
	})

	t.Run("not greater", func(t *testing.T) {
		assert.Panics(t, func() {
			a.Greater(3, 5)
		})

		assert.Panics(t, func() {
			a.Greater(5, 5)
		})
	})

	t.Run("non-int values", func(t *testing.T) {
		assert.Panics(t, func() {
			a.Greater("a", "b")
		})
	})
}

func TestAssert_Less(t *testing.T) {
	a := NewAssert(t)

	t.Run("less", func(t *testing.T) {
		a.Less(3, 5)
		a.Less(50, 100)
	})

	t.Run("not less", func(t *testing.T) {
		assert.Panics(t, func() {
			a.Less(5, 3)
		})

		assert.Panics(t, func() {
			a.Less(5, 5)
		})
	})
}

func TestAssert_WithinDuration(t *testing.T) {
	a := NewAssert(t)

	t.Run("within duration", func(t *testing.T) {
		now := time.Now()
		a.WithinDuration(now, now.Add(10*time.Millisecond), 100*time.Millisecond)
	})

	t.Run("outside duration", func(t *testing.T) {
		now := time.Now()
		assert.Panics(t, func() {
			a.WithinDuration(now, now.Add(200*time.Millisecond), 100*time.Millisecond)
		})
	})
}

func TestAssert_Panics(t *testing.T) {
	a := NewAssert(t)

	t.Run("function panics", func(t *testing.T) {
		a.Panics(func() {
			panic("test panic")
		})
	})

	t.Run("function does not panic", func(t *testing.T) {
		assert.Panics(t, func() {
			a.Panics(func() {
				// No panic
			})
		})
	})
}

func TestAssert_NotPanics(t *testing.T) {
	a := NewAssert(t)

	t.Run("function does not panic", func(t *testing.T) {
		a.NotPanics(func() {
			// No panic
		})
	})

	t.Run("function panics", func(t *testing.T) {
		assert.Panics(t, func() {
			a.NotPanics(func() {
				panic("test")
			})
		})
	})
}

func TestAssert_Eventually(t *testing.T) {
	a := NewAssert(t)

	t.Run("condition becomes true", func(t *testing.T) {
		count := 0
		a.Eventually(func() bool {
			count++
			return count >= 3
		}, 200*time.Millisecond)
	})

	t.Run("condition never true", func(t *testing.T) {
		assert.Panics(t, func() {
			a.Eventually(func() bool {
				return false
			}, 50*time.Millisecond)
		})
	})
}

func TestAssert_Consistently(t *testing.T) {
	a := NewAssert(t)

	t.Run("condition stays true", func(t *testing.T) {
		a.Consistently(func() bool {
			return true
		}, 100*time.Millisecond)
	})

	t.Run("condition becomes false", func(t *testing.T) {
		count := 0
		assert.Panics(t, func() {
			a.Consistently(func() bool {
				count++
				return count < 3
			}, 200*time.Millisecond)
		})
	})
}

func TestAssert_FileExists(t *testing.T) {
	a := NewAssert(t)

	t.Run("file exists", func(t *testing.T) {
		// Create a temp file
		tmpfile, err := os.CreateTemp("", "test")
		require.NoError(t, err)
		defer os.Remove(tmpfile.Name())

		a.FileExists(tmpfile.Name())
	})

	t.Run("file does not exist", func(t *testing.T) {
		assert.Panics(t, func() {
			a.FileExists("/nonexistent/file.txt")
		})
	})

	t.Run("directory instead of file", func(t *testing.T) {
		tmpdir, err := os.MkdirTemp("", "testdir")
		require.NoError(t, err)
		defer os.RemoveAll(tmpdir)

		assert.Panics(t, func() {
			a.FileExists(tmpdir)
		})
	})
}

func TestAssert_DirExists(t *testing.T) {
	a := NewAssert(t)

	t.Run("directory exists", func(t *testing.T) {
		tmpdir, err := os.MkdirTemp("", "testdir")
		require.NoError(t, err)
		defer os.RemoveAll(tmpdir)

		a.DirExists(tmpdir)
	})

	t.Run("directory does not exist", func(t *testing.T) {
		assert.Panics(t, func() {
			a.DirExists("/nonexistent/directory")
		})
	})

	t.Run("file instead of directory", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "test")
		require.NoError(t, err)
		defer os.Remove(tmpfile.Name())

		assert.Panics(t, func() {
			a.DirExists(tmpfile.Name())
		})
	})
}

func TestAssert_FileContains(t *testing.T) {
	a := NewAssert(t)

	t.Run("file contains substring", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "test")
		require.NoError(t, err)
		defer os.Remove(tmpfile.Name())

		content := "hello world test content"
		tmpfile.WriteString(content)
		tmpfile.Close()

		a.FileContains(tmpfile.Name(), "world")
	})

	t.Run("file does not contain substring", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "test")
		require.NoError(t, err)
		defer os.Remove(tmpfile.Name())

		tmpfile.WriteString("hello world")
		tmpfile.Close()

		assert.Panics(t, func() {
			a.FileContains(tmpfile.Name(), "missing")
		})
	})
}

func TestAssert_JSONEq(t *testing.T) {
	a := NewAssert(t)

	t.Run("equal JSON", func(t *testing.T) {
		a.JSONEq(`{"key":"value"}`, `{"key":"value"}`)
	})

	t.Run("different JSON", func(t *testing.T) {
		assert.Panics(t, func() {
			a.JSONEq(`{"key":"value1"}`, `{"key":"value2"}`)
		})
	})
}

func TestAssert_HTTPSuccess(t *testing.T) {
	a := NewAssert(t)

	t.Run("2xx status", func(t *testing.T) {
		resp := &mockHTTPResponse{statusCode: 200}
		a.HTTPSuccess(resp)

		resp = &mockHTTPResponse{statusCode: 201}
		a.HTTPSuccess(resp)

		resp = &mockHTTPResponse{statusCode: 299}
		a.HTTPSuccess(resp)
	})

	t.Run("non-2xx status", func(t *testing.T) {
		resp := &mockHTTPResponse{statusCode: 300}
		assert.Panics(t, func() {
			a.HTTPSuccess(resp)
		})

		resp = &mockHTTPResponse{statusCode: 404}
		assert.Panics(t, func() {
			a.HTTPSuccess(resp)
		})

		resp = &mockHTTPResponse{statusCode: 500}
		assert.Panics(t, func() {
			a.HTTPSuccess(resp)
		})
	})
}

func TestAssert_HTTPStatus(t *testing.T) {
	a := NewAssert(t)

	t.Run("matching status", func(t *testing.T) {
		resp := &mockHTTPResponse{statusCode: 200}
		a.HTTPStatus(resp, 200)

		resp = &mockHTTPResponse{statusCode: 404}
		a.HTTPStatus(resp, 404)
	})

	t.Run("non-matching status", func(t *testing.T) {
		resp := &mockHTTPResponse{statusCode: 200}
		assert.Panics(t, func() {
			a.HTTPStatus(resp, 404)
		})
	})
}

func TestAssert_HTTPHeader(t *testing.T) {
	a := NewAssert(t)

	t.Run("matching header", func(t *testing.T) {
		resp := &mockHTTPResponse{
			headers: http.Header{"Content-Type": []string{"application/json"}},
		}
		a.HTTPHeader(resp, "Content-Type", "application/json")
	})

	t.Run("non-matching header", func(t *testing.T) {
		resp := &mockHTTPResponse{
			headers: http.Header{"Content-Type": []string{"application/json"}},
		}
		assert.Panics(t, func() {
			a.HTTPHeader(resp, "Content-Type", "text/html")
		})
	})
}

func TestAssert_HTTPBodyContains(t *testing.T) {
	a := NewAssert(t)

	t.Run("body contains substring", func(t *testing.T) {
		resp := &mockHTTPResponse{body: []byte("hello world")}
		a.HTTPBodyContains(resp, "world")
	})

	t.Run("body does not contain", func(t *testing.T) {
		resp := &mockHTTPResponse{body: []byte("hello world")}
		assert.Panics(t, func() {
			a.HTTPBodyContains(resp, "missing")
		})
	})
}

func TestRetryable_EventuallyTrue(t *testing.T) {
	a := NewAssert(t)
	r := NewRetryable(a, 5, 10*time.Millisecond)

	t.Run("condition becomes true", func(t *testing.T) {
		count := 0
		r.EventuallyTrue(func() bool {
			count++
			return count >= 3
		})
	})

	t.Run("condition never true", func(t *testing.T) {
		assert.Panics(t, func() {
			r.EventuallyTrue(func() bool {
				return false
			})
		})
	})
}

func TestRetryable_EventuallyNoError(t *testing.T) {
	a := NewAssert(t)
	r := NewRetryable(a, 5, 10*time.Millisecond)

	t.Run("no error eventually", func(t *testing.T) {
		count := 0
		r.EventuallyNoError(func() error {
			count++
			if count < 3 {
				return errors.New("not yet")
			}
			return nil
		})
	})

	t.Run("always error", func(t *testing.T) {
		assert.Panics(t, func() {
			r.EventuallyNoError(func() error {
				return errors.New("always fails")
			})
		})
	})
}

func TestAssert_AllSatisfy(t *testing.T) {
	a := NewAssert(t)

	t.Run("all satisfy", func(t *testing.T) {
		a.AllSatisfy([]int{2, 4, 6, 8}, func(v interface{}) bool {
			return v.(int)%2 == 0
		})
	})

	t.Run("one does not satisfy", func(t *testing.T) {
		assert.Panics(t, func() {
			a.AllSatisfy([]int{2, 4, 5, 8}, func(v interface{}) bool {
				return v.(int)%2 == 0
			})
		})
	})

	t.Run("non-slice", func(t *testing.T) {
		assert.Panics(t, func() {
			a.AllSatisfy("not a slice", func(v interface{}) bool {
				return true
			})
		})
	})
}

func TestAssert_NoneSatisfy(t *testing.T) {
	a := NewAssert(t)

	t.Run("none satisfy", func(t *testing.T) {
		a.NoneSatisfy([]int{1, 3, 5, 7}, func(v interface{}) bool {
			return v.(int)%2 == 0
		})
	})

	t.Run("one satisfies", func(t *testing.T) {
		assert.Panics(t, func() {
			a.NoneSatisfy([]int{2, 3, 5, 7}, func(v interface{}) bool {
				return v.(int)%2 == 0
			})
		})
	})
}

func TestAssert_ContainsExactly(t *testing.T) {
	a := NewAssert(t)

	t.Run("exactly same elements", func(t *testing.T) {
		a.ContainsExactly([]int{1, 2, 3}, []int{1, 2, 3})
	})

	t.Run("different order", func(t *testing.T) {
		a.ContainsExactly([]int{1, 2, 3}, []int{3, 2, 1})
	})

	t.Run("different length", func(t *testing.T) {
		assert.Panics(t, func() {
			a.ContainsExactly([]int{1, 2}, []int{1, 2, 3})
		})
	})

	t.Run("missing element", func(t *testing.T) {
		assert.Panics(t, func() {
			a.ContainsExactly([]int{1, 2, 3}, []int{1, 2, 4})
		})
	})
}

func TestAssert_KeyExists(t *testing.T) {
	a := NewAssert(t)

	t.Run("key exists", func(t *testing.T) {
		m := map[string]int{"a": 1, "b": 2}
		a.KeyExists(m, "a")
	})

	t.Run("key does not exist", func(t *testing.T) {
		m := map[string]int{"a": 1, "b": 2}
		assert.Panics(t, func() {
			a.KeyExists(m, "c")
		})
	})

	t.Run("not a map", func(t *testing.T) {
		assert.Panics(t, func() {
			a.KeyExists("not a map", "key")
		})
	})
}

func TestAssert_KeyNotExists(t *testing.T) {
	a := NewAssert(t)

	t.Run("key does not exist", func(t *testing.T) {
		m := map[string]int{"a": 1, "b": 2}
		a.KeyNotExists(m, "c")
	})

	t.Run("key exists", func(t *testing.T) {
		m := map[string]int{"a": 1, "b": 2}
		assert.Panics(t, func() {
			a.KeyNotExists(m, "a")
		})
	})
}

func TestAssert_MapLen(t *testing.T) {
	a := NewAssert(t)

	t.Run("correct length", func(t *testing.T) {
		m := map[string]int{"a": 1, "b": 2, "c": 3}
		a.MapLen(m, 3)
	})

	t.Run("wrong length", func(t *testing.T) {
		m := map[string]int{"a": 1, "b": 2}
		assert.Panics(t, func() {
			a.MapLen(m, 3)
		})
	})

	t.Run("not a map", func(t *testing.T) {
		assert.Panics(t, func() {
			a.MapLen("not a map", 2)
		})
	})
}

func TestNewRequire(t *testing.T) {
	r := NewRequire(t)
	assert.NotNil(t, r)
	assert.NotNil(t, r.t)
}

func TestRequire_NoError(t *testing.T) {
	r := NewRequire(t)

	t.Run("no error", func(t *testing.T) {
		// Should not panic
		assert.NotPanics(t, func() {
			r.NoError(nil)
		})
	})

	t.Run("with error - delegated to testify", func(t *testing.T) {
		// Note: require.NoError calls t.Fatal which uses runtime.Goexit
		// This cannot be caught with recover/panic, so we skip this test
		// The behavior is tested by the testify package itself
		t.Skip("Goexit cannot be caught")
	})
}

func TestRequire_Equals(t *testing.T) {
	r := NewRequire(t)

	t.Run("equal", func(t *testing.T) {
		assert.NotPanics(t, func() {
			r.Equals("test", "test")
		})
	})

	t.Run("not equal - delegated to testify", func(t *testing.T) {
		// Note: require.Equal calls t.Fatal which uses runtime.Goexit
		// This cannot be caught with recover/panic, so we skip this test
		// The behavior is tested by the testify package itself
		t.Skip("Goexit cannot be caught")
	})
}

func TestConvenienceFunctions(t *testing.T) {
	t.Run("AssertNoError", func(t *testing.T) {
		AssertNoError(t, nil)
	})

	t.Run("AssertEquals", func(t *testing.T) {
		AssertEquals(t, "test", "test")
	})

	t.Run("AssertTrue", func(t *testing.T) {
		AssertTrue(t, true)
	})

	t.Run("AssertFalse", func(t *testing.T) {
		AssertFalse(t, false)
	})

	t.Run("AssertNil", func(t *testing.T) {
		AssertNil(t, nil)
	})

	t.Run("AssertNotNil", func(t *testing.T) {
		AssertNotNil(t, "value")
	})

	t.Run("AssertContains", func(t *testing.T) {
		AssertContains(t, "hello world", "world")
	})

	t.Run("AssertLen", func(t *testing.T) {
		AssertLen(t, "test", 4)
	})
}

// Mock implementations for testing

type mockHTTPResponse struct {
	statusCode int
	headers    http.Header
	body       []byte
}

func (m *mockHTTPResponse) StatusCode() int {
	return m.statusCode
}

func (m *mockHTTPResponse) Header() http.Header {
	if m.headers == nil {
		return http.Header{}
	}
	return m.headers
}

func (m *mockHTTPResponse) Body() []byte {
	return m.body
}

func TestAssert_WithHelper(t *testing.T) {
	// Test that t.Helper() is called
	a := NewAssert(t)
	// This should not cause issues
	a.True(true)
}

func TestAssert_MultipleAssertions(t *testing.T) {
	a := NewAssert(t)

	// Multiple assertions in sequence
	a.Equals(1, 1)
	a.True(true)
	a.NotNil("value")
	a.Contains("hello", "ell")
	a.Len("test", 4)
}

func TestAssert_ComplexTypes(t *testing.T) {
	a := NewAssert(t)

	// Test with complex types
	type Person struct {
		Name string
		Age  int
	}

	p1 := Person{Name: "Alice", Age: 30}
	p2 := Person{Name: "Alice", Age: 30}
	p3 := Person{Name: "Bob", Age: 25}

	a.Equals(p1, p2)

	assert.Panics(t, func() {
		a.Equals(p1, p3)
	})
}

func TestAssert_Slices(t *testing.T) {
	a := NewAssert(t)

	slice1 := []int{1, 2, 3}
	slice2 := []int{1, 2, 3}
	slice3 := []int{1, 2, 4}

	a.Equals(slice1, slice2)

	assert.Panics(t, func() {
		a.Equals(slice1, slice3)
	})
}

func TestAssert_Maps(t *testing.T) {
	a := NewAssert(t)

	map1 := map[string]int{"a": 1, "b": 2}
	map2 := map[string]int{"a": 1, "b": 2}
	map3 := map[string]int{"a": 1, "b": 3}

	a.Equals(map1, map2)

	assert.Panics(t, func() {
		a.Equals(map1, map3)
	})
}

func TestAssert_Pointers(t *testing.T) {
	a := NewAssert(t)

	value := 42
	ptr1 := &value
	ptr2 := &value

	a.Equals(ptr1, ptr2)

	var nilPtr *int
	a.Nil(nilPtr)
	a.NotNil(ptr1)
}

func TestAssert_TimeComparisons(t *testing.T) {
	a := NewAssert(t)

	now := time.Now()
	earlier := now.Add(-1 * time.Hour)
	later := now.Add(1 * time.Hour)

	a.True(earlier.Before(now))
	a.True(later.After(now))
	a.True(earlier.Before(later))
}

func TestAssert_Strings(t *testing.T) {
	a := NewAssert(t)

	a.Contains("Hello World", "Hello")
	a.NotContains("Hello World", "Goodbye")
	a.ContainsAll("Hello World Test", []string{"Hello", "World", "Test"})
}

func TestAssert_NestedStructs(t *testing.T) {
	a := NewAssert(t)

	type Address struct {
		City  string
		State string
	}

	type Person struct {
		Name    string
		Age     int
		Address Address
	}

	p1 := Person{
		Name: "Alice",
		Age:  30,
		Address: Address{
			City:  "Boston",
			State: "MA",
		},
	}

	p2 := Person{
		Name: "Alice",
		Age:  30,
		Address: Address{
			City:  "Boston",
			State: "MA",
		},
	}

	a.Equals(p1, p2)
}

func TestAssert_Bytes(t *testing.T) {
	a := NewAssert(t)

	bytes1 := []byte{0x01, 0x02, 0x03}
	bytes2 := []byte{0x01, 0x02, 0x03}
	bytes3 := []byte{0x01, 0x02, 0x04}

	a.Equals(bytes1, bytes2)

	assert.Panics(t, func() {
		a.Equals(bytes1, bytes3)
	})
}
