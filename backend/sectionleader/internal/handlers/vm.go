package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tongshengw/nimbus/backend/sectionleader/internal/constants"
	"github.com/tongshengw/nimbus/backend/sectionleader/internal/middle"
	"github.com/tongshengw/nimbus/backend/sectionleader/internal/models"
	"github.com/tongshengw/nimbus/backend/sectionleader/internal/shared"
)

func NewMachine(w http.ResponseWriter, r *http.Request) {
	data, ok := r.Context().Value(middle.CommonContextDataKey).(middle.CommonContextData)
	if !ok {
		logrus.Errorf("common context data not ok: %v", data)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	vmManager := data.Manager

	outputChan, err := vmManager.CreateVM()
	if err != nil {
		logrus.Errorf("create vm failed: %v", err)
		http.Error(w, "Failed to create VM", http.StatusInternalServerError)
		return
	}

	var createMachineRes *models.MachineData

	select {
	case createMachineRes = <-outputChan:
		if createMachineRes == nil {
			logrus.Errorf("create vm returned nil data pointer")
			http.Error(w, "Failed to create VM", http.StatusInternalServerError)
			return
		}
	case <-time.After(constants.CreateVmTimeout):
		logrus.Errorf("create machine timed out")
		http.Error(w, "timed out creating VM", http.StatusInternalServerError)
		return
	}

	tokenStr, err := middle.NewJwt(createMachineRes.Id, data.SecretKey)
	if err != nil {
		logrus.Errorf("new jwt failed: %v", err)
		http.Error(w, "Failed to create token", http.StatusInternalServerError)
		return
	}

	// modify frp config
	err = CreateTomlFrpcConfig(createMachineRes)
	if err != nil {
		logrus.Errorf("create toml frpc config failed: %v", err)
		http.Error(w, "Failed to create reverse proxy config", http.StatusInternalServerError)
		return
	}

	response := struct {
		MachineId   string `json:"machine_id"`
		MachineName string `json:"machine_name"`
		LocalIp     string `json:"local_ip"`
		Token       string `json:"token"`
		RemotePort  int    `json:"remote_port"`
		RemoteIp    string `json:"remote_ip"`
	}{
		MachineId:   createMachineRes.Id.String(),
		MachineName: createMachineRes.Name,
		LocalIp:     createMachineRes.LocalIp.String(),
		Token:       tokenStr,
		RemotePort:  createMachineRes.RemotePort,
		RemoteIp:    constants.PublicIpStr,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func SshKey(w http.ResponseWriter, r *http.Request) {
	machineId, ok := r.Context().Value(middle.MachineIdContextDataKey).(shared.MachineUUID)
	if !ok {
		logrus.Errorf("machine uuid data not ok")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data, ok := r.Context().Value(middle.CommonContextDataKey).(middle.CommonContextData)
	if !ok {
		logrus.Errorf("common context data not ok: %v", data)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	vmManager := data.Manager

	key, err := vmManager.GetSshKey(machineId)
	if err != nil {
		logrus.Errorf("could not load machine ssh key, %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	machineDisplayName, err := vmManager.IdNameMap.GetName(machineId)
	if err != nil {
		logrus.Errorf("could not convert to machinedisplayid: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s_ssh_key"`, machineDisplayName))

	w.Write(key)
}

func ShutdownAll(w http.ResponseWriter, r *http.Request) {
	data, ok := r.Context().Value(middle.CommonContextDataKey).(middle.CommonContextData)
	if !ok {
		logrus.Errorf("common context data not ok: %v", data)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	vmManager := data.Manager

	type requestData struct {
		SecretKey string `json:"secret_key"`
	}
	
	var reqData requestData
	err := json.NewDecoder(r.Body).Decode(&reqData)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	if reqData.SecretKey != data.SecretKey {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vmManager.GracefulShutdownAll()
}

func StopMachine(w http.ResponseWriter, r *http.Request) {

}
