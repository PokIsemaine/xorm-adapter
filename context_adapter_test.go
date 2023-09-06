// Copyright 2023 The casbin Authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package xormadapter

import (
	"context"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/casbin/casbin/v2"
	"github.com/stretchr/testify/assert"
	"xorm.io/xorm"
)

func mockExecuteWithContextTimeOut(ctx context.Context, fn func() error) error {
	done := make(chan error)
	go func() {
		time.Sleep(500 * time.Microsecond)
		done <- fn()
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}

func clearDBPolicy() (*casbin.Enforcer, *ContextAdapter) {
	ca, err := NewContextAdapter("mysql", "root:@tcp(127.0.0.1:3306)/")
	if err != nil {
		panic(err)
	}

	e, err := casbin.NewEnforcer("examples/rbac_model.conf", ca)
	if err != nil {
		panic(err)
	}

	e.ClearPolicy()
	_ = e.SavePolicy()

	return e, ca

	return e, ca
}

func TestContextAdapter_LoadPolicyCtx(t *testing.T) {
	e, ca := clearDBPolicy()

	engine, _ := xorm.NewEngine("mysql", "root:@tcp(127.0.0.1:3306)/casbin")
	policy := &CasbinRule{
		Ptype: "p",
		V0:    "alice",
		V1:    "data1",
		V2:    "read",
	}
	_, err := engine.Insert(policy)
	if err != nil {
		panic(err)
	}

	assert.NoError(t, ca.LoadPolicyCtx(context.Background(), e.GetModel()))
	e, _ = casbin.NewEnforcer(e.GetModel(), ca)
	testGetPolicy(t, e, [][]string{{"alice", "data1", "read"}})

	var p = gomonkey.ApplyFunc(executeWithContext, mockExecuteWithContextTimeOut)
	defer p.Reset()
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Microsecond)
	defer cancel()

	assert.EqualError(t, ca.LoadPolicyCtx(ctx, e.GetModel()), "context deadline exceeded")
}

func TestContextAdapter_SavePolicyCtx(t *testing.T) {
	e, ca := clearDBPolicy()

	e.EnableAutoSave(false)
	_, _ = e.AddPolicy("alice", "data1", "read")
	assert.NoError(t, ca.SavePolicyCtx(context.Background(), e.GetModel()))
	_ = e.LoadPolicy()
	testGetPolicy(t, e, [][]string{{"alice", "data1", "read"}})

	var p = gomonkey.ApplyFunc(executeWithContext, mockExecuteWithContextTimeOut)
	defer p.Reset()
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Microsecond)
	defer cancel()
	assert.EqualError(t, ca.SavePolicyCtx(ctx, e.GetModel()), "context deadline exceeded")
	// Sleep, waiting for the completion of the transaction commit
	time.Sleep(2 * time.Second)
}

func TestContextAdapter_AddPolicyCtx(t *testing.T) {
	e, ca := clearDBPolicy()

	assert.NoError(t, ca.AddPolicyCtx(context.Background(), "p", "p", []string{"alice", "data1", "read"}))
	_ = e.LoadPolicy()
	testGetPolicy(t, e, [][]string{{"alice", "data1", "read"}})

	var p = gomonkey.ApplyFunc(executeWithContext, mockExecuteWithContextTimeOut)
	defer p.Reset()
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Microsecond)
	defer cancel()
	assert.EqualError(t, ca.AddPolicyCtx(ctx, "p", "p", []string{"alice", "data1", "read"}), "context deadline exceeded")
}

func TestContextAdapter_RemovePolicyCtx(t *testing.T) {
	e, ca := clearDBPolicy()

	_ = ca.AddPolicy("p", "p", []string{"alice", "data1", "read"})
	_ = ca.AddPolicy("p", "p", []string{"alice", "data2", "read"})
	assert.NoError(t, ca.RemovePolicyCtx(context.Background(), "p", "p", []string{"alice", "data1", "read"}))
	_ = e.LoadPolicy()
	testGetPolicy(t, e, [][]string{{"alice", "data2", "read"}})

	var p = gomonkey.ApplyFunc(executeWithContext, mockExecuteWithContextTimeOut)
	defer p.Reset()
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Microsecond)
	defer cancel()
	assert.EqualError(t, ca.RemovePolicyCtx(ctx, "p", "p", []string{"alice", "data1", "read"}), "context deadline exceeded")
}

func TestContextAdapter_RemoveFilteredPolicyCtx(t *testing.T) {
	e, ca := clearDBPolicy()

	_ = ca.AddPolicy("p", "p", []string{"alice", "data1", "read"})
	_ = ca.AddPolicy("p", "p", []string{"alice", "data1", "write"})
	_ = ca.AddPolicy("p", "p", []string{"alice", "data2", "read"})
	assert.NoError(t, ca.RemoveFilteredPolicyCtx(context.Background(), "p", "p", 1, "data1"))
	_ = e.LoadPolicy()
	testGetPolicy(t, e, [][]string{{"alice", "data2", "read"}})

	var p = gomonkey.ApplyFunc(executeWithContext, mockExecuteWithContextTimeOut)
	defer p.Reset()
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Microsecond)
	defer cancel()
	assert.EqualError(t, ca.RemoveFilteredPolicyCtx(ctx, "p", "p", 1, "data1"), "context deadline exceeded")
}
