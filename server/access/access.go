package access

/* Access rules

A backend can have an ordered list of access rules. Each rule has a CEL
predicate and an action. Rules are evaluated in order, and the first rule whose
predicate evaluates to true decides the outcome. If no rule matches, the
backend's access level is used.

The predicate has access to a `request` variable with the following fields:

	request.method     string               e.g. "GET"
	request.path       string               e.g. "/api/health"
	request.host       string               lowercased, without port
	request.headers    map(string, string)  lowercased names, first value only
	request.query      map(string, string)  first value only
	request.remote_ip  string               e.g. "192.0.2.1"

Example predicates:

	request.path == "/health" || request.path.startsWith("/static/")
	request.method == "GET" && request.path.matches("^/api/v[0-9]+/public/")
	"x-api-key" in request.headers

*/

import (
	"boivie/ubergang/server/models"
	"context"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"strings"

	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/ext"
)

type Decision int

const (
	Authenticated Decision = iota
	Public
	Deny
)

func (d Decision) String() string {
	switch d {
	case Public:
		return "public"
	case Authenticated:
		return "authenticated"
	case Deny:
		return "deny"
	}
	return "unknown"
}

// Upper bound on the evaluation cost of a single predicate, to protect against
// accidentally expensive expressions.
const costLimit = 100_000

// Request is the `request` variable exposed to CEL predicates.
type Request struct {
	Method   string            `cel:"method"`
	Path     string            `cel:"path"`
	Host     string            `cel:"host"`
	Headers  map[string]string `cel:"headers"`
	Query    map[string]string `cel:"query"`
	RemoteIP string            `cel:"remote_ip"`
}

var env = func() *cel.Env {
	e, err := cel.NewEnv(
		ext.NativeTypes(reflect.TypeFor[Request](), ext.ParseStructTags(true)),
		ext.Strings(),
		cel.Variable("request", cel.ObjectType("access.Request")),
	)
	if err != nil {
		panic(err)
	}
	return e
}()

type rule struct {
	program  cel.Program
	decision Decision
}

type Policy struct {
	rules    []rule
	fallback Decision
}

// Fixed returns a policy that always results in `d`.
func Fixed(d Decision) *Policy {
	return &Policy{fallback: d}
}

func decisionFromAction(action models.AccessRuleAction) (Decision, error) {
	switch action {
	case models.AccessRuleAction_ACCESS_RULE_PUBLIC:
		return Public, nil
	case models.AccessRuleAction_ACCESS_RULE_AUTHENTICATED:
		return Authenticated, nil
	case models.AccessRuleAction_ACCESS_RULE_DENY:
		return Deny, nil
	}
	return Deny, fmt.Errorf("invalid action %v", action)
}

func compilePredicate(predicate string) (cel.Program, error) {
	ast, issues := env.Compile(predicate)
	if issues != nil && issues.Err() != nil {
		return nil, issues.Err()
	}
	if ast.OutputType() != cel.BoolType {
		return nil, fmt.Errorf("predicate must evaluate to a bool, got %v", ast.OutputType())
	}
	return env.Program(ast, cel.CostLimit(costLimit))
}

// Compile creates a policy from the backend's access rules and access level.
func Compile(rules []*models.AccessRule, accessLevel models.AccessLevel) (*Policy, error) {
	p := &Policy{fallback: Authenticated}
	if accessLevel == models.AccessLevel_PUBLIC {
		p.fallback = Public
	}
	for i, r := range rules {
		decision, err := decisionFromAction(r.Action)
		if err != nil {
			return nil, fmt.Errorf("rule %d: %w", i+1, err)
		}
		program, err := compilePredicate(r.Predicate)
		if err != nil {
			return nil, fmt.Errorf("rule %d: %w", i+1, err)
		}
		p.rules = append(p.rules, rule{program, decision})
	}
	return p, nil
}

func firstValues(values map[string][]string, lowercase bool) map[string]string {
	m := make(map[string]string, len(values))
	for k, v := range values {
		if len(v) == 0 {
			continue
		}
		if lowercase {
			k = strings.ToLower(k)
		}
		if _, ok := m[k]; !ok {
			m[k] = v[0]
		}
	}
	return m
}

func NewRequest(r *http.Request) *Request {
	host := r.Host
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	remoteIP := r.RemoteAddr
	if h, _, err := net.SplitHostPort(remoteIP); err == nil {
		remoteIP = h
	}
	return &Request{
		Method:   r.Method,
		Path:     r.URL.Path,
		Host:     strings.ToLower(host),
		Headers:  firstValues(r.Header, true),
		Query:    firstValues(r.URL.Query(), false),
		RemoteIP: remoteIP,
	}
}

// Evaluate returns the decision of the first matching rule, or the fallback if
// no rule matches. If a predicate fails to evaluate, an error is returned and
// the request must not be allowed through.
func (p *Policy) Evaluate(ctx context.Context, r *http.Request) (Decision, error) {
	if len(p.rules) == 0 {
		return p.fallback, nil
	}
	vars := map[string]any{"request": NewRequest(r)}
	for i, rule := range p.rules {
		out, _, err := rule.program.ContextEval(ctx, vars)
		if err != nil {
			return Deny, fmt.Errorf("rule %d: %w", i+1, err)
		}
		matched, ok := out.Value().(bool)
		if !ok {
			return Deny, fmt.Errorf("rule %d: predicate returned %v", i+1, out.Type())
		}
		if matched {
			return rule.decision, nil
		}
	}
	return p.fallback, nil
}
