package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"strings"

	"github.com/containers/standard/schema"
	"github.com/containers/standard/schema/types"
	"github.com/gorilla/mux"
)

type metadata struct {
	manifest schema.ContainerRuntimeManifest
	apps     map[string]*schema.AppManifest
}

var (
	metadataByIP  = make(map[string]*metadata)
	metadataByUID = make(map[types.UUID]*metadata)
	hmacKey       [sha1.Size]byte
)

const (
	myPort   = "4444"
	metaIP   = "169.254.169.255"
	metaPort = "80"
)

func setupIPTables() error {
	args := []string{"-t", "nat", "-A", "PREROUTING",
		"-p", "tcp", "-d", metaIP, "--dport", metaPort,
		"-j", "REDIRECT", "--to-port", myPort}

	return exec.Command("iptables", args...).Run()
}

func antiSpoof(brPort, ipAddr string) error {
	args := []string{"-t", "filter", "-I", "INPUT", "-i", brPort, "-p", "IPV4", "!", "--ip-source", ipAddr, "-j", "DROP"}
	return exec.Command("ebtables", args...).Run()
}

func queryValue(u *url.URL, key string) string {
	vals, ok := u.Query()[key]
	if !ok || len(vals) != 1 {
		return ""
	}
	return vals[0]
}

func handleRegisterContainer(w http.ResponseWriter, r *http.Request) {
	remoteIP := strings.Split(r.RemoteAddr, ":")[0]
	if _, ok := metadataByIP[remoteIP]; ok {
		// not allowed from container IP
		w.WriteHeader(http.StatusForbidden)
		return
	}

	containerIP := queryValue(r.URL, "container_ip")
	if containerIP == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	containerBrPort := queryValue(r.URL, "container_brport")
	if containerBrPort == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	m := &metadata{
		apps: make(map[string]*schema.AppManifest),
	}

	if err := json.NewDecoder(r.Body).Decode(&m.manifest); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := antiSpoof(containerBrPort, containerIP); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	metadataByIP[containerIP] = m
	metadataByUID[m.manifest.UUID] = m

	w.WriteHeader(http.StatusOK)
}

func handleRegisterApp(w http.ResponseWriter, r *http.Request) {
	remoteIP := strings.Split(r.RemoteAddr, ":")[0]
	if _, ok := metadataByIP[remoteIP]; ok {
		// not allowed from container IP
		w.WriteHeader(http.StatusForbidden)
		return
	}

	uid, err := types.NewUUID(mux.Vars(r)["uid"])
	if err != nil {
		fmt.Println("mulformed UUID")
		w.WriteHeader(http.StatusNotFound)
		return
	}

	m, ok := metadataByUID[*uid]
	if !ok {
		fmt.Println("metadata by UUID not found", *uid)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	an := mux.Vars(r)["app"]

	app := &schema.AppManifest{}
	if err := json.NewDecoder(r.Body).Decode(&app); err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	m.apps[an] = app

	w.WriteHeader(http.StatusOK)
}

func containerGet(h func(w http.ResponseWriter, r *http.Request, m *metadata)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		remoteIP := strings.Split(r.RemoteAddr, ":")[0]
		m, ok := metadataByIP[remoteIP]
		if !ok {
			fmt.Println("metadata by remoteIP not found ", remoteIP)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		h(w, r, m)
	}
}

func appGet(h func(w http.ResponseWriter, r *http.Request, m *metadata, am *schema.AppManifest)) func(http.ResponseWriter, *http.Request) {
	return containerGet(func(w http.ResponseWriter, r *http.Request, m *metadata) {
		appname := mux.Vars(r)["app"]

		if am, ok := m.apps[appname]; ok {
			h(w, r, m, am)
		} else {
			fmt.Println("app not found ", appname)
			w.WriteHeader(http.StatusNotFound)
		}
	})
}

func handleContainerAnnotations(w http.ResponseWriter, r *http.Request, m *metadata) {
	w.Header().Add("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)

	for k, _ := range m.manifest.Annotations {
		w.Write([]byte(k))
	}
}

func handleContainerAnnotation(w http.ResponseWriter, r *http.Request, m *metadata) {
	k := mux.Vars(r)["name"]
	v, ok := m.manifest.Annotations[types.ACLabel(k)]
	if !ok {
		fmt.Println("container annotation not found ", k)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Add("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(v))
}

func handleContainerManifest(w http.ResponseWriter, r *http.Request, m *metadata) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(m.manifest); err != nil {
		fmt.Println(err)
	}
}

func handleContainerUID(w http.ResponseWriter, r *http.Request, m *metadata) {
	uid := m.manifest.UUID.String()

	w.Header().Add("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(uid))
}

func mergeAppAnnotations(am *schema.AppManifest, cm *schema.ContainerRuntimeManifest) map[types.ACLabel]string {
	merged := make(map[types.ACLabel]string)
	for k, v := range am.Annotations {
		merged[k] = v
	}

	if app, ok := cm.Apps[am.Name]; ok {
		for k, v := range app.Annotations {
			merged[k] = v
		}
	}

	return merged
}

func handleAppAnnotations(w http.ResponseWriter, r *http.Request, m *metadata, am *schema.AppManifest) {
	w.Header().Add("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)

	for k, _ := range mergeAppAnnotations(am, &m.manifest) {
		w.Write([]byte(k + "\n"))
	}
}

func handleAppAnnotation(w http.ResponseWriter, r *http.Request, m *metadata, am *schema.AppManifest) {
	k := mux.Vars(r)["name"]

	merged := mergeAppAnnotations(am, &m.manifest)
	if v, ok := merged[types.ACLabel(k)]; ok {
		w.Header().Add("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(v))
		return
	}
	fmt.Println("app annotation not found ", k)
	w.WriteHeader(http.StatusNotFound)
}

func handleAppManifest(w http.ResponseWriter, r *http.Request, m *metadata, am *schema.AppManifest) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(*am); err != nil {
		fmt.Println(err)
	}
}

func handleAppID(w http.ResponseWriter, r *http.Request, m *metadata, am *schema.AppManifest) {
	w.Header().Add("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(m.manifest.Apps[am.Name].ImageID.String()))
}

func initCrypto() error {
	if n, err := rand.Reader.Read(hmacKey[:]); err != nil || n != len(hmacKey) {
		return fmt.Errorf("failed to generate HMAC Key")
	}
	return nil
}

func digest(r io.Reader) ([]byte, error) {
	digest := sha1.New()
	if _, err := io.Copy(digest, r); err != nil {
		return nil, err
	}
	return digest.Sum(nil), nil
}

func handleContainerSign(w http.ResponseWriter, r *http.Request) {
	remoteIP := strings.Split(r.RemoteAddr, ":")[0]
	m, ok := metadataByIP[remoteIP]
	if !ok {
		fmt.Println("metadata by remoteIP not found ", remoteIP)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// compute message digest
	d, err := digest(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// HMAC(UID:digest)
	h := hmac.New(sha1.New, hmacKey[:])
	h.Write(m.manifest.UUID[:])
	h.Write(d)

	// Send back digest:HMAC as the signature
	w.Header().Add("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	enc := base64.NewEncoder(base64.StdEncoding, w)
	enc.Write(d)
	enc.Write(h.Sum(nil))
	enc.Close()
}

func handleContainerVerify(w http.ResponseWriter, r *http.Request) {
	uid, err := types.NewUUID(r.FormValue("uid"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sig, err := base64.StdEncoding.DecodeString(r.FormValue("signature"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	digest := sig[:sha1.Size]
	sum := sig[sha1.Size:]

	h := hmac.New(sha1.New, hmacKey[:])
	h.Write(uid[:])
	h.Write(digest)

	if hmac.Equal(sum, h.Sum(nil)) {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusForbidden)
	}
}

type httpResp struct {
	writer http.ResponseWriter
	status int
}

func (r *httpResp) Header() http.Header {
	return r.writer.Header()
}

func (r *httpResp) Write(d []byte) (int, error) {
	return r.writer.Write(d)
}

func (r *httpResp) WriteHeader(status int) {
	r.status = status
	r.writer.WriteHeader(status)
}

func logReq(h func (w http.ResponseWriter, r *http.Request)) func (w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := &httpResp{w, 0}
		h(resp, r)
		fmt.Printf("%v %v - %v\n", r.Method, r.RequestURI, resp.status)
	}
}

func main() {
	if err := setupIPTables(); err != nil {
		fmt.Println(err)
		return
	}

	initCrypto()

	r := mux.NewRouter()
	r.HandleFunc("/containers/", logReq(handleRegisterContainer)).Methods("POST")
	r.HandleFunc("/containers/{uid}/{app:.*}", logReq(handleRegisterApp)).Methods("PUT")

	acRtr := r.Headers("Metadata-Flavor", "AppContainer header").
		PathPrefix("/acMetadata/v1").Subrouter()

	mr := acRtr.Methods("GET").Subrouter()

	mr.HandleFunc("/container/annotations/", logReq(containerGet(handleContainerAnnotations)))
	mr.HandleFunc("/container/annotations/{name}", logReq(containerGet(handleContainerAnnotation)))
	mr.HandleFunc("/container/manifest", logReq(containerGet(handleContainerManifest)))
	mr.HandleFunc("/container/uid", logReq(containerGet(handleContainerUID)))

	mr.HandleFunc("/apps/{app:.*}/annotations/", logReq(appGet(handleAppAnnotations)))
	mr.HandleFunc("/apps/{app:.*}/annotations/{name}", logReq(appGet(handleAppAnnotation)))
	mr.HandleFunc("/apps/{app:.*}/image/manifest", logReq(appGet(handleAppManifest)))
	mr.HandleFunc("/apps/{app:.*}/image/id", logReq(appGet(handleAppID)))

	acRtr.HandleFunc("/container/hmac/sign", logReq(handleContainerSign)).Methods("POST")
	acRtr.HandleFunc("/container/hmac/verify", logReq(handleContainerVerify)).Methods("POST")

	log.Fatal(http.ListenAndServe(":4444", r))
}
