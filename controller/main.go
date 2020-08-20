package main

import (
	"context"
	controller "controller/resource"
	"flag"
	"log"
	"net/http"
	"os"
	"strconv"

	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	serverv3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	testv3 "github.com/envoyproxy/go-control-plane/pkg/test/v3"
)

var (
	l controller.Logger

	port    uint
	CtrPort uint
	mode    string

	nodeID string
)

func init() {
	l = controller.Logger{}

	flag.BoolVar(&l.Debug, "debug", false, "Enable xDS server debug logging")

	// The port that this xDS server listens on
	flag.UintVar(&port, "port", 18000, "xDS management server port")

	// Tell Envoy to use this Node ID
	flag.StringVar(&nodeID, "nodeID", "test-id", "Node ID")
}

func main() {
	flag.Parse()

	// Create a cache
	cache := cachev3.NewSnapshotCache(false, cachev3.IDHash{}, l)

	// Create the snapshot that we'll serve to Envoy
	snapshot := controller.GenerateSnapshot()
	if err := snapshot.Consistent(); err != nil {
		l.Errorf("snapshot inconsistency: %+v\n%+v", snapshot, err)
		os.Exit(1)
	}
	l.Debugf("will serve snapshot %+v", snapshot)

	// Add the snapshot to the cache
	if err := cache.SetSnapshot(nodeID, snapshot); err != nil {
		l.Errorf("snapshot error %q for %+v", err, snapshot)
		os.Exit(1)
	}

	// Run the xDS server
	ctx := context.Background()
	cb := &testv3.Callbacks{Debug: l.Debug}
	srv := serverv3.NewServer(ctx, cache, cb)
	go controller.RunServer(ctx, srv, port)

	http.HandleFunc("/listen", func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		name := r.Form.Get("name")
		port, err := strconv.Atoi(r.Form.Get("port"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		controller.SetListener(name, port)
		controller.IncVersion()
		snapshot := controller.GenerateSnapshot()
		if err := snapshot.Consistent(); err != nil {
			l.Errorf("snapshot inconsistency: %+v\n%+v", snapshot, err)
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		l.Debugf("will serve snapshot %+v", snapshot)

		// Add the snapshot to the cache
		if err := cache.SetSnapshot(nodeID, snapshot); err != nil {
			l.Errorf("snapshot error %q for %+v", err, snapshot)
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("listen set ok"))
	})

	http.HandleFunc("/upstream", func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		host := r.Form.Get("host")
		port, err := strconv.Atoi(r.Form.Get("port"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		controller.SetUpstream(host, port)
		controller.IncVersion()
		snapshot := controller.GenerateSnapshot()
		if err := snapshot.Consistent(); err != nil {
			l.Errorf("snapshot inconsistency: %+v\n%+v", snapshot, err)
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		l.Debugf("will serve snapshot %+v", snapshot)

		// Add the snapshot to the cache
		if err := cache.SetSnapshot(nodeID, snapshot); err != nil {
			l.Errorf("snapshot error %q for %+v", err, snapshot)
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("listen set ok"))
	})

	http.HandleFunc("/delay", func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		duration, err := strconv.Atoi(r.Form.Get("duration"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		controller.SetDelay(duration)
		controller.IncVersion()
		snapshot := controller.GenerateSnapshot()
		if err := snapshot.Consistent(); err != nil {
			l.Errorf("snapshot inconsistency: %+v\n%+v", snapshot, err)
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		l.Debugf("will serve snapshot %+v", snapshot)

		// Add the snapshot to the cache
		if err := cache.SetSnapshot(nodeID, snapshot); err != nil {
			l.Errorf("snapshot error %q for %+v", err, snapshot)
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("listen set ok"))
	})
	log.Fatal(http.ListenAndServe(":15000", nil))
}
