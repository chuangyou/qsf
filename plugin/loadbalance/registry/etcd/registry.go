package etcd

import (
	"encoding/json"

	"time"

	etcd3 "github.com/coreos/etcd/clientv3"
	//"github.com/coreos/etcd/etcdserver/api/v3rpc/rpctypes"
	"golang.org/x/net/context"
	"google.golang.org/grpc/grpclog"
)

type EtcdReigistry struct {
	etcd3Client *etcd3.Client
	key         string
	value       string
	ttl         time.Duration
	ctx         context.Context
	cancel      context.CancelFunc
	deregister  chan struct{}
}

type Option struct {
	EtcdConfig  etcd3.Config
	RegistryDir string
	ServiceName string
	NodeID      string
	NData       NodeData
	Ttl         time.Duration
}

type NodeData struct {
	Addr     string
	Metadata map[string]string
}

func NewRegistry(option Option) (*EtcdReigistry, error) {
	client, err := etcd3.New(option.EtcdConfig)
	if err != nil {
		return nil, err
	}

	val, err := json.Marshal(option.NData)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	registry := &EtcdReigistry{
		etcd3Client: client,
		key:         option.RegistryDir + "/" + option.ServiceName + "/" + option.NodeID,
		value:       string(val),
		ttl:         option.Ttl,
		ctx:         ctx,
		cancel:      cancel,
		deregister:  make(chan struct{}),
	}
	return registry, nil
}

func (e *EtcdReigistry) Register() error {
	resp, err := e.etcd3Client.Grant(e.ctx, int64(e.ttl.Seconds()))
	if err != nil {
		return err
	}
	if _, err := e.etcd3Client.Put(e.ctx, e.key, e.value, etcd3.WithLease(resp.ID)); err != nil {
		grpclog.Printf("grpclb: set key '%s' with ttl to etcd3 failed: %s", e.key, err.Error())
		return err
	}

	if _, err := e.etcd3Client.KeepAlive(e.ctx, resp.ID); err != nil {
		grpclog.Printf("grpclb: refresh service '%s' with ttl to etcd3 failed: %s", e.key, err.Error())
		return err
	}
	// wait deregister then delete
	go func() {
		<-e.deregister
		e.etcd3Client.Delete(e.ctx, e.key)
		e.deregister <- struct{}{}
	}()
	//	insertFunc := func() error {
	//		resp, err := e.etcd3Client.Grant(e.ctx, int64(e.ttl.Seconds()))
	//		if err != nil {
	//			return err
	//		}
	//		_, err = e.etcd3Client.Get(e.ctx, e.key)
	//		if err != nil {
	//			if err == rpctypes.ErrKeyNotFound {
	//				if _, err := e.etcd3Client.Put(e.ctx, e.key, e.value, etcd3.WithLease(resp.ID)); err != nil {
	//					grpclog.Printf("grpclb: set key '%s' with ttl to etcd3 failed: %s", e.key, err.Error())
	//				}
	//			} else {
	//				grpclog.Printf("grpclb: key '%s' connect to etcd3 failed: %s", e.key, err.Error())
	//			}
	//			return err
	//		} else {
	//			// refresh set to true for not notifying the watcher
	//			if _, err := e.etcd3Client.Put(e.ctx, e.key, e.value, etcd3.WithLease(resp.ID)); err != nil {
	//				grpclog.Printf("grpclb: refresh key '%s' with ttl to etcd3 failed: %s", e.key, err.Error())
	//				return err
	//			}
	//		}
	//		return nil
	//	}

	//	err := insertFunc()
	//	if err != nil {
	//		return err
	//	}

	//	ticker := time.NewTicker(e.ttl / 5)
	//	for {
	//		select {
	//		case <-ticker.C:
	//			insertFunc()
	//		case <-e.ctx.Done():
	//			ticker.Stop()
	//			if _, err := e.etcd3Client.Delete(context.Background(), e.key); err != nil {
	//				grpclog.Printf("grpclb: deregister '%s' failed: %s", e.key, err.Error())
	//			}
	//			return nil
	//		}
	//	}

	//	return nil
	return nil
}

// UnRegister delete registered service from etcd
func (e *EtcdReigistry) UnRegister() {
	e.deregister <- struct{}{}
	<-e.deregister
}
