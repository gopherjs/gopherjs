package nosync

// Mutex is a dummy which is non-blocking.
type Mutex struct {
	locked bool
}

// Lock locks m. It is a run-time error if m is already locked.
func (m *Mutex) Lock() {
	if m.locked {
		panic("nosync: mutex is already locked")
	}
	m.locked = true
}

// Unlock unlocks m. It is a run-time error if m is not locked.
func (m *Mutex) Unlock() {
	if !m.locked {
		panic("nosync: unlock of unlocked mutex")
	}
	m.locked = false
}

// RWMutex is a dummy which is non-blocking.
type RWMutex struct {
	writeLocked     bool
	readLockCounter int
}

// Lock locks m for writing. It is a run-time error if rw is already locked for reading or writing.
func (rw *RWMutex) Lock() {
	println("Lock")
	if rw.readLockCounter != 0 || rw.writeLocked {
		println("FAIL")
		panic("nosync: mutex is already locked")
	}
	rw.writeLocked = true
}

// Unlock unlocks rw for writing. It is a run-time error if rw is not locked for writing.
func (rw *RWMutex) Unlock() {
	println("Unlock")
	if !rw.writeLocked {
		panic("nosync: unlock of unlocked mutex")
	}
	rw.writeLocked = false
}

// RLock locks m for reading. It is a run-time error if rw is already locked for reading or writing.
func (rw *RWMutex) RLock() {
	println("RLock")
	if rw.writeLocked {
		panic("nosync: mutex is already locked")
	}
	rw.readLockCounter++
}

// RUnlock undoes a single RLock call; it does not affect other simultaneous readers. It is a run-time error if rw is not locked for reading.
func (rw *RWMutex) RUnlock() {
	println("RUnlock")
	if rw.readLockCounter == 0 {
		panic("nosync: unlock of unlocked mutex")
	}
	rw.readLockCounter--
}
