package scripting

import (
	"boivie/ubergang/server/mqtt"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/dop251/goja"
)

/* Scripting support

An example script could look like:
```
// Register a route with a specific method and path
proxy.get('/api/health', (req, res) => {
    res.status(200).send('Proxy is Healthy');
});

// Use wildcards or logic to decide whether to proxy or block
proxy.all('/legacy/*', (req, res) => {
    // Explicitly block (403)
    res.status(403).json({ error: 'Legacy access is disabled' });
});

proxy.all('/secure/*', (req, res) => {
    const apiKey = req.headers['X-API-KEY'];

    if (apiKey === 'secret-token') {
        // Forward to the backend
        res.proxy();
    } else {
        res.status(401).send('Unauthorized');
    }
});

// If a request comes in that is unmapped,
// and no handler calls res.proxy() or res.send(),
// the system defaults to 404.
````


*/

type Route struct {
	Method  string // Empty string = all methods.
	Pattern *regexp.Regexp
	Params  []string      // Stores names like ["id", "postID"]
	Handler goja.Callable // The JS function to run
}

type JSProxy struct {
	runtime *goja.Runtime
	program *goja.Program
	routes  []Route
	mqttPub mqtt.MQTTPublisher
}

func NewJSProxy(program *goja.Program, mqttPublisher mqtt.MQTTPublisher) (*JSProxy, error) {
	p := &JSProxy{
		runtime: goja.New(),
		program: program,
		routes:  make([]Route, 0),
		mqttPub: mqttPublisher,
	}

	p.registerHandlers()

	_, err := p.runtime.RunProgram(p.program)
	if err != nil {
		return nil, err
	}

	return p, nil
}

var paramRegex = regexp.MustCompile(`:([a-zA-Z0-9_]+)`)

func convertToRegex(path string) (*regexp.Regexp, []string) {
	var paramNames []string

	// 1. Find all :names and save them
	matches := paramRegex.FindAllStringSubmatch(path, -1)
	for _, m := range matches {
		paramNames = append(paramNames, m[1])
	}

	// 2. Replace :names with a regex capture group ([^/]+)
	// QuoteMeta escapes our : symbols, so we need to account for that
	// when replacing. A simpler way is to replace :name with the group:
	pattern := paramRegex.ReplaceAllString(path, `([^/]+)`)

	// 3. Handle wildcards (*)
	pattern = strings.ReplaceAll(pattern, "*", "(.*)")

	// 4. Wrap with start/end anchors
	return regexp.MustCompile("^" + pattern + "$"), paramNames
}

func headersToMap(h http.Header) map[string]string {
	m := make(map[string]string)
	for k, v := range h {
		if len(v) > 0 {
			m[k] = v[0]
		}
	}
	return m
}

func (p *JSProxy) registerHandlers() {
	// Expose the "proxy" object to JS
	proxyObj := p.runtime.NewObject()

	addRoute := func(method string) func(call goja.FunctionCall) goja.Value {
		return func(call goja.FunctionCall) goja.Value {
			pathPattern := call.Argument(0).String()
			jsHandler, _ := goja.AssertFunction(call.Argument(1))

			// Convert '/users/*' to a regex like '^/users/.*$'
			regexPattern, params := convertToRegex(pathPattern)

			p.routes = append(p.routes, Route{
				Method:  method,
				Pattern: regexPattern,
				Params:  params,
				Handler: jsHandler,
			})
			return goja.Undefined()
		}
	}

	_ = proxyObj.Set("get", addRoute("GET"))
	_ = proxyObj.Set("post", addRoute("POST"))
	_ = proxyObj.Set("all", addRoute(""))

	_ = p.runtime.Set("proxy", proxyObj)

	// Expose MQTT API if publisher is available
	if p.mqttPub != nil {
		mqttObj := p.runtime.NewObject()

		// mqtt.publish(topic, payload, options)
		_ = mqttObj.Set("publish", func(call goja.FunctionCall) goja.Value {
			if len(call.Arguments) < 2 {
				panic(p.runtime.NewTypeError("mqtt.publish requires topic and payload"))
			}

			topic := call.Argument(0).String()
			payload := call.Argument(1).String()

			// Parse optional third argument (options object)
			qos := byte(0)
			retain := false

			if len(call.Arguments) >= 3 && !goja.IsUndefined(call.Argument(2)) && !goja.IsNull(call.Argument(2)) {
				opts := call.Argument(2).ToObject(p.runtime)
				if opts != nil {
					if qosVal := opts.Get("qos"); qosVal != nil && !goja.IsUndefined(qosVal) {
						qos = byte(qosVal.ToInteger())
					}
					if retainVal := opts.Get("retain"); retainVal != nil && !goja.IsUndefined(retainVal) {
						retain = retainVal.ToBoolean()
					}
				}
			}

			err := p.mqttPub.Publish(topic, []byte(payload), qos, retain)
			if err != nil {
				panic(p.runtime.NewGoError(err))
			}

			return goja.Undefined()
		})

		// mqtt.isConnected()
		_ = mqttObj.Set("isConnected", func(call goja.FunctionCall) goja.Value {
			return p.runtime.ToValue(p.mqttPub.IsConnected())
		})

		_ = p.runtime.Set("mqtt", mqttObj)
	}
}

// Returns true if the request should be forwarded to the backend (proxied).
func (p *JSProxy) MatchAndExecute(w http.ResponseWriter, r *http.Request) bool {
	for _, route := range p.routes {
		if route.Method == r.Method || route.Method == "" {
			matches := route.Pattern.FindStringSubmatch(r.URL.Path)
			if matches != nil {
				// matches[0] is the full string, matches[1:] are the groups
				paramsMap := make(map[string]string)

				for i, value := range matches[1:] {
					if i < len(route.Params) {
						key := route.Params[i]
						paramsMap[key] = value
					}
				}

				// Now pass paramsMap into the JS req object
				return p.executeHandler(w, r, route.Handler, paramsMap)
			}
		}
	}
	// Fallback to 404 (Outcome 1)
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte("Not Found"))
	return false
}

func (p *JSProxy) executeHandler(w http.ResponseWriter, r *http.Request, handler goja.Callable, params map[string]string) bool {
	// State to track the outcome
	shouldProxy := false
	responseSent := false

	// Create the 'req' object
	reqObj := p.runtime.NewObject()
	_ = reqObj.Set("method", r.Method)
	_ = reqObj.Set("path", r.URL.Path)
	_ = reqObj.Set("params", params)
	_ = reqObj.Set("headers", headersToMap(r.Header))

	// Optional: Parse Query String
	queryMap := make(map[string]string)
	for k, v := range r.URL.Query() {
		queryMap[k] = v[0]
	}
	_ = reqObj.Set("query", queryMap)

	// Create the 'res' object
	resObj := p.runtime.NewObject()
	statusCode := http.StatusOK

	// res.status(code)
	_ = resObj.Set("status", func(call goja.FunctionCall) goja.Value {
		statusCode = int(call.Argument(0).ToInteger())
		return resObj // Allow chaining: res.status(200).send(...)
	})

	// res.send(body)
	_ = resObj.Set("send", func(call goja.FunctionCall) goja.Value {
		if responseSent || shouldProxy {
			return goja.Undefined()
		}
		responseSent = true
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(call.Argument(0).String()))
		return goja.Undefined()
	})

	// res.json(obj)
	_ = resObj.Set("json", func(call goja.FunctionCall) goja.Value {
		if responseSent || shouldProxy {
			return goja.Undefined()
		}
		responseSent = true
		data, _ := json.Marshal(call.Argument(0).Export())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_, _ = w.Write(data)
		return goja.Undefined()
	})

	// res.proxy() - The signal to take Outcome 3
	_ = resObj.Set("proxy", func(call goja.FunctionCall) goja.Value {
		shouldProxy = true
		return goja.Undefined()
	})

	// Execute the handler
	_, err := handler(goja.Undefined(), reqObj, resObj)
	if err != nil {
		// Handle JS runtime errors as 500s
		if !responseSent {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprintf(w, "JS Error: %v", err)
		}
		return false
	}

	return shouldProxy
}
