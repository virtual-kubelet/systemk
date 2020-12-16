package systemd

import (
	"context"
	"io"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/virtual-kubelet/virtual-kubelet/node/api"
	"k8s.io/klog/v2"
)

func (p *P) GetContainerLogsHandler(w http.ResponseWriter, r *http.Request) {
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
		klog.Info(err)
		io.WriteString(w, err.Error())
		return
	}
	defer logs.Close()

	io.Copy(w, logs)
}

func (p *P) NotFound(w http.ResponseWriter, _ *http.Request) {
	klog.Info(http.StatusText(http.StatusNotFound))
	http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
}
