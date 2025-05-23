// Copyright 2021 Roger Chapman and the v8 contributors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package v8go_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zeiss/v8go"
	v8 "github.com/zeiss/v8go"
)

func TestJSONParse(t *testing.T) {
	t.Parallel()

	if _, err := v8.JSONParse(nil, "{}"); err == nil {
		t.Error("expected error but got <nil>")
	}
	ctx := v8.NewContext()
	defer ctx.Isolate().Dispose()
	defer ctx.Close()
	_, err := v8.JSONParse(ctx, "{")
	if err == nil {
		t.Error("expected error but got <nil>")
		return
	}

	var jerr *v8go.JSError
	require.ErrorAs(t, err, &jerr)
}

func TestJSONStringify(t *testing.T) {
	t.Parallel()

	ctx := v8.NewContext()
	defer ctx.Isolate().Dispose()
	defer ctx.Close()
	if _, err := v8.JSONStringify(ctx, nil); err == nil {
		t.Error("expected error but got <nil>")
	}
}

func ExampleJSONParse() {
	ctx := v8.NewContext()
	defer ctx.Isolate().Dispose()
	defer ctx.Close()
	val, _ := v8.JSONParse(ctx, `{"foo": "bar"}`)
	fmt.Println(val)
	// Output:
	// [object Object]
}

func ExampleJSONStringify() {
	ctx := v8.NewContext()
	defer ctx.Isolate().Dispose()
	defer ctx.Close()
	val, _ := v8.JSONParse(ctx, `{
		"a": 1,
		"b": "foo"
	}`)
	jsonStr, _ := v8.JSONStringify(ctx, val)
	fmt.Println(jsonStr)
	// Output:
	// {"a":1,"b":"foo"}
}
