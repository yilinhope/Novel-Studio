package store

import (
	"path/filepath"
	"testing"
	"time"
)

func TestProjectWriteLockIsSharedAcrossStoreInstances(t *testing.T) {
	root := t.TempDir()
	first := NewStore(root)
	second := NewStore(filepath.Join(root, "sub", ".."))

	releaseFirst, acquired := first.TryAcquireProjectWrite()
	if !acquired {
		t.Fatal("首次项目写锁申请应成功")
	}
	defer releaseFirst()

	if releaseSecond, acquired := second.TryAcquireProjectWrite(); acquired {
		releaseSecond()
		t.Fatal("同一项目的不同 Store 实例不应同时持有写锁")
	}

	releaseFirst()
	releaseSecond, acquired := second.TryAcquireProjectWrite()
	if !acquired {
		t.Fatal("前一个持有者释放后应允许后续项目写入")
	}
	releaseSecond()
}

func TestProjectWriteAcquireWaitsUntilCurrentWriterFinishes(t *testing.T) {
	path := t.TempDir()
	first := NewStore(path)
	second := NewStore(filepath.Join(path, "nested", ".."))
	releaseFirst, acquired := first.TryAcquireProjectWrite()
	if !acquired {
		t.Fatal("首次项目写锁申请应成功")
	}
	defer releaseFirst()

	started := make(chan struct{})
	finished := make(chan struct{})
	go func() {
		close(started)
		release, ok := second.AcquireProjectWrite()
		if ok {
			release()
		}
		close(finished)
	}()
	<-started
	select {
	case <-finished:
		t.Fatal("等待式项目写锁应等到当前写入者释放")
	case <-time.After(25 * time.Millisecond):
	}

	releaseFirst()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("当前写入者释放后，等待中的锁申请应完成")
	}
}
