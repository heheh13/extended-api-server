package main

import (
	"crypto/x509"
	"flag"
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

	var proxy = false
	flag.BoolVar(&proxy,"receive-proxy-request",proxy,"receive forwarded proxy request")
	flag.Parse()
	// i want to generate ca

	fs := afero.NewOsFs()
	store, err := certstore.NewCertStore(fs,"/tmp/extended-api-server")
	if err !=nil{
		log.Fatal(err)
	}
	err = store.InitCA("database")
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
			"jane",
		},
	})

	if err != nil{
		log.Fatal(err)
	}

	err = store.Write("jane",clientCrt,clientKey)
	if err !=nil{
		log.Fatal(err)
	}

	//things related to receiving..

	apiserverStore, err := certstore.NewCertStore(fs,"/tmp/extended-api-server")
	if err != nil{
		log.Fatal(err)
	}
	if proxy{
		err = apiserverStore.LoadCA("apiserver")
		if err != nil{
			log.Fatal(err)
		}
	}
	//------------------------------------------------------------------------------------

	rhCertPool := x509.NewCertPool()

	rhStore, err := certstore.NewCertStore(fs,"/tmp/extended-api-server")
	if err !=nil{
		log.Fatal(err)
	}
	if proxy{
		err = rhStore.LoadCA("requestheader")
		if err != nil{
			log.Fatal(err)
		}
		rhCertPool.AppendCertsFromPEM(rhStore.CACertBytes())
	}
	//---------------------------------------------------------------
	// create a server and server

	cfg := server.Config{
		Address:     "127.0.0.2:8443",
		CACertFiles: []string{
			// Only allow clients from main apiserver.
			// store.CertFile("ca"),
			//store.CertFile("ca"),
		},
		CertFile:    store.CertFile("tls"),
		KeyFile:     store.KeyFile("tls"),
	}

	if proxy{
		cfg.CACertFiles = append(cfg.CACertFiles,apiserverStore.CertFile("ca"))
		cfg.CACertFiles = append(cfg.CACertFiles,rhStore.CertFile("ca"))
	}

	srv := server.NewGenericServer(cfg)
	r := mux.NewRouter()

	r.HandleFunc("/database/{resource}", func(writer http.ResponseWriter, request *http.Request) {

		user := "system:anonymous"
		src :="-"

		if len(request.TLS.PeerCertificates) >0 {
			opts := x509.VerifyOptions{
				Roots: rhCertPool,
				KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			}
			if _, err := request.TLS.PeerCertificates[0].Verify(opts);err != nil{
				user = request.TLS.PeerCertificates[0].Subject.CommonName
				src = "Client-Cert-CN"
			}else{
				user = request.Header.Get("X-Remote-User")
				src = "X-Remote-User"
			}
		}


		vars := mux.Vars(request)
		writer.WriteHeader(http.StatusOK)
		fmt.Fprintf(writer, "Resource: %v requested by user[%s]=%s\n", vars["resource"], src, user)

	})
	r.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		fmt.Fprintln(writer,"ok")
	})


	srv.ListenAndServe(r)
}
