// Code generated by mockery 2.7.4. DO NOT EDIT.

package scheduler

import mock "github.com/stretchr/testify/mock"

// MockReconcilerInvoker is an autogenerated mock type for the reconcilerInvoker type
type MockReconcilerInvoker struct {
	mock.Mock
}

// Invoke provides a mock function with given fields: params
func (_m *MockReconcilerInvoker) Invoke(params *InvokeParams) error {
	ret := _m.Called(params)

	var r0 error
	if rf, ok := ret.Get(0).(func(*InvokeParams) error); ok {
		r0 = rf(params)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
