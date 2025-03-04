// Code generated by mockery v2.8.0. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	mongo "go.mongodb.org/mongo-driver/mongo"
)

// MongoDatabase is an autogenerated mock type for the MongoDatabase type
type MongoDatabase struct {
	mock.Mock
}

// Disconnect provides a mock function with given fields: ctx
func (_m *MongoDatabase) Disconnect(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Health provides a mock function with given fields:
func (_m *MongoDatabase) Health() map[string]error {
	ret := _m.Called()

	var r0 map[string]error
	if rf, ok := ret.Get(0).(func() map[string]error); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]error)
		}
	}

	return r0
}

// ReadDB provides a mock function with given fields:
func (_m *MongoDatabase) ReadDB() *mongo.Database {
	ret := _m.Called()

	var r0 *mongo.Database
	if rf, ok := ret.Get(0).(func() *mongo.Database); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*mongo.Database)
		}
	}

	return r0
}

// WriteDB provides a mock function with given fields:
func (_m *MongoDatabase) WriteDB() *mongo.Database {
	ret := _m.Called()

	var r0 *mongo.Database
	if rf, ok := ret.Get(0).(func() *mongo.Database); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*mongo.Database)
		}
	}

	return r0
}
