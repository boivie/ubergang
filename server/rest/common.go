package rest

import (
	"encoding/json"
	"net/http"
)

func jsonify(w http.ResponseWriter, response interface{}) {
	w.Header().Add("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func parseJsonRequest(w http.ResponseWriter, r *http.Request, req interface{}) error {
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return err
	}
	return nil
}
