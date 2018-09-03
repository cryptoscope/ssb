// Code generated by counterfeiter. DO NOT EDIT.
package mock

import (
	"context"
	"sync"

	"go.cryptoscope.co/luigi"
	"go.cryptoscope.co/muxrpc"
	"go.cryptoscope.co/sbot"
)

type FakeWantManager struct {
	WantStub        func(ref *sbot.BlobRef) error
	wantMutex       sync.RWMutex
	wantArgsForCall []struct {
		ref *sbot.BlobRef
	}
	wantReturns struct {
		result1 error
	}
	wantReturnsOnCall map[int]struct {
		result1 error
	}
	WantsStub        func(ref *sbot.BlobRef) bool
	wantsMutex       sync.RWMutex
	wantsArgsForCall []struct {
		ref *sbot.BlobRef
	}
	wantsReturns struct {
		result1 bool
	}
	wantsReturnsOnCall map[int]struct {
		result1 bool
	}
	WantWithDistStub        func(ref *sbot.BlobRef, dist int64) error
	wantWithDistMutex       sync.RWMutex
	wantWithDistArgsForCall []struct {
		ref  *sbot.BlobRef
		dist int64
	}
	wantWithDistReturns struct {
		result1 error
	}
	wantWithDistReturnsOnCall map[int]struct {
		result1 error
	}
	CreateWantsStub        func(context.Context, luigi.Sink, muxrpc.Endpoint) luigi.Sink
	createWantsMutex       sync.RWMutex
	createWantsArgsForCall []struct {
		arg1 context.Context
		arg2 luigi.Sink
		arg3 muxrpc.Endpoint
	}
	createWantsReturns struct {
		result1 luigi.Sink
	}
	createWantsReturnsOnCall map[int]struct {
		result1 luigi.Sink
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeWantManager) Want(ref *sbot.BlobRef) error {
	fake.wantMutex.Lock()
	ret, specificReturn := fake.wantReturnsOnCall[len(fake.wantArgsForCall)]
	fake.wantArgsForCall = append(fake.wantArgsForCall, struct {
		ref *sbot.BlobRef
	}{ref})
	fake.recordInvocation("Want", []interface{}{ref})
	fake.wantMutex.Unlock()
	if fake.WantStub != nil {
		return fake.WantStub(ref)
	}
	if specificReturn {
		return ret.result1
	}
	return fake.wantReturns.result1
}

func (fake *FakeWantManager) WantCallCount() int {
	fake.wantMutex.RLock()
	defer fake.wantMutex.RUnlock()
	return len(fake.wantArgsForCall)
}

func (fake *FakeWantManager) WantArgsForCall(i int) *sbot.BlobRef {
	fake.wantMutex.RLock()
	defer fake.wantMutex.RUnlock()
	return fake.wantArgsForCall[i].ref
}

func (fake *FakeWantManager) WantReturns(result1 error) {
	fake.WantStub = nil
	fake.wantReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeWantManager) WantReturnsOnCall(i int, result1 error) {
	fake.WantStub = nil
	if fake.wantReturnsOnCall == nil {
		fake.wantReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.wantReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeWantManager) Wants(ref *sbot.BlobRef) bool {
	fake.wantsMutex.Lock()
	ret, specificReturn := fake.wantsReturnsOnCall[len(fake.wantsArgsForCall)]
	fake.wantsArgsForCall = append(fake.wantsArgsForCall, struct {
		ref *sbot.BlobRef
	}{ref})
	fake.recordInvocation("Wants", []interface{}{ref})
	fake.wantsMutex.Unlock()
	if fake.WantsStub != nil {
		return fake.WantsStub(ref)
	}
	if specificReturn {
		return ret.result1
	}
	return fake.wantsReturns.result1
}

func (fake *FakeWantManager) WantsCallCount() int {
	fake.wantsMutex.RLock()
	defer fake.wantsMutex.RUnlock()
	return len(fake.wantsArgsForCall)
}

func (fake *FakeWantManager) WantsArgsForCall(i int) *sbot.BlobRef {
	fake.wantsMutex.RLock()
	defer fake.wantsMutex.RUnlock()
	return fake.wantsArgsForCall[i].ref
}

func (fake *FakeWantManager) WantsReturns(result1 bool) {
	fake.WantsStub = nil
	fake.wantsReturns = struct {
		result1 bool
	}{result1}
}

func (fake *FakeWantManager) WantsReturnsOnCall(i int, result1 bool) {
	fake.WantsStub = nil
	if fake.wantsReturnsOnCall == nil {
		fake.wantsReturnsOnCall = make(map[int]struct {
			result1 bool
		})
	}
	fake.wantsReturnsOnCall[i] = struct {
		result1 bool
	}{result1}
}

func (fake *FakeWantManager) WantWithDist(ref *sbot.BlobRef, dist int64) error {
	fake.wantWithDistMutex.Lock()
	ret, specificReturn := fake.wantWithDistReturnsOnCall[len(fake.wantWithDistArgsForCall)]
	fake.wantWithDistArgsForCall = append(fake.wantWithDistArgsForCall, struct {
		ref  *sbot.BlobRef
		dist int64
	}{ref, dist})
	fake.recordInvocation("WantWithDist", []interface{}{ref, dist})
	fake.wantWithDistMutex.Unlock()
	if fake.WantWithDistStub != nil {
		return fake.WantWithDistStub(ref, dist)
	}
	if specificReturn {
		return ret.result1
	}
	return fake.wantWithDistReturns.result1
}

func (fake *FakeWantManager) WantWithDistCallCount() int {
	fake.wantWithDistMutex.RLock()
	defer fake.wantWithDistMutex.RUnlock()
	return len(fake.wantWithDistArgsForCall)
}

func (fake *FakeWantManager) WantWithDistArgsForCall(i int) (*sbot.BlobRef, int64) {
	fake.wantWithDistMutex.RLock()
	defer fake.wantWithDistMutex.RUnlock()
	return fake.wantWithDistArgsForCall[i].ref, fake.wantWithDistArgsForCall[i].dist
}

func (fake *FakeWantManager) WantWithDistReturns(result1 error) {
	fake.WantWithDistStub = nil
	fake.wantWithDistReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeWantManager) WantWithDistReturnsOnCall(i int, result1 error) {
	fake.WantWithDistStub = nil
	if fake.wantWithDistReturnsOnCall == nil {
		fake.wantWithDistReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.wantWithDistReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeWantManager) CreateWants(arg1 context.Context, arg2 luigi.Sink, arg3 muxrpc.Endpoint) luigi.Sink {
	fake.createWantsMutex.Lock()
	ret, specificReturn := fake.createWantsReturnsOnCall[len(fake.createWantsArgsForCall)]
	fake.createWantsArgsForCall = append(fake.createWantsArgsForCall, struct {
		arg1 context.Context
		arg2 luigi.Sink
		arg3 muxrpc.Endpoint
	}{arg1, arg2, arg3})
	fake.recordInvocation("CreateWants", []interface{}{arg1, arg2, arg3})
	fake.createWantsMutex.Unlock()
	if fake.CreateWantsStub != nil {
		return fake.CreateWantsStub(arg1, arg2, arg3)
	}
	if specificReturn {
		return ret.result1
	}
	return fake.createWantsReturns.result1
}

func (fake *FakeWantManager) CreateWantsCallCount() int {
	fake.createWantsMutex.RLock()
	defer fake.createWantsMutex.RUnlock()
	return len(fake.createWantsArgsForCall)
}

func (fake *FakeWantManager) CreateWantsArgsForCall(i int) (context.Context, luigi.Sink, muxrpc.Endpoint) {
	fake.createWantsMutex.RLock()
	defer fake.createWantsMutex.RUnlock()
	return fake.createWantsArgsForCall[i].arg1, fake.createWantsArgsForCall[i].arg2, fake.createWantsArgsForCall[i].arg3
}

func (fake *FakeWantManager) CreateWantsReturns(result1 luigi.Sink) {
	fake.CreateWantsStub = nil
	fake.createWantsReturns = struct {
		result1 luigi.Sink
	}{result1}
}

func (fake *FakeWantManager) CreateWantsReturnsOnCall(i int, result1 luigi.Sink) {
	fake.CreateWantsStub = nil
	if fake.createWantsReturnsOnCall == nil {
		fake.createWantsReturnsOnCall = make(map[int]struct {
			result1 luigi.Sink
		})
	}
	fake.createWantsReturnsOnCall[i] = struct {
		result1 luigi.Sink
	}{result1}
}

func (fake *FakeWantManager) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.wantMutex.RLock()
	defer fake.wantMutex.RUnlock()
	fake.wantsMutex.RLock()
	defer fake.wantsMutex.RUnlock()
	fake.wantWithDistMutex.RLock()
	defer fake.wantWithDistMutex.RUnlock()
	fake.createWantsMutex.RLock()
	defer fake.createWantsMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeWantManager) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ sbot.WantManager = new(FakeWantManager)