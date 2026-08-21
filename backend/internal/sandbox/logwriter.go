package sandbox

import "log"

type logWriter struct {
	prefix string
}

func (w *logWriter) Write(p []byte) (int, error) {
	log.Print(w.prefix + string(p))
	return len(p), nil
}
