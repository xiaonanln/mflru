package mflru

import (
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof" //"time"
	"strconv"
	"testing"
	"time"
)

func TestNewMFLRU(t *testing.T) {
	mflru := NewMFLRU(1024, time.Second, func(key string, val []byte) {
		t.Logf("evicted: %s = %s", key, string(val))
	})

	mflru.Put("a", []byte("a"))
	x, ok := mflru.Get("a")
	if !ok || !equalsBytes([]byte("a"), x) {
		t.Fatalf("should ok")
	}
}

func TestMFLRUEvictTimeout(t *testing.T) {
	mflru := NewMFLRU(1024, time.Millisecond*100, func(key string, val []byte) {
		t.Logf("evicted: %s = %s", key, string(val))
	})
	mflru.Put("a", []byte("a"))
	mflru.Put("b", []byte("b"))
	time.Sleep(time.Millisecond * 50)
	if _, ok := mflru.Get("a"); !ok {
		t.Fatalf("should not evict")
	}
	if _, ok := mflru.Get("b"); !ok {
		t.Fatalf("should not evict")
	}
	time.Sleep(time.Millisecond * 101)
	if _, ok := mflru.Get("a"); ok {
		t.Fatalf("should evict")
	}
	if _, ok := mflru.Get("b"); ok {
		t.Fatalf("should evict")
	}
}

func TestMFLRUWithoutEvictTimeout(t *testing.T) {
	mflru := NewMFLRU(1024, 0, func(key string, val []byte) {
		t.Logf("evicted: %s = %s", key, string(val))
	})
	mflru.Put("a", []byte("a"))
	mflru.Put("b", []byte("b"))
	time.Sleep(time.Millisecond * 50)
	if _, ok := mflru.Get("a"); !ok {
		t.Fatalf("should not evict")
	}
	if _, ok := mflru.Get("b"); !ok {
		t.Fatalf("should not evict")
	}
}

func TestMFLRU_SetEvictTimeout(t *testing.T) {
	mflru := NewMFLRU(512, time.Second, func(key string, val []byte) {
	})

	mflru.Put("a", []byte("a"))
	mflru.Put("b", []byte("b"))
	time.Sleep(time.Millisecond * 50)
	if _, ok := mflru.Get("a"); !ok {
		t.Fatalf("should not evict")
	}
	if _, ok := mflru.Get("b"); !ok {
		t.Fatalf("should not evict")
	}
	mflru.SetEvictTimeout(time.Millisecond * 50)

	time.Sleep(time.Millisecond * 51)
	if _, ok := mflru.Get("a"); ok {
		t.Fatalf("should evict")
	}
	if _, ok := mflru.Get("b"); ok {
		t.Fatalf("should evict")
	}
}

func TestMFLRU_SetMemoryLimit(t *testing.T) {
	mflru := NewMFLRU(10*1024*1024, time.Second, func(key string, val []byte) {
	})

	mflru.Put("a", []byte("a"))
	mflru.Put("b", []byte("b"))
	mflru.Put("c", []byte("c"))
	mflru.SetMemoryLimit(mflru.memorySize)

	mflru.Put("d", []byte("d"))
	if _, ok := mflru.Get("a"); ok {
		t.Fatalf("should evict")
	}

	if _, ok := mflru.Get("b"); !ok {
		t.Fatalf("should not evict")
	}

	mflru.Put("f", []byte("f"))
	if _, ok := mflru.Get("c"); ok {
		t.Fatalf("should evict")
	}
}

func TestMFLRUEvictMemoryLimit(t *testing.T) {
	expectEvict := 0
	mflru := NewMFLRU(512, time.Second, func(key string, val []byte) {
		vi, err := strconv.Atoi(key)
		if err != nil {
			t.Fatal(err)
		}
		if vi != expectEvict {
			t.Fatalf("expect evict %d, but evict %d", expectEvict, vi)
		}
		expectEvict += 1
	})
	for i := 0; i < 1000; i++ {
		is := strconv.Itoa(i)
		mflru.Put(is, []byte(is))
	}
}

func TestMFLRU_Fuzzy(t *testing.T) {
	keyset := map[string]struct{}{}
	incahe := func(key string) bool {
		_, ok := keyset[key]
		return ok
	}

	mflru := NewMFLRU(1024*1024, time.Millisecond*100, func(key string, val []byte) {
		//t.Logf("evicted %s", key)
		if !incahe(key) {
			t.Fatalf("key not in keyset")
		}
		delete(keyset, key)
	})

	for i := 0; i < 100000; i++ {
		k := strconv.Itoa(rand.Intn(100000))
		v, ok := mflru.Get(k)
		if !incahe(k) {
			if ok {
				t.Fatalf("should not Get ok")
			}
		} else {
			if !ok || !equalsBytes(v, []byte(k)) {
				t.Fatalf("Get error: %s = %v %v", k, v, ok)
			}
		}

		mflru.Put(k, []byte(k))
		keyset[k] = struct{}{}
		v, ok = mflru.Get(k)
		if !ok || !equalsBytes(v, []byte(k)) {
			t.Fatalf("get wrong:%s = %v %v", k, v, ok)
		}
	}

}

func equalsBytes(b1, b2 []byte) bool {
	if len(b1) != len(b2) {
		return false
	}

	for i, b := range b1 {
		if b2[i] != b {
			return false
		}
	}
	return true
}

func TestMFLRU_MemorySize(t *testing.T) {
	t.SkipNow()

	go func() {
		pprofAddr := "localhost:9958"
		log.Printf("starting pprof server on %s ...", pprofAddr)
		if err := http.ListenAndServe(pprofAddr, nil); err != nil {
			t.Error(err)
		}
	}()

	var mflru *MFLRU

	mflru = NewMFLRU(100*1024*1024, 0, func(key string, val []byte) {
		log.Printf("MFLRU cache is full: %d", mflru.MemorySize())
		time.Sleep(time.Second * 60)
	})

	for {
		b := make([]byte, 16)
		rand.Read(b)
		mflru.Put(string(b), b)
	}
}
