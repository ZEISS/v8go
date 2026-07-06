// Copyright 2019 Roger Chapman and the v8go contributors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package v8go_test

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	v8 "github.com/zeiss/v8go"
)

func TestIsolateTerminateExecution(t *testing.T) {
	iso := v8.NewIsolate()
	defer iso.Dispose()

	if iso.IsExecutionTerminating() {
		t.Error("expected no execution to be terminating")
	}

	var terminating bool
	fooFn := v8.NewFunctionTemplate(iso, func(info *v8.FunctionCallbackInfo) *v8.Value {
		loop, _ := info.Args()[0].AsFunction()
		go func() {
			iso.TerminateExecution()
		}()
		loop.Call(v8.Undefined(iso))

		terminating = iso.IsExecutionTerminating()
		return nil
	})

	global := v8.NewObjectTemplate(iso)
	global.Set("foo", fooFn)

	ctx := v8.NewContext(iso, global)
	defer ctx.Close()

	script := `function loop() { while (true) { } }; foo(loop);`
	_, e := ctx.RunScript(script, "forever.js")
	if e == nil || !strings.HasPrefix(e.Error(), "ExecutionTerminated") {
		t.Errorf("unexpected error: %v", e)
	}

	if !terminating {
		t.Error("expected execution to have been terminating in function")
	}
}

func TestIsolateCompileUnboundScript(t *testing.T) {
	s := "function foo() { return 'bar'; }; foo()"

	i1 := v8.NewIsolate()
	defer i1.Dispose()
	c1 := v8.NewContext(i1)
	defer c1.Close()

	_, err := i1.CompileUnboundScript("invalid js", "filename", v8.CompileOptions{})
	if err == nil {
		t.Fatal("expected error")
	}

	us, err := i1.CompileUnboundScript(s, "script.js", v8.CompileOptions{Mode: v8.CompileModeEager})
	fatalIf(t, err)

	val, err := us.Run(c1)
	fatalIf(t, err)
	if val.String() != "bar" {
		t.Fatalf("invalid value returned, expected bar got %v", val)
	}

	cachedData := us.CreateCodeCache()

	i2 := v8.NewIsolate()
	defer i2.Dispose()
	c2 := v8.NewContext(i2)
	defer c2.Close()

	opts := v8.CompileOptions{CachedData: cachedData}
	usWithCachedData, err := i2.CompileUnboundScript(s, "script.js", opts)
	fatalIf(t, err)
	if usWithCachedData == nil {
		t.Fatal("expected unbound script from cached data not to be nil")
	}
	if opts.CachedData.Rejected {
		t.Fatal("expected cached data to be used, not rejected")
	}

	val, err = usWithCachedData.Run(c2)
	fatalIf(t, err)
	if val.String() != "bar" {
		t.Fatalf("invalid value returned, expected bar got %v", val)
	}
}

func TestIsolateCompileUnboundScript_CachedDataRejected(t *testing.T) {
	s := "function foo() { return 'bar'; }; foo()"
	iso := v8.NewIsolate()
	defer iso.Dispose()

	// Try to compile an unbound script using cached data that does not match this source
	opts := v8.CompileOptions{CachedData: &v8.CompilerCachedData{Bytes: []byte("Math.sqrt(4)")}}
	us, err := iso.CompileUnboundScript(s, "script.js", opts)
	fatalIf(t, err)
	if !opts.CachedData.Rejected {
		t.Error("expected cached data to be rejected")
	}

	ctx := v8.NewContext(iso)
	defer ctx.Close()

	// Verify that unbound script is still compiled and able to be used
	val, err := us.Run(ctx)
	fatalIf(t, err)
	if val.String() != "bar" {
		t.Errorf("invalid value returned, expected bar got %v", val)
	}
}

func TestIsolateCompileUnboundScript_InvalidOptions(t *testing.T) {
	iso := v8.NewIsolate()
	defer iso.Dispose()

	opts := v8.CompileOptions{
		CachedData: &v8.CompilerCachedData{Bytes: []byte("unused")},
		Mode:       v8.CompileModeEager,
	}
	panicErr := recoverPanic(func() { iso.CompileUnboundScript("console.log(1)", "script.js", opts) })
	if panicErr == nil {
		t.Error("expected panic")
	}
	if panicErr != "On CompileOptions, Mode and CachedData can't both be set" {
		t.Errorf("unexpected panic: %v\n", panicErr)
	}
}

func TestIsolateGetHeapStatistics(t *testing.T) {
	iso := v8.NewIsolate()
	defer iso.Dispose()
	ctx1 := v8.NewContext(iso)
	defer ctx1.Close()
	ctx2 := v8.NewContext(iso)
	defer ctx2.Close()

	hs := iso.GetHeapStatistics()

	if hs.NumberOfNativeContexts != 3 {
		t.Error("expect NumberOfNativeContexts return 3, got", hs.NumberOfNativeContexts)
	}

	if hs.NumberOfDetachedContexts != 0 {
		t.Error("expect NumberOfDetachedContexts return 0, got", hs.NumberOfDetachedContexts)
	}
}

func TestCallbackRegistry(t *testing.T) {
	iso := v8.NewIsolate()
	defer iso.Dispose()
	cb := func(*v8.FunctionCallbackInfo) (*v8.Value, error) { return nil, nil }

	cb0 := iso.GetCallback(0)
	if cb0 != nil {
		t.Error("expected callback function to be <nil>")
	}
	ref1 := iso.RegisterCallback(cb)
	if ref1 != 1 {
		t.Errorf("expected callback ref == 1, got %d", ref1)
	}
	cb1 := iso.GetCallback(1)
	if fmt.Sprintf("%p", cb1) != fmt.Sprintf("%p", cb) {
		t.Errorf("unexpected callback function; want %p, got %p", cb, cb1)
	}
}

func TestIsolateDispose(t *testing.T) {
	iso := v8.NewIsolate()
	if iso.GetHeapStatistics().TotalHeapSize == 0 {
		t.Error("Isolate incorrectly allocated")
	}

	iso.Dispose()
	// noop when called multiple times
	iso.Dispose()
	// deprecated
	iso.Close()

	if iso.GetHeapStatistics().TotalHeapSize != 0 {
		t.Error("Isolate not disposed correctly")
	}
}

func TestIsolateThrowException(t *testing.T) {
	iso := v8.NewIsolate()

	strErr, _ := v8.NewValue(iso, "some type error")

	throwError := func(val *v8.Value) {
		v := iso.ThrowException(val)

		if !v.IsNullOrUndefined() {
			t.Error("expected result to be null or undefined")
		}
	}

	// Function that throws a simple string error from within the function. It is meant
	// to emulate when an error is returned within Go.
	fn := v8.NewFunctionTemplate(iso, func(info *v8.FunctionCallbackInfo) *v8.Value {
		throwError(strErr)

		return nil
	})

	// Function that is passed a TypeError from JavaScript.
	fn2 := v8.NewFunctionTemplate(iso, func(info *v8.FunctionCallbackInfo) *v8.Value {
		typeErr := info.Args()[0]

		throwError(typeErr)

		return nil
	})

	global := v8.NewObjectTemplate(iso)
	global.Set("foo", fn)
	global.Set("foo2", fn2)

	ctx := v8.NewContext(iso, global)

	_, e := ctx.RunScript("foo()", "foo.js")

	if e.Error() != "some type error" {
		t.Errorf("expected \"some type error\" error but got: %v", e)
	}

	_, e = ctx.RunScript("foo2(new TypeError('this is a test'))", "foo.js")

	if e.Error() != "TypeError: this is a test" {
		t.Errorf("expected \"TypeError: this is a test\" error but got: %v", e)
	}

	ctx.Close()
	iso.Dispose()
	if recoverPanic(func() { iso.ThrowException(strErr) }) == nil {
		t.Error("expected panic")
	}
}

func TestNewIsolateWithConstraints(t *testing.T) {
	iso := v8.NewIsolate(v8.WithResourceConstraints(
		8*1024*1024,
		16*1024*1024,
	))
	defer iso.Dispose()

	ctx := v8.NewContext(iso)
	defer ctx.Close()

	// First test - should work fine
	val, err := ctx.RunScript("1 + 2", "test.js")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !val.IsNumber() || val.Number() != 3 {
		t.Errorf("expected 3, got %v", val)
	}

	// Second test - should run out of memory without crashing the process
	val, err = ctx.RunScript(`
			const data = [];
			for (let i = 0; i < 1000 * 1000; i++) {
					data.push("large data chunk ".repeat(1000));
			}
			data.length;
		`, "memory-test.js")
	if err != nil {
		t.Logf("Memory test correctly returned error: %v", err)
	} else {
		t.Fatalf("Memory test completed unexpectedly: %v", val)
	}
}

func BenchmarkIsolateInitialization(b *testing.B) {
	b.ReportAllocs()

	for n := 0; n < b.N; n++ {
		vm := v8.NewIsolate()
		vm.Close() // force disposal of the VM
	}
}

const script = `
	const process = (record) => {
		const res = [];
		for (let [k, v] of Object.entries(record)) {
			res.push({
				name: k,
				value: v,
			});
		}
		return JSON.stringify(res);
	};
`

func BenchmarkIsolateInitAndRunConcurrent(b *testing.B) {
	b.ReportAllocs()

	clients := []int{1000, 5000, 10000, 50000, 100000}
	for _, c := range clients {
		b.Run(fmt.Sprintf("process(%d)", c), func(b *testing.B) {
			sem := make(chan struct{}, c)
			wg := sync.WaitGroup{}
			for n := 0; n < b.N; n++ {
				wg.Add(1)

				go func() {
					defer wg.Done()
					sem <- struct{}{}        // acquire semaphore
					defer func() { <-sem }() // release semaphore

					iso := v8.NewIsolate()
					ctx := v8.NewContext(iso)
					defer iso.Dispose()
					defer ctx.Close()

					_, err := ctx.RunScript(script, "script.js")
					require.NoError(b, err)

					obj := makeObject()
					jsonStr, err := json.Marshal(obj)
					require.NoError(b, err)

					val, err := ctx.RunScript(fmt.Sprintf("process(%s)", jsonStr), "process.js")
					require.NoError(b, err)

					var res []map[string]interface{}
					err = json.Unmarshal([]byte(val.String()), &res)
					require.NoError(b, err)
				}()
			}

			wg.Wait()
		})
	}
}

func BenchmarkIsolateInitRun(b *testing.B) {
	b.ReportAllocs()

	for n := 0; n < b.N; n++ {
		iso := v8.NewIsolate()
		ctx := v8.NewContext(iso)
		_, err := ctx.RunScript(script, "script.js")
		require.NoError(b, err)

		obj := makeObject()
		jsonStr, err := json.Marshal(obj)
		require.NoError(b, err)

		val, err := ctx.RunScript(fmt.Sprintf("process(%s)", jsonStr), "process.js")
		require.NoError(b, err)

		var res []map[string]interface{}
		err = json.Unmarshal([]byte(val.String()), &res)
		require.NoError(b, err)

		ctx.Close()
		iso.Dispose()
	}
}

func BenchmarkIsolateCodeCache(b *testing.B) {
	b.ReportAllocs()

	iso := v8.NewIsolate()
	defer iso.Dispose()

	scrpt, err := iso.CompileUnboundScript(string(script), "main.js", v8.CompileOptions{})
	cache := scrpt.CreateCodeCache()
	opts := v8.CompileOptions{CachedData: cache}
	require.NoError(b, err)

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		iso := v8.NewIsolate()
		ctx := v8.NewContext(iso)
		script, _ := iso.CompileUnboundScript(string(script), "main.js", opts)
		script.Run(ctx)
		ctx.Close()
		iso.Dispose()
	}
}

func makeObject() interface{} {
	return map[string]interface{}{
		"a": rand.Intn(1000000), //nolint:gosec
		"b": "AAAABBBBAAAABBBBAAAABBBBAAAABBBBAAAABBBB",
	}
}
