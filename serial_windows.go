package serial
import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)
type Port struct {
	config Config
	handle syscall.Handle
}
func (p * Port) configure(cfg Config) (err error) {
	p.config = cfg
	return
}
func (p * Port) open() (err error) {
	err = p.openHandle()
	if err != nil {
		return fmt.Errorf("error opening serial device %s: %s", p.config.Device, err)
	}
	err = p.setTimeouts()
	if err != nil {
		return fmt.Errorf("error setting timeouts: %s", err)
	}
	err = p.setCommState()
	if err != nil {
		return fmt.Errorf("error applying serial settings: %s", err)
	}
	err = p.reset()
	if err != nil {
		return fmt.Errorf("error during reset: %s", err)
	}
	return
}
func (p * Port) openHandle() (err error) {
	device := p.config.Device
	if !strings.HasPrefix(device, `\\.\`) {
		device = `\\.\` + device
	}
	device_utf16, err := syscall.UTF16PtrFromString(device)
	if err != nil {
		return
	}
	p.handle, err = syscall.CreateFile(device_utf16, syscall.GENERIC_READ | syscall.GENERIC_WRITE, 0, nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	return
}
func (p * Port) setCommState() (err error) {
	dcb := &_DCB {
		DCBlength: 28,
	}
	err = _GetCommState(p.handle, dcb)
	if err != nil {
		return
	}
	f := _DCB_Flags {
		Binary: true,
		DtrControl: 0x01,
		DsrSensitivity: false,
		TXContinueOnXoff: false,
		OutX: false,
		InX: false,
		ErrorChar: false,
		Null: false,
		RtsControl: 0x01,
		AbortOnError: false,
		OutxCtsFlow: false,
		OutxDsrFlow: false,
	}
	dcb.Flags = f.Get()
	if v, ok := settings[p.config.BaudRate]; ok {
		dcb.BaudRate = uint32(v)
	} else {
		return fmt.Errorf("unsupported baud rate: %d", p.config.BaudRate)
	}
	if v, ok := settings[p.config.DataBits]; ok {
		dcb.ByteSize = byte(v)
	} else {
		return fmt.Errorf("unsupported data bits: %d", p.config.DataBits)
	}
	if v, ok := settings[p.config.Parity]; ok {
		dcb.Parity = byte(v)
	} else {
		return fmt.Errorf("unsupported parity: %d", p.config.Parity)
	}
	if v, ok := settings[p.config.StopBits]; ok {
		dcb.StopBits = byte(v)
	} else {
		return fmt.Errorf("unsupported stop bits: %d", p.config.StopBits)
	}
	err = _SetCommState(p.handle, dcb)
	return
}
func (p * Port) setTimeouts() (err error) {
	timeouts := &_COMMTIMEOUTS {
		ReadIntervalTimeout: p.config.ReadIntervalTimeout,
		ReadTotalTimeoutMultiplier: p.config.ReadTotalTimeoutMultiplier,
		ReadTotalTimeoutConstant: p.config.ReadTotalTimeoutConstant,
		WriteTotalTimeoutMultiplier: p.config.WriteTotalTimeoutMultiplier,
		WriteTotalTimeoutConstant: p.config.WriteTotalTimeoutConstant,
	}
	err = _SetCommTimeouts(p.handle, timeouts)
	if err != nil {
		return
	}
	readback := new(_COMMTIMEOUTS)
	err = _GetCommTimeouts(p.handle, readback)
	if err != nil {
		return
	}
	if * timeouts != * readback {
		err = fmt.Errorf("timeout settings overridden by serial device:\nrequested:\n%v\nactual:\n%v", * timeouts, * readback)
		return
	}
	return
}
func (p * Port) reset() (err error) {
	err = syscall.FlushFileBuffers(p.handle)
	if err != nil {
		return
	}
	err = _PurgeComm(p.handle, _PURGE_TXABORT | _PURGE_RXABORT | _PURGE_TXCLEAR | _PURGE_RXCLEAR)
	if err != nil {
		return
	}
	err = _ClearCommError(p.handle)
	if err != nil {
		return
	}
	return
}
func (p * Port) close() (err error) {
	return syscall.CloseHandle(p.handle)
}
func (p * Port) read(b []byte) (n int, err error) {
	var done uint32
	err = syscall.ReadFile(p.handle, b, &done, nil)
	n = int(done)
	return
}
func (p * Port) write(b []byte) (n int, err error) {
	n = 0;
	for {
		var done uint32
		err = syscall.WriteFile(p.handle, b[n:], &done, nil)
		n += int(done)
		if n == len(b) {
			break
		}
	}
	return
}
func (p * Port) flush() (err error) {
	return syscall.FlushFileBuffers(p.handle)
}
func (p * Port) signal(s Signal, value bool) (err error) {
	switch {
	case s == DTR && value == false:
		return _EscapeCommFunction(p.handle, _CLRDTR)
	case s == DTR && value == true:
		return _EscapeCommFunction(p.handle, _SETDTR)
	case s == RTS && value == false:
		return _EscapeCommFunction(p.handle, _CLRRTS)
	case s == RTS && value == true:
		return _EscapeCommFunction(p.handle, _SETRTS)
	default:
		return fmt.Errorf("Unreconized signal: %v %v", s, value)
	}
}
var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	pGetCommState = kernel32.NewProc("GetCommState")
	pSetCommState = kernel32.NewProc("SetCommState")
	pSetCommTimeouts = kernel32.NewProc("SetCommTimeouts")
	pGetCommTimeouts = kernel32.NewProc("GetCommTimeouts")
	pEscapeCommFunction = kernel32.NewProc("EscapeCommFunction")
	pPurgeComm = kernel32.NewProc("PurgeComm")
	pClearCommError = kernel32.NewProc("ClearCommError")
	settings = map[int] int {
		DataBits_5: 5,
		DataBits_6: 6,
		DataBits_7: 7,
		DataBits_8: 8,
		StopBits_1: 0,
		StopBits_1_5: 1,
		StopBits_2: 2,
		Parity_None: 0,
		Parity_Odd: 1,
		Parity_Even: 2,
		Parity_Mark: 3,
		Parity_Space: 4,
		BaudRate_9600: 9600,
		BaudRate_19200: 19200,
		BaudRate_38400: 38400,
		BaudRate_57600: 57600,
		BaudRate_115200: 115200,
	}
)
type _DCB_Flags struct {
	Binary bool
	Parity bool
	OutxCtsFlow bool
	OutxDsrFlow bool
	DtrControl byte
	DsrSensitivity bool
	TXContinueOnXoff bool
	OutX bool
	InX bool
	ErrorChar bool
	Null bool
	RtsControl byte
	AbortOnError bool
}
func (f *_DCB_Flags) Set(v uint32) {
	bit := func(n uint32) bool {
		var mask uint32 = 1 << n
		return (mask & v) != 0
	}
	bit2 := func(n uint32) byte {
		x := v >> n
		x = x & 0x3
		return byte(x)
	}
	f.Binary = bit(0)
	f.Parity = bit(1)
	f.OutxCtsFlow = bit(2)
	f.OutxDsrFlow = bit(3)
	f.DtrControl = bit2(4)
	f.DsrSensitivity = bit(6)
	f.TXContinueOnXoff = bit(7)
	f.OutX = bit(8)
	f.InX = bit(9)
	f.ErrorChar = bit(10)
	f.Null = bit(11)
	f.RtsControl = bit2(12)
	f.AbortOnError = bit(14)
}
func (f _DCB_Flags) Get() (v uint32) {
	bit := func(b bool, n uint32) (v uint32) {
		if b {
			v = 1
		}
		v = v << n
		return
	}
	bit2 := func(b byte, n uint32) (v uint32) {
		v = uint32(b & 0x03)
		v = v << n
		return
	}
	v = v | bit(f.Binary, 0)
	v = v | bit(f.Parity, 1)
	v = v | bit(f.OutxCtsFlow, 2)
	v = v | bit(f.OutxDsrFlow, 3)
	v = v | bit2(f.DtrControl, 4)
	v = v | bit(f.DsrSensitivity, 6)
	v = v | bit(f.TXContinueOnXoff, 7)
	v = v | bit(f.OutX, 8)
	v = v | bit(f.InX, 9)
	v = v | bit(f.ErrorChar, 10)
	v = v | bit(f.Null, 11)
	v = v | bit2(f.RtsControl, 12)
	v = v | bit(f.AbortOnError, 14)
	return
}
type _DCB struct {
	DCBlength uint32
	BaudRate uint32
	Flags uint32
	Reserved1 uint16
	XonLim uint16
	XoffLim uint16
	ByteSize byte
	Parity byte
	StopBits byte
	XonChar byte
	XoffChar byte
	ErrorChar byte
	EofChar byte
	EvtChar byte
	Reserved2 uint16
}
type _COMMTIMEOUTS struct {
	ReadIntervalTimeout uint32
	ReadTotalTimeoutMultiplier uint32
	ReadTotalTimeoutConstant uint32
	WriteTotalTimeoutMultiplier uint32
	WriteTotalTimeoutConstant uint32
}
func (c _COMMTIMEOUTS) String() string {
	return fmt.Sprintf("\n  ReadIntervalTimeout:         %08x" + "\n  ReadTotalTimeoutMultiplier:  %08x" +
		"\n  ReadTotalTimeoutConstant:    %08x" + "\n  WriteTotalTimeoutMultiplier: %08x" +
		"\n  WriteTotalTimeoutConstant:   %08x" +
		"\n", c.ReadIntervalTimeout, c.ReadTotalTimeoutMultiplier, c.ReadTotalTimeoutConstant, c.WriteTotalTimeoutMultiplier, c.WriteTotalTimeoutConstant)
}
func _GetCommTimeouts(handle syscall.Handle, timeouts * _COMMTIMEOUTS) (err error) {
	r0, _, e1 := syscall.Syscall(pGetCommTimeouts.Addr(), 2, uintptr(handle), uintptr(unsafe.Pointer(timeouts)), 0)
	if r0 == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}
func _SetCommTimeouts(handle syscall.Handle, timeouts * _COMMTIMEOUTS) (err error) {
	r0, _, e1 := syscall.Syscall(pSetCommTimeouts.Addr(), 2, uintptr(handle), uintptr(unsafe.Pointer(timeouts)), 0)
	if r0 == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}
func _GetCommState(handle syscall.Handle, dcb * _DCB) (err error) {
	r0, _, e1 := syscall.Syscall(pGetCommState.Addr(), 2, uintptr(handle), uintptr(unsafe.Pointer(dcb)), 0)
	if r0 == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}
func _SetCommState(handle syscall.Handle, dcb * _DCB) (err error) {
	r0, _, e1 := syscall.Syscall(pSetCommState.Addr(), 2, uintptr(handle), uintptr(unsafe.Pointer(dcb)), 0)
	if r0 == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}
type purgeFlag int
const (
	_PURGE_TXABORT purgeFlag = 0x01
	_PURGE_RXABORT = 0x02
	_PURGE_TXCLEAR = 0x04
	_PURGE_RXCLEAR = 0x08
)
func _PurgeComm(handle syscall.Handle, purge purgeFlag) (err error) {
	r0, _, e1 := syscall.Syscall(pPurgeComm.Addr(), 2, uintptr(handle), uintptr(purge), 0)
	if r0 == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}
type escapeFn int
const (
	_SETXOFF  escapeFn = 1
	_SETXON = 2
	_SETRTS = 3
	_CLRRTS = 4
	_SETDTR = 5
	_CLRDTR = 6
	_RESETDEV = 7
	_SETBREAK = 8
	_CLRBREAK = 9
)
func _EscapeCommFunction(handle syscall.Handle, escape escapeFn) (err error) {
	r0, _, e1 := syscall.Syscall(pEscapeCommFunction.Addr(), 2, uintptr(handle), uintptr(escape), 0)
	if r0 == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}
func _ClearCommError(handle syscall.Handle) (err error) {
	r0, _, e1 := syscall.Syscall(pClearCommError.Addr(), 3, uintptr(handle), 0, 0)
	if r0 == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}
