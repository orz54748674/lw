package consul

import ()
import "vn/framework/mqant/registry"

func NewRegistry(opts ...registry.Option) registry.Registry {
	return registry.NewRegistry(opts...)
}
