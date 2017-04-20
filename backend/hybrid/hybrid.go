package hybrid

import (
	"fmt"

	"github.com/coreos/flannel/backend"
	"github.com/coreos/flannel/pkg/ip"
	"github.com/coreos/flannel/subnet"
	"golang.org/x/net/context"
)

func init() {
	backend.Register("hybrid", New)
}

const (
	routeCheckRetries = 10
)

type HybridBackend struct {
	sm       subnet.Manager
	extIface *backend.ExternalInterface
	networks map[string]*network
	extra    string
}

func New(sm subnet.Manager, extIface *backend.ExternalInterface) (backend.Backend, error) {
	if !extIface.ExtAddr.Equal(extIface.IfaceAddr) {
		return nil, fmt.Errorf("your PublicIP differs from interface IP, meaning that probably you're on a NAT, which is not supported by host-gw backend")
	}

	be := &HybridBackend{
		sm:       sm,
		extIface: extIface,
		networks: make(map[string]*network),
	}

	return be, nil
}

func (_ *HybridBackend) Run(ctx context.Context) {
	<-ctx.Done()
}

func (be *HybridBackend) RegisterNetwork(ctx context.Context, config *subnet.Config) (backend.Network, error) {
	n := &network{
		extIface: be.extIface,
		sm:       be.sm,
	}

	attrs := subnet.LeaseAttrs{
		PublicIP:    ip.FromIP(be.extIface.ExtAddr),
		BackendType: "host-gw",
	}

	l, err := be.sm.AcquireLease(ctx, &attrs)
	switch err {
	case nil:
		n.lease = l

	case context.Canceled, context.DeadlineExceeded:
		return nil, err

	default:
		return nil, fmt.Errorf("failed to acquire lease: %v", err)
	}

	/* NB: docker will create the local route to `sn` */

	return n, nil
}
