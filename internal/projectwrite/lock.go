// Package projectwrite 提供按项目目录共享的进程内写入互斥。
package projectwrite

import (
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type entry struct {
	mu   sync.Mutex
	refs int
}

var registry = struct {
	sync.Mutex
	entries map[string]*entry
}{entries: make(map[string]*entry)}

// TryAcquire 尝试独占项目写入；release 必须且只需调用一次。
// 不同 Store/Host 实例只要指向同一目录，就会共享此边界。
func TryAcquire(path string) (release func(), acquired bool) {
	key, ok := normalize(path)
	if !ok {
		return nil, false
	}
	item := retain(key)
	if !item.mu.TryLock() {
		drop(key, item)
		return nil, false
	}
	return makeRelease(key, item), true
}

// Acquire 等待并独占项目写入；release 必须且只需调用一次。
func Acquire(path string) (release func(), acquired bool) {
	key, ok := normalize(path)
	if !ok {
		return nil, false
	}
	item := retain(key)
	item.mu.Lock()
	return makeRelease(key, item), true
}

func normalize(path string) (string, bool) {
	if strings.TrimSpace(path) == "" {
		return "", false
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", false
	}
	key := filepath.Clean(abs)
	if runtime.GOOS == "windows" {
		key = strings.ToLower(key)
	}
	return key, true
}

func retain(key string) *entry {
	registry.Lock()
	defer registry.Unlock()
	item := registry.entries[key]
	if item == nil {
		item = &entry{}
		registry.entries[key] = item
	}
	item.refs++
	return item
}

func drop(key string, item *entry) {
	registry.Lock()
	defer registry.Unlock()
	item.refs--
	if item.refs == 0 {
		delete(registry.entries, key)
	}
}

func makeRelease(key string, item *entry) func() {
	var once sync.Once
	return func() {
		once.Do(func() {
			item.mu.Unlock()
			drop(key, item)
		})
	}
}
