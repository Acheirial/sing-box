package obfs

import (
	"fmt"
	"net"
)

type Base struct {
	Host   string
	Port   int
	Key    []byte
	IVSize int
	Param  string
}

type Obfs interface {
	StreamConn(net.Conn) net.Conn
}

type obfsCreator func(b *Base) Obfs

var obfsList = make(map[string]struct {
	overhead int
	new      obfsCreator
})

func register(name string, c obfsCreator, o int) {
	obfsList[name] = struct {
		overhead int
		new      obfsCreator
	}{overhead: o, new: c}
}

// PickObfs returns an Obfs of the given name and its overhead.
func PickObfs(name string, b *Base) (Obfs, int, error) {
	if choice, ok := obfsList[name]; ok {
		return choice.new(b), choice.overhead, nil
	}
	return nil, 0, fmt.Errorf("obfs %s not supported", name)
}
