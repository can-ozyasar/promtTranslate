// Package hotkey implements a global hotkey listener using Linux's evdev
// interface (/dev/input/event*) without requiring root privileges.
// The user must be in the "input" group, or logind must have granted ACLs.
package hotkey

import (
	"encoding/binary"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// EventType constants from linux/input.h
const (
	evSync  uint16 = 0x00
	evKey   uint16 = 0x01
)

// Key state constants
const (
	keyUp      int32 = 0
	keyDown    int32 = 1
	keyRepeat  int32 = 2
)

// inputEvent mirrors struct input_event from linux/input.h.
// Size: 24 bytes on 64-bit systems (timeval=16, type=2, code=2, value=4).
type inputEvent struct {
	TimeSec  int64
	TimeUSec int64
	Type     uint16
	Code     uint16
	Value    int32
}

// Handler is called when a registered combo fires.
type Handler func(comboName string)

// Listener monitors all keyboard devices and fires Handler on matching combos.
type Listener struct {
	combos   []Combo
	handler  Handler
	mu       sync.Mutex
	held     map[Key]bool // currently pressed keys
	stopCh   chan struct{}
	devices  []string
}

// NewListener creates a Listener with the given combos and handler.
func NewListener(combos []Combo, handler Handler) *Listener {
	return &Listener{
		combos:  combos,
		handler: handler,
		held:    make(map[Key]bool),
		stopCh:  make(chan struct{}),
	}
}

// Start discovers keyboard event devices and begins listening. Blocks until Stop is called.
func (l *Listener) Start() error {
	devices, err := findKeyboards()
	if err != nil {
		return fmt.Errorf("hotkey: discover devices: %w", err)
	}
	if len(devices) == 0 {
		return fmt.Errorf("hotkey: no keyboard devices found in /dev/input — " +
			"ensure your user is in the 'input' group or run as a systemd user service")
	}
	l.devices = devices
	slog.Info("hotkey: monitoring keyboards", "count", len(devices), "paths", devices)

	var wg sync.WaitGroup
	for _, dev := range devices {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			l.readDevice(path)
		}(dev)
	}
	wg.Wait()
	return nil
}

// Stop signals all device goroutines to exit.
func (l *Listener) Stop() {
	close(l.stopCh)
}

// readDevice opens a single event device and reads key events until Stop is called.
func (l *Listener) readDevice(path string) {
	f, err := os.Open(path)
	if err != nil {
		slog.Warn("hotkey: open device", "path", path, "err", err)
		return
	}
	defer f.Close()

	var ev inputEvent
	for {
		select {
		case <-l.stopCh:
			return
		default:
		}

		f.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		if err := binary.Read(f, binary.LittleEndian, &ev); err != nil {
			if os.IsTimeout(err) {
				continue
			}
			slog.Warn("hotkey: read event", "path", path, "err", err)
			return
		}

		if ev.Type != evKey {
			continue
		}

		key := Key(ev.Code)
		l.mu.Lock()
		switch ev.Value {
		case keyDown, keyRepeat:
			l.held[key] = true
		case keyUp:
			delete(l.held, key)
		}
		if ev.Value == keyDown {
			l.checkCombos()
		}
		l.mu.Unlock()
	}
}

// checkCombos checks if any registered combo is currently fully pressed.
// Must be called with l.mu held.
func (l *Listener) checkCombos() {
	// Build current modifier mask.
	var mods Modifier
	for k := range l.held {
		if m := ModifierOf(k); m != 0 {
			mods |= m
		}
	}

	for _, c := range l.combos {
		if c.Mods != mods {
			continue
		}
		if !l.held[c.Key] {
			continue
		}
		if IsModifier(c.Key) {
			continue
		}
		name := c.Name
		go l.handler(name) // fire asynchronously to avoid blocking the read loop
		return
	}
}

// findKeyboards returns paths to /dev/input/event* devices that advertise
// key events. This heuristic selects actual keyboards over mice/touchpads.
func findKeyboards() ([]string, error) {
	entries, err := filepath.Glob("/dev/input/event*")
	if err != nil || len(entries) == 0 {
		return nil, fmt.Errorf("no /dev/input/event* devices found")
	}

	var keyboards []string
	for _, path := range entries {
		if isKeyboard(path) {
			keyboards = append(keyboards, path)
		}
	}
	return keyboards, nil
}

// isKeyboard uses /proc/bus/input/devices to check if a device is a keyboard.
// Falls back to attempting to open the device if /proc info is unavailable.
func isKeyboard(eventPath string) bool {
	// Try to read /proc/bus/input/devices and match the event handler.
	data, err := os.ReadFile("/proc/bus/input/devices")
	if err != nil {
		// If we can't read /proc, just try to open the device.
		f, err := os.Open(eventPath)
		if err == nil {
			f.Close()
			return true
		}
		return false
	}

	base := filepath.Base(eventPath)
	// Find the section for this event node and check if it contains keyboard-related flags.
	sections := strings.Split(string(data), "\n\n")
	for _, section := range sections {
		if !strings.Contains(section, base) {
			continue
		}
		// Check EV= line for key events (bit 1 set → EV_KEY)
		for _, line := range strings.Split(section, "\n") {
			if strings.HasPrefix(line, "B: EV=") {
				evHex := strings.TrimPrefix(line, "B: EV=")
				evHex = strings.TrimSpace(evHex)
				// Parse hex; bit 1 (0x2) means EV_KEY
				var evBits uint64
				fmt.Sscanf(evHex, "%x", &evBits)
				if evBits&0x2 != 0 {
					return true
				}
			}
		}
	}
	return false
}
