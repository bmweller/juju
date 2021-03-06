// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package runcommands_test

import (
	"github.com/juju/testing"
	"github.com/juju/utils/v2/exec"

	"github.com/juju/juju/worker/uniter/operation"
	"github.com/juju/juju/worker/uniter/runner"
	"github.com/juju/juju/worker/uniter/runner/context"
)

type mockRunnerFactory struct {
	runner.Factory
	newCommandRunner func(context.CommandInfo) (runner.Runner, error)
}

func (f *mockRunnerFactory) NewCommandRunner(info context.CommandInfo) (runner.Runner, error) {
	return f.newCommandRunner(info)
}

type mockRunner struct {
	runner.Runner
	runCommands func(string, runner.RunLocation) (*exec.ExecResponse, error)
}

func (r *mockRunner) Context() runner.Context {
	return &mockRunnerContext{}
}

func (r *mockRunner) RunCommands(commands string, runLocation runner.RunLocation) (*exec.ExecResponse, error) {
	return r.runCommands(commands, runLocation)
}

type mockRunnerContext struct {
	runner.Context
}

func (*mockRunnerContext) Prepare() error {
	return nil
}

type mockCallbacks struct {
	testing.Stub
	operation.Callbacks
}

func (c *mockCallbacks) SetExecutingStatus(status string) error {
	c.MethodCall(c, "SetExecutingStatus", status)
	return c.NextErr()
}
