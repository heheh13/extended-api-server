package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/spf13/afero"
	"github.com/tamalsaha/DIY-k8s-extended-apiserver/lib/certstore"
	"github.com/tamalsaha/DIY-k8s-extended-apiserver/lib/server"
	"k8s.io/client-go/util/cert"
	"log"
	"net"
	"net/http"
)

func main()  {
	// i want to generate ca

	fs := afero.NewOsFs()
	store, err := certstore.NewCertStore(fs,"tmp/extended-api-server")
	if err !=nil{
		log.Fatal(err)
	}
	err = store.NewCA("database")
	if err !=nil{
		log.Fatal(err)
	}
	// generate server key
	serverCrt, serverKey, err := store.NewServerCertPair(cert.AltNames{
		IPs: []net.IP{
			net.ParseIP("127.0.0.2"),
		},
	})
	if err != nil{
		log.Fatal(err)
	}
	// need to write some where

	err = store.Write("tls",serverCrt,serverKey)
	if err !=nil{
		log.Fatal(err)
	}

	// generate client crt, key

	clientCrt, clientKey , err := store.NewClientCertPair(cert.AltNames{
		DNSNames: []string{
			"jhon",
		},
	})

	if err != nil{
		log.Fatal(err)
	}

	err = store.Write("jhon",clientCrt,clientKey)
	if err !=nil{
		log.Fatal(err)
	}

	// create a server and server

	cfg := server.Config{
		Address:     "127.0.0.2:8443",
		CACertFiles: []string{
			store.CertFile("ca"),
		},
		CertFile:    store.CertFile("tls"),
		KeyFile:     store.KeyFile("tls"),
	}

	srv := server.NewGenericServer(cfg)
	r := mux.NewRouter()

	r.HandleFunc("/database/{resource}", func(writer http.ResponseWriter, request *http.Request) {
		vars := mux.Vars(request)
		writer.WriteHeader(http.StatusOK)
		fmt.Fprintf(writer,"resource : %s\n",vars["resource"])
	})
	r.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		fmt.Fprintln(writer,"ok")
	})


	srv.ListenAndServe(r)
}
