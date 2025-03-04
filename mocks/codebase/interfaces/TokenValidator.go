// Code generated by mockery v2.8.0. DO NOT EDIT.

package mocks

import (
	context "context"

	candishared "github.com/golangid/candi/candishared"

	mock "github.com/stretchr/testify/mock"
)

// TokenValidator is an autogenerated mock type for the TokenValidator type
type TokenValidator struct {
	mock.Mock
}

// ValidateToken provides a mock function with given fields: ctx, token
func (_m *TokenValidator) ValidateToken(ctx context.Context, token string) (*candishared.TokenClaim, error) {
	ret := _m.Called(ctx, token)

	var r0 *candishared.TokenClaim
	if rf, ok := ret.Get(0).(func(context.Context, string) *candishared.TokenClaim); ok {
		r0 = rf(ctx, token)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*candishared.TokenClaim)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, token)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
