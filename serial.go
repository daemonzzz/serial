package serial
type Signal byte
const (
	DTR Signal = iota
	RTS
)
func NewPort() (p * Port) {
	return &Port {
		config: Config {
			BaudRate: BaudRate_9600,
			DataBits: DataBits_8,
			Parity: Parity_None,
			StopBits: StopBits_1,
			VMIN: 1,
			VTIME: 0,
			ReadIntervalTimeout: 1000,
			ReadTotalTimeoutMultiplier: 0,
			ReadTotalTimeoutConstant: 0,
			WriteTotalTimeoutMultiplier: 0,
			WriteTotalTimeoutConstant: 5000,
		},
	}
}
func Open(device string, baud int) (p * Port, err error) {
	p = NewPort()
	config := p.Config()
	config.Device = device
	config.BaudRate = baud
	err = p.Configure(config)
	if err != nil {
		return
	}
	err = p.Open()
	if err != nil {
		return
	}
	return
}
func (p * Port) Config() Config {
	return p.config
}
func (p * Port) Configure(cfg Config) (err error) {
	return p.configure(cfg)
}
func (p * Port) Open() (err error) {
	return p.open()
}
func (p * Port) Close() (err error) {
	return p.close()
}
func (p * Port) Read(b []byte) (n int, err error) {
	return p.read(b)
}
func (p * Port) Write(b []byte) (n int, err error) {
	return p.write(b)
}
func (p * Port) Signal(s Signal, value bool) (err error) {
	return p.signal(s, value)
}
