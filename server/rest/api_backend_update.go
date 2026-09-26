package rest

import (
	"boivie/ubergang/server/access"
	"boivie/ubergang/server/api"
	"boivie/ubergang/server/models"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func parseAccessRules(rules []api.ApiAccessRule) ([]*models.AccessRule, error) {
	out := make([]*models.AccessRule, 0, len(rules))
	for i, r := range rules {
		var action models.AccessRuleAction
		switch r.Action {
		case "PUBLIC":
			action = models.AccessRuleAction_ACCESS_RULE_PUBLIC
		case "AUTHENTICATED":
			action = models.AccessRuleAction_ACCESS_RULE_AUTHENTICATED
		case "DENY":
			action = models.AccessRuleAction_ACCESS_RULE_DENY
		default:
			return nil, fmt.Errorf("rule %d: invalid action %q", i+1, r.Action)
		}
		out = append(out, &models.AccessRule{Predicate: r.Predicate, Action: action})
	}
	// Validate that all predicates compile. The access level doesn't matter here.
	if _, err := access.Compile(out, models.AccessLevel_NORMAL); err != nil {
		return nil, err
	}
	return out, nil
}

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

	var accessRules []*models.AccessRule
	if req.AccessRules != nil {
		accessRules, err = parseAccessRules(*req.AccessRules)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
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

		if req.AccessRules != nil {
			old.AccessRules = accessRules
		}

		if req.JsScript != "" {
			old.ScriptHandler = &models.ScriptHandler{
				JsScript: req.JsScript,
			}
		} else {
			old.ScriptHandler = nil
		}

		if req.PinnedCertificateFingerprint != nil {
			old.PinnedCertificateFingerprint = *req.PinnedCertificateFingerprint
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
