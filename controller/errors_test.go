package controller

import (
	"errors"
	"testing"
)

func TestRequestErrorMessage(t *testing.T) {
	e := &RequestError{Code: -30109, Msg: "session expired"}
	if e.Error() != "Omada controller responded 'session expired' (-30109)" {
		t.Errorf("got %q", e.Error())
	}
}

func TestRequestErrorIs(t *testing.T) {
	if !errors.Is(&RequestError{Code: -30109, Msg: "m"}, ErrLoginFailed) {
		t.Error("-30109 should match ErrLoginFailed")
	}
	if errors.Is(&RequestError{Code: 123, Msg: "m"}, ErrLoginFailed) {
		t.Error("other codes must not match")
	}
	if !errors.Is(ErrLoginFailed, ErrLoginFailed) {
		t.Error("sentinel must match itself")
	}
	if errors.Is(ErrSiteNotFound, ErrLoginFailed) {
		t.Error("unrelated sentinels must not match")
	}
}

func TestValidationError(t *testing.T) {
	var verr *ValidationError
	err := &ValidationError{Msg: "VLAN 1 is protected"}
	if !errors.As(err, &verr) || verr.Msg != "VLAN 1 is protected" {
		t.Error("errors.As failed")
	}
	if verr.Error() != "VLAN 1 is protected" {
		t.Errorf("got %q", verr.Error())
	}
}
