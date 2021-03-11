package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/spf13/afero"
	"github.com/tamalsaha/DIY-k8s-extended-apiserver/lib/certstore"
	"github.com/tamalsaha/DIY-k8s-extended-apiserver/lib/server"

	"log"
	"net"

	"k8s.io/client-go/util/cert"
)

func main() {

	var proxy = false
	flag.BoolVar(&proxy, "send-proxy-request", proxy, "forward request to database extended apiserver")
	flag.Parse()
	//Creates a directory to store certificates at tmp/extended-apiserver
	fs := afero.NewOsFs()
	store, err := certstore.NewCertStore(fs, "/tmp/extended-api-server")
	if err != nil {
		log.Fatal(err)
	}

	//creates a new ca and store in tmp/extended-apiserver path

	//err = store.NewCA("apiserver")
	//if err !=nil{
	//	log.Fatal(err)
	//}

	err = store.InitCA("apiserver")
	if err != nil {
		log.Fatal(err)
	}
	serverCert, serverKey, err := store.NewServerCertPair(cert.AltNames{
		IPs: []net.IP{
			net.ParseIP("127.0.0.1"),
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	err = store.Write("tls", serverCert, serverKey)

	if err != nil {
		log.Fatal(err)
	}

	clientCert, clientKey, err := store.NewClientCertPair(cert.AltNames{
		DNSNames: []string{"jhon"},
	})

	if err != nil {
		log.Fatal(err)
	}

	err = store.Write("jhon", clientCert, clientKey)

	if err != nil {
		log.Fatal(err)
	}
	//--------------------------------------------------------------------------
	rhStore, err := certstore.NewCertStore(fs, "/tmp/extended-api-server")
	if err != nil {
		log.Fatal(err)
	}
	err = rhStore.InitCA("requestheader")
	if err != nil {
		log.Fatal(err)
	}
	rhClientCert, rhClientKey, err := rhStore.NewClientCertPair(cert.AltNames{
		DNSNames: []string{"apiserver"}, // because apiserver is making the calls to database eas
	})
	if err != nil {
		log.Fatalln(err)
	}

	err = rhStore.Write("apiserver", rhClientCert, rhClientKey)
	if err != nil {
		log.Fatal(err)
	}
	rhcert, err := tls.LoadX509KeyPair(rhStore.CertFile("apiserver"), rhStore.KeyFile("apiserver"))
	if err != nil {
		log.Fatal(err)
	}
	//--------------------------------------------------------------------------

	easCertPool := x509.NewCertPool()

	if proxy {
		easStore, err := certstore.NewCertStore(fs, "/tmp/extended-api-server")
		if err != nil {
			log.Fatal(err)
		}
		err = easStore.LoadCA("database")
		if err != nil {
			log.Fatal(err)
		}
		easCertPool.AppendCertsFromPEM(easStore.CACertBytes())
	}

	//----------------------------------------------------------------------------
	cfg := server.Config{
		Address: "127.0.0.1:8443",
		CACertFiles: []string{
			store.CertFile("ca"),
		},
		CertFile: store.CertFile("tls"),
		KeyFile:  store.KeyFile("tls"),
	}

	srv := server.NewGenericServer(cfg)

	r := mux.NewRouter()

	r.HandleFunc("/core/{resource}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "resource :%v\n", vars["resource"])

	})

	//---------------------------------------------------
	if proxy {
		r.HandleFunc("/database/{resource}", func(writer http.ResponseWriter, request *http.Request) {
			tr := &http.Transport{
				MaxConnsPerHost: 10,
				TLSClientConfig: &tls.Config{
					Certificates: []tls.Certificate{rhcert},
					RootCAs:      easCertPool,
				},
			}
			client := http.Client{
				Transport: tr,
				Timeout:   time.Duration(30 * time.Second),
			}

			u := *request.URL

			u.Scheme = "https"

			u.Host = "127.0.0.2:8443"
			fmt.Printf("forwarding request to %v\n", u.String())
			fmt.Println(u.String())
			req, err := http.NewRequest(request.Method, u.String(), nil)
			if err != nil {
				log.Fatal(err)
			}

			if len(request.TLS.PeerCertificates) > 0 {
				fmt.Print("heerer\n\n")
				req.Header.Set("X-Remote-User", req.TLS.PeerCertificates[0].Subject.CommonName)
			}
			fmt.Printf("hi\n\n")

			resp, err := client.Do(req)

			if err != nil {
				writer.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(writer, "error : %v\n", err.Error())
			}
			defer resp.Body.Close()

			writer.WriteHeader(http.StatusOK)
			io.Copy(writer, resp.Body)
		})
	}
	//=---------------------------------------------------

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})
	srv.ListenAndServe(r)

}
