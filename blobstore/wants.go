package blobstore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/cryptix/go/logging"
	"github.com/pkg/errors"
	"github.com/shurcooL/go-goon"

	"go.cryptoscope.co/luigi"
	"go.cryptoscope.co/muxrpc"
	"go.cryptoscope.co/sbot"
)

var log logging.Interface

func dump(v interface{}) {
	if msg, ok := v.(WantMsg); ok {
		v = &msg
	}

	if msg, ok := v.(*WantMsg); ok {
		m := make(map[string]int64)
		for _, w := range *msg {
			m[w.Ref.Ref()] = w.Dist
		}
		v = m
	}

	goon.Dump(v)
}

func init() {
	log = logging.Logger("blobstore")
}

func NewWantManager(bs sbot.BlobStore) sbot.WantManager {
	wmgr := &wantManager{
		bs:    bs,
		wants: make(map[string]int64),
	}

	wmgr.wantSink, wmgr.Broadcast = luigi.NewBroadcast()

	bs.Changes().Register(luigi.FuncSink(func(ctx context.Context, v interface{}, doClose bool) error {
		wmgr.l.Lock()
		defer wmgr.l.Unlock()

		n, ok := v.(sbot.BlobStoreNotification)
		if ok && n.Op == sbot.BlobStoreOpPut {
			if _, ok := wmgr.wants[n.Ref.Ref()]; ok {
				delete(wmgr.wants, n.Ref.Ref())
			}
		}

		return nil
	}))

	return wmgr
}

type wantManager struct {
	luigi.Broadcast

	bs sbot.BlobStore

	wants    map[string]int64
	wantSink luigi.Sink

	l sync.Mutex
}

func (wmgr *wantManager) Wants(ref *sbot.BlobRef) bool {
	wmgr.l.Lock()
	defer wmgr.l.Unlock()

	_, ok := wmgr.wants[ref.Ref()]
	return ok
}

func (wmgr *wantManager) Want(ref *sbot.BlobRef) error {
	return wmgr.WantWithDist(ref, -1)
}

func (wmgr *wantManager) WantWithDist(ref *sbot.BlobRef, dist int64) error {
	f, err := wmgr.bs.Get(ref)
	if err == nil {
		return f.(io.Closer).Close()
	}

	wmgr.l.Lock()
	defer wmgr.l.Unlock()

	wmgr.wants[ref.Ref()] = dist

	err = wmgr.wantSink.Pour(context.TODO(), want{ref, dist})
	err = errors.Wrap(err, "error pouring want to broadcast")
	return err
}

func (wmgr *wantManager) CreateWants(ctx context.Context, sink luigi.Sink, edp muxrpc.Endpoint) luigi.Sink {
	proc := &wantProc{
		bs:          wmgr.bs,
		wmgr:        wmgr,
		out:         sink,
		remoteWants: make(map[string]int64),
		edp:         edp,
	}

	proc.init()

	return proc
}

type want struct {
	Ref *sbot.BlobRef

	// if Dist is negative, it is the hop count to the original wanter.
	// if it is positive, it is the size of the blob.
	Dist int64
}

type wantProc struct {
	l sync.Mutex

	bs          sbot.BlobStore
	wmgr        *wantManager
	out         luigi.Sink
	remoteWants map[string]int64
	done        func(func())
	edp         muxrpc.Endpoint
}

func (proc *wantProc) init() {
	cancel := proc.bs.Changes().Register(
		luigi.FuncSink(func(ctx context.Context, v interface{}, doClose bool) error {
			proc.l.Lock()
			defer proc.l.Unlock()

			notif := v.(sbot.BlobStoreNotification)
			fmt.Println("wantProc received notification from blobstore:", notif.Op, notif.Ref.Ref())
			_, ok := proc.remoteWants[notif.Ref.Ref()]
			if ok {
				sz, err := proc.bs.Size(notif.Ref)
				if err != nil {
					return errors.Wrap(err, "error getting blob size")
				}

				m := map[string]int64{notif.Ref.Ref(): sz}
				err = proc.out.Pour(ctx, m)
				log.Log("event", "createWants.Out", "cause", "changesnotification")
				dump(m)
				return errors.Wrap(err, "errors pouring into sink")
			}

			return nil
		}))

	oldDone := proc.done
	proc.done = func(next func()) {
		cancel()
		if oldDone != nil {
			oldDone(nil)
		}
	}

	err := proc.out.Pour(context.TODO(), proc.wmgr.wants)
	log.Log("event", "createWants.Out", "cause", "initial wants")
	dump(proc.wmgr.wants)
	if err != nil {
		log.Log("event", "wantProc.init/Pour", "err", err.Error())
	}
}

func (proc *wantProc) Close() error {
	defer proc.done(nil)
	return errors.Wrap(proc.out.Close(), "error in lower-layer close")
}

func (proc *wantProc) Pour(ctx context.Context, v interface{}) error {
	log.Log("event", "createWants.In", "cause", "got called")
	dump(v)
	proc.l.Lock()
	defer proc.l.Unlock()

	mIn := v.(*WantMsg)
	mOut := make(map[string]int64)

	for _, w := range *mIn {
		if w.Dist < 0 {
			s, err := proc.bs.Size(w.Ref)
			if err != nil {
				if err == ErrNoSuchBlob {
					proc.remoteWants[w.Ref.Ref()] = w.Dist - 1
					continue
				}

				return errors.Wrap(err, "error getting blob size")
			}

			delete(proc.remoteWants, w.Ref.Ref())
			mOut[w.Ref.Ref()] = s
		} else {
			if proc.wmgr.Wants(w.Ref) {
				go func(ref *sbot.BlobRef) {
					src, err := proc.edp.Source(ctx, &WantMsg{}, muxrpc.Method{"blobs", "get"}, ref.Ref())
					if err != nil {
						log.Log("event", "blob fetch err", "ref", ref.Ref(), "error", err.Error())
						return
					}

					r := muxrpc.NewSourceReader(src)
					newBr, err := proc.bs.Put(r)
					if err != nil {
						log.Log("event", "blob fetch err", "ref", ref.Ref(), "error", err.Error())
						return
					}

					if newBr.Ref() != ref.Ref() {
						log.Log("event", "blob fetch err", "actualRef", newBr.Ref(), "expectedRef", ref.Ref(), "error", "ref did not match expected ref")
						return
					}
				}(w.Ref)
			}
		}
	}

	return errors.Wrap(proc.out.Pour(ctx, mOut), "error responding to wants")
}

type WantMsg []want

func (msg *WantMsg) UnmarshalJSON(data []byte) error {
	r := bytes.NewBuffer(data)
	dec := json.NewDecoder(r)

	type pState int
	const (
		begin pState = iota
		expRef
		expDist
		expEnd
		expEOF
	)

	var (
		state pState
		next  want
	)

	for {
		tok, err := dec.Token()
		if err != nil {
			if err == io.EOF {
				if state == expEOF {
					return nil
				}

				return errors.New("unexpected end of file")
			}

			return err
		}

		if state == expEOF {
			return errors.New("expected EOF but read data")
		}

		switch state {
		case begin:
			del, ok := tok.(json.Delim)
			if !ok {
				return errors.Errorf("expected delimiter { but got %v", tok)
			}
			if del.String() != "{" {
				return errors.Errorf("expected delimiter { but got %v", tok)
			}

			state = expRef
		case expRef:
			str, ok := tok.(string)
			if !ok {
				return errors.Errorf("expected blob reference string but got %v", tok)
			}

			ref, err := sbot.ParseRef(str)
			if err != nil {
				return errors.Wrap(err, "error parsing blob reference")
			}

			br, ok := ref.(*sbot.BlobRef)
			if !ok {
				return errors.Errorf("expected *sbot.BlobRef but got %T", ref)
			}

			next.Ref = br
			state = expDist
		case expDist:
			fDist, ok := tok.(float64)
			if !ok {
				return errors.Errorf("expected blob distance float64 but got %v", tok)
			}

			next.Dist = int64(fDist)
			*msg = append(*msg, next)

			if dec.More() {
				state = expRef
			} else {
				state = expEnd
			}
		case expEnd:
			del, ok := tok.(json.Delim)
			if !ok {
				return errors.Errorf("expected delimiter } but got %v", tok)
			}
			if del.String() != "}" {
				return errors.Errorf("expected delimiter } but got %v", tok)
			}

			state = expEOF
		default:
			panic("undefined state in blobs.wantMsg.UnmarshalJSON with data\n" + string(data))
		}
	}

	return nil
}
