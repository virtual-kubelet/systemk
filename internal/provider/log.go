package provider

import (
	"context"
	"io"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/virtual-kubelet/virtual-kubelet/node/api"
)

func (p *p) GetContainerLogsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if len(vars) != 3 {
		p.NotFound(w, r)
		return
	}

	namespace := vars["namespace"]
	pod := vars["pod"]
	container := vars["container"]

	opts := api.ContainerLogOpts{}
	q := r.URL.Query()
	// various options
	tailLines := q.Get("tailLines")
	if tailLines != "" {
		t, err := strconv.Atoi(tailLines)
		if err != nil {
			opts.Tail = t
		}
	}

	ctx := context.TODO()
	logs, err := p.GetContainerLogs(ctx, namespace, pod, container, opts)
	if err != nil {
		io.WriteString(w, err.Error())
		return
	}
	defer logs.Close()

	io.Copy(w, logs)
}

func (p *p) NotFound(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
}
