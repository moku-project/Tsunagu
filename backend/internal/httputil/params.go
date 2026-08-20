package httputil

import "net/http"

func RequireParams(w http.ResponseWriter, r *http.Request, names ...string) (map[string]string, bool) {
	if err := r.ParseForm(); err != nil {
		Error(w, "invalid form data", http.StatusBadRequest)
		return nil, false
	}

	values := make(map[string]string, len(names))
	missing := make([]string, 0)

	for _, name := range names {
		v := r.Form.Get(name)
		if v == "" {
			missing = append(missing, name)
			continue
		}
		values[name] = v
	}

	if len(missing) > 0 {
		msg := missing[0]
		for _, m := range missing[1:] {
			msg += ", " + m
		}
		Error(w, msg+" is required", http.StatusBadRequest)
		return nil, false
	}

	return values, true
}