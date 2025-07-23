package shared

import (
	"database/sql/driver"
	"fmt"
	"net"

	"github.com/google/uuid"
)

type Ipv4 net.IP

func (ip Ipv4) String() string {
	return net.IP(ip).String()
}

func (ip *Ipv4) Scan(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan ipv4: %v", value)
	}
	parsed := net.ParseIP(str).To4()
	if parsed == nil {
		return fmt.Errorf("ipv4 parse error")
	}

	*ip = Ipv4(parsed)
	return nil
}

func (ip Ipv4) Value() (driver.Value, error) {
	return ip.String(), nil
}

type MachineUUID uuid.UUID

func (o MachineUUID) String() string {
	return uuid.UUID(o).String()
}

func (id *MachineUUID) Scan(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan UUID: %v", value)
	}
	parsed, err := uuid.Parse(str)
	if err != nil {
		return err
	}
	*id = MachineUUID(parsed)
	return nil
}

func (id MachineUUID) Value() (driver.Value, error) {
	return id.String(), nil
}

type IdNameMap struct {
	idToName map[MachineUUID]string
	nameToId map[string]MachineUUID
}

func NewIdNameMap() *IdNameMap {
	return &IdNameMap{
		idToName: make(map[MachineUUID]string),
		nameToId: make(map[string]MachineUUID),
	}
}

func (m *IdNameMap) GenerateNewName(id MachineUUID) (string, error) {
	name := GeneratePetname()
	_, ok := m.nameToId[name]

	counter := 0
	for ok && counter < 10 {
		name = GeneratePetname()
		_, ok = m.nameToId[name]
		counter++
	}
	if counter == 9 {
		return "", fmt.Errorf("could not generate new name")
	}

	m.idToName[id] = name
	m.nameToId[name] = id
	return name, nil
}

func (m *IdNameMap) GetName(id MachineUUID) (string, error) {
	name, ok := m.idToName[id]
	if ok {
		return name, nil
	}

	return "", fmt.Errorf("id not found")
}

func (m *IdNameMap) GetId(name string) (MachineUUID, error) {
	id, ok := m.nameToId[name]
	if ok {
		return id, nil
	}

	return MachineUUID{}, fmt.Errorf("id not found")

}
