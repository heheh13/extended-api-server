package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/spf13/afero"
	"github.com/tamalsaha/DIY-k8s-extended-apiserver/lib/certstore"
	"github.com/tamalsaha/DIY-k8s-extended-apiserver/lib/server"
	"net/http"

	"k8s.io/client-go/util/cert"
	"log"
	"net"
)

func main()  {

	//Creates a directory to store certificates at tmp/extended-apiserver
	fs :=afero.NewOsFs()
	store , err := certstore.NewCertStore(fs,"tmp/extended-api-server")
	if err !=nil{
		log.Fatal(err)
	}

	//creates a new ca and store in tmp/extended-apiserver path

	err = store.NewCA("apiserver")
	if err !=nil{
		log.Fatal(err)
	}
	serverCert, serverKey ,err := store.NewServerCertPair(cert.AltNames{
		IPs:      []net.IP{
			net.ParseIP("127.0.0.1"),
		},
	})
	if err !=nil{
		log.Fatal(err)
	}

	err = store.Write("tls",serverCert,serverKey)

	if err != nil{
		log.Fatal(err)
	}


	clientCert, clientKey , err := store.NewClientCertPair(cert.AltNames{
		DNSNames: []string{"jhon"},
	})

	if err !=nil{
		log.Fatal(err)
	}

	err = store.Write("jhon",clientCert,clientKey)

	if err != nil{
		log.Fatal(err)
	}

	cfg := server.Config{
		Address:     "127.0.0.1:8443",
		CACertFiles: []string{
			store.CertFile("ca"),
		},
		CertFile:   store.CertFile("tls"),
		KeyFile:     store.KeyFile("tls"),
	}

	srv := server.NewGenericServer(cfg)

	r := mux.NewRouter()

	r.HandleFunc("/core/{resource}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w,"resource :%v\n",vars["resource"])

	})

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w,"ok")
	})
	srv.ListenAndServe(r)

}
