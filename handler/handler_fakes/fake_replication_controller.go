// This file was generated by counterfeiter
package handler_fakes

import (
	"sync"

	"k8s.io/kubernetes/pkg/api"
	v1api "k8s.io/kubernetes/pkg/api/v1"
	"k8s.io/kubernetes/pkg/client/clientset_generated/release_1_3/typed/core/v1"
	"k8s.io/kubernetes/pkg/watch"
)

type FakeReplicationController struct {
	CreateStub        func(*v1api.ReplicationController) (*v1api.ReplicationController, error)
	createMutex       sync.RWMutex
	createArgsForCall []struct {
		arg1 *v1api.ReplicationController
	}
	createReturns struct {
		result1 *v1api.ReplicationController
		result2 error
	}
	UpdateStub        func(*v1api.ReplicationController) (*v1api.ReplicationController, error)
	updateMutex       sync.RWMutex
	updateArgsForCall []struct {
		arg1 *v1api.ReplicationController
	}
	updateReturns struct {
		result1 *v1api.ReplicationController
		result2 error
	}
	UpdateStatusStub        func(*v1api.ReplicationController) (*v1api.ReplicationController, error)
	updateStatusMutex       sync.RWMutex
	updateStatusArgsForCall []struct {
		arg1 *v1api.ReplicationController
	}
	updateStatusReturns struct {
		result1 *v1api.ReplicationController
		result2 error
	}
	DeleteStub        func(name string, options *api.DeleteOptions) error
	deleteMutex       sync.RWMutex
	deleteArgsForCall []struct {
		name    string
		options *api.DeleteOptions
	}
	deleteReturns struct {
		result1 error
	}
	DeleteCollectionStub        func(options *api.DeleteOptions, listOptions api.ListOptions) error
	deleteCollectionMutex       sync.RWMutex
	deleteCollectionArgsForCall []struct {
		options     *api.DeleteOptions
		listOptions api.ListOptions
	}
	deleteCollectionReturns struct {
		result1 error
	}
	GetStub        func(name string) (*v1api.ReplicationController, error)
	getMutex       sync.RWMutex
	getArgsForCall []struct {
		name string
	}
	getReturns struct {
		result1 *v1api.ReplicationController
		result2 error
	}
	ListStub        func(opts api.ListOptions) (*v1api.ReplicationControllerList, error)
	listMutex       sync.RWMutex
	listArgsForCall []struct {
		opts api.ListOptions
	}
	listReturns struct {
		result1 *v1api.ReplicationControllerList
		result2 error
	}
	WatchStub        func(opts api.ListOptions) (watch.Interface, error)
	watchMutex       sync.RWMutex
	watchArgsForCall []struct {
		opts api.ListOptions
	}
	watchReturns struct {
		result1 watch.Interface
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeReplicationController) Create(arg1 *v1api.ReplicationController) (*v1api.ReplicationController, error) {
	fake.createMutex.Lock()
	fake.createArgsForCall = append(fake.createArgsForCall, struct {
		arg1 *v1api.ReplicationController
	}{arg1})
	fake.recordInvocation("Create", []interface{}{arg1})
	fake.createMutex.Unlock()
	if fake.CreateStub != nil {
		return fake.CreateStub(arg1)
	} else {
		return fake.createReturns.result1, fake.createReturns.result2
	}
}

func (fake *FakeReplicationController) CreateCallCount() int {
	fake.createMutex.RLock()
	defer fake.createMutex.RUnlock()
	return len(fake.createArgsForCall)
}

func (fake *FakeReplicationController) CreateArgsForCall(i int) *v1api.ReplicationController {
	fake.createMutex.RLock()
	defer fake.createMutex.RUnlock()
	return fake.createArgsForCall[i].arg1
}

func (fake *FakeReplicationController) CreateReturns(result1 *v1api.ReplicationController, result2 error) {
	fake.CreateStub = nil
	fake.createReturns = struct {
		result1 *v1api.ReplicationController
		result2 error
	}{result1, result2}
}

func (fake *FakeReplicationController) Update(arg1 *v1api.ReplicationController) (*v1api.ReplicationController, error) {
	fake.updateMutex.Lock()
	fake.updateArgsForCall = append(fake.updateArgsForCall, struct {
		arg1 *v1api.ReplicationController
	}{arg1})
	fake.recordInvocation("Update", []interface{}{arg1})
	fake.updateMutex.Unlock()
	if fake.UpdateStub != nil {
		return fake.UpdateStub(arg1)
	} else {
		return fake.updateReturns.result1, fake.updateReturns.result2
	}
}

func (fake *FakeReplicationController) UpdateCallCount() int {
	fake.updateMutex.RLock()
	defer fake.updateMutex.RUnlock()
	return len(fake.updateArgsForCall)
}

func (fake *FakeReplicationController) UpdateArgsForCall(i int) *v1api.ReplicationController {
	fake.updateMutex.RLock()
	defer fake.updateMutex.RUnlock()
	return fake.updateArgsForCall[i].arg1
}

func (fake *FakeReplicationController) UpdateReturns(result1 *v1api.ReplicationController, result2 error) {
	fake.UpdateStub = nil
	fake.updateReturns = struct {
		result1 *v1api.ReplicationController
		result2 error
	}{result1, result2}
}

func (fake *FakeReplicationController) UpdateStatus(arg1 *v1api.ReplicationController) (*v1api.ReplicationController, error) {
	fake.updateStatusMutex.Lock()
	fake.updateStatusArgsForCall = append(fake.updateStatusArgsForCall, struct {
		arg1 *v1api.ReplicationController
	}{arg1})
	fake.recordInvocation("UpdateStatus", []interface{}{arg1})
	fake.updateStatusMutex.Unlock()
	if fake.UpdateStatusStub != nil {
		return fake.UpdateStatusStub(arg1)
	} else {
		return fake.updateStatusReturns.result1, fake.updateStatusReturns.result2
	}
}

func (fake *FakeReplicationController) UpdateStatusCallCount() int {
	fake.updateStatusMutex.RLock()
	defer fake.updateStatusMutex.RUnlock()
	return len(fake.updateStatusArgsForCall)
}

func (fake *FakeReplicationController) UpdateStatusArgsForCall(i int) *v1api.ReplicationController {
	fake.updateStatusMutex.RLock()
	defer fake.updateStatusMutex.RUnlock()
	return fake.updateStatusArgsForCall[i].arg1
}

func (fake *FakeReplicationController) UpdateStatusReturns(result1 *v1api.ReplicationController, result2 error) {
	fake.UpdateStatusStub = nil
	fake.updateStatusReturns = struct {
		result1 *v1api.ReplicationController
		result2 error
	}{result1, result2}
}

func (fake *FakeReplicationController) Delete(name string, options *api.DeleteOptions) error {
	fake.deleteMutex.Lock()
	fake.deleteArgsForCall = append(fake.deleteArgsForCall, struct {
		name    string
		options *api.DeleteOptions
	}{name, options})
	fake.recordInvocation("Delete", []interface{}{name, options})
	fake.deleteMutex.Unlock()
	if fake.DeleteStub != nil {
		return fake.DeleteStub(name, options)
	} else {
		return fake.deleteReturns.result1
	}
}

func (fake *FakeReplicationController) DeleteCallCount() int {
	fake.deleteMutex.RLock()
	defer fake.deleteMutex.RUnlock()
	return len(fake.deleteArgsForCall)
}

func (fake *FakeReplicationController) DeleteArgsForCall(i int) (string, *api.DeleteOptions) {
	fake.deleteMutex.RLock()
	defer fake.deleteMutex.RUnlock()
	return fake.deleteArgsForCall[i].name, fake.deleteArgsForCall[i].options
}

func (fake *FakeReplicationController) DeleteReturns(result1 error) {
	fake.DeleteStub = nil
	fake.deleteReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeReplicationController) DeleteCollection(options *api.DeleteOptions, listOptions api.ListOptions) error {
	fake.deleteCollectionMutex.Lock()
	fake.deleteCollectionArgsForCall = append(fake.deleteCollectionArgsForCall, struct {
		options     *api.DeleteOptions
		listOptions api.ListOptions
	}{options, listOptions})
	fake.recordInvocation("DeleteCollection", []interface{}{options, listOptions})
	fake.deleteCollectionMutex.Unlock()
	if fake.DeleteCollectionStub != nil {
		return fake.DeleteCollectionStub(options, listOptions)
	} else {
		return fake.deleteCollectionReturns.result1
	}
}

func (fake *FakeReplicationController) DeleteCollectionCallCount() int {
	fake.deleteCollectionMutex.RLock()
	defer fake.deleteCollectionMutex.RUnlock()
	return len(fake.deleteCollectionArgsForCall)
}

func (fake *FakeReplicationController) DeleteCollectionArgsForCall(i int) (*api.DeleteOptions, api.ListOptions) {
	fake.deleteCollectionMutex.RLock()
	defer fake.deleteCollectionMutex.RUnlock()
	return fake.deleteCollectionArgsForCall[i].options, fake.deleteCollectionArgsForCall[i].listOptions
}

func (fake *FakeReplicationController) DeleteCollectionReturns(result1 error) {
	fake.DeleteCollectionStub = nil
	fake.deleteCollectionReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeReplicationController) Get(name string) (*v1api.ReplicationController, error) {
	fake.getMutex.Lock()
	fake.getArgsForCall = append(fake.getArgsForCall, struct {
		name string
	}{name})
	fake.recordInvocation("Get", []interface{}{name})
	fake.getMutex.Unlock()
	if fake.GetStub != nil {
		return fake.GetStub(name)
	} else {
		return fake.getReturns.result1, fake.getReturns.result2
	}
}

func (fake *FakeReplicationController) GetCallCount() int {
	fake.getMutex.RLock()
	defer fake.getMutex.RUnlock()
	return len(fake.getArgsForCall)
}

func (fake *FakeReplicationController) GetArgsForCall(i int) string {
	fake.getMutex.RLock()
	defer fake.getMutex.RUnlock()
	return fake.getArgsForCall[i].name
}

func (fake *FakeReplicationController) GetReturns(result1 *v1api.ReplicationController, result2 error) {
	fake.GetStub = nil
	fake.getReturns = struct {
		result1 *v1api.ReplicationController
		result2 error
	}{result1, result2}
}

func (fake *FakeReplicationController) List(opts api.ListOptions) (*v1api.ReplicationControllerList, error) {
	fake.listMutex.Lock()
	fake.listArgsForCall = append(fake.listArgsForCall, struct {
		opts api.ListOptions
	}{opts})
	fake.recordInvocation("List", []interface{}{opts})
	fake.listMutex.Unlock()
	if fake.ListStub != nil {
		return fake.ListStub(opts)
	} else {
		return fake.listReturns.result1, fake.listReturns.result2
	}
}

func (fake *FakeReplicationController) ListCallCount() int {
	fake.listMutex.RLock()
	defer fake.listMutex.RUnlock()
	return len(fake.listArgsForCall)
}

func (fake *FakeReplicationController) ListArgsForCall(i int) api.ListOptions {
	fake.listMutex.RLock()
	defer fake.listMutex.RUnlock()
	return fake.listArgsForCall[i].opts
}

func (fake *FakeReplicationController) ListReturns(result1 *v1api.ReplicationControllerList, result2 error) {
	fake.ListStub = nil
	fake.listReturns = struct {
		result1 *v1api.ReplicationControllerList
		result2 error
	}{result1, result2}
}

func (fake *FakeReplicationController) Watch(opts api.ListOptions) (watch.Interface, error) {
	fake.watchMutex.Lock()
	fake.watchArgsForCall = append(fake.watchArgsForCall, struct {
		opts api.ListOptions
	}{opts})
	fake.recordInvocation("Watch", []interface{}{opts})
	fake.watchMutex.Unlock()
	if fake.WatchStub != nil {
		return fake.WatchStub(opts)
	} else {
		return fake.watchReturns.result1, fake.watchReturns.result2
	}
}

func (fake *FakeReplicationController) WatchCallCount() int {
	fake.watchMutex.RLock()
	defer fake.watchMutex.RUnlock()
	return len(fake.watchArgsForCall)
}

func (fake *FakeReplicationController) WatchArgsForCall(i int) api.ListOptions {
	fake.watchMutex.RLock()
	defer fake.watchMutex.RUnlock()
	return fake.watchArgsForCall[i].opts
}

func (fake *FakeReplicationController) WatchReturns(result1 watch.Interface, result2 error) {
	fake.WatchStub = nil
	fake.watchReturns = struct {
		result1 watch.Interface
		result2 error
	}{result1, result2}
}

func (fake *FakeReplicationController) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.createMutex.RLock()
	defer fake.createMutex.RUnlock()
	fake.updateMutex.RLock()
	defer fake.updateMutex.RUnlock()
	fake.updateStatusMutex.RLock()
	defer fake.updateStatusMutex.RUnlock()
	fake.deleteMutex.RLock()
	defer fake.deleteMutex.RUnlock()
	fake.deleteCollectionMutex.RLock()
	defer fake.deleteCollectionMutex.RUnlock()
	fake.getMutex.RLock()
	defer fake.getMutex.RUnlock()
	fake.listMutex.RLock()
	defer fake.listMutex.RUnlock()
	fake.watchMutex.RLock()
	defer fake.watchMutex.RUnlock()
	return fake.invocations
}

func (fake *FakeReplicationController) recordInvocation(key string, args []interface{}) {
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

var _ v1.ReplicationControllerInterface = new(FakeReplicationController)
