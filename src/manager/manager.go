package manager

import (
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"config"
	"datamodel"
	pb "datamodel/protobuf"
	"storage"
	"utils"
)

func isValidType(info *datamodel.Info) bool {
	if info.Type == nil {
		return false
	}
	return len(datamodel.GetTypeString(info.GetType())) != 0
}

// Manager is responsible for manipulating the sketches and syncing to disk
type Manager struct {
	infos    *infoManager
	sketches *sketchManager
	domains  *domainManager
	lock     sync.RWMutex
	ticker   *time.Ticker
	storage  *storage.Manager
}

func (m *Manager) saveSketch(id string) error {
	return m.sketches.save(id)
}

func (m *Manager) saveSketches() error {
	var wg sync.WaitGroup
	running := 0
	for _, v := range m.infos.info {
		wg.Add(1)
		running++
		go func(info *datamodel.Info) {
			// a) save sketch
			if err := m.saveSketch(info.ID()); err != nil {
				// TODO: log something here
				fmt.Println(err)
			}
			// b) replay from AOF (SELECT * FROM ops WHERE sketchId = ?)
			// TODO: Replay from AOF
			// c) unlock sketch

			wg.Done()
		}(v)
		// Just 4 at a time
		if running%4 == 0 {
			wg.Wait()
		}
	}
	wg.Wait()
	return nil
}

func (m *Manager) setLockSketches(b bool) {
	m.sketches.setLockAll(b)
}

// Save ...
func (m *Manager) Save() error {
	m.lock.Lock()
	defer m.lock.Unlock()

	// 1) save DEFAULT SETTINGS
	// TODO: save defaut settings

	// 2) lock all sketches from being allowed to do ADD
	m.setLockSketches(true)

	// 3) Clear AOF
	// TODO: clear AOF

	// 4) Save deep copied sketches info from previously
	if err := m.infos.save(); err != nil {
		// TODO: Do somthing here?
	}
	if err := m.domains.save(); err != nil {
		// TODO: Do somthing here?
	}

	// 5) For each sketch
	if err := m.saveSketches(); err != nil {
		return err
	}

	// 6) Unlock sketches
	m.setLockSketches(false)

	return nil
}

// NewManager ...
func NewManager() *Manager {
	storage := storage.NewManager()
	sketches := newSketchManager(storage)
	infos := newInfoManager(storage)
	domains := newDomainManager(infos, sketches, storage)

	m := &Manager{
		sketches: sketches,
		infos:    infos,
		domains:  domains,
		lock:     sync.RWMutex{},
		ticker:   time.NewTicker(time.Second * time.Duration(config.GetConfig().SaveThresholdSeconds)),
		storage:  storage,
	}

	for _, info := range infos.info {
		utils.PanicOnError(sketches.load(info))
	}

	// Set up saving on intervals
	go func() {
		for _ = range m.ticker.C {
			if m.Save() != nil {
				// FIXME: print out something
			}
		}
	}()
	return m
}

// CreateSketch ...
func (m *Manager) CreateSketch(info *datamodel.Info) error {
	if !isValidType(info) {
		return fmt.Errorf("Can not create sketch of type %s, invalid type.", info.Type)
	}
	if err := m.infos.create(info); err != nil {
		return err
	}
	if err := m.sketches.create(info); err != nil {
		// If error occurred during creation of sketch, delete info
		if err2 := m.infos.delete(info.ID()); err2 != nil {
			return fmt.Errorf("%q\n%q ", err, err2)
		}
		return err
	}
	return nil
}

// CreateDomain ...
func (m *Manager) CreateDomain(info *datamodel.Info) error {
	infos := make(map[string]*datamodel.Info)
	for _, typ := range datamodel.GetTypesPb() {
		styp := pb.SketchType(typ)
		tmpInfo := info.Copy()
		tmpInfo.Type = &styp
		infos[tmpInfo.ID()] = tmpInfo
	}
	return m.domains.create(info.GetName(), infos)
}

// AddToSketch ...
func (m *Manager) AddToSketch(id string, values []string) error {
	return m.sketches.add(id, values)
}

// AddToDomain ...
func (m *Manager) AddToDomain(id string, values []string) error {
	return m.domains.add(id, values)
}

// DeleteSketch ...
func (m *Manager) DeleteSketch(id string) error {
	if err := m.infos.delete(id); err != nil {
		return err
	}
	return m.sketches.delete(id)
}

// DeleteDomain ...
func (m *Manager) DeleteDomain(id string) error {
	return m.domains.delete(id)
}

type tupleResult [][2]string

func (slice tupleResult) Len() int {
	return len(slice)
}

func (slice tupleResult) Less(i, j int) bool {
	if slice[i][0] == slice[j][0] {
		return slice[i][1] < slice[j][1]
	}
	return slice[i][0] < slice[j][0]
}

func (slice tupleResult) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

// GetSketches return a list of sketch tuples [name, type]
func (m *Manager) GetSketches() [][2]string {
	sketches := tupleResult{}
	for _, v := range m.infos.info {
		sketches = append(sketches,
			[2]string{v.GetName(),
				datamodel.GetTypeString(v.GetType())})
	}
	sort.Sort(sketches)
	return sketches
}

// GetDomains return a list of sketch tuples [name, type]
func (m *Manager) GetDomains() [][2]string {
	domains := tupleResult{}
	for k, v := range m.domains.domains {
		domains = append(domains, [2]string{k, strconv.Itoa(len(v))})
	}
	sort.Sort(domains)
	return domains
}

// GetSketch ...
func (m *Manager) GetSketch(id string) (*datamodel.Info, error) {
	info := m.infos.get(id)
	if info == nil {
		return nil, fmt.Errorf("No such sketch %s", id)
	}
	return info, nil
}

// GetDomain ...
func (m *Manager) GetDomain(id string) (*pb.Domain, error) {
	return m.domains.get(id)
}

// GetFromSketch ...
func (m *Manager) GetFromSketch(id string, data interface{}) (interface{}, error) {
	return m.sketches.get(id, data)
}

// Destroy ...
func (m *Manager) Destroy() {
	m.ticker.Stop()
	_ = m.storage.Close()
}