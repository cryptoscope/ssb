package private

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"go.cryptoscope.co/librarian"
	"go.cryptoscope.co/margaret"
	"go.cryptoscope.co/ssb/private/box2"

	"go.cryptoscope.co/ssb/keys"
	refs "go.mindeco.de/ssb-refs"
)

type groupInit struct {
	Type string `json:"type"`
	Name string `json:"name"`

	Tangles refs.Tangles `json:"tangles"`
}

// returns cloaked id and public root
func (mgr *Manager) Init(name string) (*refs.MessageRef, *refs.MessageRef, error) {
	// roll new key
	var r keys.Recipient
	r.Scheme = keys.SchemeLargeSymmetricGroup
	r.Key = make([]byte, 32) // TODO: key size const
	_, err := rand.Read(r.Key)
	if err != nil {
		return nil, nil, err
	}

	// prepare init content
	var gi groupInit
	gi.Type = "group/init"
	gi.Name = name
	gi.Tangles = make(refs.Tangles)
	gi.Tangles["group"] = refs.TanglePoint{}   // empty/root
	gi.Tangles["members"] = refs.TanglePoint{} // empty/root

	jsonContent, err := json.Marshal(gi)
	if err != nil {
		return nil, nil, err
	}

	// encrypt the group/init message

	publicRoot, err := mgr.encryptAndPublish(jsonContent, keys.Recipients{r})
	if err != nil {
		return nil, nil, err
	}

	var cloakedID refs.MessageRef
	cloakedID.Algo = refs.RefAlgoCloakedGroup
	cloakedID.Hash = make([]byte, 32)
	err = box2.DeriveTo(cloakedID.Hash, r.Key, []byte("cloaked_msg_id"), publicRoot.Hash)
	if err != nil {
		return nil, nil, err
	}

	err = mgr.keymgr.AddKey(r.Scheme, cloakedID.Hash, r.Key)
	if err != nil {
		return nil, nil, err
	}

	// my keys
	err = mgr.keymgr.AddKey(r.Scheme, sortAndConcat(mgr.author.ID, mgr.author.ID), r.Key)
	if err != nil {
		return nil, nil, err
	}

	return &cloakedID, publicRoot, nil
}

const exampleAddMemberContent = `ex:
var content = {
	type: 'group/add-member',
	version: 'v1',
	groupKey: '3YUat1ylIUVGaCjotAvof09DhyFxE8iGbF6QxLlCWWc=',
	initialMsg: '%THxjTGPuXvvxnbnAV7xVuVXdhDcmoNtDDN0j3UTxcd8=.sha256',
	text: 'welcome keks!',                                      // optional
	recps: [
	  '%vof09Dhy3YUat1ylIUVGaCjotAFxE8iGbF6QxLlCWWc=.cloaked',  // group_id
	  '@YXkE3TikkY4GFMX3lzXUllRkNTbj5E+604AkaO1xbz8=.ed25519'   // feed_id (for new person)
	],
  
	tangles: {
	  group: {
		root: '%THxjTGPuXvvxnbnAV7xVuVXdhDcmoNtDDN0j3UTxcd8=.sha256',
		previous: [
		  '%Sp294oBk7OJxizvPOlm6Sqk3fFJA2EQFiyJ1MS/BZ9E=.sha256'
		]
	  },
	  members: {
		root: '%THxjTGPuXvvxnbnAV7xVuVXdhDcmoNtDDN0j3UTxcd8=.sha256',
		previous: [
		  '%lm6Sqk3fFJA2EQFiyJ1MSASDASDASDASDASDAS/BZ9E=.sha256',
		  '%Sp294oBk7OJxizvPOlm6Sqk3fFJA2EQFiyJ1MS/BZ9E=.sha256'
		]
	  }
	}
  }`

type groupAddMember struct {
	Type string `json:"type"`
	Text string `json:"text"`

	Version string `json:"version"`

	GroupKey       keys.Base64String `json:"groupKey"`
	InitialMessage *refs.MessageRef  `json:"initialMsg"`

	Recps []refs.Ref `json:"recps"`

	Tangles refs.Tangles `json:"tangles"`
}

func (mgr *Manager) AddMember(groupID *refs.MessageRef, r *refs.FeedRef, welcome string) (*refs.MessageRef, error) {
	if groupID.Algo != refs.RefAlgoCloakedGroup {
		return nil, fmt.Errorf("not a group")
	}

	gskey, err := mgr.keymgr.GetKeys(keys.SchemeLargeSymmetricGroup, groupID.Hash)
	if err != nil {
		return nil, err
	}

	if n := len(gskey); n != 1 {
		return nil, fmt.Errorf("inconsistent group-key count: %d", n)
	}

	groupRoot, err := mgr.getGroupRoot(groupID)
	if err != nil {
		return nil, err
	}

	// prepare init content
	var ga groupAddMember
	ga.Type = "group/add-member"
	ga.Version = "v1"
	ga.Text = welcome

	ga.GroupKey = keys.Base64String(gskey[0].Key)
	ga.InitialMessage = groupRoot

	ga.Recps = []refs.Ref{groupID, r}

	ga.Tangles = make(refs.Tangles)

	ga.Tangles["group"] = mgr.getTangleState(groupRoot, "group")
	ga.Tangles["members"] = mgr.getTangleState(groupRoot, "members")

	jsonContent, err := json.Marshal(ga)
	if err != nil {
		return nil, err
	}

	return mgr.encryptAndPublish(jsonContent, gskey)
}

func (mgr *Manager) PublishTo(groupID *refs.MessageRef, content []byte) (*refs.MessageRef, error) {
	if groupID.Algo != refs.RefAlgoCloakedGroup {
		return nil, fmt.Errorf("not a group")
	}
	r, err := mgr.keymgr.GetKeys(keys.SchemeLargeSymmetricGroup, groupID.Hash)
	if err != nil {
		return nil, err
	}

	return mgr.encryptAndPublish(content, r)
}

func (mgr *Manager) PublishPostTo(groupID *refs.MessageRef, text string) (*refs.MessageRef, error) {
	if groupID.Algo != refs.RefAlgoCloakedGroup {
		return nil, fmt.Errorf("not a group")
	}
	r, err := mgr.keymgr.GetKeys(keys.SchemeLargeSymmetricGroup, groupID.Hash)
	if err != nil {
		return nil, err
	}

	var p refs.Post
	p.Type = "post"
	p.Text = text
	p.Recps = refs.MessageRefs{groupID}
	p.Tangles = make(refs.Tangles)

	groupRoot, err := mgr.getGroupRoot(groupID)
	if err != nil {
		return nil, err
	}
	p.Tangles["group"] = mgr.getTangleState(groupRoot, "group")

	content, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return mgr.encryptAndPublish(content, r)
}

// fake hacks BIG TODO

func (mgr *Manager) getGroupRoot(groupID *refs.MessageRef) (*refs.MessageRef, error) {
	if groupID.Algo != refs.RefAlgoCloakedGroup {
		return nil, fmt.Errorf("not a group")
	}

	return &refs.MessageRef{Algo: "fake", Hash: bytes.Repeat([]byte("1312"), 8)}, nil
}

func (mgr *Manager) getTangleState(root *refs.MessageRef, tname string) refs.TanglePoint {
	addr := librarian.Addr(append(root.Hash, []byte(tname)...))
	thandle, err := mgr.tangles.Get(addr)
	if err != nil {
		return refs.TanglePoint{Root: root, Previous: []*refs.MessageRef{root}}
	}

	prev := getLooseEnds(thandle)
	// TODO!!!
	return refs.TanglePoint{Root: root, Previous: append(prev, root)}
}

func getLooseEnds(_ margaret.Log) refs.MessageRefs {
	return []*refs.MessageRef{
		{Algo: "fake", Hash: bytes.Repeat([]byte("acab"), 8)},
	}
}

// utils

// TODO: protect against race of changing previous
// mgr.publog.Lock()
func (mgr *Manager) encryptAndPublish(c []byte, recps keys.Recipients) (*refs.MessageRef, error) {
	fmt.Printf("encryptig :%s\n", string(c))

	prev, err := mgr.getPrevious()
	if err != nil {
		return nil, err
	}
	// now create the ciphertext
	bxr := box2.NewBoxer(mgr.rand)
	ciphertext, err := bxr.Encrypt(c, mgr.author, prev, recps)
	if err != nil {
		return nil, err
	}

	return mgr.publishCiphertext(ciphertext)
}

func (mgr *Manager) getPrevious() (*refs.MessageRef, error) {
	// get previous
	currSeqV, err := mgr.publog.Seq().Value()
	if err != nil {
		return nil, err
	}
	currSeq := currSeqV.(margaret.Seq)
	msgV, err := mgr.publog.Get(currSeq)
	if err != nil {
		return nil, err
	}
	msg := msgV.(refs.Message)
	prev := msg.Key()
	return prev, nil
}

func (mgr *Manager) publishCiphertext(ctxt []byte) (*refs.MessageRef, error) {
	// TODO: format check for gabbygrove
	content := base64.StdEncoding.EncodeToString(ctxt) + ".box2"

	r, err := mgr.publog.Publish(content)
	if err != nil {
		return nil, err
	}
	return r, nil
}
