package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	gotgproto "github.com/krau/mygotg"
)

type loginFlow struct {
	id     string
	phone  string
	ctx    context.Context
	cancel context.CancelFunc

	mu     sync.Mutex
	state  string
	errMsg string
	codeCh chan string
	pwdCh  chan string
}

func newLoginFlow(parent context.Context, id, phone string) *loginFlow {
	ctx, cancel := context.WithTimeout(parent, 5*time.Minute)
	return &loginFlow{
		id:     id,
		phone:  phone,
		ctx:    ctx,
		cancel: cancel,
		state:  "starting",
		codeCh: make(chan string, 1),
		pwdCh:  make(chan string, 1),
	}
}

func (f *loginFlow) setState(state, errMsg string) {
	f.mu.Lock()
	f.state = state
	f.errMsg = errMsg
	f.mu.Unlock()
}

func (f *loginFlow) status() (string, string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.state, f.errMsg
}

func (f *loginFlow) markDone() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.state == "cancelled" {
		return false
	}
	f.state = "done"
	return true
}

func (f *loginFlow) markFailed(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.state == "cancelled" {
		return
	}
	f.state = "failed"
	f.errMsg = err.Error()
}

func (f *loginFlow) submitCode(code string) error {
	state, _ := f.status()
	if state != "code_required" {
		return fmt.Errorf("login flow is not waiting for a code (state %s)", state)
	}
	select {
	case f.codeCh <- code:
		return nil
	case <-f.ctx.Done():
		return f.ctx.Err()
	}
}

func (f *loginFlow) submitPassword(password string) error {
	state, _ := f.status()
	if state != "password_required" {
		return fmt.Errorf("login flow is not waiting for a password (state %s)", state)
	}
	select {
	case f.pwdCh <- password:
		return nil
	case <-f.ctx.Done():
		return f.ctx.Err()
	}
}

type webConversator struct {
	flow *loginFlow
}

func (c *webConversator) AskPhoneNumber() (string, error) { return c.flow.phone, nil }

func (c *webConversator) AskCode() (string, error) {
	c.flow.setState("code_required", "")
	select {
	case <-c.flow.ctx.Done():
		return "", c.flow.ctx.Err()
	case code := <-c.flow.codeCh:
		c.flow.setState("verifying", "")
		return code, nil
	}
}

func (c *webConversator) AskPassword() (string, error) {
	c.flow.setState("password_required", "")
	select {
	case <-c.flow.ctx.Done():
		return "", c.flow.ctx.Err()
	case password := <-c.flow.pwdCh:
		c.flow.setState("verifying", "")
		return password, nil
	}
}

func (c *webConversator) AuthStatus(status gotgproto.AuthStatus) {
	switch status.Event {
	case gotgproto.AuthStatusPhoneCodeAsked:
		c.flow.setState("code_required", "")
	case gotgproto.AuthStatusPasswordAsked:
		c.flow.setState("password_required", "")
	case gotgproto.AuthStatusPhoneCodeRetrial, gotgproto.AuthStatusPasswordRetrial, gotgproto.AuthStatusPhoneRetrial:
		c.flow.setState("code_required", fmt.Sprintf("incorrect credential, attempts left: %d", status.AttemptsLeft))
	case gotgproto.AuthStatusPhoneCodeFailed, gotgproto.AuthStatusPasswordFailed:
		c.flow.setState("failed", "credential verification failed")
	case gotgproto.AuthStatusSuccess:
		// The flow is marked done only after the client is fully started and
		// the enabled flag has been persisted.
		c.flow.setState("verifying", "")
	}
}
