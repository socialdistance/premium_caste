// Code generated by mockery v2.53.3. DO NOT EDIT.

package mocks

import (
	context "context"

	uuid "github.com/google/uuid"
	mock "github.com/stretchr/testify/mock"
)

// UserSaver is an autogenerated mock type for the UserSaver type
type UserSaver struct {
	mock.Mock
}

// SaveUser provides a mock function with given fields: ctx, name, email, phone, password, permissionId, basketId
func (_m *UserSaver) SaveUser(ctx context.Context, name string, email string, phone string, password []byte, permissionId int, basketId uuid.UUID) (int64, error) {
	ret := _m.Called(ctx, name, email, phone, password, permissionId, basketId)

	if len(ret) == 0 {
		panic("no return value specified for SaveUser")
	}

	var r0 int64
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string, []byte, int, uuid.UUID) (int64, error)); ok {
		return rf(ctx, name, email, phone, password, permissionId, basketId)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string, []byte, int, uuid.UUID) int64); ok {
		r0 = rf(ctx, name, email, phone, password, permissionId, basketId)
	} else {
		r0 = ret.Get(0).(int64)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string, string, []byte, int, uuid.UUID) error); ok {
		r1 = rf(ctx, name, email, phone, password, permissionId, basketId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewUserSaver creates a new instance of UserSaver. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewUserSaver(t interface {
	mock.TestingT
	Cleanup(func())
}) *UserSaver {
	mock := &UserSaver{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
