module github.com/intermernet/gopherclaw

go 1.24.5

require tinygo.org/x/drivers/bno08x v0.0.0-replace

require (
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	tinygo.org/x/drivers v0.33.0 // indirect
	tinygo.org/x/drivers/internal/pin v0.0.0-replace // indirect
)

replace tinygo.org/x/drivers/bno08x => ../../../go/tinygo-drivers/bno08x

replace tinygo.org/x/drivers/internal/pin => ../../../go/tinygo-drivers/internal/pin
