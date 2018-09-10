package main

import (
	"flag"
	"net/http/httputil"
	"net/http"
	"fmt"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/api/core/v1"
	"github.com/golang/glog"
	)

type Endpoint struct {
	Address string
	Port    int32
}

func (e *Endpoint) FullAddress() string {
	return fmt.Sprintf("%s:%d", e.Address, e.Port)
}

var opts struct {
	ListenPort  int
	Namespace   string
	ServiceName string
	ServicePort string
	Kubeconfig  string
}

func main() {
	flag.IntVar(&opts.ListenPort, "port", 8080, "HTTP port to listen on")
	flag.StringVar(&opts.Namespace, "namespace", "default", "namespace in which the target service is defined")
	flag.StringVar(&opts.ServiceName, "service", "", "service to proxy to")
	flag.StringVar(&opts.ServicePort, "target-port", "http", "port name")
	flag.StringVar(&opts.Kubeconfig, "kubeconfig", "", "kubeconfig file to use")

	flag.Parse()

	var config *rest.Config
	var err error
	var client kubernetes.Interface

	if opts.Kubeconfig == "" {
		glog.Infof("using in-cluster configuration")
		config, err = rest.InClusterConfig()
	} else {
		glog.Infof("using configuration from '%s'", opts.Kubeconfig)
		config, err = clientcmd.BuildConfigFromFlags("", opts.Kubeconfig)
	}

	if err != nil {
		panic(err)
	}

	client = kubernetes.NewForConfigOrDie(config)

	w, err := client.CoreV1().Endpoints(opts.Namespace).Watch(meta_v1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", opts.ServiceName),
	})

	if err != nil {
		panic(err)
	}

	ep := Endpoint{}
	WatchPrimaryEndpoint(w, &ep)

	proxy := httputil.ReverseProxy{
		Director: func(r *http.Request) {
			r.URL.Scheme = "http"
			r.URL.Host = ep.FullAddress()
		},
	}

	addr := fmt.Sprintf(":%d", opts.ListenPort)

	glog.Infof("listening at %s", addr)
	err = http.ListenAndServe(addr, &proxy)
	if err != nil {
		panic(err)
	}
}

func WatchPrimaryEndpoint(w watch.Interface, ep *Endpoint) {
	c := w.ResultChan()
	for ev := range c {
		if ev.Type != watch.Added && ev.Type != watch.Modified {
			continue
		}

		endpoint := ev.Object.(*v1.Endpoints)

		if len(endpoint.Subsets) == 0 || len(endpoint.Subsets[0].Addresses) == 0 {
			glog.Warningf("service '%s' has no endpoints", opts.ServiceName)
			continue
		}

		if ep.Address == "" {
			if len(endpoint.Subsets) > 0 && len(endpoint.Subsets[0].Addresses) > 0 {
				foundPort := int32(80)

				for _, port := range endpoint.Subsets[0].Ports {
					if port.Name == "http" {
						foundPort = port.Port
					}
				}

				ep.Address = endpoint.Subsets[0].Addresses[0].IP
				ep.Port = foundPort

				glog.Infof("initializing endpoint with '%s'", ep.FullAddress())
			}
		} else {
			endpointRemains := false
			for _, addr := range endpoint.Subsets[0].Addresses {
				if addr.IP == ep.Address {
					endpointRemains = true
					glog.Infof("endpoint '%s' is still available", ep.Address)
					continue
				}
			}

			if !endpointRemains {
				previous := *ep
				foundPort := int32(80)

				for _, port := range endpoint.Subsets[0].Ports {
					if port.Name == "http" {
						foundPort = port.Port
					}
				}

				ep.Address = endpoint.Subsets[0].Addresses[0].IP
				ep.Port = foundPort

				glog.Infof("endpoint '%s' is not available any more. Switching to '%s'", previous.FullAddress(), ep.FullAddress())
			}
		}
	}
}
