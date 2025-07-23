package app

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/sirupsen/logrus"
	"github.com/tongshengw/nimbus/backend/sectionleader/internal/constants"
	"github.com/tongshengw/nimbus/backend/sectionleader/internal/db"
	"github.com/tongshengw/nimbus/backend/sectionleader/internal/models"
	"github.com/tongshengw/nimbus/backend/sectionleader/internal/shared"
)

type VMState int

const (
	StateActive VMState = iota
	StatePaused
	StateStopped
)

type VM struct {
	Machine *firecracker.Machine
	Id      shared.MachineUUID
	State   VMState
	cancel  context.CancelFunc
	data    models.MachineData
}

type VMManager struct {
	mutex         sync.Mutex
	createVmMutex sync.Mutex
	IdNameMap     *shared.IdNameMap
	VMs           map[shared.MachineUUID]*VM
}

func NewVMManager() *VMManager {
	return &VMManager{
		mutex:         sync.Mutex{},
		createVmMutex: sync.Mutex{},
		IdNameMap:     shared.NewIdNameMap(),
		VMs:           make(map[shared.MachineUUID]*VM),
	}
}

func (manager *VMManager) CreateVM() (<-chan *models.MachineData, error) {
	// has to be withcancel as this is the context that lives with the machine
	ctx, cancelFunc := context.WithCancel(context.Background())
	outputChannel := make(chan *models.MachineData)

	go func() {
		manager.createVmMutex.Lock()
		defer manager.createVmMutex.Unlock()

		machine, id, ip, err := SpawnNewVM(ctx)
		if err != nil {
			logrus.Errorf("failed to spawn VM: %v", err)
			outputChannel <- nil
			return
		}

		if machine == nil {
			logrus.Errorf("spawnvm return error")
			outputChannel <- nil
			return
		}

		manager.mutex.Lock()
		defer manager.mutex.Unlock()

		vmName, err := manager.IdNameMap.GenerateNewName(id)
		if err != nil {
			logrus.Errorf("could not generate name for new vm: %v", err)
			return
		}

		newMachineData := models.MachineData{
			UUID:         id,
			Name:         vmName,
			LocalIp:      ip,
			CreationTime: time.Now(),
			SshPort:      constants.MinRemotePort + len(manager.VMs),
			ForwardedPorts: []models.ForwardedPort{
				{
					MachinePort: 22,
					RemotePort:  constants.MinRemotePort + len(manager.VMs),
					Protocol:    "tcp",
				},
			},
		}
		err = db.CreateMachine(&newMachineData)
		if err != nil {
			logrus.Errorf("failed to save machine to database: %v", err)
			outputChannel <- nil
			return
		}

		vmPtr := &VM{
			Machine: machine,
			Id:      id,
			State:   StateActive,
			cancel:  cancelFunc,
			data:    newMachineData,
		}

		manager.VMs[id] = vmPtr
		outputChannel <- &vmPtr.data
	}()

	return outputChannel, nil
}

func (manager *VMManager) PauseVM(id shared.MachineUUID) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*5)

	go func(ctx context.Context, cancelFunc context.CancelFunc, manager *VMManager, id shared.MachineUUID) {
		defer cancelFunc()

		manager.mutex.Lock()
		defer manager.mutex.Unlock()

		vmPtr := manager.VMs[id]
		if vmPtr.State != StateActive {
			logrus.Errorf("machine not active, cannot be paused, id: %s", id.String())
			return
		}

		err := vmPtr.Machine.PauseVM(ctx)
		if err != nil {
			logrus.Errorf("pause vm error")
			return
		}

		vmPtr.State = StatePaused
	}(ctx, cancelFunc, manager, id)
}

func (manager *VMManager) ResumeVM(id shared.MachineUUID) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*5)

	go func(ctx context.Context, cancelFunc context.CancelFunc, manager *VMManager, id shared.MachineUUID) {
		defer cancelFunc()

		vmPtr := manager.VMs[id]
		if vmPtr.State != StatePaused {
			logrus.Errorf("machine not paused, cannot be resumed, id %s", id.String())
			return
		}
		err := vmPtr.Machine.PauseVM(ctx)
		if err != nil {
			logrus.Errorf("resume vm error")
			return
		}

		vmPtr.State = StateActive
	}(ctx, cancelFunc, manager, id)
}

func (manager *VMManager) GracefulShutdownVM(id shared.MachineUUID) <-chan bool {
	ctx, cancelFunc := context.WithTimeout(context.Background(), constants.DefaultTimeout*5)
	logrus.Infof("requested machine %s shutdown", id.String())
	outputChan := make(chan bool)

	go func() {
		defer cancelFunc()
		defer close(outputChan)

		manager.mutex.Lock()
		defer manager.mutex.Unlock()

		vmPtr := manager.VMs[id]

		if vmPtr.State == StateStopped {
			logrus.Errorf("attempted to shutdown stopped machine, id: %s", id.String())
			return
		}

		vmPtr.State = StateStopped
		err := vmPtr.Machine.Shutdown(ctx)
		if err != nil {
			logrus.Errorf("machine shutdown err, id: %s, err %v, forcing shutdown", id.String(), err)
			if forceErr := vmPtr.Machine.StopVMM(); forceErr != nil {
				logrus.Errorf("force shutdown failed, id: %s, err %v", id.String(), forceErr)
				outputChan <- false
				return
			}
			return
		}

		outputChan <- true
		logrus.Infof("machine %s successfully shut down", id.String())
	}()

	return outputChan
}

func (manager *VMManager) GracefulShutdownAll() error {
	shutdownChans := make([]<-chan bool, len(manager.VMs))

	counter := 0
	for id := range manager.VMs {
		shutdownChans[counter] = manager.GracefulShutdownVM(id)
		counter++
	}

	for _, c := range shutdownChans {
		select {
		case <-time.After(constants.DefaultTimeout * 5):
			logrus.Errorf("vm shutdown timeout")
			return fmt.Errorf("vm shutdown timeout")
		case <-c:
			// correct, proceed
		}
	}

	return nil
}

func (manager *VMManager) GetSshKey(id shared.MachineUUID) ([]byte, error) {
	if _, ok := manager.VMs[id]; !ok {
		return nil, fmt.Errorf("machine does not exist")
	}

	key, err := os.ReadFile(constants.DataDirPath + "/" + id.String() + "/id_rsa")
	if err != nil {
		return nil, err
	}

	return key, nil
}

func (manager *VMManager) ResurrectAllVMFromDB() (error) {
	machines, err := db.GetAllMachines()
	if err != nil {
		return err
	}

	for _, machine := range machines {
		c, err := manager.ResurrectVM(machine.UUID)
		if err != nil {
			return err
		}
		<-c
	}
	return nil
}

func (manager *VMManager) ResurrectVM(id shared.MachineUUID) (<-chan *models.MachineData, error) {
	// has to be withcancel as this is the context that lives with the machine
	ctx, cancelFunc := context.WithCancel(context.Background())
	outputChannel := make(chan *models.MachineData)

	go func() {
		defer close(outputChannel)

		// Get existing machine data from database
		machines, err := db.GetMachineByUUID(id)
		if err != nil {
			logrus.Errorf("failed to get machine data from database: %v", err)
			outputChannel <- nil
			return
		}

		if len(machines) == 0 {
			logrus.Errorf("machine with UUID %s not found in database", id.String())
			outputChannel <- nil
			return
		}

		machineData := machines[0] // Get the first (and should be only) result

		manager.mutex.Lock()
		defer manager.mutex.Unlock()

		// Check if VM is already running
		if _, exists := manager.VMs[id]; exists {
			logrus.Errorf("VM with UUID %s is already running", id.String())
			outputChannel <- nil
			return
		}

		// Restore the existing VM
		machine, err := RestoreExistingVM(ctx, id, machineData.LocalIp)
		if err != nil {
			logrus.Errorf("failed to restore VM: %v", err)
			outputChannel <- nil
			return
		}

		if machine == nil {
			logrus.Errorf("RestoreExistingVM returned nil machine")
			outputChannel <- nil
			return
		}

		// Update the name mapping
		manager.IdNameMap.RestoreMapping(id, machineData.Name)

		vmPtr := &VM{
			Machine: machine,
			Id:      id,
			State:   StateActive,
			cancel:  cancelFunc,
			data:    machineData,
		}

		manager.VMs[id] = vmPtr
		logrus.Infof("successfully resurrected VM %s with IP %s", id.String(), string(machineData.LocalIp))
		outputChannel <- &vmPtr.data
	}()

	return outputChannel, nil
}
