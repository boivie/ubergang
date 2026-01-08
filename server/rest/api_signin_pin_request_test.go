package rest

import (
	"boivie/ubergang/server/api"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSigninPinRequest(t *testing.T) {
	t.Run("returns error for non-existing user", func(t *testing.T) {
		f := CreateFixture(t)
		req := &api.ApiRequestSigninPinRequest{
			Email: "notexisting@example.com",
		}
		resp := &api.ApiRequestSigninPinResponse{}
		f.request("POST", "/api/signin/pin/request", req, nil, resp)
		assert.NotNil(t, resp.Error, "Expected an error for non-existing email")
		assert.True(t, resp.Error.InvalidEmail, "Expected InvalidEmail error")
	})

	t.Run("returns error for empty email", func(t *testing.T) {
		f := CreateFixture(t)
		req := &api.ApiRequestSigninPinRequest{
			Email: "",
		}
		resp := &api.ApiRequestSigninPinResponse{}
		f.request("POST", "/api/signin/pin/request", req, nil, resp)
		assert.NotNil(t, resp.Error, "Expected an error for empty email")
		assert.True(t, resp.Error.InvalidEmail, "Expected InvalidEmail error")
	})

	t.Run("returns ID for existing user", func(t *testing.T) {
		f := CreateFixture(t)
		f.CreateUser("test@example.com")
		req := &api.ApiRequestSigninPinRequest{
			Email: "test@example.com",
		}
		resp := &api.ApiRequestSigninPinResponse{}
		f.request("POST", "/api/signin/pin/request", req, nil, resp)
		assert.Nil(t, resp.Error, "Did not expect an error")
		assert.NotEmpty(t, resp.ID, "Expected a non-empty ID")
	})
}
