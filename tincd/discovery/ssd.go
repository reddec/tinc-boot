package discovery

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// Single Source Distribution
// - name
// - version (must always grow)

type Entity struct {
	Name    string
	Version int64
}

func NewSSD(filename string) *SSD {
	return &SSD{
		file: filename,
	}
}

type SSD struct {
	lock     sync.RWMutex
	entities map[string]Entity
	file     string
	fileLock sync.Mutex
}

func (ssd *SSD) unsafeIsNewer(name string, desired int64) bool {
	old, exists := ssd.entities[name]
	return !exists || desired > old.Version
}

func (ssd *SSD) CanBeMerged(entity Entity) bool {
	ssd.lock.RLock()
	defer ssd.lock.RUnlock()
	return ssd.unsafeIsNewer(entity.Name, entity.Version)
}

func (ssd *SSD) ReplaceIfNewer(entity Entity, block func() bool) bool {
	ssd.lock.Lock()
	defer ssd.lock.Unlock()
	if ssd.entities == nil {
		ssd.entities = make(map[string]Entity)
	}
	if !ssd.unsafeIsNewer(entity.Name, entity.Version) {
		return false
	}
	if block != nil && !block() {
		return false
	}
	ssd.entities[entity.Name] = entity
	return true
}

func (ssd *SSD) Replace(entity Entity) {
	ssd.lock.Lock()
	defer ssd.lock.Unlock()
	if ssd.entities == nil {
		ssd.entities = make(map[string]Entity)
	}
	ssd.entities[entity.Name] = entity
}

func (ssd *SSD) GetAfter(name string, version int64) (Entity, bool) {
	ssd.lock.RLock()
	defer ssd.lock.RUnlock()
	old, hasOld := ssd.entities[name]
	if !hasOld || old.Version <= version {
		return old, false
	}
	return old, true
}

func (ssd *SSD) Header() []Entity {
	ssd.lock.RLock()
	defer ssd.lock.RUnlock()
	var ans = make([]Entity, 0, len(ssd.entities))
	for _, v := range ssd.entities {
		ans = append(ans, v)
	}
	return ans
}

func (ssd *SSD) Marshal(writer io.Writer) error {
	ssd.lock.RLock()
	defer ssd.lock.RUnlock()

	items := ssd.Header() // double RLock, but it's ok

	enc := json.NewEncoder(writer)
	enc.SetIndent("", " ")
	return enc.Encode(items)
}

func (ssd *SSD) Unmarshal(reader io.Reader) error {
	var items []Entity
	err := json.NewDecoder(reader).Decode(&items)
	if err != nil {
		return err
	}

	var data = make(map[string]Entity)
	for _, v := range items {
		data[v.Name] = v
	}

	ssd.lock.Lock()
	defer ssd.lock.Unlock()

	ssd.entities = data
	return nil
}

func (ssd *SSD) SaveFile(filename string) error {
	f, err := os.CreateTemp(filepath.Dir(filename), filepath.Base(filename)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	err = ssd.Marshal(f)
	if err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return fmt.Errorf("marshal entities: %w", err)
	}
	err = f.Sync()
	if err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return fmt.Errorf("sync changes: %w", err)
	}
	err = f.Close()
	if err != nil {
		_ = os.Remove(f.Name())
		return fmt.Errorf("close temp file: %w", err)
	}
	err = os.Rename(f.Name(), filename)
	if err != nil {
		_ = os.Remove(f.Name())
		return fmt.Errorf("swap temp file: %w", err)
	}
	return nil
}

func (ssd *SSD) ReadFile(filename string) error {
	f, err := os.Open(filename)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("open data file: %w", err)
	}
	defer f.Close()
	return ssd.Unmarshal(f)
}

func (ssd *SSD) Read() error {
	ssd.fileLock.Lock()
	defer ssd.fileLock.Unlock()
	return ssd.ReadFile(ssd.file)
}

func (ssd *SSD) Save() error {
	ssd.fileLock.Lock()
	defer ssd.fileLock.Unlock()
	return ssd.SaveFile(ssd.file)
}

func (ssd *SSD) Filename() string {
	return ssd.file
}
