package goroutines

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"testing"
	"time"
)

var (
	testErr = errors.New("Generic test error")

	testStrings = []string{
		"a", "be", "cee", "deep", "geee", "geeee", "geeeee", "geeeeee", "geeeeeee",
		"geeeeeeee", "geeeeeeeee", "geeeeeeeeee", "geeeeeeeeeee", "geeeeeeeeeeee", "geeeeeeeeeeeee",
		"a", "be", "cee", "deep", "geee", "geeee", "geeeee", "geeeeee", "geeeeeee",
		"geeeeeeee", "geeeeeeeee", "geeeeeeeeee", "geeeeeeeeeee", "geeeeeeeeeeee", "geeeeeeeeeeeee"}

	testInts = []int{
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18,
		19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36,
		37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54,
		55, 56, 57, 58, 59, 60,
	}
)

func TestSearch(t *testing.T) {
	sleepTime := 100 * time.Millisecond
	tests := []struct {
		name       string
		fn         func(context.Context)
		cancelTime time.Duration
		timeLimit  time.Duration
	}{
		{
			name: "basic unordered search",
			fn: func(_ context.Context) {
				v, err := SearchUnordered(5, func(s string) (int, error) {
					if len(s)%2 == 1 {
						time.Sleep(sleepTime)
					}
					if len(s) == 5 {
						return len(s) + 100, ErrSearchSuccess
					}
					return 0, nil
				}, testStrings)

				if v != 105 {
					t.Errorf("Expected search to return result=%v but received result=%v", 105, v)
				}
				if err != nil {
					t.Errorf("Expected error=%v but received error=%v", nil, err)
				}
			},
		},
		{
			name: "basic ordered search",
			fn: func(_ context.Context) {
				v, err := SearchUnordered(5, func(s string) (int, error) {
					if len(s)%2 == 1 {
						time.Sleep(sleepTime)
					}
					if len(s) == 5 {
						return len(s) + 100, ErrSearchSuccess
					}
					return 0, nil
				}, testStrings)

				if v != 105 {
					t.Errorf("Expected search to return result=%v but received result=%v", 105, v)
				}
				if err != nil {
					t.Errorf("Expected error=%v but received error=%v", nil, err)
				}
			},
		},
		{
			name: "basic unordered search failure",
			fn: func(_ context.Context) {
				_, err := SearchUnordered(5, func(s string) (int, error) {
					if len(s)%2 == 1 {
						time.Sleep(sleepTime)
					}
					return 0, nil
				}, testStrings)

				if err != ErrSearchFailure {
					t.Errorf("Expected error=%v but received error=%v", ErrSearchFailure, err)
				}
			},
		},
		{
			name: "basic ordered search failure",
			fn: func(_ context.Context) {
				_, err := Search(5, func(s string) (int, error) {
					if len(s)%2 == 1 {
						time.Sleep(sleepTime)
					}
					return 0, nil
				}, testStrings)

				if err != ErrSearchFailure {
					t.Errorf("Expected error=%v but received error=%v", ErrSearchFailure, err)
				}
			},
		},
		{
			name: "basic ordered search with error",
			fn: func(_ context.Context) {
				v, err := Search(5, func(s string) (int, error) {
					if len(s)%2 == 1 {
						time.Sleep(sleepTime)
					}
					if len(s) == 14 {
						return len(s), testErr
					}
					return len(s), nil
				}, testStrings)

				if v != 14 {
					t.Errorf("Expected search to return result=%v but received result=%v", 14, v)
				}
				if err != testErr {
					t.Errorf("Expected error=%v but received error=%v", testErr, err)
				}
			},
		},
		{
			name: "basic unordered search with error",
			fn: func(_ context.Context) {
				v, err := SearchUnordered(5, func(s string) (int, error) {
					if len(s)%2 == 1 {
						time.Sleep(sleepTime)
					}
					if len(s) == 14 {
						return len(s), testErr
					}
					return len(s), nil
				}, testStrings)

				if v != 14 {
					t.Errorf("Expected search to return result=%v but received result=%v", 14, v)
				}
				if err != testErr {
					t.Errorf("Expected error=%v but received error=%v", testErr, err)
				}
			},
		},
		{
			name: "basic ordered search with context cancellation",
			fn: func(ctx context.Context) {
				_, err := SearchWithContext(ctx, 5, func(s string) (int, error) {
					if len(s)%2 == 1 {
						time.Sleep(sleepTime)
					}
					if len(s) == 14 {
						return len(s), testErr
					}
					return len(s), nil
				}, testStrings)

				if err != context.Canceled {
					t.Errorf("Expected error=\"%v\" but received error=\"%v\"", context.Canceled, err)
				}
			},
			cancelTime: 50 * time.Millisecond,
		},
		{
			name: "basic unordered search with context cancellation",
			fn: func(ctx context.Context) {
				_, err := SearchUnorderedWithContext(ctx, 5, func(s string) (int, error) {
					if len(s)%2 == 1 {
						time.Sleep(sleepTime)
					}
					if len(s) == 14 {
						return len(s), testErr
					}
					return len(s), nil
				}, testStrings)

				if err != context.Canceled {
					t.Errorf("Expected error=\"%v\" but received error=\"%v\"", context.Canceled, err)
				}
			},
			cancelTime: 50 * time.Millisecond,
		},
		{
			name: "basic ordered search with error and context cancellation",
			fn: func(ctx context.Context) {
				_, err := SearchWithContext(ctx, 5, func(s string) (int, error) {
					if len(s)%2 == 1 {
						time.Sleep(sleepTime)
					}
					if len(s) == 14 {
						return len(s), testErr
					}
					return len(s), nil
				}, testStrings)

				if err != testErr {
					t.Errorf("Expected error=\"%v\" but received error=\"%v\"", context.Canceled, err)
				}
			},
			cancelTime: 500 * time.Millisecond,
		},
		{
			name: "basic unordered search with error and context cancellation",
			fn: func(ctx context.Context) {
				_, err := SearchUnorderedWithContext(ctx, 5, func(s string) (int, error) {
					if len(s)%2 == 1 {
						time.Sleep(sleepTime)
					}
					if len(s) == 14 {
						return len(s), testErr
					}
					return len(s), nil
				}, testStrings)

				if err != testErr {
					t.Errorf("Expected error=\"%v\" but received error=\"%v\"", context.Canceled, err)
				}
			},
			cancelTime: 500 * time.Millisecond,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.cancelTime > 0 {
				var cancel func()
				ctx, cancel = context.WithCancel(context.Background())
				time.AfterFunc(tt.cancelTime, func() {
					cancel()
				})
			}
			tt.fn(ctx)
		})
	}
}

func TestInject(t *testing.T) {
	sleepTime := 100 * time.Millisecond
	tests := []struct {
		name       string
		fn         func(context.Context)
		cancelTime time.Duration
		timeLimit  time.Duration
	}{
		{
			name: "basic convert int to string",
			fn: func(_ context.Context) {
				v, err := Inject(5, []string{}, func(n int) (int, error) {
					if n%2 == 1 {
						time.Sleep(sleepTime)
					}
					return n + 10, nil
				}, func(a []string, b int) ([]string, error) {
					return append(a, strconv.Itoa(b)), nil
				}, testInts)

				if len(v) != len(testInts) {
					t.Fatalf("Expected result slice len=%v but received len=%v", len(testInts), len(v))
				}
				for i, l := range v {
					if l != strconv.Itoa(testInts[i]+10) {
						t.Errorf("Expected search to return result=%v but received result=%v", 105, v)
					}
				}
				if err != nil {
					t.Errorf("Expected error=%v but received error=%v", nil, err)
				}
			},
		},
		{
			name: "basic unordered convert int to string",
			fn: func(_ context.Context) {
				v, err := InjectUnordered(5, []string{}, func(n int) (int, error) {
					if n%2 == 1 {
						time.Sleep(sleepTime)
					}
					return n + 10, nil
				}, func(a []string, b int) ([]string, error) {
					return append(a, strconv.Itoa(b)), nil
				}, testInts)

				if len(v) != len(testInts) {
					t.Fatalf("Expected result slice len=%v but received len=%v", len(testInts), len(v))
				}
				sort.Strings(v)
				for i, l := range v {
					if l != strconv.Itoa(testInts[i]+10) {
						t.Errorf("Expected search to return result=%v but received result=%v", 105, v)
					}
				}
				if err != nil {
					t.Errorf("Expected error=%v but received error=%v", nil, err)
				}
			},
		},
		{
			name: "error during mapping",
			fn: func(_ context.Context) {
				_, err := Inject(5, []string{}, func(n int) (int, error) {
					if n%2 == 1 {
						time.Sleep(sleepTime)
					}
					if n%15 == 0 {
						return n, testErr
					}
					return n + 10, nil
				}, func(a []string, b int) ([]string, error) {
					return append(a, strconv.Itoa(b)), nil
				}, testInts)

				if err != testErr {
					t.Errorf("Expected error=%v but received error=%v", testErr, err)
				}
			},
		},
		{
			name: "unordered error during mapping",
			fn: func(_ context.Context) {
				_, err := InjectUnordered(5, []string{}, func(n int) (int, error) {
					if n%2 == 1 {
						time.Sleep(sleepTime)
					}
					if n%15 == 0 {
						return n, testErr
					}
					return n + 10, nil
				}, func(a []string, b int) ([]string, error) {
					return append(a, strconv.Itoa(b)), nil
				}, testInts)

				if err != testErr {
					t.Errorf("Expected error=%v but received error=%v", testErr, err)
				}
			},
		},
		{
			name: "error during fold",
			fn: func(_ context.Context) {
				v, err := Inject(5, []string{}, func(n int) (int, error) {
					if n%2 == 1 {
						time.Sleep(sleepTime)
					}
					return n + 10, nil
				}, func(a []string, b int) ([]string, error) {
					if b%25 == 0 {
						return a, testErr
					}
					return append(a, strconv.Itoa(b)), nil
				}, testInts)

				if len(v) != 14 {
					t.Fatalf("Expected result slice len=%v but received len=%v", len(testInts), 14)
				}
				for i, l := range v {
					if l != strconv.Itoa(testInts[i]+10) {
						t.Errorf("Expected search to return result=%v but received result=%v", 105, v)
					}
				}
				if err != testErr {
					t.Errorf("Expected error=%v but received error=%v", testErr, err)
				}
			},
		},
		{
			name: "unordered error during fold",
			fn: func(_ context.Context) {
				_, err := InjectUnordered(5, []string{}, func(n int) (int, error) {
					if n%2 == 1 {
						time.Sleep(sleepTime)
					}
					return n + 10, nil
				}, func(a []string, b int) ([]string, error) {
					if b%25 == 0 {
						return a, testErr
					}
					return append(a, strconv.Itoa(b)), nil
				}, testInts)

				if err != testErr {
					t.Errorf("Expected error=%v but received error=%v", testErr, err)
				}
			},
		},
		{
			name: "with context",
			fn: func(ctx context.Context) {
				_, err := InjectWithContext(ctx, 5, []string{}, func(n int) (int, error) {
					if n%2 == 1 {
						time.Sleep(sleepTime)
					}
					return n + 10, nil
				}, func(a []string, b int) ([]string, error) {
					if b%25 == 0 {
						return a, testErr
					}
					return append(a, strconv.Itoa(b)), nil
				}, testInts)

				if err != context.Canceled {
					t.Errorf("Expected error=%v but received error=%v", context.Canceled, err)
				}
			},
			cancelTime: 50 * time.Millisecond,
		},
		{
			name: "unordered with context",
			fn: func(ctx context.Context) {
				_, err := InjectUnorderedWithContext(ctx, 5, []string{}, func(n int) (int, error) {
					if n%2 == 1 {
						time.Sleep(sleepTime)
					}
					return n + 10, nil
				}, func(a []string, b int) ([]string, error) {
					if b%25 == 0 {
						return a, testErr
					}
					return append(a, strconv.Itoa(b)), nil
				}, testInts)

				if err != context.Canceled {
					t.Errorf("Expected error=%v but received error=%v", context.Canceled, err)
				}
			},
			cancelTime: 50 * time.Millisecond,
		},
		{
			name: "with error and context",
			fn: func(ctx context.Context) {
				_, err := InjectWithContext(ctx, 5, []string{}, func(n int) (int, error) {
					if n%2 == 1 {
						time.Sleep(sleepTime)
					}
					return n + 10, nil
				}, func(a []string, b int) ([]string, error) {
					if b%25 == 0 {
						return a, testErr
					}
					return append(a, strconv.Itoa(b)), nil
				}, testInts)

				if err != testErr {
					t.Errorf("Expected error=%v but received error=%v", testErr, err)
				}
			},
			cancelTime: 500 * time.Millisecond,
		},
		{
			name: "unordered with error and context",
			fn: func(ctx context.Context) {
				_, err := InjectUnorderedWithContext(ctx, 5, []string{}, func(n int) (int, error) {
					if n%2 == 1 {
						time.Sleep(sleepTime)
					}
					return n + 10, nil
				}, func(a []string, b int) ([]string, error) {
					if b%25 == 0 {
						return a, testErr
					}
					return append(a, strconv.Itoa(b)), nil
				}, testInts)

				if err != testErr {
					t.Errorf("Expected error=%v but received error=%v", testErr, err)
				}
			},
			cancelTime: 500 * time.Millisecond,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.cancelTime > 0 {
				var cancel func()
				ctx, cancel = context.WithCancel(context.Background())
				time.AfterFunc(tt.cancelTime, func() {
					cancel()
				})
			}
			tt.fn(ctx)
		})
	}
}

func TestMap(t *testing.T) {
	sleepTime := 100 * time.Millisecond
	tests := []struct {
		name       string
		fn         func(context.Context)
		cancelTime time.Duration
		timeLimit  time.Duration
	}{
		{
			name: "basic convert int to string",
			fn: func(_ context.Context) {
				results := Map(5, func(n int) string {
					if n%2 == 1 {
						time.Sleep(sleepTime)
					}
					return strconv.Itoa(n + 10)
				}, testInts)

				v := make([]string, 0, len(testInts))
				for r := range results {
					v = append(v, r)
				}

				if len(v) != len(testInts) {
					t.Fatalf("Expected result slice len=%v but received len=%v", len(testInts), len(v))
				}
				for i, l := range v {
					if l != strconv.Itoa(testInts[i]+10) {
						t.Errorf("Expected search to return result=%v but received result=%v", testInts[i]+10, v)
					}
				}
			},
		},
		{
			name: "basic unordered convert int to string",
			fn: func(_ context.Context) {
				results := MapUnordered(5, func(n int) string {
					if n%2 == 1 {
						time.Sleep(sleepTime)
					}
					return strconv.Itoa(n + 10)
				}, testInts)

				v := make([]string, 0, len(testInts))
				for r := range results {
					v = append(v, r)
				}
				sort.Strings(v)

				if len(v) != len(testInts) {
					t.Fatalf("Expected result slice len=%v but received len=%v", len(testInts), len(v))
				}
				for i, l := range v {
					if l != strconv.Itoa(testInts[i]+10) {
						t.Errorf("Expected search to return result=%v but received result=%v", testInts[i]+10, v)
					}
				}
			},
		},
		{
			name: "basic convert int to string with context cancel",
			fn: func(ctx context.Context) {
				results := MapWithContext(ctx, 5, func(n int) string {
					if n%2 == 1 {
						time.Sleep(sleepTime)
					}
					return strconv.Itoa(n + 10)
				}, testInts)

				v := make([]string, 0, len(testInts))
				for r := range results {
					v = append(v, r)
				}

				if len(v) >= len(testInts) {
					t.Fatalf("Expected result slice len=%v > len=%v", len(testInts), len(v))
				}
				for i, l := range v {
					if l != strconv.Itoa(testInts[i]+10) {
						t.Errorf("Expected search to return result=%v but received result=%v", testInts[i]+10, v)
					}
				}
			},
			cancelTime: sleepTime + (50 * time.Millisecond),
		},
		{
			name: "basic unordered convert int to string  with context cancel",
			fn: func(ctx context.Context) {
				results := MapUnorderedWithContext(ctx, 5, func(n int) string {
					if n%2 == 1 {
						time.Sleep(sleepTime)
					}
					return strconv.Itoa(n + 10)
				}, testInts)

				v := make([]string, 0, len(testInts))
				for r := range results {
					v = append(v, r)
				}
				sort.Strings(v)

				if len(v) >= len(testInts) {
					t.Fatalf("Expected result slice len=%v > len=%v", len(testInts), len(v))
				}
			},
			cancelTime: sleepTime + (50 * time.Millisecond),
		},
		{
			name: "huge pool convert int to string",
			fn: func(_ context.Context) {
				results := Map(len(testInts)+20, func(n int) string {
					if n%2 == 1 {
						time.Sleep(sleepTime)
					}
					return strconv.Itoa(n + 10)
				}, testInts)

				v := make([]string, 0, len(testInts))
				for r := range results {
					v = append(v, r)
				}

				if len(v) != len(testInts) {
					t.Fatalf("Expected result slice len=%v but received len=%v", len(testInts), len(v))
				}
				for i, l := range v {
					if l != strconv.Itoa(testInts[i]+10) {
						t.Errorf("Expected search to return result=%v but received result=%v", testInts[i]+10, v)
					}
				}
			},
		},
		{
			name: "huge pool unordered convert int to string",
			fn: func(_ context.Context) {
				results := MapUnordered(len(testInts)+20, func(n int) string {
					if n%2 == 1 {
						time.Sleep(sleepTime)
					}
					return strconv.Itoa(n + 10)
				}, testInts)

				v := make([]string, 0, len(testInts))
				for r := range results {
					v = append(v, r)
				}
				sort.Strings(v)

				if len(v) != len(testInts) {
					t.Fatalf("Expected result slice len=%v but received len=%v", len(testInts), len(v))
				}
				for i, l := range v {
					if l != strconv.Itoa(testInts[i]+10) {
						t.Errorf("Expected search to return result=%v but received result=%v", testInts[i]+10, v)
					}
				}
			},
		},
		{
			name: "huge pool convert int to string with context cancel",
			fn: func(ctx context.Context) {
				results := MapWithContext(ctx, len(testInts)+20, func(n int) string {
					if n%2 == 1 {
						time.Sleep(sleepTime)
					}
					return strconv.Itoa(n + 10)
				}, testInts)

				v := make([]string, 0, len(testInts))
				for r := range results {
					v = append(v, r)
				}

				if len(v) >= len(testInts) {
					t.Fatalf("Expected result slice len=%v > len=%v", len(testInts), len(v))
				}
				for i, l := range v {
					if l != strconv.Itoa(testInts[i]+10) {
						t.Errorf("Expected search to return result=%v but received result=%v", testInts[i]+10, v)
					}
				}
			},
			cancelTime: 50 * time.Millisecond,
		},
		{
			name: "huge pool unordered convert int to string with context cancel",
			fn: func(ctx context.Context) {
				results := MapUnorderedWithContext(ctx, len(testInts)+20, func(n int) string {
					if n%2 == 1 {
						time.Sleep(sleepTime)
					}
					return strconv.Itoa(n + 10)
				}, testInts)

				v := make([]string, 0, len(testInts))
				for r := range results {
					v = append(v, r)
				}
				sort.Strings(v)

				if len(v) >= len(testInts) {
					t.Fatalf("Expected result slice len=%v > len=%v", len(testInts), len(v))
				}
			},
			cancelTime: 50 * time.Millisecond,
		},
		{
			name: "negative pool convert int to string",
			fn: func(_ context.Context) {
				results := Map(-100, func(n int) string {
					if n%2 == 1 {
						time.Sleep(sleepTime)
					}
					return strconv.Itoa(n + 10)
				}, testInts)

				v := make([]string, 0, len(testInts))
				for r := range results {
					v = append(v, r)
				}

				if len(v) != len(testInts) {
					t.Fatalf("Expected result slice len=%v but received len=%v", len(testInts), len(v))
				}
				for i, l := range v {
					if l != strconv.Itoa(testInts[i]+10) {
						t.Errorf("Expected search to return result=%v but received result=%v", testInts[i]+10, v)
					}
				}
			},
		},
		{
			name: "negative pool unordered convert int to string",
			fn: func(_ context.Context) {
				results := MapUnordered(-100, func(n int) string {
					if n%2 == 1 {
						time.Sleep(sleepTime)
					}
					return strconv.Itoa(n + 10)
				}, testInts)

				v := make([]string, 0, len(testInts))
				for r := range results {
					v = append(v, r)
				}
				sort.Strings(v)

				if len(v) != len(testInts) {
					t.Fatalf("Expected result slice len=%v but received len=%v", len(testInts), len(v))
				}
				for i, l := range v {
					if l != strconv.Itoa(testInts[i]+10) {
						t.Errorf("Expected search to return result=%v but received result=%v", testInts[i]+10, v)
					}
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.cancelTime > 0 {
				var cancel func()
				ctx, cancel = context.WithCancel(context.Background())
				time.AfterFunc(tt.cancelTime, func() {
					cancel()
				})
			}
			tt.fn(ctx)
		})
	}
}

func TestMapErr(t *testing.T) {
	sleepTime := 100 * time.Millisecond
	tests := []struct {
		name       string
		fn         func(context.Context)
		cancelTime time.Duration
		timeLimit  time.Duration
	}{
		{
			name: "basic convert int to string",
			fn: func(_ context.Context) {
				next := MapErr(5, func(s string) (int, error) {
					if len(s)%2 == 1 {
						time.Sleep(sleepTime)
					}
					return len(s), nil
				}, testStrings)

				v := make([]int, 0, len(testStrings))
				for n, err, ok := next(); ok; n, err, ok = next() {
					if err != nil {
						t.Fatalf("Expected no error")
					}
					v = append(v, n)
				}

				if len(v) != len(testStrings) {
					t.Fatalf("Expected result slice len=%v but received len=%v", len(testStrings), len(v))
				}
				for i, l := range v {
					if l != len(testStrings[i]) {
						t.Errorf("Expected search to return result=%v but received result=%v", testStrings[i], v)
					}
				}
			},
		},
		{
			name: "basic unordered convert int to string",
			fn: func(_ context.Context) {
				next := MapErrUnordered(5, func(s string) (int, error) {
					if len(s)%2 == 1 {
						time.Sleep(sleepTime)
					}
					return len(s), nil
				}, testStrings)

				v := make([]int, 0, len(testStrings))
				for n, err, ok := next(); ok; n, err, ok = next() {
					if err != nil {
						t.Fatalf("Expected no error")
					}
					v = append(v, n)
				}
				sort.Ints(v)
				st := make([]string, len(testStrings))
				copy(st, testStrings)
				sort.Strings(st)

				if len(v) != len(testStrings) {
					t.Fatalf("Expected result slice len=%v but received len=%v", len(testStrings), len(v))
				}
				for i, l := range v {
					if l != len(st[i]) {
						t.Errorf("Expected search to return result=%v but received result=%v", testStrings[i], v)
					}
				}
			},
		},
		{
			name: "convert int to string error",
			fn: func(_ context.Context) {
				next := MapErr(5, func(s string) (int, error) {
					if len(s)%2 == 1 {
						time.Sleep(sleepTime)
					}
					if len(s)%14 == 0 {
						return len(s), testErr
					}
					return len(s), nil
				}, testStrings)

				v := make([]int, 0, len(testStrings))
				var foundErr bool
				for n, err, ok := next(); ok; n, err, ok = next() {
					if err == testErr {
						foundErr = true
					}
					v = append(v, n)
				}

				if !foundErr {
					t.Errorf("Expected an error")
				}
				if len(v) >= len(testStrings) {
					t.Fatalf("Expected result slice len=%v > len=%v", len(testStrings), len(v))
				}
				for i, l := range v {
					if l != len(testStrings[i]) {
						t.Errorf("Expected search to return result=%v but received result=%v", testStrings[i], v)
					}
				}
			},
		},
		{
			name: "unordered convert int to string error",
			fn: func(_ context.Context) {
				next := MapErrUnordered(5, func(s string) (int, error) {
					if len(s)%2 == 1 {
						time.Sleep(sleepTime)
					}
					if len(s)%14 == 0 {
						return len(s), testErr
					}
					return len(s), nil
				}, testStrings)

				v := make([]int, 0, len(testStrings))
				var foundErr bool
				for n, err, ok := next(); ok; n, err, ok = next() {
					if err == testErr {
						foundErr = true
						break
					}
					v = append(v, n)
				}

				if !foundErr {
					t.Errorf("Expected an error")
				}
				if len(v) >= len(testStrings) {
					t.Fatalf("Expected result slice len=%v > len=%v", len(testStrings), len(v))
				}
			},
		},
		{
			name: "basic convert int to string with context cancel",
			fn: func(ctx context.Context) {
				next := MapErrWithContext(ctx, 5, func(s string) (int, error) {
					if len(s)%2 == 1 {
						time.Sleep(sleepTime)
					}
					return len(s), nil
				}, testStrings)

				v := make([]int, 0, len(testStrings))
				var foundErr bool
				for n, err, ok := next(); ok; n, err, ok = next() {
					if err == context.Canceled {
						foundErr = true
						break
					}
					v = append(v, n)
				}

				if !foundErr {
					t.Errorf("Expected an error")
				}
				if len(v) >= len(testStrings) {
					t.Fatalf("Expected result slice len=%v < len=%v", len(testStrings), len(v))
				}
				for i, l := range v {
					if l != len(testStrings[i]) {
						t.Errorf("Expected search to return result=%v > result=%v", testStrings[i], v)
					}
				}
			},
			cancelTime: sleepTime + (50 * time.Millisecond),
		},
		{
			name: "basic unordered convert int to string with context cancel",
			fn: func(ctx context.Context) {
				next := MapErrUnorderedWithContext(ctx, 5, func(s string) (int, error) {
					if len(s)%2 == 1 {
						time.Sleep(sleepTime)
					}
					return len(s), nil
				}, testStrings)

				v := make([]int, 0, len(testStrings))
				var foundErr bool
				for n, err, ok := next(); ok; n, err, ok = next() {
					if err == context.Canceled {
						foundErr = true
						break
					}
					v = append(v, n)
				}

				if !foundErr {
					t.Errorf("Expected an error")
				}
				if len(v) >= len(testStrings) {
					t.Fatalf("Expected result slice len=%v > len=%v", len(testStrings), len(v))
				}
			},
			cancelTime: sleepTime + (50 * time.Millisecond),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.cancelTime > 0 {
				var cancel func()
				ctx, cancel = context.WithCancel(context.Background())
				time.AfterFunc(tt.cancelTime, func() {
					cancel()
				})
			}
			tt.fn(ctx)
		})
	}
}

func TestReduce(t *testing.T) {
	sleepTime := 100 * time.Millisecond
	tests := []struct {
		name       string
		fn         func(context.Context)
		cancelTime time.Duration
		timeLimit  time.Duration
	}{
		{
			name: "basic convert int to string",
			fn: func(_ context.Context) {
				v, err := Reduce(5, func(s string) (int, error) {
					if len(s)%2 == 1 {
						time.Sleep(sleepTime)
					}
					return len(s), nil
				}, func(n, m int) (int, error) {
					return n + m, nil
				}, testStrings)

				if err != nil {
					t.Fatalf("Expected no error")
				}

				if v != 218 {
					t.Errorf("Expected string len sum=%v but received sum=%v", 218, v)
				}
			},
		},
		{
			name: "basic unordered convert int to string",
			fn: func(_ context.Context) {
				v, err := ReduceUnordered(5, func(s string) (int, error) {
					if len(s)%2 == 1 {
						time.Sleep(sleepTime)
					}
					return len(s), nil
				}, func(n, m int) (int, error) {
					return n + m, nil
				}, testStrings)

				if err != nil {
					t.Fatalf("Expected no error")
				}

				if v != 218 {
					t.Errorf("Expected string len sum=%v but received sum=%v", 218, v)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.cancelTime > 0 {
				var cancel func()
				ctx, cancel = context.WithCancel(context.Background())
				time.AfterFunc(tt.cancelTime, func() {
					cancel()
				})
			}
			tt.fn(ctx)
		})
	}
}

// Modified example from code x/sync/errgroup
// https://pkg.go.dev/golang.org/x/sync/errgroup
var (
	Web   = fakeSearch("web")
	Image = fakeSearch("image")
	Video = fakeSearch("video")
)

type aResult string
type aSearch func(ctx context.Context, query string) (aResult, error)

func fakeSearch(kind string) aSearch {
	return func(_ context.Context, query string) (aResult, error) {
		return aResult(fmt.Sprintf("%s result for %q", kind, query)), nil
	}
}

func TestCollect(t *testing.T) {
	expectV := []string{
		fmt.Sprintf("%s result for %q", "web", "golang"),
		fmt.Sprintf("%s result for %q", "image", "golang"),
		fmt.Sprintf("%s result for %q", "video", "golang"),
	}
	tests := []struct {
		name       string
		fn         func(context.Context)
		cancelTime time.Duration
		timeLimit  time.Duration
	}{
		{
			name: "basic errgroup parallel",
			fn: func(ctx context.Context) {
				results, err := Collect(3, func(search aSearch) (aResult, error) {
					return search(ctx, "golang")
				}, []aSearch{Web, Image, Video})

				if err != nil {
					t.Fatalf("Expected no error")
				}

				if len(results) != 3 {
					t.Fatalf("Expected result length=%v but length=%v", 3, len(results))
				}

				for i, s := range results {
					if string(s) != expectV[i] {
						t.Errorf("Expected string=%v but received string=%v", expectV[i], s)
					}
				}
			},
		},
		{
			name: "basic unordered errgroup parallel",
			fn: func(ctx context.Context) {
				results, err := Collect(3, func(search aSearch) (aResult, error) {
					return search(ctx, "golang")
				}, []aSearch{Web, Image, Video})

				if err != nil {
					t.Fatalf("Expected no error")
				}

				if len(results) != 3 {
					t.Fatalf("Expected result length=%v but length=%v", 3, len(results))
				}

				sta := make([]string, len(expectV))
				copy(sta, expectV)
				sort.Strings(sta)

				stb := make([]string, len(expectV))
				for i, result := range results {
					stb[i] = string(result)
				}
				sort.Strings(stb)

				for i, s := range stb {
					if s != sta[i] {
						t.Errorf("Expected string=%v but received string=%v", sta[i], s)
					}
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.cancelTime > 0 {
				var cancel func()
				ctx, cancel = context.WithCancel(context.Background())
				time.AfterFunc(tt.cancelTime, func() {
					cancel()
				})
			}
			tt.fn(ctx)
		})
	}
}

func TestForEach(t *testing.T) {
	tests := []struct {
		name       string
		fn         func(context.Context)
		cancelTime time.Duration
		timeLimit  time.Duration
	}{
		{
			name: "without an error",
			fn: func(ctx context.Context) {
				var mu sync.Mutex
				var total int
				err := ForEach(3, func(n int) error {
					time.Sleep(100 * time.Millisecond)
					mu.Lock()
					defer mu.Unlock()
					total = total + n
					return nil
				}, []int{1, 2, 3, 4, 5, 6, 7, 8, 9})

				if err != nil {
					t.Fatalf("Expected no error")
				}

				if total != 45 {
					t.Fatalf("Expected total=%v but found=%v", 45, total)
				}
			},
		},
		{
			name: "with an error",
			fn: func(ctx context.Context) {
				var mu sync.Mutex
				var total int
				err := ForEach(3, func(n int) error {
					time.Sleep(100 * time.Millisecond)
					mu.Lock()
					defer mu.Unlock()
					if n == 4 {
						return testErr
					}
					total = total + n
					return nil
				}, []int{1, 2, 3, 4, 5, 6, 7, 8, 9})

				if err != testErr {
					t.Fatalf("Expected an error")
				}

				if total >= 45 {
					t.Fatalf("Expected total<%v but found=%v", 45, total)
				}
			},
		},
		{
			name: "with ErrSearchSuccess",
			fn: func(ctx context.Context) {
				var mu sync.Mutex
				var total int
				err := ForEach(3, func(n int) error {
					time.Sleep(100 * time.Millisecond)
					mu.Lock()
					defer mu.Unlock()
					if n == 4 {
						return ErrSearchSuccess
					}
					total = total + n
					return nil
				}, []int{1, 2, 3, 4, 5, 6, 7, 8, 9})

				if err != ErrSearchSuccess {
					t.Fatalf("Expected ErrSearchSuccess")
				}

				if total >= 45 {
					t.Fatalf("Expected total<%v but found=%v", 45, total)
				}
			},
		},
		{
			name: "unordered without an error",
			fn: func(ctx context.Context) {
				var mu sync.Mutex
				var total int
				err := ForEachUnordered(3, func(n int) error {
					time.Sleep(100 * time.Millisecond)
					mu.Lock()
					defer mu.Unlock()
					total = total + n
					return nil
				}, []int{1, 2, 3, 4, 5, 6, 7, 8, 9})

				if err != nil {
					t.Fatalf("Expected no error")
				}

				if total != 45 {
					t.Fatalf("Expected total=%v but found=%v", 45, total)
				}
			},
		},
		{
			name: "unordered with an error",
			fn: func(ctx context.Context) {
				var mu sync.Mutex
				var total int
				err := ForEachUnordered(3, func(n int) error {
					time.Sleep(100 * time.Millisecond)
					mu.Lock()
					defer mu.Unlock()
					if n == 4 {
						return testErr
					}
					total = total + n
					return nil
				}, []int{1, 2, 3, 4, 5, 6, 7, 8, 9})

				if err != testErr {
					t.Fatalf("Expected an error")
				}

				if total >= 45 {
					t.Fatalf("Expected total<%v but found=%v", 45, total)
				}
			},
		},
		{
			name: "unordered with ErrSearchSuccess",
			fn: func(ctx context.Context) {
				var mu sync.Mutex
				var total int
				err := ForEachUnordered(3, func(n int) error {
					time.Sleep(100 * time.Millisecond)
					mu.Lock()
					defer mu.Unlock()
					if n == 4 {
						return ErrSearchSuccess
					}
					total = total + n
					return nil
				}, []int{1, 2, 3, 4, 5, 6, 7, 8, 9})

				if err != ErrSearchSuccess {
					t.Fatalf("Expected ErrSearchSuccess")
				}

				if total >= 45 {
					t.Fatalf("Expected total<%v but found=%v", 45, total)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.cancelTime > 0 {
				var cancel func()
				ctx, cancel = context.WithCancel(context.Background())
				time.AfterFunc(tt.cancelTime, func() {
					cancel()
				})
			}
			tt.fn(ctx)
		})
	}
}
