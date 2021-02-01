package provider

import (
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/virtual-kubelet/virtual-kubelet/errdefs"
	nodeapi "github.com/virtual-kubelet/virtual-kubelet/node/api"
)

func (p *p) GetContainerLogsHandler(w http.ResponseWriter, r *http.Request) {
	handleError(func(w http.ResponseWriter, req *http.Request) error {
		vars := mux.Vars(r)
		if len(vars) != 3 {
			return errdefs.NotFound("not found")
		}

		namespace := vars["namespace"]
		pod := vars["pod"]
		container := vars["container"]

		query := r.URL.Query()
		opts, err := parseLogOptions(query)
		if err != nil {
			return err
		}

		r.Header.Set("Transfer-Encoding", "chunked")

		logsReader, cancel, err := journalReader(namespace, pod, container, opts)
		if err != nil {
			return errors.Wrap(err, "failed to get systemd journal logs reader")
		}
		defer logsReader.Close()
		defer cancel()

		// ResponseWriter must be flushed after each write.
		if _, ok := w.(writeFlusher); !ok {
			log.Warn("HTTP response writer does not support flushes")
		}
		fw := flushOnWrite(w)

		if !opts.Follow {
			io.Copy(fw, logsReader)
			return nil
		}

		// If in follow mode, follow until interrupted.
		untilTime := make(chan time.Time, 1)
		errChan := make(chan error, 1)

		go func(w io.Writer, errChan chan error) {
			err := journalFollow(untilTime, logsReader, w)
			if err != nil && err != ErrExpired {
				err = errors.Wrap(err, "failed to follow systemd journal logs")
			}
			errChan <- err
		}(fw, errChan)

		// Stop following logs if request context is completed.
		select {
		case err := <-errChan:
			return err
		case <-r.Context().Done():
			close(untilTime)
		}
		return nil
	})(w, r)
}

func (p *p) notFound(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
}

func parseLogOptions(q url.Values) (opts nodeapi.ContainerLogOpts, err error) {
	if tailLines := q.Get("tailLines"); tailLines != "" {
		opts.Tail, err = strconv.Atoi(tailLines)
		if err != nil {
			return opts, errdefs.AsInvalidInput(errors.Wrap(err, "could not parse \"tailLines\""))
		}
		if opts.Tail < 0 {
			return opts, errdefs.InvalidInputf("\"tailLines\" is %d", opts.Tail)
		}
	}
	if follow := q.Get("follow"); follow != "" {
		opts.Follow, err = strconv.ParseBool(follow)
		if err != nil {
			return opts, errdefs.AsInvalidInput(errors.Wrap(err, "could not parse \"follow\""))
		}
	}
	if limitBytes := q.Get("limitBytes"); limitBytes != "" {
		opts.LimitBytes, err = strconv.Atoi(limitBytes)
		if err != nil {
			return opts, errdefs.AsInvalidInput(errors.Wrap(err, "could not parse \"limitBytes\""))
		}
		if opts.LimitBytes < 1 {
			return opts, errdefs.InvalidInputf("\"limitBytes\" is %d", opts.LimitBytes)
		}
	}
	if previous := q.Get("previous"); previous != "" {
		opts.Previous, err = strconv.ParseBool(previous)
		if err != nil {
			return opts, errdefs.AsInvalidInput(errors.Wrap(err, "could not parse \"previous\""))
		}
	}
	if sinceSeconds := q.Get("sinceSeconds"); sinceSeconds != "" {
		opts.SinceSeconds, err = strconv.Atoi(sinceSeconds)
		if err != nil {
			return opts, errdefs.AsInvalidInput(errors.Wrap(err, "could not parse \"sinceSeconds\""))
		}
		if opts.SinceSeconds < 1 {
			return opts, errdefs.InvalidInputf("\"sinceSeconds\" is %d", opts.SinceSeconds)
		}
	}
	if sinceTime := q.Get("sinceTime"); sinceTime != "" {
		opts.SinceTime, err = time.Parse(time.RFC3339, sinceTime)
		if err != nil {
			return opts, errdefs.AsInvalidInput(errors.Wrap(err, "could not parse \"sinceTime\""))
		}
		if opts.SinceSeconds > 0 {
			return opts, errdefs.InvalidInput("both \"sinceSeconds\" and \"sinceTime\" are set")
		}
	}
	if timestamps := q.Get("timestamps"); timestamps != "" {
		opts.Timestamps, err = strconv.ParseBool(timestamps)
		if err != nil {
			return opts, errdefs.AsInvalidInput(errors.Wrap(err, "could not parse \"timestamps\""))
		}
	}
	return opts, nil
}

type handlerFunc func(http.ResponseWriter, *http.Request) error

func handleError(f handlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		err := f(w, req)
		if err == nil {
			return
		}

		code := httpStatusCode(err)
		w.WriteHeader(code)
		io.WriteString(w, err.Error())
	}
}

func flushOnWrite(w io.Writer) io.Writer {
	if fw, ok := w.(writeFlusher); ok {
		return &flushWriter{fw}
	}
	return w
}

type flushWriter struct {
	w writeFlusher
}

type writeFlusher interface {
	Flush()
	Write([]byte) (int, error)
}

func (fw *flushWriter) Write(p []byte) (int, error) {
	n, err := fw.w.Write(p)
	if n > 0 {
		fw.w.Flush()
	}
	return n, err
}

func httpStatusCode(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
	case errdefs.IsNotFound(err):
		return http.StatusNotFound
	case errdefs.IsInvalidInput(err):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
