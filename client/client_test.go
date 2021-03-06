// SPDX-License-Identifier: MIT

package client_test

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.cryptoscope.co/margaret"
	"go.cryptoscope.co/muxrpc/v2"
	"golang.org/x/sync/errgroup"

	"go.cryptoscope.co/ssb"
	"go.cryptoscope.co/ssb/client"
	"go.cryptoscope.co/ssb/internal/testutils"
	"go.cryptoscope.co/ssb/message"
	"go.cryptoscope.co/ssb/network"
	"go.cryptoscope.co/ssb/sbot"
	refs "go.mindeco.de/ssb-refs"
)

func TestUnixSock(t *testing.T) {
	r, a := require.New(t), assert.New(t)

	srvRepo := filepath.Join("testrun", t.Name(), "serv")
	os.RemoveAll(srvRepo)
	srvLog := testutils.NewRelativeTimeLogger(nil)

	srv, err := sbot.New(
		sbot.WithInfo(srvLog),
		sbot.WithRepoPath(srvRepo),
		// sbot.DisableNetworkNode(), skips muxrpc handler
		sbot.WithListenAddr(":0"),
		sbot.LateOption(sbot.WithUNIXSocket()),
	)
	r.NoError(err, "sbot srv init failed")

	c, err := client.NewUnix(filepath.Join(srvRepo, "socket"))
	r.NoError(err, "failed to make client connection")
	// end test boilerplate

	ref, err := c.Whoami()
	r.NoError(err, "failed to call whoami")
	r.NotNil(ref)
	a.Equal(srv.KeyPair.Id.Ref(), ref.Ref())

	// make sure we can publish
	var msgs []*refs.MessageRef
	const msgCount = 15
	for i := 0; i < msgCount; i++ {
		ref, err := c.Publish(struct {
			Type string `json:"type"`
			Test int
		}{"test", i})
		r.NoError(err)
		r.NotNil(ref)
		msgs = append(msgs, ref)
	}

	// and stream those messages back
	var o message.CreateHistArgs
	o.ID = srv.KeyPair.Id
	o.Keys = true
	o.Limit = -1
	src, err := c.CreateHistoryStream(o)
	r.NoError(err)
	r.NotNil(src)

	ctx := context.TODO()
	i := 0
	for src.Next(ctx) {

		var msg refs.KeyValueRaw
		err = src.Reader(func(r io.Reader) error {
			return json.NewDecoder(r).Decode(&msg)
		})
		r.NoError(err)

		r.True(msg.Key().Equal(msgs[i]), "wrong message %d", i)
		i++
	}
	r.NoError(src.Err())
	r.Equal(msgCount, i, "did not get all messages")

	a.NoError(c.Close())
	srv.Shutdown()
	r.NoError(srv.Close())
}

func TestWhoami(t *testing.T) {
	r, a := require.New(t), assert.New(t)

	srvRepo := filepath.Join("testrun", t.Name(), "serv")
	os.RemoveAll(srvRepo)
	srvLog := testutils.NewRelativeTimeLogger(nil)

	srv, err := sbot.New(
		sbot.WithInfo(srvLog),
		sbot.WithRepoPath(srvRepo),
		sbot.WithListenAddr(":0"))
	r.NoError(err, "sbot srv init failed")

	var srvErrc = make(chan error, 1)
	go func() {
		err := srv.Network.Serve(context.TODO())
		if err != nil {
			srvErrc <- errors.Wrap(err, "ali serve exited")
		}
		close(srvErrc)
	}()

	kp, err := ssb.LoadKeyPair(filepath.Join(srvRepo, "secret"))
	r.NoError(err, "failed to load servers keypair")
	srvAddr := srv.Network.GetListenAddr()

	c, err := client.NewTCP(kp, srvAddr)
	r.NoError(err, "failed to make client connection")
	// end test boilerplate

	ref, err := c.Whoami()
	r.NoError(err, "failed to call whoami")
	r.NotNil(ref)
	a.Equal(kp.Id.Ref(), ref.Ref())

	a.NoError(c.Close())

	srv.Shutdown()
	r.NoError(srv.Close())
	r.NoError(<-srvErrc)
}

func TestLotsOfWhoami(t *testing.T) {
	// defer leakcheck.Check(t)
	r, a := require.New(t), assert.New(t)

	srvRepo := filepath.Join("testrun", t.Name(), "serv")
	os.RemoveAll(srvRepo)
	srvLog := testutils.NewRelativeTimeLogger(nil)

	srv, err := sbot.New(
		sbot.WithInfo(srvLog),
		sbot.WithRepoPath(srvRepo),
		// sbot.DisableNetworkNode(), skips muxrpc handler
		sbot.WithListenAddr(":0"),
		sbot.LateOption(sbot.WithUNIXSocket()),
	)
	r.NoError(err, "sbot srv init failed")

	c, err := client.NewUnix(filepath.Join(srvRepo, "socket"))
	r.NoError(err, "failed to make client connection")
	// end test boilerplate

	for i := 50; i > 0; i-- {
		ref, err := c.Whoami()
		r.NoError(err, "call %d errored", i)
		r.NotNil(ref)
		a.Equal(srv.KeyPair.Id.Ref(), ref.Ref(), "call %d has wrong result", i)
	}

	a.NoError(c.Close())

	srv.Shutdown()
	r.NoError(srv.Close())
}

func TestStatusCalls(t *testing.T) {
	// defer leakcheck.Check(t)

	mkTCP := func(t *testing.T, opts ...sbot.Option) (*sbot.Sbot, mkClient) {
		r := require.New(t)

		srvRepo := filepath.Join("testrun", t.Name(), "serv")
		os.RemoveAll(srvRepo)
		srvLog := testutils.NewRelativeTimeLogger(nil)

		defOpts := []sbot.Option{
			sbot.WithInfo(srvLog),
			sbot.WithRepoPath(srvRepo),
			sbot.WithListenAddr(":0"),
		}

		srv, err := sbot.New(append(defOpts, opts...)...)
		r.NoError(err, "sbot srv init failed")
		ctx, cancel := context.WithCancel(context.TODO())
		t.Cleanup(func() {
			cancel()
		})
		go func() {
			err := srv.Network.Serve(ctx)
			t.Log("tcp bot serve exited", err)
			if err != nil && !errors.Is(err, context.Canceled) {
				panic(err)
			}
		}()

		kp, err := ssb.LoadKeyPair(filepath.Join(srvRepo, "secret"))
		r.NoError(err, "failed to load servers keypair")
		srvAddr := srv.Network.GetListenAddr()
		return srv, func(ctx context.Context) (*client.Client, error) {
			c, err := client.NewTCP(kp, srvAddr, client.WithContext(ctx))
			if err != nil {
				return nil, errors.Wrap(err, "failed to make TCP client connection")
			}
			return c, nil
		}
	}

	mkUnix := func(t *testing.T, opts ...sbot.Option) (*sbot.Sbot, mkClient) {
		r := require.New(t)

		srvRepo := filepath.Join("testrun", t.Name(), "serv")
		os.RemoveAll(srvRepo)
		srvLog := testutils.NewRelativeTimeLogger(nil)

		defOpts := []sbot.Option{
			sbot.WithInfo(srvLog),
			sbot.WithRepoPath(srvRepo),
			// sbot.DisableNetworkNode(), skips muxrpc handler
			sbot.WithListenAddr(":0"),
			sbot.LateOption(sbot.WithUNIXSocket()),
		}

		srv, err := sbot.New(append(defOpts, opts...)...)
		r.NoError(err, "sbot srv init failed")

		return srv, func(ctx context.Context) (*client.Client, error) {
			c, err := client.NewUnix(filepath.Join(srvRepo, "socket"), client.WithContext(ctx))
			if err != nil {
				return nil, errors.Wrap(err, "failed to make unix client connection")
			}
			return c, nil
		}
	}

	t.Run("tcp", LotsOfStatusCalls(mkTCP))
	t.Run("unix", LotsOfStatusCalls(mkUnix))
}

type mkClient func(context.Context) (*client.Client, error)
type mkPair func(t *testing.T, opts ...sbot.Option) (*sbot.Sbot, mkClient)

func LotsOfStatusCalls(newPair mkPair) func(t *testing.T) {

	return func(t *testing.T) {
		r, a := require.New(t), assert.New(t)

		srv, mkClient := newPair(t,
			// this test needs multiple stable client connections
			// the default, LastWinsTracker disconnects the previous connection
			sbot.WithNetworkConnTracker(network.NewAcceptAllTracker()),
		)
		r.NotNil(srv, "no server from init func")

		ctx, done := context.WithCancel(context.Background())

		g, ctx := errgroup.WithContext(ctx)

		var statusCalls uint32

		n := 25 // spawn n clients
		for i := n; i > 0; i-- {
			fn := func() error {
				tick := time.NewTicker(250 * time.Millisecond)
				for {
					select {
					case <-ctx.Done():
						return nil
					case <-tick.C:
					}

					c, err := mkClient(ctx)
					if err != nil {
						if errors.Cause(err) == context.Canceled {
							return nil
						}
						return errors.Wrapf(err, "tick%p failed", tick)
					}

					var status map[string]interface{}
					err = c.Async(ctx, &status, muxrpc.TypeJSON, muxrpc.Method{"status"})
					if err != nil {
						causeErr := errors.Cause(err)
						if causeErr == context.Canceled || causeErr == muxrpc.ErrSessionTerminated {
							return nil
						}
						return errors.Wrapf(err, "tick%p failed", tick)
					}
					if err := c.Close(); err != nil {
						return errors.Wrapf(err, "tick%p failed close", tick)
					}
					// fmt.Println(resp)
					atomic.AddUint32(&statusCalls, 1)
				}
			}
			g.Go(fn)
		}

		c, err := mkClient(ctx)
		r.NoError(err, "failed to make client connection")

		// check that we can read messages as we create them from the same connection
		var lopt message.CreateLogArgs
		lopt.Live = true
		lopt.Keys = true
		lopt.Limit = -1
		src, err := c.CreateLogStream(lopt)
		r.NoError(err)

		for i := 25; i > 0; i-- {
			time.Sleep(500 * time.Millisecond)
			ref, err := c.Publish(struct {
				Type string `json:"type"`
				Test int
			}{"test", i})
			r.NoError(err, "publish %d errored", i)
			r.NotNil(ref)

			r.True(src.Next(ctx))
			var msg refs.KeyValueRaw
			err = src.Reader(func(r io.Reader) error {
				return json.NewDecoder(r).Decode(&msg)
			})
			r.NoError(err, "message live err %d errored", i)

			a.Equal(msg.Key().Hash, ref.Hash, "wrong message: %d - %s", i, ref.Ref())
		}
		time.Sleep(1 * time.Second)
		a.NoError(c.Close())

		done()
		r.NoError(g.Wait())

		a.GreaterOrEqual(statusCalls, uint32(1000), "expected more status calls")

		v, err := srv.ReceiveLog.Seq().Value()
		r.NoError(err)
		r.EqualValues(24, v)

		srv.Shutdown()
		r.NoError(srv.Close())
	}
}

type testMsg struct {
	Type string `json:"type"`
	Foo  string
	Bar  int
}

func TestPublish(t *testing.T) {
	// defer leakcheck.Check(t)
	r, a := require.New(t), assert.New(t)

	srvRepo := filepath.Join("testrun", t.Name(), "serv")
	os.RemoveAll(srvRepo)

	srvLog := testutils.NewRelativeTimeLogger(nil)

	srv, err := sbot.New(
		sbot.WithInfo(srvLog),
		sbot.WithRepoPath(srvRepo),
		sbot.WithListenAddr(":0"))
	r.NoError(err, "sbot srv init failed")

	var srvErrc = make(chan error, 1)
	go func() {
		err := srv.Network.Serve(context.TODO())
		if err != nil {
			srvErrc <- errors.Wrap(err, "ali serve exited")
		}
		close(srvErrc)
	}()

	kp, err := ssb.LoadKeyPair(filepath.Join(srvRepo, "secret"))
	r.NoError(err, "failed to load servers keypair")
	srvAddr := srv.Network.GetListenAddr()

	c, err := client.NewTCP(kp, srvAddr)
	r.NoError(err, "failed to make client connection")
	// end test boilerplate

	// no messages yet
	seqv, err := srv.ReceiveLog.Seq().Value()
	r.NoError(err, "failed to get root log sequence")
	r.Equal(margaret.SeqEmpty, seqv)

	msg := testMsg{"test", "hello", 23}
	ref, err := c.Publish(msg)
	r.NoError(err, "failed to call publish")
	r.NotNil(ref)

	// get stored message from the log
	seqv, err = srv.ReceiveLog.Seq().Value()
	r.NoError(err, "failed to get root log sequence")
	wantSeq := margaret.BaseSeq(0)
	a.Equal(wantSeq, seqv)
	msgv, err := srv.ReceiveLog.Get(wantSeq)
	r.NoError(err)
	newMsg, ok := msgv.(refs.Message)
	r.True(ok)
	r.Equal(newMsg.Key(), ref)

	opts := message.CreateLogArgs{}
	opts.Limit = 1
	opts.Keys = true
	src, err := c.CreateLogStream(opts)
	r.NoError(err)

	r.True(src.Next(context.TODO()))
	var streamMsg refs.KeyValueRaw
	err = src.Reader(func(r io.Reader) error {
		return json.NewDecoder(r).Decode(&streamMsg)
	})
	r.NoError(err)

	a.Equal(newMsg.Author().Ref(), streamMsg.Author().Ref())
	a.EqualValues(newMsg.Seq(), streamMsg.Seq())

	r.False(src.Next(context.TODO()))

	a.NoError(c.Close())

	srv.Shutdown()
	r.NoError(srv.Close())
	r.NoError(<-srvErrc)
}

func TestTangles(t *testing.T) {
	// defer leakcheck.Check(t)
	r, a := require.New(t), assert.New(t)

	srvRepo := filepath.Join("testrun", t.Name(), "serv")
	os.RemoveAll(srvRepo)
	srvLog := testutils.NewRelativeTimeLogger(nil)

	srv, err := sbot.New(
		sbot.WithInfo(srvLog),
		sbot.WithRepoPath(srvRepo),
		sbot.WithListenAddr(":0"),
	)
	r.NoError(err, "sbot srv init failed")

	var srvErrc = make(chan error, 1)
	go func() {
		err := srv.Network.Serve(context.TODO())
		if err != nil {
			srvErrc <- errors.Wrap(err, "ali serve exited")
		}
		close(srvErrc)
	}()

	kp, err := ssb.LoadKeyPair(filepath.Join(srvRepo, "secret"))
	r.NoError(err, "failed to load servers keypair")
	srvAddr := srv.Network.GetListenAddr()

	c, err := client.NewTCP(kp, srvAddr)
	r.NoError(err, "failed to make client connection")
	// end test boilerplate

	type testMsg struct {
		Type string `json:"type"`
		Foo  string
		Bar  int
		Root *refs.MessageRef `json:"root,omitempty"`
	}
	msg := testMsg{"test", "hello", 23, nil}
	rootRef, err := c.Publish(msg)
	r.NoError(err, "failed to call publish")
	r.NotNil(rootRef)

	rep1 := testMsg{"test", "reply", 1, rootRef}
	rep1Ref, err := c.Publish(rep1)
	r.NoError(err, "failed to call publish")
	r.NotNil(rep1Ref)
	rep2 := testMsg{"test", "reply", 2, rootRef}
	rep2Ref, err := c.Publish(rep2)
	r.NoError(err, "failed to call publish")
	r.NotNil(rep2Ref)

	opts := message.TanglesArgs{}
	opts.Root = rootRef
	opts.Limit = 2
	opts.Keys = true
	src, err := c.Tangles(opts)
	r.NoError(err)

	ctx := context.TODO()
	r.True(src.Next(ctx))
	var streamMsg refs.KeyValueRaw
	err = src.Reader(func(r io.Reader) error {
		return json.NewDecoder(r).Decode(&streamMsg)
	})
	r.NoError(err)
	a.EqualValues(2, streamMsg.Seq())

	r.True(src.Next(ctx))
	err = src.Reader(func(r io.Reader) error {
		return json.NewDecoder(r).Decode(&streamMsg)
	})
	r.NoError(err)
	a.EqualValues(3, streamMsg.Seq())

	r.False(src.Next(ctx))
	r.NoError(src.Err())

	a.NoError(c.Close())

	srv.Shutdown()
	r.NoError(srv.Close())
	r.NoError(<-srvErrc)
}
