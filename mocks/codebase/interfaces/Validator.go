// Code generated by mockery v2.8.0. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"

// Validator is an autogenerated mock type for the Validator type
type Validator struct {
	mock.Mock
}

// ValidateDocument provides a mock function with given fields: reference, document
func (_m *Validator) ValidateDocument(reference string, document interface{}) error {
	ret := _m.Called(reference, document)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, interface{}) error); ok {
		r0 = rf(reference, document)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ValidateStruct provides a mock function with given fields: data
func (_m *Validator) ValidateStruct(data interface{}) error {
	ret := _m.Called(data)

	var r0 error
	if rf, ok := ret.Get(0).(func(interface{}) error); ok {
		r0 = rf(data)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
