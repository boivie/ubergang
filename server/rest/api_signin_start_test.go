package rest

import (
	"boivie/ubergang/server/api"
	"testing"
)

func TestSigninStart(t *testing.T) {
	f := CreateFixture(t)
	resp := &api.ApiStartSigninResponse{}
	f.request("GET", "/api/signin/start", nil, nil, resp)
	if resp.Token == "" {
		t.Errorf("Invalid token")
	} else if resp.AssertionRequest.Challenge == "" {
		t.Errorf("Invalid challenge")
	}
}
