package etcdclient

import (
	"context"
	"time"

	"github.com/coreos/etcd/clientv3"
)

type EtcdV3Client struct {
	dialTimeout    time.Duration
	requestTimeout time.Duration
	endpoints      []string
}

func NewEtcdV3Client(dialTimeout, requestTimeout time.Duration, endpoints []string) *EtcdV3Client {
	return &EtcdV3Client{
		dialTimeout:    dialTimeout,
		requestTimeout: requestTimeout,
		endpoints:      endpoints,
	}
}

func (e *EtcdV3Client) Put(key, value string) (*clientv3.PutResponse, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   e.endpoints,
		DialTimeout: e.dialTimeout,
	})

	if err != nil {
		return nil, err
	}
	defer cli.Close() // make sure to close the client

	ctx, cancel := context.WithTimeout(context.Background(), e.requestTimeout)
	resp, err := cli.Put(ctx, key, value)
	cancel()

	return resp, err
}

func (e *EtcdV3Client) Get(key string, withPrefix bool) (*clientv3.GetResponse, error) {
	var resp *clientv3.GetResponse

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   e.endpoints,
		DialTimeout: e.dialTimeout,
	})
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), e.requestTimeout)
	if withPrefix {
		resp, err = cli.Get(ctx, key, clientv3.WithPrefix())
	} else {
		resp, err = cli.Get(ctx, key)
	}
	cancel()

	return resp, err
	//for _, ev := range resp.Kvs {
	//	fmt.Printf("%s : %s\n", ev.Key, ev.Value)
	//}
	// Output: foo : bar
}

func (e *EtcdV3Client) Watch(prefix string) clientv3.WatchChan {
	cli, _ := clientv3.New(clientv3.Config{
		Endpoints:   e.endpoints,
		DialTimeout: e.dialTimeout,
	})

	return cli.Watch(context.Background(), prefix, clientv3.WithPrefix())
}

func (e *EtcdV3Client) Delete(key string, withPrefix bool) (*clientv3.DeleteResponse, error) {
	var resp *clientv3.DeleteResponse

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   e.endpoints,
		DialTimeout: e.dialTimeout,
	})
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), e.requestTimeout)
	defer cancel()

	// delete the keys
	if withPrefix {
		resp, err = cli.Delete(ctx, key, clientv3.WithPrefix(), clientv3.WithPrevKV())
	} else {
		resp, err = cli.Delete(ctx, key)
	}

	return resp, err
}

func (e *EtcdV3Client) HasEntry(key string) bool {
	resp, err := e.Get(key, false)
	if err != nil {
		return false
	}

	if len(resp.Kvs) == 1 {
		return true
	}

	return false
}

func (e *EtcdV3Client) Txn() *EtcdV3Txn {
	return &EtcdV3Txn{
		dialTimeout:    e.dialTimeout,
		requestTimeout: e.requestTimeout,
		endpoints:      e.endpoints,
	}
}

type EtcdV3Txn struct {
	dialTimeout    time.Duration
	requestTimeout time.Duration
	endpoints      []string

	ops []clientv3.Op
}

// func NewTxn() *EtcdV3Txn {
// 	return &EtcdV3Txn{}
// }

func (t *EtcdV3Txn) AddPutOp(key, value string) *EtcdV3Txn {
	t.ops = append(t.ops, clientv3.OpPut(key, value))

	return t
}

func (t *EtcdV3Txn) AddDeleteOp(key string, withPrefix bool) *EtcdV3Txn {
	var op clientv3.Op

	if withPrefix {
		op = clientv3.OpDelete(key, clientv3.WithPrefix())
	} else {
		op = clientv3.OpDelete(key)
	}

	t.ops = append(t.ops, op)

	return t
}

func (t *EtcdV3Txn) Commit() (*clientv3.TxnResponse, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   t.endpoints,
		DialTimeout: t.dialTimeout,
	})

	if err != nil {
		return nil, err
	}
	defer cli.Close() // make sure to close the client

	kvc := clientv3.NewKV(cli)

	ctx, cancel := context.WithTimeout(context.Background(), t.requestTimeout)

	resp, err := kvc.Txn(ctx).Then(t.ops...).Commit()
	cancel()

	return resp, err
}
