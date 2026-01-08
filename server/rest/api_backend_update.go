package rest

import (
	"boivie/ubergang/server/api"
	"boivie/ubergang/server/models"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *ApiModule) handleBackendUpdate(w http.ResponseWriter, r *http.Request) {
	user, _, err := s.session.GetAndValidate(w, r)
	if err != nil {
		return
	}

	if !user.IsAdmin {
		http.Error(w, "Not authorized", http.StatusForbidden)
		return
	}

	var req api.ApiUpdateBackendRequest
	err = parseJsonRequest(w, r, &req)
	if err != nil {
		return
	}

	fqdn := strings.ToLower(mux.Vars(r)["fqdn"])

	err = s.db.UpdateBackend(fqdn, func(old *models.Backend) (*models.Backend, error) {
		now := time.Now()
		if old == nil {
			old = &models.Backend{
				Fqdn:        fqdn,
				CreatedAt:   timestamppb.New(now),
				UpdatedAt:   timestamppb.New(now),
				AccessLevel: models.AccessLevel_NORMAL,
			}
		}
		if req.UpstreamUrl != nil {
			old.UpstreamUrl = *req.UpstreamUrl
		}

		if req.Headers != nil {
			old.Headers = make([]*models.Header, 0)
			for _, h := range *req.Headers {
				old.Headers = append(old.Headers, &models.Header{
					Name:  h.Name,
					Value: h.Value,
				})
			}
		}

		if req.AccessLevel != nil {
			switch *req.AccessLevel {
			case "PUBLIC":
				old.AccessLevel = models.AccessLevel_PUBLIC
			case "NORMAL":
				old.AccessLevel = models.AccessLevel_NORMAL
			default:
				old.AccessLevel = models.AccessLevel_NORMAL
			}
		}

		if req.JsScript != "" {
			old.ScriptHandler = &models.ScriptHandler{
				JsScript: req.JsScript,
			}
		} else {
			old.ScriptHandler = nil
		}

		old.UpdatedAt = timestamppb.New(now)
		return old, nil
	})

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	jsonify(w, api.ApiUpdateBackendResponse{})
}
