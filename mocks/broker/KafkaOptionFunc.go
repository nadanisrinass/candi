// Code generated by mockery v2.8.0. DO NOT EDIT.

package mocks

import (
	broker "github.com/golangid/candi/broker"
	mock "github.com/stretchr/testify/mock"
)

// KafkaOptionFunc is an autogenerated mock type for the KafkaOptionFunc type
type KafkaOptionFunc struct {
	mock.Mock
}

// Execute provides a mock function with given fields: _a0
func (_m *KafkaOptionFunc) Execute(_a0 *broker.KafkaBroker) {
	_m.Called(_a0)
}
