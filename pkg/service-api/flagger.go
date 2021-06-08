package routes

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"

	"github.com/gorilla/mux"
	"github.com/nwillems/ctff-server/pkg/storage"
)

var (
	logger = log.New(os.Stdout, "flagger: ", log.LstdFlags)
)

type FlaggerServer struct {
	store storage.FeatureFlagStore
}

func NewFlaggerServer() *FlaggerServer {
	store := storage.NewInMemory()
	return &FlaggerServer{store: store}
}

type RegisterFeatureFlags struct {
	Flags []string `json:"flags"`
}

func (fs *FlaggerServer) RegisterFeatureFlagsHandler(rw http.ResponseWriter, req *http.Request) {
	logger.Println("Received request to register")
	requestDump, _ := httputil.DumpRequest(req, true)
	logger.Println(string(requestDump))

	vars := mux.Vars(req)
	decoder := json.NewDecoder(req.Body)

	var flagRegistration RegisterFeatureFlags
	err := decoder.Decode(&flagRegistration)
	if err != nil {
		rw.WriteHeader(401)
		fmt.Fprintf(rw, "Wrongly formatted flags registration")
		return
	}

	identity := vars["identity"]
	var featureFlags []*storage.FeatureFlag
	for _, flag := range flagRegistration.Flags {
		featureFlags = append(featureFlags, &storage.FeatureFlag{Name: flag})
	}

	logger.Printf("Registering for \"%s\": %v\n", identity, featureFlags)
	fs.store.RegisterFeatureFlags(identity, featureFlags)

	rw.WriteHeader(201)
	fmt.Fprintf(rw, "Flags registered")
}

func (fs *FlaggerServer) GetFeatureFlagStateHandler(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)

	flag, err := fs.store.GetFeatureFlagState(vars["identity"], vars["flag_name"])
	if err != nil {
		rw.WriteHeader(401)
		fmt.Fprintf(rw, "Error: %v", err)
	} else {
		json.NewEncoder(rw).Encode(flag)
	}
}

func (fs *FlaggerServer) ListAllFeatureFlagsHandler(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	identity := vars["identity"]

	featureFlags, err := fs.store.GetAllFeatureFlags(identity)
	if err != nil {
		rw.WriteHeader(401)
		fmt.Fprintf(rw, "Error: %v", err)
	} else {
		json.NewEncoder(rw).Encode(featureFlags)
	}
}

func (fs *FlaggerServer) SetFeatureFlagStateHandler(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	identity := vars["identity"]
	flag_name := vars["flag_name"]

	raw_flag, err := ioutil.ReadAll(req.Body)
	if err != nil {
		rw.WriteHeader(401)
		fmt.Fprintf(rw, "Error: %v", err)
		return
	}
	flag_state := string(raw_flag)

	err = fs.store.SetFeatureFlagState(identity, flag_name, flag_state)
	if err != nil {
		rw.WriteHeader(401)
		fmt.Fprintf(rw, "Error: %v", err)
	} else {
		rw.WriteHeader(201)
		fmt.Fprintf(rw, "Flag value changed")
	}

}
