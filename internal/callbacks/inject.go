/*
 * Copyright 2024 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package callbacks

import (
	"context"

	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/eino/utils/generic"
)

type CtxManagerKey struct{}

func InitCallbacks(ctx context.Context, info *RunInfo, handlers ...Handler) context.Context {
	mgr, ok := newManager(info, handlers...)
	if ok {
		return ctxWithManager(ctx, mgr)
	}

	return ctxWithManager(ctx, nil)
}

func InitCallbacksWithExistingHandlers(ctx context.Context, info *RunInfo) context.Context {
	cbm, ok := managerFromCtx(ctx)
	if !ok {
		return ctx
	}

	return ctxWithManager(ctx, cbm.withRunInfo(info))
}

type Handle[T any] func(context.Context, T, *RunInfo, []Handler) (context.Context, T)

func On[T any](ctx context.Context, inOut T, handle Handle[T], timing CallbackTiming) (context.Context, T) {
	mgr, ok := managerFromCtx(ctx)
	if !ok {
		return ctx, inOut
	}

	hs := make([]Handler, 0, len(mgr.handlers))
	for _, handler := range mgr.handlers {
		timingChecker, ok_ := handler.(TimingChecker)
		if !ok_ || timingChecker.Needed(ctx, mgr.runInfo, timing) {
			hs = append(hs, handler)
		}
	}

	return handle(ctx, inOut, mgr.runInfo, hs)
}

func OnStartHandle[T any](ctx context.Context, input T,
	runInfo *RunInfo, handlers []Handler) (context.Context, T) {

	for i := len(handlers) - 1; i >= 0; i-- {
		ctx = handlers[i].OnStart(ctx, runInfo, input)
	}

	return ctx, input
}

func OnEndHandle[T any](ctx context.Context, output T,
	runInfo *RunInfo, handlers []Handler) (context.Context, T) {

	for _, handler := range handlers {
		ctx = handler.OnEnd(ctx, runInfo, output)
	}

	return ctx, output
}

func OnWithStream[S any](
	ctx context.Context,
	inOut S,
	handlers []Handler,
	cpy func(int) []S,
	handle func(Handler, S) context.Context) (context.Context, S) {

	if len(handlers) == 0 {
		return ctx, inOut
	}

	inOuts := cpy(len(handlers) + 1)

	for i, handler := range handlers {
		ctx = handle(handler, inOuts[i])
	}

	return ctx, inOuts[len(inOuts)-1]
}

func OnStartWithStreamInputHandle[T any](ctx context.Context, input *schema.StreamReader[T],
	runInfo *RunInfo, handlers []Handler) (context.Context, *schema.StreamReader[T]) {

	generic.Reverse(handlers)

	cpy := input.Copy

	handle := func(handler Handler, in *schema.StreamReader[T]) context.Context {
		in_ := schema.StreamReaderWithConvert(in, func(i T) (CallbackInput, error) {
			return i, nil
		})
		return handler.OnStartWithStreamInput(ctx, runInfo, in_)
	}

	return OnWithStream(ctx, input, handlers, cpy, handle)
}

func OnEndWithStreamOutputHandle[T any](ctx context.Context, output *schema.StreamReader[T],
	runInfo *RunInfo, handlers []Handler) (context.Context, *schema.StreamReader[T]) {

	cpy := output.Copy

	handle := func(handler Handler, out *schema.StreamReader[T]) context.Context {
		out_ := schema.StreamReaderWithConvert(out, func(i T) (CallbackOutput, error) {
			return i, nil
		})
		return handler.OnEndWithStreamOutput(ctx, runInfo, out_)
	}

	return OnWithStream(ctx, output, handlers, cpy, handle)
}

func OnErrorHandle(ctx context.Context, err error,
	runInfo *RunInfo, handlers []Handler) (context.Context, error) {

	for _, handler := range handlers {
		ctx = handler.OnError(ctx, runInfo, err)
	}

	return ctx, err
}
